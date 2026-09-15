package exit

import (
	"encoding/binary"
	"io"
	"net"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// fakeResolver is a TCP-only DNS server: A for example.test., a large TXT
// set for big.test., NXDOMAIN otherwise. AAAA must never reach it.
type fakeResolver struct {
	ln      net.Listener
	queries atomic.Int32
	aaaa    atomic.Int32
	addr    netip.AddrPort
}

func newFakeResolver(t *testing.T) *fakeResolver {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	r := &fakeResolver{ln: ln, addr: ln.Addr().(*net.TCPAddr).AddrPort()}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go r.serve(c)
		}
	}()
	return r
}

func (r *fakeResolver) serve(c net.Conn) {
	defer c.Close()
	for {
		q, err := readTCPMessage(c)
		if err != nil {
			return
		}
		r.queries.Add(1)
		var m dnsmessage.Message
		if err := m.Unpack(q); err != nil || len(m.Questions) != 1 {
			return
		}
		question := m.Questions[0]
		if question.Type == dnsmessage.TypeAAAA {
			r.aaaa.Add(1)
		}
		resp := dnsmessage.Message{
			Header:    dnsmessage.Header{ID: m.ID, Response: true, RecursionAvailable: true, RecursionDesired: m.RecursionDesired},
			Questions: []dnsmessage.Question{question},
		}
		switch {
		case question.Name.String() == "example.test." && question.Type == dnsmessage.TypeA:
			resp.Answers = []dnsmessage.Resource{{
				Header: dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 60},
				Body:   &dnsmessage.AResource{A: [4]byte{192, 0, 2, 10}},
			}}
		case question.Name.String() == "long.test." && question.Type == dnsmessage.TypeA:
			resp.Answers = []dnsmessage.Resource{{
				Header: dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 86400},
				Body:   &dnsmessage.AResource{A: [4]byte{192, 0, 2, 11}},
			}}
		case question.Name.String() == "big.test." && question.Type == dnsmessage.TypeTXT:
			for i := 0; i < 8; i++ {
				resp.Answers = append(resp.Answers, dnsmessage.Resource{
					Header: dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeTXT, Class: dnsmessage.ClassINET, TTL: 30},
					Body:   &dnsmessage.TXTResource{TXT: []string{strings.Repeat("x", 200)}},
				})
			}
		default:
			resp.RCode = dnsmessage.RCodeNameError
		}
		out, err := resp.Pack()
		if err != nil {
			return
		}
		if err := writeTCPMessage(c, out); err != nil {
			return
		}
	}
}

// dnsHarness is a Forwarder bound on loopback whose upstream is reached
// through a front door and a fake exit that maps the pinned resolver address
// onto the fake resolver.
type dnsHarness struct {
	fd        *fdHarness
	fwd       *Forwarder
	res       *fakeResolver
	connected atomic.Bool
	upstreams atomic.Pointer[[]netip.Addr]
	udpAddr   string
	tcpAddr   string
}

var (
	resolverA = netip.MustParseAddr("192.0.2.53")
	resolverB = netip.MustParseAddr("192.0.2.54")
	deadRes   = netip.MustParseAddr("192.0.2.52")
)

func newDNSHarness(t *testing.T) *dnsHarness {
	t.Helper()
	h := &dnsHarness{fd: newFrontDoor(t), res: newFakeResolver(t)}
	ups := []netip.Addr{resolverA}
	h.upstreams.Store(&ups)
	h.connected.Store(true)

	f := NewFakeExit()
	f.Connect = func(dst netip.AddrPort) (net.Conn, byte) {
		switch dst.Addr() {
		case resolverA, resolverB:
			if dst.Port() != 53 {
				return nil, RepConnRefused
			}
			return defaultConnect(h.res.addr)
		}
		return nil, RepConnRefused
	}
	h.fd.attachNative(t, f)

	h.fwd = NewForwarder(h.fd.slot, func() []netip.Addr { return *h.upstreams.Load() }, h.connected.Load)
	if err := h.fwd.Bind(netip.MustParseAddr("127.0.0.1")); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	t.Cleanup(h.fwd.Stop)
	h.udpAddr, h.tcpAddr = h.fwd.boundAddrs()
	return h
}

func buildQuery(t *testing.T, id uint16, name string, typ dnsmessage.Type, edns int) []byte {
	t.Helper()
	m := dnsmessage.Message{
		Header:    dnsmessage.Header{ID: id, RecursionDesired: true},
		Questions: []dnsmessage.Question{{Name: dnsmessage.MustNewName(name), Type: typ, Class: dnsmessage.ClassINET}},
	}
	if edns > 0 {
		m.Additionals = []dnsmessage.Resource{{
			Header: dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName("."), Type: dnsmessage.TypeOPT, Class: dnsmessage.Class(edns)},
			Body:   &dnsmessage.OPTResource{},
		}}
	}
	q, err := m.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func (h *dnsHarness) queryUDP(t *testing.T, q []byte) dnsmessage.Message {
	t.Helper()
	c, err := net.Dial("udp", h.udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := c.Write(q); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("udp answer: %v", err)
	}
	var m dnsmessage.Message
	if err := m.Unpack(buf[:n]); err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if m.ID != binary.BigEndian.Uint16(q) {
		t.Fatalf("answer id %d for query %d", m.ID, binary.BigEndian.Uint16(q))
	}
	return m
}

func (h *dnsHarness) queryTCP(t *testing.T, q []byte) dnsmessage.Message {
	t.Helper()
	c, err := net.Dial("tcp", h.tcpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if err := writeTCPMessage(c, q); err != nil {
		t.Fatal(err)
	}
	resp, err := readTCPMessage(c)
	if err != nil {
		t.Fatalf("tcp answer: %v", err)
	}
	var m dnsmessage.Message
	if err := m.Unpack(resp); err != nil {
		t.Fatalf("unpack: %v", err)
	}
	return m
}

func TestForwarderAnswersThroughTheExit(t *testing.T) {
	h := newDNSHarness(t)
	if !h.fwd.Bound() {
		t.Fatal("not bound")
	}
	if !strings.HasPrefix(h.udpAddr, "127.0.0.1:") || !strings.HasPrefix(h.tcpAddr, "127.0.0.1:") {
		t.Fatalf("bound to %s / %s", h.udpAddr, h.tcpAddr)
	}

	m := h.queryUDP(t, buildQuery(t, 0x1234, "example.test.", dnsmessage.TypeA, 0))
	if m.RCode != dnsmessage.RCodeSuccess || len(m.Answers) != 1 || !m.RecursionAvailable {
		t.Fatalf("answer %+v", m)
	}
	a, ok := m.Answers[0].Body.(*dnsmessage.AResource)
	if !ok || a.A != [4]byte{192, 0, 2, 10} {
		t.Fatalf("A record %+v", m.Answers[0].Body)
	}
	if h.res.queries.Load() != 1 {
		t.Fatalf("resolver saw %d queries", h.res.queries.Load())
	}

	// Same question again: served from cache with a TTL no larger than the original.
	m2 := h.queryUDP(t, buildQuery(t, 0x4321, "EXAMPLE.test.", dnsmessage.TypeA, 0))
	if len(m2.Answers) != 1 || m2.Answers[0].Header.TTL > 60 || m2.Answers[0].Header.TTL == 0 {
		t.Fatalf("cached answer %+v", m2.Answers)
	}
	if h.res.queries.Load() != 1 {
		t.Fatalf("cache miss: resolver saw %d queries", h.res.queries.Load())
	}

	// TCP works too and shares the cache.
	m3 := h.queryTCP(t, buildQuery(t, 7, "example.test.", dnsmessage.TypeA, 0))
	if len(m3.Answers) != 1 || h.res.queries.Load() != 1 {
		t.Fatalf("tcp answer %+v, resolver queries %d", m3.Answers, h.res.queries.Load())
	}

	// NXDOMAIN passes through and is not cached.
	m4 := h.queryUDP(t, buildQuery(t, 8, "nope.test.", dnsmessage.TypeA, 0))
	if m4.RCode != dnsmessage.RCodeNameError {
		t.Fatalf("rcode %v", m4.RCode)
	}
	h.queryUDP(t, buildQuery(t, 9, "nope.test.", dnsmessage.TypeA, 0))
	if h.res.queries.Load() != 3 {
		t.Fatalf("negative answer cached: resolver queries %d", h.res.queries.Load())
	}

	// TTLs are capped at 300 s in the cache.
	h.queryUDP(t, buildQuery(t, 10, "long.test.", dnsmessage.TypeA, 0))
	m5 := h.queryUDP(t, buildQuery(t, 11, "long.test.", dnsmessage.TypeA, 0))
	if m5.Answers[0].Header.TTL > 86400 || h.res.queries.Load() != 4 {
		t.Fatalf("long ttl handling: ttl %d, queries %d", m5.Answers[0].Header.TTL, h.res.queries.Load())
	}

	st := h.fwd.Stats()
	if st.Queries != 7 || st.Failures != 0 {
		t.Fatalf("stats %+v", st)
	}
}

func TestForwarderAAAAIsEmptyNoError(t *testing.T) {
	h := newDNSHarness(t)
	m := h.queryUDP(t, buildQuery(t, 1, "example.test.", dnsmessage.TypeAAAA, 0))
	if m.RCode != dnsmessage.RCodeSuccess || len(m.Answers) != 0 || len(m.Questions) != 1 {
		t.Fatalf("AAAA answer %+v", m)
	}
	if h.res.queries.Load() != 0 || h.res.aaaa.Load() != 0 {
		t.Fatal("AAAA reached the resolver")
	}
	// Even when no exit is connected, AAAA is not SERVFAIL.
	h.connected.Store(false)
	m = h.queryUDP(t, buildQuery(t, 2, "example.test.", dnsmessage.TypeAAAA, 0))
	if m.RCode != dnsmessage.RCodeSuccess {
		t.Fatalf("AAAA while disconnected: %v", m.RCode)
	}
}

func TestForwarderServfailWithoutExit(t *testing.T) {
	h := newDNSHarness(t)
	h.connected.Store(false)
	start := time.Now()
	m := h.queryUDP(t, buildQuery(t, 1, "example.test.", dnsmessage.TypeA, 0))
	if m.RCode != dnsmessage.RCodeServerFailure {
		t.Fatalf("rcode %v", m.RCode)
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("SERVFAIL was slow")
	}
	if h.res.queries.Load() != 0 {
		t.Fatal("query forwarded while disconnected")
	}
	if st := h.fwd.Stats(); st.Queries != 1 || st.Failures != 1 {
		t.Fatalf("stats %+v", st)
	}
	// A cached answer is still served while disconnected.
	h.connected.Store(true)
	h.queryUDP(t, buildQuery(t, 2, "example.test.", dnsmessage.TypeA, 0))
	h.connected.Store(false)
	m = h.queryUDP(t, buildQuery(t, 3, "example.test.", dnsmessage.TypeA, 0))
	if m.RCode != dnsmessage.RCodeSuccess || len(m.Answers) != 1 {
		t.Fatalf("cached answer while disconnected: %+v", m)
	}
}

func TestForwarderUpstreamFailover(t *testing.T) {
	h := newDNSHarness(t)
	ups := []netip.Addr{deadRes, resolverB}
	h.upstreams.Store(&ups)
	m := h.queryUDP(t, buildQuery(t, 1, "example.test.", dnsmessage.TypeA, 0))
	if m.RCode != dnsmessage.RCodeSuccess || len(m.Answers) != 1 {
		t.Fatalf("failover answer %+v", m)
	}
	// Every upstream dead: SERVFAIL within the budget, counted as a failure.
	ups = []netip.Addr{deadRes}
	h.upstreams.Store(&ups)
	start := time.Now()
	m = h.queryUDP(t, buildQuery(t, 2, "other.test.", dnsmessage.TypeA, 0))
	if m.RCode != dnsmessage.RCodeServerFailure {
		t.Fatalf("rcode %v", m.RCode)
	}
	if d := time.Since(start); d > dnsQueryBudget+time.Second {
		t.Fatalf("all-dead SERVFAIL took %s", d)
	}
	if st := h.fwd.Stats(); st.Failures != 1 {
		t.Fatalf("stats %+v", st)
	}
	// No upstreams configured at all.
	ups = nil
	h.upstreams.Store(&ups)
	if m := h.queryUDP(t, buildQuery(t, 3, "other.test.", dnsmessage.TypeA, 0)); m.RCode != dnsmessage.RCodeServerFailure {
		t.Fatalf("no upstreams: %v", m.RCode)
	}
}

func TestForwarderTruncatesForUDP(t *testing.T) {
	h := newDNSHarness(t)
	m := h.queryUDP(t, buildQuery(t, 1, "big.test.", dnsmessage.TypeTXT, 0))
	if !m.Truncated || len(m.Answers) != 0 {
		t.Fatalf("plain UDP: truncated=%v answers=%d", m.Truncated, len(m.Answers))
	}
	m = h.queryUDP(t, buildQuery(t, 2, "big.test.", dnsmessage.TypeTXT, 4096))
	if m.Truncated || len(m.Answers) != 8 {
		t.Fatalf("EDNS UDP: truncated=%v answers=%d", m.Truncated, len(m.Answers))
	}
	m = h.queryTCP(t, buildQuery(t, 3, "big.test.", dnsmessage.TypeTXT, 0))
	if m.Truncated || len(m.Answers) != 8 {
		t.Fatalf("TCP: truncated=%v answers=%d", m.Truncated, len(m.Answers))
	}
}

func TestForwarderIgnoresGarbage(t *testing.T) {
	h := newDNSHarness(t)
	c, err := net.Dial("udp", h.udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, _ = c.Write([]byte("nope"))
	_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := c.Read(make([]byte, 64)); err == nil {
		t.Fatal("garbage was answered")
	}
	// A response sent to the server is dropped, not reflected.
	resp := buildResponse(dnsmessage.Header{ID: 9}, nil, dnsmessage.RCodeSuccess, nil, false)
	_, _ = c.Write(resp)
	_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := c.Read(make([]byte, 64)); err == nil {
		t.Fatal("a response was reflected")
	}
	if st := h.fwd.Stats(); st.Queries != 0 {
		t.Fatalf("garbage counted: %+v", st)
	}
	// A TCP client that sends a bad length is dropped.
	tc, err := net.Dial("tcp", h.tcpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer tc.Close()
	_, _ = tc.Write([]byte{0, 0})
	_ = tc.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := tc.Read(make([]byte, 1)); err == nil || !isEOF(err) {
		t.Fatalf("bad length: %v", err)
	}
}

func isEOF(err error) bool {
	return err == io.EOF || strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "reset")
}

func TestForwarderBindStopRebind(t *testing.T) {
	h := newDNSHarness(t)
	udp1, _ := h.fwd.boundAddrs()
	h.fwd.Stop()
	if h.fwd.Bound() || h.fwd.Addr().IsValid() {
		t.Fatal("still bound after Stop")
	}
	if st := h.fwd.Stats(); st.Queries != 0 {
		t.Fatalf("stats reset: %+v", st)
	}
	c, _ := net.Dial("udp", udp1)
	_, _ = c.Write(buildQuery(t, 1, "example.test.", dnsmessage.TypeA, 0))
	_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := c.Read(make([]byte, 64)); err == nil {
		t.Fatal("old listener still answers")
	}
	_ = c.Close()

	if err := h.fwd.Bind(netip.MustParseAddr("127.0.0.1")); err != nil {
		t.Fatalf("rebind: %v", err)
	}
	h.udpAddr, h.tcpAddr = h.fwd.boundAddrs()
	if m := h.queryUDP(t, buildQuery(t, 2, "example.test.", dnsmessage.TypeA, 0)); len(m.Answers) != 1 {
		t.Fatalf("after rebind %+v", m)
	}
	// Bind on an address that is already taken fails and leaves nothing bound.
	taken, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	old := dnsPort
	dnsPort = taken.LocalAddr().(*net.UDPAddr).Port
	err = h.fwd.Bind(netip.MustParseAddr("127.0.0.1"))
	dnsPort = old
	if err == nil {
		t.Fatal("Bind on a taken port succeeded")
	}
	if h.fwd.Bound() {
		t.Fatal("half bound after failure")
	}
	// Stop on an unbound forwarder is harmless.
	h.fwd.Stop()
}

func TestDNSCacheBounds(t *testing.T) {
	c := newDNSCache()
	name := func(i int) dnsmessage.Question {
		return dnsmessage.Question{Name: dnsmessage.MustNewName("h" + strings.Repeat("a", i%50) + "." + strings.Repeat("b", i/50+1) + ".test."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}
	}
	rr := func(ttl uint32) []dnsmessage.Resource {
		return []dnsmessage.Resource{{Header: dnsmessage.ResourceHeader{TTL: ttl}, Body: &dnsmessage.AResource{}}}
	}
	for i := 0; i < dnsCacheSize+250; i++ {
		c.put(name(i), rr(60))
	}
	if c.len() > dnsCacheSize {
		t.Fatalf("cache holds %d entries", c.len())
	}
	// TTL 0 is never cached; the cap applies to long TTLs.
	zero := dnsmessage.Question{Name: dnsmessage.MustNewName("zero.test."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}
	c.put(zero, rr(0))
	if _, ok := c.get(zero); ok {
		t.Fatal("TTL 0 cached")
	}
	c.put(name(2), rr(100000))
	e := c.entries[keyOf(name(2))]
	if time.Until(e.expires) > dnsCacheMaxTTL+time.Second {
		t.Fatalf("expiry %s beyond cap", time.Until(e.expires))
	}
	// Names are matched case-insensitively.
	q := name(3)
	c.put(q, rr(60))
	q.Name = dnsmessage.MustNewName(strings.ToUpper(q.Name.String()))
	if _, ok := c.get(q); !ok {
		t.Fatal("case-insensitive lookup failed")
	}
}
