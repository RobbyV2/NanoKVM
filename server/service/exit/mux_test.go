package exit

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

// tcpPair returns two ends of one loopback TCP connection, so tests can
// half-close (net.Pipe cannot).
func tcpPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			accepted <- nil
			return
		}
		accepted <- c
	}()
	a, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	b := <-accepted
	if b == nil {
		t.Fatal("accept failed")
	}
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
	return a, b
}

// pipeConnect is a FakeExit.Connect that hands the mux one end of a fresh
// net.Pipe for any destination; the other end is dropped.
func pipeConnect(netip.AddrPort) (net.Conn, byte) { a, _ := net.Pipe(); return a, 0 }

// testDst is a TEST-NET-1 destination the policy allows; the fake maps it
// to whatever loopback socket a test wants.
var testDst = netip.MustParseAddrPort("192.0.2.1:80")

type testCounter struct {
	up, down atomic.Int64
}

func (c *testCounter) AddUp(n int64)   { c.up.Add(n) }
func (c *testCounter) AddDown(n int64) { c.down.Add(n) }
func (c *testCounter) Totals() proto.ExitBytes {
	return proto.ExitBytes{Up: uint64(c.up.Load()), Down: uint64(c.down.Load())}
}

// muxHarness is a Mux behind an httptest server. The handler takes the
// peer's address from X-Test-Remote when present so tests can play several
// exits from one machine.
type muxHarness struct {
	mux      *Mux
	srv      *httptest.Server
	url      string
	sessions chan Backend
	closes   chan string
	bytes    *testCounter
	policy   atomic.Pointer[Policy]
}

func newMuxHarness(t *testing.T) *muxHarness {
	t.Helper()
	h := &muxHarness{
		sessions: make(chan Backend, 16),
		closes:   make(chan string, 16),
		bytes:    &testCounter{},
	}
	h.policy.Store(&Policy{AllowPrivate: true})
	h.mux = NewMux(MustSlot("0"), MuxHooks{
		OnSession: func(s Backend) { h.sessions <- s },
		OnClose:   func(_ Backend, reason string) { h.closes <- reason },
		Policy:    func() Policy { return *h.policy.Load() },
		Bytes:     h.bytes,
	})
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if remote := r.Header.Get("X-Test-Remote"); remote != "" {
			r.RemoteAddr = remote
		}
		h.mux.ServeNative(w, r)
	}))
	h.url = "ws" + strings.TrimPrefix(h.srv.URL, "http") + "/exit/0/native"
	t.Cleanup(func() {
		h.mux.CloseAll("test over")
		h.srv.Close()
	})
	return h
}

func (h *muxHarness) dial(t *testing.T, f *FakeExit, remote string) Backend {
	t.Helper()
	hdr := http.Header{}
	if remote != "" {
		hdr.Set("X-Test-Remote", remote)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.Dial(ctx, h.url, hdr); err != nil {
		t.Fatalf("fake exit dial: %v", err)
	}
	if f.NoHello {
		return nil
	}
	select {
	case s := <-h.sessions:
		return s
	case <-time.After(3 * time.Second):
		t.Fatal("OnSession did not fire")
		return nil
	}
}

func (h *muxHarness) waitClose(t *testing.T) string {
	t.Helper()
	select {
	case r := <-h.closes:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("OnClose did not fire")
		return ""
	}
}

// echoServer is a loopback TCP echo that also half-closes after the client does.
func echoServer(t *testing.T) netip.AddrPort {
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
				_, _ = io.Copy(c, c)
				_ = c.(*net.TCPConn).CloseWrite()
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).AddrPort()
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestMuxHandshake(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.Hostname = "robs-laptop"
	f.OS = "darwin"
	s := h.dial(t, f, "203.0.113.7:4242")

	sw, cw, ms := f.Welcome()
	if sw != StreamWindow || cw != ConnWindow || ms != MaxStreams {
		t.Fatalf("WELCOME = %d/%d/%d", sw, cw, ms)
	}
	p := s.Peer()
	if p.Addr != "203.0.113.7" || p.Hostname != "robs-laptop" || p.OS != "darwin" || p.Transport != proto.ExitModeNative {
		t.Fatalf("peer = %+v", p)
	}
	if s.ConnectedAt().IsZero() {
		t.Fatal("ConnectedAt zero")
	}
	if h.mux.Current() != s {
		t.Fatal("Current() is not the session")
	}
	f.Close()
	h.waitClose(t)
	waitFor(t, "Current nil", func() bool { return h.mux.Current() == nil })
}

func TestMuxConnectRoundTrip(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	var opened atomic.Int32
	f.OnOpen = func(id uint32, p byte, _ netip.AddrPort) {
		if id%2 == 0 {
			t.Errorf("even stream id %d", id)
		}
		if p == protoTCP {
			opened.Add(1)
		}
	}
	echo := echoServer(t)
	f.Connect = func(dst netip.AddrPort) (net.Conn, byte) {
		if dst != testDst {
			t.Errorf("OPEN for %s, want %s", dst, testDst)
		}
		return defaultConnect(echo)
	}
	s := h.dial(t, f, "")

	conn, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatalf("DialTCP: %v", err)
	}
	if conn.RemoteAddr().String() != testDst.String() {
		t.Fatalf("RemoteAddr = %s", conn.RemoteAddr())
	}
	payload := make([]byte, 300<<10) // more than two stream windows
	_, _ = rand.Read(payload)
	go func() {
		_, _ = conn.Write(payload)
		_ = conn.(interface{ CloseWrite() error }).CloseWrite()
	}()
	got, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("echo mismatch: %d vs %d bytes", len(got), len(payload))
	}
	if opened.Load() != 1 {
		t.Fatalf("OPEN count %d", opened.Load())
	}
	// Both EOFs exchanged: the stream is released on both sides without RST.
	sess := s.(*session)
	waitFor(t, "stream release", func() bool { return sess.streamCount() == 0 && f.Streams() == 0 })
	if err := conn.Close(); err != nil {
		t.Fatalf("Close after clean EOF: %v", err)
	}
	tot := h.bytes.Totals()
	if tot.Up != uint64(len(payload)) || tot.Down != uint64(len(payload)) {
		t.Fatalf("bytes = %+v", tot)
	}
	// The exit credits the connection window only once half of it has been
	// consumed, so after 300 KiB exactly that much is still outstanding.
	if got := sess.connSendAvailable(); got != int64(ConnWindow-len(payload)) {
		t.Fatalf("conn credit after round trip = %d, want %d", got, ConnWindow-len(payload))
	}
}

// TestMuxLargeTransfer moves more than the connection window in each
// direction; it would stall at 4 MiB if either side failed to credit stream 0.
func TestMuxLargeTransfer(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	echo := echoServer(t)
	f.Connect = func(netip.AddrPort) (net.Conn, byte) { return defaultConnect(echo) }
	var connWindows atomic.Int64
	f.OnFrame = func(typ byte, id uint32, payload []byte) {
		if typ == frameWindow && id == 0 {
			connWindows.Add(int64(uint32(payload[0])<<24 | uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3])))
		}
	}
	s := h.dial(t, f, "")
	conn, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	const size = 5 << 20
	payload := make([]byte, size)
	_, _ = rand.Read(payload)
	go func() {
		_, _ = conn.Write(payload)
		_ = conn.(interface{ CloseWrite() error }).CloseWrite()
	}()
	done := make(chan []byte, 1)
	go func() {
		got, _ := io.ReadAll(conn)
		done <- got
	}()
	select {
	case got := <-done:
		if !bytes.Equal(got, payload) {
			t.Fatalf("mismatch: %d vs %d bytes", len(got), len(payload))
		}
	case <-time.After(20 * time.Second):
		t.Fatal("transfer stalled")
	}
	if connWindows.Load() < ConnWindow/2 {
		t.Fatalf("kvm sent only %d bytes of connection WINDOW", connWindows.Load())
	}
}

func TestMuxHalfCloseFromExit(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	// The destination writes a greeting and half-closes; then reads what the
	// kvm sends and records it.
	var mu sync.Mutex
	var received []byte
	a, b := tcpPair(t)
	f.Connect = func(netip.AddrPort) (net.Conn, byte) {
		go func() {
			_, _ = b.Write([]byte("hello from exit"))
			_ = b.(*net.TCPConn).CloseWrite()
		}()
		go func() {
			data, _ := io.ReadAll(b)
			mu.Lock()
			received = data
			mu.Unlock()
			_ = b.Close()
		}()
		return a, 0
	}
	s := h.dial(t, f, "")
	conn, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(conn)
	if err != nil || string(got) != "hello from exit" {
		t.Fatalf("read = %q, %v", got, err)
	}
	// Our direction is still open after the exit's EOF.
	if _, err := conn.Write([]byte("reply")); err != nil {
		t.Fatalf("write after remote EOF: %v", err)
	}
	_ = conn.(interface{ CloseWrite() error }).CloseWrite()
	waitFor(t, "exit received reply", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return string(received) == "reply"
	})
	waitFor(t, "release", func() bool { return s.(*session).streamCount() == 0 })
}

func TestMuxOpenFailMapping(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	var reason atomic.Uint32
	f.Connect = func(netip.AddrPort) (net.Conn, byte) { return nil, byte(reason.Load()) }
	s := h.dial(t, f, "")
	for _, rep := range []byte{RepConnRefused, RepHostUnreach, RepNetworkUnreach, RepTTLExpired, RepAddrTypeNotSupp} {
		reason.Store(uint32(rep))
		_, err := s.DialTCP(context.Background(), testDst)
		if RepOf(err) != rep {
			t.Fatalf("reason %#x mapped to %#x (%v)", rep, RepOf(err), err)
		}
	}
	waitFor(t, "no streams", func() bool { return s.(*session).streamCount() == 0 })

	// The default connect maps a real refusal through the errno table.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	closed := ln.Addr().(*net.TCPAddr).AddrPort()
	_ = ln.Close()
	if c, rep := defaultConnect(closed); c != nil || rep != RepConnRefused {
		t.Fatalf("refused port mapped to %#x", rep)
	}
}

func TestMuxPolicyAtOpen(t *testing.T) {
	h := newMuxHarness(t)
	h.policy.Store(&Policy{})
	f := NewFakeExit()
	s := h.dial(t, f, "")
	for _, dst := range []string{"10.1.2.3:80", "127.0.0.1:80", "198.18.0.1:80", "[fe80::1]:80"} {
		_, err := s.DialTCP(context.Background(), netip.MustParseAddrPort(dst))
		if RepOf(err) != RepNotAllowed {
			t.Fatalf("%s: rep %#x", dst, RepOf(err))
		}
	}
	h.policy.Store(&Policy{AllowPrivate: true})
	f.Connect = pipeConnect
	if _, err := s.DialTCP(context.Background(), netip.MustParseAddrPort("10.1.2.3:80")); err != nil {
		t.Fatalf("allowPrivate: %v", err)
	}
}

func TestMuxOpenTimeout(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	block := make(chan struct{})
	f.Connect = func(netip.AddrPort) (net.Conn, byte) { <-block; return nil, RepGeneralFailure }
	defer close(block)
	s := h.dial(t, f, "")
	var rsts atomic.Int32
	f.OnFrame = func(typ byte, _ uint32, _ []byte) {
		if typ == frameRST {
			rsts.Add(1)
		}
	}
	start := time.Now()
	_, err := s.DialTCP(context.Background(), testDst)
	if RepOf(err) != RepTTLExpired {
		t.Fatalf("rep %#x, err %v", RepOf(err), err)
	}
	if d := time.Since(start); d < openTimeout || d > openTimeout*3 {
		t.Fatalf("timed out after %s", d)
	}
	waitFor(t, "RST after timeout", func() bool { return rsts.Load() == 1 })
	if s.(*session).streamCount() != 0 {
		t.Fatal("stream not released after timeout")
	}
}

func TestMuxCreditExhaustionAndWindow(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.ManualWindow = true
	sink := make(chan int, 1024)
	f.Connect = func(netip.AddrPort) (net.Conn, byte) {
		a, b := net.Pipe()
		go func() {
			buf := make([]byte, 64<<10)
			for {
				n, err := b.Read(buf)
				if n > 0 {
					sink <- n
				}
				if err != nil {
					return
				}
			}
		}()
		return a, 0
	}
	s := h.dial(t, f, "")
	conn, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	// Exactly one window goes through without credit coming back.
	first := make([]byte, StreamWindow)
	if _, err := conn.Write(first); err != nil {
		t.Fatalf("write window: %v", err)
	}
	drained := 0
	waitFor(t, "window drained by exit", func() bool {
		for {
			select {
			case n := <-sink:
				drained += n
			default:
				return drained == StreamWindow
			}
		}
	})
	// The next byte blocks until a WINDOW arrives.
	wrote := make(chan error, 1)
	go func() {
		_, err := conn.Write([]byte{1})
		wrote <- err
	}()
	select {
	case err := <-wrote:
		t.Fatalf("write did not block without credit (err=%v)", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := f.SendWindow(f.StreamIDs()[0], 1); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-wrote:
		if err != nil {
			t.Fatalf("write after WINDOW: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("write did not resume after WINDOW")
	}
	// A short WINDOW deadline surfaces as a timeout, not a hang.
	_ = conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
	_, err = conn.Write([]byte{2})
	if !errors.Is(err, os.ErrDeadlineExceeded) && !isTimeout(err) {
		t.Fatalf("write without credit under deadline: %v", err)
	}
	_ = conn.Close()
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func TestMuxKvmSendsWindowAtHalf(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	src := make([]byte, StreamWindow) // one full window from the exit
	a, b := tcpPair(t)
	f.Connect = func(netip.AddrPort) (net.Conn, byte) {
		go func() {
			_, _ = b.Write(src)
			_ = b.(*net.TCPConn).CloseWrite()
		}()
		return a, 0
	}
	var windows atomic.Int64
	var connWindows atomic.Int64
	f.OnFrame = func(typ byte, id uint32, payload []byte) {
		if typ == frameWindow {
			inc := int64(uint32(payload[0])<<24 | uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3]))
			if id == 0 {
				connWindows.Add(inc)
			} else {
				windows.Add(inc)
			}
		}
	}
	s := h.dial(t, f, "")
	conn, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(conn)
	if err != nil || len(got) != len(src) {
		t.Fatalf("read %d bytes, %v", len(got), err)
	}
	// WINDOW fires once half the window has been passed to hev; what is
	// consumed after that (less than half) is not credited before the
	// stream is released, so the total lands in [half, full].
	waitFor(t, "stream WINDOW", func() bool { return windows.Load() >= StreamWindow/2 })
	if w := windows.Load(); w > StreamWindow {
		t.Fatalf("over-credited: %d", w)
	}
	// The connection window is only credited at half of ConnWindow; one
	// stream window is far below that, so no conn WINDOW yet.
	if connWindows.Load() != 0 {
		t.Fatalf("premature connection WINDOW of %d", connWindows.Load())
	}
}

func TestMuxMaxStreams(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.Connect = pipeConnect
	s := h.dial(t, f, "")
	conns := make([]net.Conn, 0, MaxStreams)
	for i := 0; i < MaxStreams; i++ {
		c, err := s.DialTCP(context.Background(), testDst)
		if err != nil {
			t.Fatalf("stream %d: %v", i, err)
		}
		conns = append(conns, c)
	}
	_, err := s.DialTCP(context.Background(), testDst)
	if RepOf(err) != RepGeneralFailure {
		t.Fatalf("stream %d: rep %#x (%v)", MaxStreams+1, RepOf(err), err)
	}
	_ = conns[0].Close()
	waitFor(t, "slot freed", func() bool { return s.(*session).streamCount() == MaxStreams-1 })
	if _, err := s.DialTCP(context.Background(), testDst); err != nil {
		t.Fatalf("after close: %v", err)
	}
	for _, c := range conns[1:] {
		_ = c.Close()
	}
}

func TestMuxUDPRoundTrip(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	s := h.dial(t, f, "")

	echo, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		buf := make([]byte, 2048)
		for {
			n, addr, err := echo.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = echo.WriteTo(append([]byte("echo:"), buf[:n]...), addr)
		}
	}()
	dst := echo.LocalAddr().(*net.UDPAddr).AddrPort()

	us, err := s.OpenUDP(context.Background())
	if err != nil {
		t.Fatalf("OpenUDP: %v", err)
	}
	if err := us.Send(dst, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	src, payload, err := us.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if src != dst || string(payload) != "echo:ping" {
		t.Fatalf("got %s from %s", payload, src)
	}
	// Oversized datagrams are dropped by the sender, not sent.
	if err := us.Send(dst, make([]byte, MaxFrame)); err != nil {
		t.Fatalf("oversized send: %v", err)
	}
	_ = us.Close()
	waitFor(t, "udp stream released", func() bool { return s.(*session).streamCount() == 0 && f.Streams() == 0 })
	tot := h.bytes.Totals()
	if tot.Up != 4 || tot.Down != 9 {
		t.Fatalf("udp bytes = %+v", tot)
	}
}

func TestMuxSupersede(t *testing.T) {
	h := newMuxHarness(t)
	f1 := NewFakeExit()
	s1 := h.dial(t, f1, "198.51.100.1:1000")
	f2 := NewFakeExit()
	s2 := h.dial(t, f2, "198.51.100.2:1000")

	if !f1.Wait(3 * time.Second) {
		t.Fatal("first exit not closed")
	}
	select {
	case <-s1.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("first session Done not closed")
	}
	reason := h.waitClose(t)
	if !strings.Contains(reason, "superseded by 198.51.100.2") {
		t.Fatalf("close reason %q", reason)
	}
	if h.mux.Current() != s2 {
		t.Fatal("Current is not the second session")
	}
	select {
	case <-s2.Done():
		t.Fatal("second session closed")
	default:
	}
}

func TestMuxPinPeer(t *testing.T) {
	h := newMuxHarness(t)
	f1 := NewFakeExit()
	s1 := h.dial(t, f1, "198.51.100.1:1000")
	h.mux.SetPinPeer(true)

	f2 := NewFakeExit()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	hdr := http.Header{"X-Test-Remote": {"198.51.100.2:1000"}}
	err := f2.Dial(ctx, h.url, hdr)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("pinned: dial from other host = %v", err)
	}
	select {
	case <-s1.Done():
		t.Fatal("pinned session was closed by a refused peer")
	default:
	}
	// Same host may supersede.
	f3 := NewFakeExit()
	h.dial(t, f3, "198.51.100.1:2000")
	if !f1.Wait(3 * time.Second) {
		t.Fatal("same-host supersede did not close the first")
	}
	// Clearing the pin admits a new host.
	h.mux.SetPinPeer(false)
	f4 := NewFakeExit()
	h.dial(t, f4, "198.51.100.9:1000")
}

func TestMuxHelloTimeout(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.NoHello = true
	h.dial(t, f, "")
	waitFor(t, "connecting", func() bool { return h.mux.Connecting() })
	start := time.Now()
	if !f.Wait(3 * time.Second) {
		t.Fatal("silent exit was not closed")
	}
	if d := time.Since(start); d > helloTimeout*4 {
		t.Fatalf("closed after %s", d)
	}
	if h.mux.Current() != nil || h.mux.Connecting() {
		t.Fatal("session state left behind")
	}
	select {
	case <-h.sessions:
		t.Fatal("OnSession fired without HELLO")
	default:
	}
}

func TestMuxPingTimeout(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.NoPong = true
	s := h.dial(t, f, "")
	start := time.Now()
	select {
	case <-s.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("deaf exit was not closed")
	}
	d := time.Since(start)
	want := pingInterval * time.Duration(pingMisses+1)
	if d < want-pingInterval || d > want*3 {
		t.Fatalf("closed after %s, expected about %s", d, want)
	}
	if !strings.Contains(h.waitClose(t), "ping timeout") {
		t.Fatal("close reason is not the ping timeout")
	}
}

func TestMuxPongKeepsSessionAlive(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	s := h.dial(t, f, "")
	select {
	case <-s.Done():
		t.Fatal("responsive exit was closed")
	case <-time.After(pingInterval * time.Duration(pingMisses+3)):
	}
}

func TestMuxRSTAllOnWSLoss(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.Connect = pipeConnect
	s := h.dial(t, f, "")
	var conns []net.Conn
	for i := 0; i < 3; i++ {
		c, err := s.DialTCP(context.Background(), testDst)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, c)
	}
	us, err := s.OpenUDP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	readErr := make(chan error, len(conns))
	for _, c := range conns {
		go func(c net.Conn) {
			_, err := c.Read(make([]byte, 1))
			readErr <- err
		}(c)
	}
	f.Close() // the WS drops
	for range conns {
		select {
		case err := <-readErr:
			if err == nil || errors.Is(err, io.EOF) {
				t.Fatalf("stream read after WS loss = %v, want an abort", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("stream read still blocked after WS loss")
		}
	}
	if _, err := conns[0].Write([]byte{1}); err == nil {
		t.Fatal("write succeeded after WS loss")
	}
	if _, _, err := us.Recv(context.Background()); err == nil {
		t.Fatal("udp Recv succeeded after WS loss")
	}
	select {
	case <-s.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed")
	}
	h.waitClose(t)
}

func TestMuxProtocolErrorsCloseSession(t *testing.T) {
	cases := []struct {
		name string
		act  func(f *FakeExit, s Backend)
	}{
		{"text message", func(f *FakeExit, _ Backend) { _ = f.SendText("hi") }},
		{"second HELLO", func(f *FakeExit, _ Backend) { _ = f.SendRaw(frameHello, 0, encodeHello(hello{version: 1})) }},
		{"OPEN from exit", func(f *FakeExit, _ Backend) { _ = f.SendRaw(frameOpen, 7, []byte{protoTCP}) }},
		{"DATA before OPENED", func(f *FakeExit, s Backend) {
			block := make(chan struct{})
			f.Connect = func(netip.AddrPort) (net.Conn, byte) { <-block; return nil, 1 }
			go func() { _, _ = s.DialTCP(context.Background(), testDst) }()
			var id uint32
			for id == 0 {
				for _, sid := range s.(*session).streamIDs() {
					id = sid
				}
				time.Sleep(time.Millisecond)
			}
			_ = f.SendRaw(frameData, id, []byte("early"))
			close(block)
		}},
		{"oversized payload", func(f *FakeExit, _ Backend) { _ = f.SendRaw(frameWindow, 0, make([]byte, MaxFrame+1)) }},
		{"unknown type", func(f *FakeExit, _ Backend) { _ = f.SendRaw(0x7f, 0, nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newMuxHarness(t)
			f := NewFakeExit()
			s := h.dial(t, f, "")
			tc.act(f, s)
			select {
			case <-s.Done():
			case <-time.After(3 * time.Second):
				t.Fatal("session survived the protocol error")
			}
			reason := h.waitClose(t)
			if !strings.Contains(reason, "protocol error") && !strings.Contains(reason, "read:") {
				t.Fatalf("reason %q", reason)
			}
		})
	}
}

func TestMuxDataAfterEOFIsRST(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.Connect = pipeConnect
	s := h.dial(t, f, "")
	conn, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	id := s.(*session).streamIDs()[0]
	var rst atomic.Bool
	f.OnFrame = func(typ byte, fid uint32, _ []byte) {
		if typ == frameRST && fid == id {
			rst.Store(true)
		}
	}
	_ = f.SendRaw(frameEOF, id, nil)
	_ = f.SendRaw(frameData, id, []byte("late"))
	waitFor(t, "RST for DATA after EOF", rst.Load)
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("read succeeded on reset stream")
	}
	select {
	case <-s.Done():
		t.Fatal("session closed for a stream-level error")
	default:
	}
}

func TestMuxCloseAll(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	s := h.dial(t, f, "")
	h.mux.CloseAll("operator disconnect")
	select {
	case <-s.Done():
	default:
		t.Fatal("CloseAll returned before Done")
	}
	if r := h.waitClose(t); r != "operator disconnect" {
		t.Fatalf("reason %q", r)
	}
	if !f.Wait(3 * time.Second) {
		t.Fatal("exit did not see the close")
	}
}

func (s *session) streamIDs() []uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]uint32, 0, len(s.streams))
	for id := range s.streams {
		out = append(out, id)
	}
	return out
}

// connWindowsOf sums the stream-0 WINDOW increments the fake receives.
func connWindowsOf(f *FakeExit) *atomic.Int64 {
	var total atomic.Int64
	f.OnFrame = func(typ byte, id uint32, payload []byte) {
		if typ == frameWindow && id == 0 && len(payload) >= 4 {
			total.Add(int64(uint32(payload[0])<<24 | uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3])))
		}
	}
	return &total
}

// TestMuxCreditsDataOnReleasedStream: DATA that raced the kvm's RST was
// debited from the exit's connection credit when it left the exit, so the
// kvm must still account it on stream 0 even though the stream is gone.
// UDP DATA is never credited, released or not.
func TestMuxCreditsDataOnReleasedStream(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.Connect = pipeConnect
	windows := connWindowsOf(f)
	s := h.dial(t, f, "")
	sess := s.(*session)

	c, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	id := sess.streamIDs()[0]
	_ = c.Close()
	waitFor(t, "stream released on both sides", func() bool { return sess.streamCount() == 0 && f.Streams() == 0 })

	chunk := make([]byte, MaxFrame)
	for i := 0; i < ConnWindow/2/MaxFrame; i++ {
		if err := f.SendRaw(frameData, id, chunk); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, "stream-0 WINDOW for DATA on a released id", func() bool { return windows.Load() >= ConnWindow/2 })
	select {
	case <-s.Done():
		t.Fatal("DATA on a released id closed the session")
	default:
	}
	sess.flowMu.Lock()
	outstanding := sess.connOutstanding
	sess.flowMu.Unlock()
	if outstanding != 0 {
		t.Fatalf("connOutstanding = %d after DATA on a released id", outstanding)
	}

	// The lookup -> release -> enqueue race inside onData ends the same way.
	c2, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	st := c2.(*stream)
	_ = c2.Close()
	if res := st.enqueue([]byte("x")); res != enqueueReleased {
		t.Fatalf("enqueue on a released stream = %d", res)
	}
	before := windows.Load()
	if err := sess.onData(st, make([]byte, ConnWindow/2)); err != nil {
		t.Fatalf("onData on a released stream: %v", err)
	}
	waitFor(t, "stream-0 WINDOW for the enqueue race", func() bool { return windows.Load() >= before+ConnWindow/2 })
	sess.flowMu.Lock()
	outstanding = sess.connOutstanding
	sess.flowMu.Unlock()
	if outstanding != 0 {
		t.Fatalf("connOutstanding = %d after the enqueue race", outstanding)
	}

	// UDP: released id, DATA with a source header, no credit.
	us, err := s.OpenUDP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	uid := sess.streamIDs()[0]
	_ = us.Close()
	waitFor(t, "udp stream released", func() bool { return sess.streamCount() == 0 && f.Streams() == 0 })
	before = windows.Load()
	dgram := append(encodeAddrPort(testDst), make([]byte, MaxFrame-64)...)
	for i := 0; i < ConnWindow/2/MaxFrame+1; i++ {
		if err := f.SendRaw(frameData, uid, dgram); err != nil {
			t.Fatal(err)
		}
	}
	// One more TCP frame proves the reader has processed everything above.
	fence := make(chan struct{})
	f.OnFrame = func(typ byte, id uint32, payload []byte) {
		if typ == frameWindow && id == 0 {
			select {
			case <-fence:
			default:
				close(fence)
			}
		}
	}
	_ = f.SendRaw(frameData, id, make([]byte, MaxFrame))
	for i := 0; i < ConnWindow/2/MaxFrame; i++ {
		_ = f.SendRaw(frameData, id, chunk)
	}
	select {
	case <-fence:
	case <-time.After(3 * time.Second):
		t.Fatal("no WINDOW after the fence frames")
	}
	if windows.Load() != before {
		t.Fatalf("UDP DATA on a released id was credited: %d -> %d", before, windows.Load())
	}
}

// TestMuxCloseAfterCleanReleaseCreditsQueuedBytes: both EOFs exchanged, the
// exit's bytes still queued for hev, then hev dies before reading them. The
// bytes are dropped and the connection window credited, exactly as an abort
// on a live stream would.
func TestMuxCloseAfterCleanReleaseCreditsQueuedBytes(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	a, b := tcpPair(t)
	f.Connect = func(netip.AddrPort) (net.Conn, byte) { return a, 0 }
	var rsts atomic.Int32
	f.OnFrame = func(typ byte, _ uint32, _ []byte) {
		if typ == frameRST {
			rsts.Add(1)
		}
	}
	s := h.dial(t, f, "")
	sess := s.(*session)
	c, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	st := c.(*stream)
	const size = 64 << 10
	if _, err := b.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
	_ = b.(*net.TCPConn).CloseWrite()
	waitFor(t, "exit EOF and bytes queued", func() bool {
		st.mu.Lock()
		defer st.mu.Unlock()
		return st.remoteEOF && st.recvBytes == size
	})
	if err := st.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "clean release", func() bool { return sess.streamCount() == 0 })
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	sess.flowMu.Lock()
	outstanding, consumed := sess.connOutstanding, sess.connConsumed
	sess.flowMu.Unlock()
	if outstanding != 0 || consumed != size {
		t.Fatalf("connOutstanding=%d connConsumed=%d after clean release then Close, want 0 and %d", outstanding, consumed, size)
	}
	if rsts.Load() != 0 {
		t.Fatal("Close after a clean release sent RST")
	}
	if _, err := c.Read(make([]byte, 1)); err == nil {
		t.Fatal("read after Close returned data")
	}
}

// TestFakeExitCreditsDataOnReleasedStream holds the Go oracle to the same
// rule as the kvm: TCP DATA arriving for an id the exit has already dropped
// is credited on stream 0; UDP DATA is not.
func TestFakeExitCreditsDataOnReleasedStream(t *testing.T) {
	h := newMuxHarness(t)
	f := NewFakeExit()
	f.Connect = pipeConnect
	s := h.dial(t, f, "")
	sess := s.(*session)
	c, err := s.DialTCP(context.Background(), testDst)
	if err != nil {
		t.Fatal(err)
	}
	id := sess.streamIDs()[0]
	us, err := s.OpenUDP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var uid uint32
	for _, sid := range sess.streamIDs() {
		if sid != id {
			uid = sid
		}
	}
	// The exit aborts both streams on its side (a socket error, say).
	f.drop(f.lookup(id))
	f.drop(f.lookup(uid))
	before := sess.connSendAvailable()

	// DATA the kvm had in flight lands on the dropped ids.
	chunk := make([]byte, MaxFrame)
	for i := 0; i < ConnWindow/2/MaxFrame; i++ {
		if err := f.handle(frame{typ: frameData, id: uid, payload: append(encodeAddrPort(testDst), chunk[:MaxFrame-64]...)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.handle(frame{typ: frameData, id: id, payload: chunk}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if got := sess.connSendAvailable(); got != before {
		t.Fatalf("connection credit moved before half a window: %d -> %d", before, got)
	}
	for i := 1; i < ConnWindow/2/MaxFrame; i++ {
		if err := f.handle(frame{typ: frameData, id: id, payload: chunk}); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, "stream-0 WINDOW from the fake", func() bool { return sess.connSendAvailable() == before+ConnWindow/2 })
	_ = c.Close()
	_ = us.Close()
}
