package exit

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

// Address and listener indirections so tests can run on ephemeral ports and
// prove nothing binds 0.0.0.0. Production values come from Slot.
var (
	socksListenAddr     = func(s Slot) string { return s.SocksAddr() }
	socksDialAddr       = func(s Slot) string { return s.SocksAddr() }
	wstunnelReverseAddr = func(s Slot) string { return s.WstunnelReverseAddr() }
	listenTCP           = utils.Listen
	listenPacket        = func(network, addr string) (net.PacketConn, error) { return net.ListenPacket(network, addr) }
	frontDoorHandshake  = FrontDoorHandshake
	relayDialTimeout    = 3 * time.Second
)

const (
	socksVersion      byte = 0x05
	socksNoAuth       byte = 0x00
	socksNoAcceptable byte = 0xff
	socksCmdConnect   byte = 0x01
	socksCmdUDPAssoc  byte = 0x03
)

// FrontDoor is the loopback SOCKS5 listener hev-socks5-tunnel dials (D1). It
// dispatches CONNECT and UDP ASSOCIATE to the native Backend (Mode A) or
// relays the SOCKS bytes verbatim into wstunnel's reverse listener (Mode B),
// applies the destination policy first, and answers 0x03 in under a
// millisecond when no exit is attached.
type FrontDoor struct {
	slot   Slot
	policy func() Policy
	bytes  ByteCounter

	mu      sync.Mutex
	ln      net.Listener
	native  Backend
	relay   RelayGate
	conns   map[net.Conn]struct{}
	stopped bool
	wg      sync.WaitGroup
}

// NewFrontDoor builds the front door; nothing listens until Start.
func NewFrontDoor(slot Slot, policy func() Policy, bytes ByteCounter) *FrontDoor {
	if policy == nil {
		policy = func() Policy { return Policy{} }
	}
	return &FrontDoor{slot: slot, policy: policy, bytes: bytes, conns: make(map[net.Conn]struct{})}
}

// Start binds the slot's SOCKS address. A taken port is an error the enable
// transaction treats as fatal (D23).
func (f *FrontDoor) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ln != nil {
		return nil
	}
	ln, err := listenTCP(socksListenAddr(f.slot))
	if err != nil {
		return fmt.Errorf("%s front door: %w", f.slot.Name(), err)
	}
	f.ln = ln
	f.stopped = false
	f.wg.Add(1)
	go f.accept(ln)
	return nil
}

// Addr is the bound address, "" before Start.
func (f *FrontDoor) Addr() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ln == nil {
		return ""
	}
	return f.ln.Addr().String()
}

// Stop closes the listener and every hev connection and waits for them.
func (f *FrontDoor) Stop() {
	f.mu.Lock()
	f.stopped = true
	ln := f.ln
	f.ln = nil
	conns := make([]net.Conn, 0, len(f.conns))
	for c := range f.conns {
		conns = append(conns, c)
	}
	f.mu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}
	for _, c := range conns {
		_ = c.Close()
	}
	f.wg.Wait()
}

// SetNative attaches (or with nil detaches) the Mode A backend.
func (f *FrontDoor) SetNative(b Backend) {
	f.mu.Lock()
	f.native = b
	f.mu.Unlock()
}

// SetRelay attaches (or with nil detaches) the Mode B gate.
func (f *FrontDoor) SetRelay(gate RelayGate) {
	f.mu.Lock()
	f.relay = gate
	f.mu.Unlock()
}

// Attached reports the tunnel state as the front door sees it: connected
// with a live native session or a relay gate that has proven its listener,
// connecting while a Mode B peer is tracked but not yet proven, otherwise
// disconnected.
func (f *FrontDoor) Attached() (proto.ExitTunnelState, *proto.ExitPeer, *time.Time) {
	f.mu.Lock()
	native, relay := f.native, f.relay
	f.mu.Unlock()
	if native != nil {
		select {
		case <-native.Done():
		default:
			p := native.Peer()
			at := native.ConnectedAt()
			return proto.ExitConnected, &p, &at
		}
	}
	if relay != nil {
		if relay.Connected() {
			return proto.ExitConnected, relay.Peer(), relay.ConnectedAt()
		}
		if p := relay.Peer(); p != nil {
			return proto.ExitConnecting, p, nil
		}
	}
	return proto.ExitDisconnected, nil, nil
}

// backend returns the live native session, or nil.
func (f *FrontDoor) backend() Backend {
	f.mu.Lock()
	b := f.native
	f.mu.Unlock()
	if b == nil {
		return nil
	}
	select {
	case <-b.Done():
		return nil
	default:
		return b
	}
}

func (f *FrontDoor) gate() RelayGate {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.relay
}

func (f *FrontDoor) track(c net.Conn) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopped {
		return false
	}
	f.conns[c] = struct{}{}
	return true
}

func (f *FrontDoor) untrack(c net.Conn) {
	f.mu.Lock()
	delete(f.conns, c)
	f.mu.Unlock()
}

func (f *FrontDoor) accept(ln net.Listener) {
	defer f.wg.Done()
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		if !f.track(c) {
			_ = c.Close()
			return
		}
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			defer f.untrack(c)
			defer c.Close()
			f.serve(c)
		}()
	}
}

// socksRequest is a parsed SOCKS5 request together with its raw bytes, which
// Mode B forwards verbatim.
type socksRequest struct {
	cmd  byte
	atyp byte
	dst  netip.AddrPort // valid for atyp 1 and 4
	raw  []byte
}

// serve runs one hev connection: greeting, request, dispatch.
func (f *FrontDoor) serve(c net.Conn) {
	_ = c.SetDeadline(time.Now().Add(frontDoorHandshake))
	if err := f.greet(c); err != nil {
		return
	}
	req, err := readSocksRequest(c)
	if err != nil {
		if errors.Is(err, errSocksAddrType) {
			_ = writeSocksReply(c, RepAddrTypeNotSupp, netip.AddrPort{})
		}
		return
	}
	_ = c.SetDeadline(time.Time{})

	switch req.cmd {
	case socksCmdConnect:
		if req.atyp == atypDomain {
			_ = writeSocksReply(c, RepAddrTypeNotSupp, netip.AddrPort{})
			return
		}
		if !f.policy().Allow(req.dst.Addr()) {
			_ = writeSocksReply(c, RepNotAllowed, netip.AddrPort{})
			return
		}
	case socksCmdUDPAssoc:
	default:
		_ = writeSocksReply(c, RepCommandNotSupp, netip.AddrPort{})
		return
	}

	if b := f.backend(); b != nil {
		f.serveNative(c, b, req)
		return
	}
	if g := f.gate(); g != nil && g.Connected() {
		f.serveRelay(c, req)
		return
	}
	_ = writeSocksReply(c, RepNetworkUnreach, netip.AddrPort{})
}

func (f *FrontDoor) greet(c net.Conn) error {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return err
	}
	if hdr[0] != socksVersion || hdr[1] == 0 {
		return errors.New("not socks5")
	}
	methods := make([]byte, int(hdr[1]))
	if _, err := io.ReadFull(c, methods); err != nil {
		return err
	}
	for _, m := range methods {
		if m == socksNoAuth {
			_, err := c.Write([]byte{socksVersion, socksNoAuth})
			return err
		}
	}
	_, _ = c.Write([]byte{socksVersion, socksNoAcceptable})
	return errors.New("no acceptable auth method")
}

var errSocksAddrType = errors.New("socks: unsupported address type")

func readSocksRequest(r io.Reader) (socksRequest, error) {
	head := make([]byte, 4)
	if _, err := io.ReadFull(r, head); err != nil {
		return socksRequest{}, err
	}
	if head[0] != socksVersion {
		return socksRequest{}, errors.New("socks: bad version in request")
	}
	req := socksRequest{cmd: head[1], atyp: head[3], raw: append([]byte(nil), head...)}
	switch req.atyp {
	case atypIPv4:
		b := make([]byte, 6)
		if _, err := io.ReadFull(r, b); err != nil {
			return socksRequest{}, err
		}
		req.dst = netip.AddrPortFrom(netip.AddrFrom4([4]byte{b[0], b[1], b[2], b[3]}), binary.BigEndian.Uint16(b[4:]))
		req.raw = append(req.raw, b...)
	case atypIPv6:
		b := make([]byte, 18)
		if _, err := io.ReadFull(r, b); err != nil {
			return socksRequest{}, err
		}
		var a [16]byte
		copy(a[:], b[:16])
		req.dst = netip.AddrPortFrom(netip.AddrFrom16(a), binary.BigEndian.Uint16(b[16:]))
		req.raw = append(req.raw, b...)
	case atypDomain:
		n := make([]byte, 1)
		if _, err := io.ReadFull(r, n); err != nil {
			return socksRequest{}, err
		}
		b := make([]byte, int(n[0])+2)
		if _, err := io.ReadFull(r, b); err != nil {
			return socksRequest{}, err
		}
		req.raw = append(append(req.raw, n...), b...)
	default:
		return socksRequest{}, errSocksAddrType
	}
	return req, nil
}

// writeSocksReply writes VER REP RSV ATYP BND.ADDR BND.PORT. An invalid bnd
// is rendered as 0.0.0.0:0.
func writeSocksReply(w io.Writer, rep byte, bnd netip.AddrPort) error {
	if !bnd.IsValid() {
		bnd = netip.AddrPortFrom(netip.IPv4Unspecified(), 0)
	}
	out := []byte{socksVersion, rep, 0x00}
	out = append(out, encodeAddrPort(bnd)...)
	_, err := w.Write(out)
	return err
}

// serveNative dispatches to the attached nexit/1 session.
func (f *FrontDoor) serveNative(c net.Conn, b Backend, req socksRequest) {
	switch req.cmd {
	case socksCmdConnect:
		ctx, cancel := context.WithTimeout(context.Background(), openTimeout+time.Second)
		stream, err := b.DialTCP(ctx, req.dst)
		cancel()
		if err != nil {
			_ = writeSocksReply(c, RepOf(err), netip.AddrPort{})
			return
		}
		if err := writeSocksReply(c, RepSucceeded, addrPortOf(stream.LocalAddr())); err != nil {
			_ = stream.Close()
			return
		}
		// The mux counts bytes in Mode A, so pass nil here.
		relayTCP(c, stream, nil)
	case socksCmdUDPAssoc:
		ctx, cancel := context.WithTimeout(context.Background(), openTimeout+time.Second)
		stream, err := b.OpenUDP(ctx)
		cancel()
		if err != nil {
			_ = writeSocksReply(c, RepOf(err), netip.AddrPort{})
			return
		}
		f.serveUDPAssociate(c, stream)
	}
}

// serveRelay is Mode B: dial wstunnel's reverse-SOCKS listener, redo the
// greeting toward it, then forward the request bytes verbatim and copy both
// ways. wstunnel's own reply (including its UDP BND.ADDR) reaches hev as is.
func (f *FrontDoor) serveRelay(c net.Conn, req socksRequest) {
	up, err := net.DialTimeout("tcp", wstunnelReverseAddr(f.slot), relayDialTimeout)
	if err != nil {
		log.Warnf("%s: reverse listener %s: %s", f.slot.Name(), wstunnelReverseAddr(f.slot), err)
		_ = writeSocksReply(c, RepNetworkUnreach, netip.AddrPort{})
		return
	}
	defer up.Close()
	_ = up.SetDeadline(time.Now().Add(frontDoorHandshake))
	if _, err := up.Write([]byte{socksVersion, 1, socksNoAuth}); err != nil {
		_ = writeSocksReply(c, RepGeneralFailure, netip.AddrPort{})
		return
	}
	ack := make([]byte, 2)
	if _, err := io.ReadFull(up, ack); err != nil || ack[0] != socksVersion || ack[1] != socksNoAuth {
		_ = writeSocksReply(c, RepGeneralFailure, netip.AddrPort{})
		return
	}
	if _, err := up.Write(req.raw); err != nil {
		_ = writeSocksReply(c, RepGeneralFailure, netip.AddrPort{})
		return
	}
	reply, err := readSocksReply(up)
	if err != nil {
		_ = writeSocksReply(c, RepGeneralFailure, netip.AddrPort{})
		return
	}
	// wstunnel's reply, BND.ADDR included, reaches hev as is.
	if _, err := c.Write(reply); err != nil || reply[1] != RepSucceeded {
		return
	}
	_ = up.SetDeadline(time.Time{})
	relayTCP(c, up, f.bytes)
}

// readSocksReply reads one complete VER REP RSV ATYP BND.ADDR BND.PORT reply.
func readSocksReply(r io.Reader) ([]byte, error) {
	head := make([]byte, 4)
	if _, err := io.ReadFull(r, head); err != nil {
		return nil, err
	}
	if head[0] != socksVersion {
		return nil, errors.New("socks: bad version in reply")
	}
	var rest []byte
	switch head[3] {
	case atypIPv4:
		rest = make([]byte, 6)
	case atypIPv6:
		rest = make([]byte, 18)
	case atypDomain:
		n := make([]byte, 1)
		if _, err := io.ReadFull(r, n); err != nil {
			return nil, err
		}
		head = append(head, n...)
		rest = make([]byte, int(n[0])+2)
	default:
		return nil, errSocksAddrType
	}
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, err
	}
	return append(head, rest...), nil
}

// relayTCP copies both directions between hev and the exit-side conn,
// propagating half closes, until both directions have ended or one side
// aborts. counter, when non-nil, records hev->exit as Up and exit->hev as Down.
func relayTCP(hev, exit net.Conn, counter ByteCounter) {
	var wg sync.WaitGroup
	wg.Add(2)
	abort := func() {
		_ = hev.Close()
		_ = exit.Close()
	}
	go func() {
		defer wg.Done()
		var w io.Writer = exit
		if counter != nil {
			w = &countingWriter{w: exit, add: counter.AddUp}
		}
		_, err := io.Copy(w, hev)
		if err != nil {
			abort()
			return
		}
		if !closeWrite(exit) {
			abort()
		}
	}()
	go func() {
		defer wg.Done()
		var w io.Writer = hev
		if counter != nil {
			w = &countingWriter{w: hev, add: counter.AddDown}
		}
		_, err := io.Copy(w, exit)
		if err != nil {
			abort()
			return
		}
		if !closeWrite(hev) {
			abort()
		}
	}()
	wg.Wait()
	_ = hev.Close()
	_ = exit.Close()
}

func closeWrite(c net.Conn) bool {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite() == nil
	}
	return false
}

type countingWriter struct {
	w   io.Writer
	add func(int64)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if n > 0 {
		c.add(int64(n))
	}
	return n, err
}

func addrPortOf(a net.Addr) netip.AddrPort {
	switch v := a.(type) {
	case *net.TCPAddr:
		return v.AddrPort()
	case *net.UDPAddr:
		return v.AddrPort()
	}
	return netip.AddrPort{}
}

// dialThroughFrontDoor is the SOCKS5 client the forwarder and prober use:
// a plain CONNECT to dst via the slot's front door, so both work identically
// in either mode.
func dialThroughFrontDoor(ctx context.Context, slot Slot, dst netip.AddrPort) (net.Conn, error) {
	var d net.Dialer
	c, err := d.DialContext(ctx, "tcp", socksDialAddr(slot))
	if err != nil {
		return nil, err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetDeadline(dl)
	}
	if _, err := c.Write([]byte{socksVersion, 1, socksNoAuth}); err != nil {
		_ = c.Close()
		return nil, err
	}
	ack := make([]byte, 2)
	if _, err := io.ReadFull(c, ack); err != nil {
		_ = c.Close()
		return nil, err
	}
	if ack[0] != socksVersion || ack[1] != socksNoAuth {
		_ = c.Close()
		return nil, errors.New("socks: greeting refused")
	}
	req := []byte{socksVersion, socksCmdConnect, 0x00}
	req = append(req, encodeAddrPort(dst)...)
	if _, err := c.Write(req); err != nil {
		_ = c.Close()
		return nil, err
	}
	reply, err := readSocksReply(c)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	if reply[1] != RepSucceeded {
		_ = c.Close()
		return nil, &RepError{Rep: reply[1], Err: fmt.Errorf("socks connect to %s", dst)}
	}
	_ = c.SetDeadline(time.Time{})
	return c, nil
}
