package exit

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/proto"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/dns/dnsmessage"
)

var (
	dnsPort        = 53
	dnsQueryBudget = 3 * time.Second
	dnsCacheSize   = 1000
	dnsCacheMaxTTL = 300 * time.Second
	dnsTCPIdle     = 10 * time.Second
	dnsMaxMessage  = 4096
)

// Forwarder is the in-process DNS forwarder of D3: bound to the gadget
// address on UDP and TCP 53, it forwards every query over TCP through the
// slot's SOCKS front door to pinned public resolvers, never consults
// /etc/resolv.conf, answers SERVFAIL when no exit is connected and answers
// AAAA with an empty NOERROR while the slot carries no IPv6 (D22).
type Forwarder struct {
	slot      Slot
	upstreams func() []netip.Addr
	connected func() bool

	// opMu serialises Bind against Stop (and Bind against Bind): without it a
	// Stop could Wait on the WaitGroup while a concurrent Bind then Add(2)s to
	// it (the documented misuse that panics), or a Bind could finish after a
	// Stop and leave :53 bound on a forwarder nobody references (MGR-10). mu
	// guards only the published listeners.
	opMu sync.Mutex
	mu   sync.Mutex
	udp  net.PacketConn
	tcp  net.Listener
	addr netip.Addr
	wg   sync.WaitGroup

	queries  atomic.Uint64
	failures atomic.Uint64

	cache *dnsCache
}

// NewForwarder builds a forwarder; nothing listens until Bind.
func NewForwarder(slot Slot, upstreams func() []netip.Addr, connected func() bool) *Forwarder {
	if connected == nil {
		connected = func() bool { return false }
	}
	if upstreams == nil {
		upstreams = func() []netip.Addr { return nil }
	}
	return &Forwarder{slot: slot, upstreams: upstreams, connected: connected, cache: newDNSCache()}
}

// Bind (re)binds UDP and TCP :53 on addr. Any failure leaves nothing bound
// and surfaces as downstream.dns=false.
func (f *Forwarder) Bind(addr netip.Addr) error {
	f.opMu.Lock()
	defer f.opMu.Unlock()
	f.stopLocked()
	hostport := net.JoinHostPort(addr.String(), strconv.Itoa(dnsPort))
	network := "udp4"
	if addr.Is6() {
		network = "udp6"
	}
	udp, err := listenPacket(network, hostport)
	if err != nil {
		return fmt.Errorf("%s dns udp: %w", f.slot.Name(), err)
	}
	tcp, err := listenTCP(hostport)
	if err != nil {
		_ = udp.Close()
		return fmt.Errorf("%s dns tcp: %w", f.slot.Name(), err)
	}
	// Add before publishing and before the goroutines: opMu keeps this the
	// only Add, so a Stop's Wait can never race it.
	f.wg.Add(2)
	f.mu.Lock()
	f.udp, f.tcp, f.addr = udp, tcp, addr
	f.mu.Unlock()
	go f.serveUDP(udp)
	go f.serveTCP(tcp)
	return nil
}

// Stop closes both listeners.
func (f *Forwarder) Stop() {
	f.opMu.Lock()
	defer f.opMu.Unlock()
	f.stopLocked()
}

// stopLocked closes both listeners and waits for their goroutines. opMu must
// be held, so it never runs beside a Bind.
func (f *Forwarder) stopLocked() {
	f.mu.Lock()
	udp, tcp := f.udp, f.tcp
	f.udp, f.tcp = nil, nil
	f.mu.Unlock()
	if udp != nil {
		_ = udp.Close()
	}
	if tcp != nil {
		_ = tcp.Close()
	}
	f.wg.Wait()
}

// Bound reports whether both listeners are up.
func (f *Forwarder) Bound() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.udp != nil && f.tcp != nil
}

// Addr is the bound address; the zero Addr when unbound.
func (f *Forwarder) Addr() netip.Addr {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.udp == nil {
		return netip.Addr{}
	}
	return f.addr
}

// boundAddrs are the two listeners' host:port strings ("" when unbound).
// Tests bind port 0 and need them; production binds a fixed :53.
func (f *Forwarder) boundAddrs() (udp, tcp string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.udp != nil {
		udp = f.udp.LocalAddr().String()
	}
	if f.tcp != nil {
		tcp = f.tcp.Addr().String()
	}
	return udp, tcp
}

// Stats are the forwarder's counters. Redirected is a conntrack figure the
// manager fills in; the forwarder cannot tell a DNAT'd query from a direct one.
func (f *Forwarder) Stats() proto.ExitDNSStats {
	return proto.ExitDNSStats{Queries: f.queries.Load(), Failures: f.failures.Load()}
}

func (f *Forwarder) serveUDP(pc net.PacketConn) {
	defer f.wg.Done()
	buf := make([]byte, dnsMaxMessage)
	for {
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}
		q := make([]byte, n)
		copy(q, buf[:n])
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), dnsQueryBudget)
			defer cancel()
			resp := f.answer(ctx, q, true)
			if resp != nil {
				_, _ = pc.WriteTo(resp, from)
			}
		}()
	}
}

func (f *Forwarder) serveTCP(ln net.Listener) {
	defer f.wg.Done()
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			defer c.Close()
			for {
				_ = c.SetReadDeadline(time.Now().Add(dnsTCPIdle))
				q, err := readTCPMessage(c)
				if err != nil {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), dnsQueryBudget)
				resp := f.answer(ctx, q, false)
				cancel()
				if resp == nil {
					return
				}
				_ = c.SetWriteDeadline(time.Now().Add(dnsTCPIdle))
				if err := writeTCPMessage(c, resp); err != nil {
					return
				}
			}
		}()
	}
}

func readTCPMessage(r io.Reader) ([]byte, error) {
	var l [2]byte
	if _, err := io.ReadFull(r, l[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint16(l[:]))
	if n == 0 || n > 65535 {
		return nil, errors.New("dns: bad length")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

func writeTCPMessage(w io.Writer, msg []byte) error {
	out := make([]byte, 2+len(msg))
	binary.BigEndian.PutUint16(out, uint16(len(msg)))
	copy(out[2:], msg)
	_, err := w.Write(out)
	return err
}

// answer produces the response for one query, or nil when the query is not
// even a DNS header. udp asks for truncation to the client's payload size.
func (f *Forwarder) answer(ctx context.Context, q []byte, udp bool) []byte {
	var p dnsmessage.Parser
	hdr, err := p.Start(q)
	if err != nil {
		return nil
	}
	if hdr.Response {
		return nil
	}
	f.queries.Add(1)
	question, err := p.Question()
	if err != nil {
		return buildResponse(hdr, nil, dnsmessage.RCodeFormatError, nil, false)
	}
	maxSize := 512
	if udp {
		maxSize = ednsPayloadSize(&p, maxSize)
	}

	if question.Type == dnsmessage.TypeAAAA {
		return buildResponse(hdr, &question, dnsmessage.RCodeSuccess, nil, false)
	}

	if answers, ok := f.cache.get(question); ok {
		return f.finish(buildResponse(hdr, &question, dnsmessage.RCodeSuccess, answers, false), hdr, question, udp, maxSize)
	}

	if !f.connected() {
		f.failures.Add(1)
		return buildResponse(hdr, &question, dnsmessage.RCodeServerFailure, nil, false)
	}

	resp, err := f.forward(ctx, q)
	if err != nil {
		f.failures.Add(1)
		log.Debugf("%s: dns %s %s: %s", f.slot.Name(), question.Type, question.Name, err)
		return buildResponse(hdr, &question, dnsmessage.RCodeServerFailure, nil, false)
	}

	var m dnsmessage.Message
	if err := m.Unpack(resp); err == nil {
		if m.RCode == dnsmessage.RCodeSuccess && len(m.Answers) > 0 {
			f.cache.put(question, m.Answers)
		}
		if m.Header.ID != hdr.ID {
			m.Header.ID = hdr.ID
			if fixed, err := m.Pack(); err == nil {
				resp = fixed
			}
		}
	}
	return f.finish(resp, hdr, question, udp, maxSize)
}

// finish truncates a UDP response the client cannot take.
func (f *Forwarder) finish(resp []byte, hdr dnsmessage.Header, q dnsmessage.Question, udp bool, maxSize int) []byte {
	if udp && len(resp) > maxSize {
		return buildResponse(hdr, &q, dnsmessage.RCodeSuccess, nil, true)
	}
	return resp
}

// forward tries each upstream in order, over TCP through the front door,
// within ctx's budget.
func (f *Forwarder) forward(ctx context.Context, q []byte) ([]byte, error) {
	ups := f.upstreams()
	if len(ups) == 0 {
		return nil, errors.New("no upstream resolvers")
	}
	var lastErr error
	for _, up := range ups {
		if ctx.Err() != nil {
			break
		}
		resp, err := f.exchange(ctx, netip.AddrPortFrom(up, 53), q)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = ctx.Err()
	}
	return nil, lastErr
}

func (f *Forwarder) exchange(ctx context.Context, upstream netip.AddrPort, q []byte) ([]byte, error) {
	c, err := dialThroughFrontDoor(ctx, f.slot, upstream)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetDeadline(dl)
	}
	if err := writeTCPMessage(c, q); err != nil {
		return nil, err
	}
	resp, err := readTCPMessage(c)
	if err != nil {
		return nil, err
	}
	if len(resp) < 12 || binary.BigEndian.Uint16(resp) != binary.BigEndian.Uint16(q) {
		return nil, errors.New("dns: response id mismatch")
	}
	return resp, nil
}

// ednsPayloadSize reads the client's OPT UDP payload size when present.
func ednsPayloadSize(p *dnsmessage.Parser, def int) int {
	if err := p.SkipAllQuestions(); err != nil {
		return def
	}
	if err := p.SkipAllAnswers(); err != nil {
		return def
	}
	if err := p.SkipAllAuthorities(); err != nil {
		return def
	}
	for {
		h, err := p.AdditionalHeader()
		if err != nil {
			return def
		}
		if h.Type == dnsmessage.TypeOPT {
			size := int(h.Class)
			if size < 512 {
				size = 512
			}
			if size > dnsMaxMessage {
				size = dnsMaxMessage
			}
			return size
		}
		if err := p.SkipAdditional(); err != nil {
			return def
		}
	}
}

// buildResponse renders a reply to hdr's query with the given answers.
func buildResponse(hdr dnsmessage.Header, q *dnsmessage.Question, rcode dnsmessage.RCode, answers []dnsmessage.Resource, truncated bool) []byte {
	out := dnsmessage.Header{
		ID:                 hdr.ID,
		Response:           true,
		OpCode:             hdr.OpCode,
		RecursionDesired:   hdr.RecursionDesired,
		RecursionAvailable: true,
		Truncated:          truncated,
		RCode:              rcode,
	}
	b := dnsmessage.NewBuilder(make([]byte, 0, 512), out)
	b.EnableCompression()
	if q != nil {
		if err := b.StartQuestions(); err != nil {
			return nil
		}
		if err := b.Question(*q); err != nil {
			return nil
		}
	}
	if len(answers) > 0 {
		if err := b.StartAnswers(); err != nil {
			return nil
		}
		for _, r := range answers {
			if err := appendResource(&b, r); err != nil {
				return nil
			}
		}
	}
	msg, err := b.Finish()
	if err != nil {
		return nil
	}
	return msg
}

func appendResource(b *dnsmessage.Builder, r dnsmessage.Resource) error {
	switch body := r.Body.(type) {
	case *dnsmessage.AResource:
		return b.AResource(r.Header, *body)
	case *dnsmessage.AAAAResource:
		return b.AAAAResource(r.Header, *body)
	case *dnsmessage.CNAMEResource:
		return b.CNAMEResource(r.Header, *body)
	case *dnsmessage.NSResource:
		return b.NSResource(r.Header, *body)
	case *dnsmessage.PTRResource:
		return b.PTRResource(r.Header, *body)
	case *dnsmessage.MXResource:
		return b.MXResource(r.Header, *body)
	case *dnsmessage.TXTResource:
		return b.TXTResource(r.Header, *body)
	case *dnsmessage.SOAResource:
		return b.SOAResource(r.Header, *body)
	case *dnsmessage.SRVResource:
		return b.SRVResource(r.Header, *body)
	case *dnsmessage.UnknownResource:
		return b.UnknownResource(r.Header, *body)
	default:
		return fmt.Errorf("dns: unsupported record type %T", r.Body)
	}
}

// dnsCache is the 1 000-entry positive cache honouring TTL, capped at 300 s.
type dnsCache struct {
	mu      sync.Mutex
	entries map[dnsCacheKey]dnsCacheEntry
}

type dnsCacheKey struct {
	name  string
	typ   dnsmessage.Type
	class dnsmessage.Class
}

type dnsCacheEntry struct {
	answers []dnsmessage.Resource
	expires time.Time
	stored  time.Time
}

func newDNSCache() *dnsCache {
	return &dnsCache{entries: make(map[dnsCacheKey]dnsCacheEntry)}
}

func keyOf(q dnsmessage.Question) dnsCacheKey {
	return dnsCacheKey{name: strings.ToLower(q.Name.String()), typ: q.Type, class: q.Class}
}

func (c *dnsCache) get(q dnsmessage.Question) ([]dnsmessage.Resource, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[keyOf(q)]
	if !ok {
		return nil, false
	}
	now := time.Now()
	if !now.Before(e.expires) {
		delete(c.entries, keyOf(q))
		return nil, false
	}
	elapsed := uint32(now.Sub(e.stored) / time.Second)
	out := make([]dnsmessage.Resource, len(e.answers))
	for i, r := range e.answers {
		out[i] = r
		if r.Header.TTL > elapsed {
			out[i].Header.TTL = r.Header.TTL - elapsed
		} else {
			out[i].Header.TTL = 1
		}
	}
	return out, true
}

func (c *dnsCache) put(q dnsmessage.Question, answers []dnsmessage.Resource) {
	minTTL := uint32(dnsCacheMaxTTL / time.Second)
	for _, r := range answers {
		if r.Header.TTL == 0 {
			return
		}
		if r.Header.TTL < minTTL {
			minTTL = r.Header.TTL
		}
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= dnsCacheSize {
		for k, e := range c.entries {
			if !now.Before(e.expires) {
				delete(c.entries, k)
			}
		}
		for k := range c.entries {
			if len(c.entries) < dnsCacheSize {
				break
			}
			delete(c.entries, k)
		}
	}
	c.entries[keyOf(q)] = dnsCacheEntry{
		answers: answers,
		expires: now.Add(time.Duration(minTTL) * time.Second),
		stored:  now,
	}
}

func (c *dnsCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
