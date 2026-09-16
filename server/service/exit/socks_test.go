package exit

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"golang.org/x/net/proxy"
)

// fdHarness is a started FrontDoor on an ephemeral loopback port.
type fdHarness struct {
	fd     *FrontDoor
	slot   Slot
	policy atomic.Pointer[Policy]
	bytes  *testCounter
}

func newFrontDoor(t *testing.T) *fdHarness {
	t.Helper()
	h := &fdHarness{slot: MustSlot("0"), bytes: &testCounter{}}
	h.policy.Store(&Policy{})
	h.fd = NewFrontDoor(h.slot, func() Policy { return *h.policy.Load() }, h.bytes)
	if err := h.fd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	setTestSocksAddr(h.slot, h.fd.Addr())
	t.Cleanup(h.fd.Stop)
	return h
}

// dialer is golang.org/x/net/proxy's SOCKS5 client pointed at the front door.
func (h *fdHarness) dialer(t *testing.T) proxy.ContextDialer {
	t.Helper()
	d, err := proxy.SOCKS5("tcp", h.fd.Addr(), nil, proxy.Direct)
	if err != nil {
		t.Fatal(err)
	}
	return d.(proxy.ContextDialer)
}

// attachNative wires a mux session into the front door and returns the
// fake exit that serves it. The mux shares the front door's counter, as
// wire.go has it, so a byte counted twice shows up as twice.
func (h *fdHarness) attachNative(t *testing.T, f *FakeExit) (*muxHarness, Backend) {
	t.Helper()
	mh := newMuxHarnessWith(t, h.bytes)
	mh.policy.Store(h.policy.Load())
	s := mh.dial(t, f, "")
	h.fd.SetNative(s)
	return mh, s
}

// redirectPacketConn makes every WriteTo go to one fixed address, so a fake
// exit's UDP socket reaches a loopback echo while the policy sees TEST-NET.
type redirectPacketConn struct {
	net.PacketConn
	to net.Addr
}

func (r *redirectPacketConn) WriteTo(b []byte, _ net.Addr) (int, error) {
	return r.PacketConn.WriteTo(b, r.to)
}

func udpEcho(t *testing.T) *net.UDPAddr {
	t.Helper()
	pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	go func() {
		buf := make([]byte, 64<<10)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = pc.WriteTo(append([]byte("echo:"), buf[:n]...), addr)
		}
	}()
	return pc.LocalAddr().(*net.UDPAddr)
}

// rawSocks performs the greeting and one request, returning the reply.
func rawSocks(t *testing.T, addr string, cmd byte, dst netip.AddrPort) (net.Conn, byte, netip.AddrPort) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := c.Write([]byte{5, 1, 0}); err != nil {
		t.Fatal(err)
	}
	ack := make([]byte, 2)
	if _, err := io.ReadFull(c, ack); err != nil || ack[0] != 5 || ack[1] != 0 {
		t.Fatalf("greeting reply %v %v", ack, err)
	}
	req := append([]byte{5, cmd, 0}, encodeAddrPort(dst)...)
	if _, err := c.Write(req); err != nil {
		t.Fatal(err)
	}
	head := make([]byte, 4)
	if _, err := io.ReadFull(c, head); err != nil {
		t.Fatalf("reply: %v", err)
	}
	var bnd netip.AddrPort
	switch head[3] {
	case atypIPv4:
		b := make([]byte, 6)
		if _, err := io.ReadFull(c, b); err != nil {
			t.Fatal(err)
		}
		bnd, _, _ = decodeAddrPort(append([]byte{atypIPv4}, b...))
	case atypIPv6:
		b := make([]byte, 18)
		if _, err := io.ReadFull(c, b); err != nil {
			t.Fatal(err)
		}
		bnd, _, _ = decodeAddrPort(append([]byte{atypIPv6}, b...))
	default:
		t.Fatalf("reply atyp %d", head[3])
	}
	_ = c.SetDeadline(time.Time{})
	return c, head[1], bnd
}

func TestFrontDoorRefusesFastWithoutExit(t *testing.T) {
	h := newFrontDoor(t)
	if !strings.HasPrefix(h.fd.Addr(), "127.0.0.1:") {
		t.Fatalf("bound to %s", h.fd.Addr())
	}
	start := time.Now()
	_, err := h.dialer(t).DialContext(context.Background(), "tcp", testDst.String())
	if err == nil || !strings.Contains(err.Error(), "network unreachable") {
		t.Fatalf("CONNECT without exit: %v", err)
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Fatalf("refusal took %s", d)
	}
	_, rep, _ := rawSocks(t, h.fd.Addr(), socksCmdUDPAssoc, netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
	if rep != RepNetworkUnreach {
		t.Fatalf("UDP ASSOCIATE without exit: rep %#x", rep)
	}
	state, peer, since := h.fd.Attached()
	if state != proto.ExitDisconnected || peer != nil || since != nil {
		t.Fatalf("Attached = %s %v %v", state, peer, since)
	}
}

func TestFrontDoorConnectThroughNative(t *testing.T) {
	h := newFrontDoor(t)
	echo := echoServer(t)
	f := NewFakeExit()
	f.Connect = func(dst netip.AddrPort) (net.Conn, byte) { return defaultConnect(echo) }
	_, s := h.attachNative(t, f)

	state, peer, since := h.fd.Attached()
	if state != proto.ExitConnected || peer == nil || peer.Hostname != "fake-exit" || since == nil {
		t.Fatalf("Attached = %s %v %v", state, peer, since)
	}

	c, rep, bnd := rawSocks(t, h.fd.Addr(), socksCmdConnect, testDst)
	if rep != RepSucceeded {
		t.Fatalf("CONNECT rep %#x", rep)
	}
	if !bnd.Addr().IsLoopback() || bnd.Port() == 0 {
		t.Fatalf("BND.ADDR %s is not the exit's bound socket", bnd)
	}
	payload := make([]byte, 200<<10)
	_, _ = rand.Read(payload)
	go func() {
		_, _ = c.Write(payload)
		_ = c.(*net.TCPConn).CloseWrite()
	}()
	got, err := io.ReadAll(c)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("echo: %d bytes, %v", len(got), err)
	}
	_ = c.Close()
	waitFor(t, "stream release", func() bool { return s.(*session).streamCount() == 0 })
	// Mode A bytes are counted once, by the mux; the front door shares the
	// counter and adds nothing.
	if tot := h.bytes.Totals(); tot.Up != uint64(len(payload)) || tot.Down != uint64(len(payload)) {
		t.Fatalf("Mode A bytes = %+v, want %d each way counted once", tot, len(payload))
	}
}

func TestFrontDoorHalfCloseFromExit(t *testing.T) {
	h := newFrontDoor(t)
	f := NewFakeExit()
	a, b := tcpPair(t)
	f.Connect = func(netip.AddrPort) (net.Conn, byte) {
		go func() {
			_, _ = b.Write([]byte("greeting"))
			_ = b.(*net.TCPConn).CloseWrite()
		}()
		return a, 0
	}
	h.attachNative(t, f)
	c, err := h.dialer(t).DialContext(context.Background(), "tcp", testDst.String())
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(c)
	if err != nil || string(got) != "greeting" {
		t.Fatalf("read %q %v", got, err)
	}
	// hev's direction is still open.
	if _, err := c.Write([]byte("reply")); err != nil {
		t.Fatalf("write after exit EOF: %v", err)
	}
	buf := make([]byte, 5)
	_ = b.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(b, buf); err != nil || string(buf) != "reply" {
		t.Fatalf("exit read %q %v", buf, err)
	}
	_ = c.Close()
}

func TestFrontDoorReplyCodes(t *testing.T) {
	h := newFrontDoor(t)
	var reason atomic.Uint32
	f := NewFakeExit()
	f.Connect = func(netip.AddrPort) (net.Conn, byte) { return nil, byte(reason.Load()) }
	h.attachNative(t, f)
	d := h.dialer(t)

	cases := []struct {
		rep  byte
		want string
	}{
		{RepConnRefused, "connection refused"},
		{RepHostUnreach, "host unreachable"},
		{RepNetworkUnreach, "network unreachable"},
		{RepTTLExpired, "TTL expired"},
		{RepGeneralFailure, "general SOCKS server failure"},
	}
	for _, tc := range cases {
		reason.Store(uint32(tc.rep))
		_, err := d.DialContext(context.Background(), "tcp", testDst.String())
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("rep %#x: %v", tc.rep, err)
		}
	}

	// Domain destinations are refused before any OPEN.
	var opens atomic.Int32
	f.OnOpen = func(uint32, byte, netip.AddrPort) { opens.Add(1) }
	_, err := d.DialContext(context.Background(), "tcp", "example.com:80")
	if err == nil || !strings.Contains(err.Error(), "address type not supported") {
		t.Fatalf("domain CONNECT: %v", err)
	}
	// So is a denied destination.
	_, err = d.DialContext(context.Background(), "tcp", "10.0.0.1:80")
	if err == nil || !strings.Contains(err.Error(), "not allowed by ruleset") {
		t.Fatalf("private CONNECT: %v", err)
	}
	_, err = d.DialContext(context.Background(), "tcp", "127.0.0.1:80")
	if err == nil || !strings.Contains(err.Error(), "not allowed by ruleset") {
		t.Fatalf("loopback CONNECT: %v", err)
	}
	if opens.Load() != 0 {
		t.Fatalf("%d OPEN frames reached the exit for refused destinations", opens.Load())
	}
	// BIND is not supported.
	_, rep, _ := rawSocks(t, h.fd.Addr(), 0x02, testDst)
	if rep != RepCommandNotSupp {
		t.Fatalf("BIND rep %#x", rep)
	}
	// An IPv6 destination reaches the exit, which answers 0x08 without a v6 socket.
	f.Connect = func(dst netip.AddrPort) (net.Conn, byte) {
		if dst.Addr().Is6() {
			return nil, RepAddrTypeNotSupp
		}
		return nil, RepGeneralFailure
	}
	_, err = d.DialContext(context.Background(), "tcp", "[2001:db8::1]:443")
	if err == nil || !strings.Contains(err.Error(), "address type not supported") {
		t.Fatalf("v6 CONNECT: %v", err)
	}
}

func TestFrontDoorRejectsAuthMethods(t *testing.T) {
	h := newFrontDoor(t)
	c, err := net.Dial("tcp", h.fd.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	_, _ = c.Write([]byte{5, 1, 2}) // username/password only
	ack := make([]byte, 2)
	if _, err := io.ReadFull(c, ack); err != nil || ack[1] != socksNoAcceptable {
		t.Fatalf("ack %v %v", ack, err)
	}
	if _, err := c.Read(ack); err == nil {
		t.Fatal("connection stayed open after no acceptable method")
	}
}

func TestFrontDoorDetachedBackendRefuses(t *testing.T) {
	h := newFrontDoor(t)
	f := NewFakeExit()
	_, s := h.attachNative(t, f)
	f.Close()
	select {
	case <-s.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("session did not end")
	}
	// The manager has not called SetNative(nil) yet; the front door notices Done.
	_, err := h.dialer(t).DialContext(context.Background(), "tcp", testDst.String())
	if err == nil || !strings.Contains(err.Error(), "network unreachable") {
		t.Fatalf("CONNECT after session end: %v", err)
	}
	if state, _, _ := h.fd.Attached(); state != proto.ExitDisconnected {
		t.Fatalf("Attached = %s", state)
	}
	h.fd.SetNative(nil)
}

func TestFrontDoorUDPAssociate(t *testing.T) {
	h := newFrontDoor(t)
	echo := udpEcho(t)
	f := NewFakeExit()
	f.ListenUDP = func() (net.PacketConn, error) {
		pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}
		return &redirectPacketConn{PacketConn: pc, to: echo}, nil
	}
	_, s := h.attachNative(t, f)

	control, rep, bnd := rawSocks(t, h.fd.Addr(), socksCmdUDPAssoc, netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
	if rep != RepSucceeded {
		t.Fatalf("UDP ASSOCIATE rep %#x", rep)
	}
	if !bnd.Addr().IsLoopback() || bnd.Port() == 0 {
		t.Fatalf("BND.ADDR %s", bnd)
	}
	relay := net.UDPAddrFromAddrPort(bnd)

	// hev's datagram, addressed to TEST-NET; the fake redirects to the echo.
	client, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	dgram := append([]byte{0, 0, 0}, encodeAddrPort(testDst)...)
	dgram = append(dgram, []byte("ping")...)
	if _, err := client.WriteTo(dgram, relay); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2048)
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, from, err := client.ReadFrom(buf)
	if err != nil {
		t.Fatalf("no reply: %v", err)
	}
	if from.String() != relay.String() {
		t.Fatalf("reply from %s, want %s", from, relay)
	}
	src, payload, ok := decodeSocksUDP(buf[:n])
	if !ok || string(payload) != "echo:ping" {
		t.Fatalf("reply %q ok=%v", buf[:n], ok)
	}
	// Full cone: the header carries the real source the exit saw.
	if src != echo.AddrPort() {
		t.Fatalf("reply source %s, want %s", src, echo.AddrPort())
	}

	// A second source cannot hijack the association.
	other, _ := net.ListenPacket("udp4", "127.0.0.1:0")
	defer other.Close()
	_, _ = other.WriteTo(dgram, relay)
	_ = other.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, _, err := other.ReadFrom(buf); err == nil {
		t.Fatal("datagram from another source was relayed")
	}
	// Fragments and denied destinations are dropped.
	frag := append([]byte{0, 0, 1}, encodeAddrPort(testDst)...)
	_, _ = client.WriteTo(append(frag, 'x'), relay)
	denied := append([]byte{0, 0, 0}, encodeAddrPort(netip.MustParseAddrPort("10.0.0.1:53"))...)
	_, _ = client.WriteTo(append(denied, 'x'), relay)
	_ = client.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, _, err := client.ReadFrom(buf); err == nil {
		t.Fatal("fragment or denied destination was relayed")
	}
	if tot := h.bytes.Totals(); tot.Up != 4 || tot.Down != 9 {
		t.Fatalf("udp bytes %+v", tot)
	}

	// Closing the control connection tears the association down.
	_ = control.Close()
	waitFor(t, "association released", func() bool { return s.(*session).streamCount() == 0 && f.Streams() == 0 })
	_ = client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _ = client.WriteTo(dgram, relay)
	if _, _, err := client.ReadFrom(buf); err == nil {
		t.Fatal("relay still answering after control close")
	}
}

// blockingUDPStream is a UDPStream whose Close does not return until the
// test lets it, standing in for a stream whose RST is stuck behind a stalled
// WebSocket writer.
type blockingUDPStream struct {
	closing chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingUDPStream) Send(netip.AddrPort, []byte) error { return nil }
func (b *blockingUDPStream) Recv(ctx context.Context) (netip.AddrPort, []byte, error) {
	<-ctx.Done()
	return netip.AddrPort{}, nil, ctx.Err()
}
func (b *blockingUDPStream) Close() error {
	b.once.Do(func() { close(b.closing) })
	<-b.release
	return nil
}

// TestFrontDoorUDPTeardownClosesControlBeforeStream: the idle teardown must
// free hev's control connection even while the stream's Close is stuck on a
// backed-up WebSocket writer; the association's end must not wait for the
// exit to make progress.
func TestFrontDoorUDPTeardownClosesControlBeforeStream(t *testing.T) {
	fd := NewFrontDoor(MustSlot("0"), nil, nil)
	stream := &blockingUDPStream{closing: make(chan struct{}), release: make(chan struct{})}
	release := sync.OnceFunc(func() { close(stream.release) })
	defer release()
	hevSide, control := tcpPair(t)
	served := make(chan struct{})
	go func() {
		defer close(served)
		fd.serveUDPAssociate(control, stream)
	}()
	reply, err := readSocksReply(hevSide)
	if err != nil || reply[1] != RepSucceeded {
		t.Fatalf("UDP ASSOCIATE reply %v, %v", reply, err)
	}
	select {
	case <-stream.closing:
	case <-time.After(udpIdleTimeout * 4):
		t.Fatal("idle teardown did not start")
	}
	// The stream's Close is now blocked. hev's control connection must
	// already be closed.
	_ = hevSide.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := hevSide.Read(make([]byte, 1)); err == nil || isTimeout(err) {
		t.Fatalf("control read = %v, want closed while stream.Close blocks", err)
	}
	release()
	select {
	case <-served:
	case <-time.After(3 * time.Second):
		t.Fatal("serveUDPAssociate did not return once Close was released")
	}
}

func TestFrontDoorUDPIdleTeardown(t *testing.T) {
	h := newFrontDoor(t)
	f := NewFakeExit()
	_, s := h.attachNative(t, f)
	control, rep, _ := rawSocks(t, h.fd.Addr(), socksCmdUDPAssoc, netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
	if rep != RepSucceeded {
		t.Fatalf("rep %#x", rep)
	}
	start := time.Now()
	_ = control.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := control.Read(make([]byte, 1)); err == nil || isTimeout(err) {
		t.Fatalf("control read = %v, want close after idle", err)
	}
	if d := time.Since(start); d < udpIdleTimeout/2 || d > udpIdleTimeout*4 {
		t.Fatalf("torn down after %s (idle %s)", d, udpIdleTimeout)
	}
	waitFor(t, "stream released", func() bool { return s.(*session).streamCount() == 0 })
}

// fakeReverseSocks stands in for wstunnel's reverse-SOCKS listener: CONNECT
// is answered by dialling echo (whatever the destination), UDP ASSOCIATE by
// the fixed BND address, so the test can prove the reply reached hev verbatim.
func fakeReverseSocks(t *testing.T, echo netip.AddrPort, udpBnd netip.AddrPort) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				hdr := make([]byte, 2)
				if _, err := io.ReadFull(c, hdr); err != nil {
					return
				}
				methods := make([]byte, int(hdr[1]))
				if _, err := io.ReadFull(c, methods); err != nil {
					return
				}
				_, _ = c.Write([]byte{5, 0})
				req, err := readSocksRequest(c)
				if err != nil {
					return
				}
				switch req.cmd {
				case socksCmdConnect:
					up, err := net.Dial("tcp", echo.String())
					if err != nil {
						_ = writeSocksReply(c, RepConnRefused, netip.AddrPort{})
						return
					}
					_ = writeSocksReply(c, RepSucceeded, addrPortOf(up.LocalAddr()))
					relayTCP(c, up, nil)
				case socksCmdUDPAssoc:
					_ = writeSocksReply(c, RepSucceeded, udpBnd)
					_, _ = io.Copy(io.Discard, c)
				}
			}()
		}
	}()
	return ln
}

type fakeGate struct {
	connected atomic.Bool
	peer      *proto.ExitPeer
	at        *time.Time
}

func (g *fakeGate) Connected() bool         { return g.connected.Load() }
func (g *fakeGate) Peer() *proto.ExitPeer   { return g.peer }
func (g *fakeGate) ConnectedAt() *time.Time { return g.at }
func (g *fakeGate) set(on bool)             { g.connected.Store(on) }
func (g *fakeGate) withPeer(addr string) *fakeGate {
	now := time.Now()
	g.peer = &proto.ExitPeer{Addr: addr, Transport: proto.ExitModeWstunnel}
	g.at = &now
	return g
}

func TestFrontDoorRelayModeB(t *testing.T) {
	h := newFrontDoor(t)
	echo := echoServer(t)
	udpBnd := netip.MustParseAddrPort("127.0.0.1:4242")
	rev := fakeReverseSocks(t, echo, udpBnd)
	setTestReverseAddr(h.slot, rev.Addr().String())
	gate := (&fakeGate{}).withPeer("203.0.113.9")
	h.fd.SetRelay(gate)

	// Gate closed: 0x03 without touching the listener.
	if state, peer, _ := h.fd.Attached(); state != proto.ExitConnecting || peer == nil {
		t.Fatalf("Attached with unproven gate = %s %v", state, peer)
	}
	_, err := h.dialer(t).DialContext(context.Background(), "tcp", testDst.String())
	if err == nil || !strings.Contains(err.Error(), "network unreachable") {
		t.Fatalf("relay with closed gate: %v", err)
	}

	gate.set(true)
	if state, peer, since := h.fd.Attached(); state != proto.ExitConnected || peer.Addr != "203.0.113.9" || since == nil {
		t.Fatalf("Attached = %s %v %v", state, peer, since)
	}
	c, rep, _ := rawSocks(t, h.fd.Addr(), socksCmdConnect, testDst)
	if rep != RepSucceeded {
		t.Fatalf("relay CONNECT rep %#x", rep)
	}
	payload := make([]byte, 100<<10)
	_, _ = rand.Read(payload)
	go func() {
		_, _ = c.Write(payload)
		_ = c.(*net.TCPConn).CloseWrite()
	}()
	got, err := io.ReadAll(c)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("relayed echo: %d bytes, %v", len(got), err)
	}
	_ = c.Close()
	if tot := h.bytes.Totals(); tot.Up != uint64(len(payload)) || tot.Down != uint64(len(payload)) {
		t.Fatalf("relay bytes %+v", tot)
	}

	// Policy still applies in front of the relay, and domains are refused.
	_, err = h.dialer(t).DialContext(context.Background(), "tcp", "192.168.1.1:80")
	if err == nil || !strings.Contains(err.Error(), "not allowed by ruleset") {
		t.Fatalf("private through relay: %v", err)
	}
	_, err = h.dialer(t).DialContext(context.Background(), "tcp", "example.com:80")
	if err == nil || !strings.Contains(err.Error(), "address type not supported") {
		t.Fatalf("domain through relay: %v", err)
	}

	// wstunnel's own UDP BND.ADDR reaches hev verbatim.
	control, rep, bnd := rawSocks(t, h.fd.Addr(), socksCmdUDPAssoc, netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
	if rep != RepSucceeded || bnd != udpBnd {
		t.Fatalf("relayed UDP ASSOCIATE: rep %#x bnd %s", rep, bnd)
	}
	_ = control.Close()

	// Listener gone: 0x03, not a hang.
	_ = rev.Close()
	_, err = h.dialer(t).DialContext(context.Background(), "tcp", testDst.String())
	if err == nil || !strings.Contains(err.Error(), "network unreachable") {
		t.Fatalf("relay with dead listener: %v", err)
	}
	h.fd.SetRelay(nil)
	if state, _, _ := h.fd.Attached(); state != proto.ExitDisconnected {
		t.Fatalf("Attached after detach = %s", state)
	}
}

func TestFrontDoorNativeWinsOverRelay(t *testing.T) {
	h := newFrontDoor(t)
	gate := (&fakeGate{}).withPeer("203.0.113.9")
	gate.set(true)
	h.fd.SetRelay(gate)
	f := NewFakeExit()
	var opens atomic.Int32
	f.OnOpen = func(uint32, byte, netip.AddrPort) { opens.Add(1) }
	f.Connect = func(netip.AddrPort) (net.Conn, byte) { return nil, RepConnRefused }
	h.attachNative(t, f)
	_, err := h.dialer(t).DialContext(context.Background(), "tcp", testDst.String())
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("CONNECT: %v", err)
	}
	if opens.Load() != 1 {
		t.Fatal("native backend was not preferred")
	}
}

func TestFrontDoorStopClosesConnections(t *testing.T) {
	h := newFrontDoor(t)
	f := NewFakeExit()
	f.Connect = pipeConnect
	h.attachNative(t, f)
	c, err := h.dialer(t).DialContext(context.Background(), "tcp", testDst.String())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		h.fd.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop hung on an active connection")
	}
	_ = c.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := c.Read(make([]byte, 1)); err == nil || isTimeout(err) {
		t.Fatalf("hev connection survived Stop: %v", err)
	}
	if _, err := net.DialTimeout("tcp", h.fd.Addr(), 200*time.Millisecond); err == nil {
		t.Fatal("listener still accepting after Stop")
	}
	// Start again works and the second Stop is harmless.
	if err := h.fd.Start(); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if h.fd.Addr() == "" {
		t.Fatal("restart did not bind")
	}
}

func TestFrontDoorStartTwiceAndPortTaken(t *testing.T) {
	h := newFrontDoor(t)
	if err := h.fd.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	other := NewFrontDoor(MustSlot("1"), nil, nil)
	old := socksListenAddr
	socksListenAddr = func(Slot) string { return h.fd.Addr() }
	err := other.Start()
	socksListenAddr = old
	if err == nil {
		other.Stop()
		t.Fatal("Start on a taken port succeeded")
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) && !strings.Contains(err.Error(), "address already in use") {
		t.Fatalf("error %v", err)
	}
}

func TestSocksUDPHeaderRoundTrip(t *testing.T) {
	src := netip.MustParseAddrPort("[2001:db8::7]:5353")
	b := encodeSocksUDP(src, []byte("x"))
	got, payload, ok := decodeSocksUDP(b)
	if !ok || got != src || string(payload) != "x" {
		t.Fatalf("decode = %s %q %v", got, payload, ok)
	}
	if _, _, ok := decodeSocksUDP([]byte{0, 0, 0, atypDomain, 3, 'a', 'b', 'c', 0, 53}); ok {
		t.Fatal("domain datagram accepted")
	}
	if _, _, ok := decodeSocksUDP(b[:5]); ok {
		t.Fatal("short datagram accepted")
	}
	port := binary.BigEndian.Uint16(b[3+17:])
	if port != 5353 {
		t.Fatalf("port %d", port)
	}
}
