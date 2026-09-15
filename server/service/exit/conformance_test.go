//go:build conformance

package exit

// The conformance suite of the plan's Wave 2: the real Python and Perl nexit/1
// clients, templated exactly as the token-gated surface would serve them, run
// as subprocesses against the real Mux and FrontDoor, and traffic is driven
// through the front door with golang.org/x/net/proxy as hev would.
//
//	go test -race -tags conformance -run Conformance ./service/exit/ -v
//
// It needs python3 and perl on PATH and one non-loopback IPv4 address on this
// host: the destination policy (D21) always denies 127/8, in the front door,
// in the mux and in the clients, so the echo servers listen on the host's LAN
// address (the way clients/testdata/host_tests.py takes --bind) and the slot
// runs with allowPrivate so an RFC 1918 LAN passes. Without an interpreter or
// an address the suite skips and says why. TestMain keeps the production
// protocol timers when the -run filter names this suite.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/net/proxy"
)

const (
	conformanceSessionWait = 20 * time.Second
	conformanceBulkSize    = 20 << 20
)

var conformanceClients = []struct {
	name, interpreter string
}{
	{"client.py", "python3"},
	{"client.pl", "perl"},
}

func TestConformanceClients(t *testing.T) {
	if pingInterval != PingInterval {
		t.Skip("protocol timers are shortened; run with -run Conformance so TestMain keeps the production values")
	}
	for _, c := range conformanceClients {
		t.Run(strings.TrimPrefix(c.interpreter, "client."), func(t *testing.T) {
			h := newConformance(t)
			if _, err := exec.LookPath(c.interpreter); err != nil {
				t.Skipf("%s not on PATH: %v", c.interpreter, err)
			}
			first := h.startClient(t, c.name, c.interpreter)
			t.Run("tcp-echo", func(t *testing.T) { h.tcpEcho(t, 1<<20) })
			t.Run("bulk-20MB-both-ways", func(t *testing.T) { h.tcpEcho(t, conformanceBulkSize) })
			t.Run("half-close", h.halfClose)
			t.Run("open-fail-refused", h.openFailRefused)
			t.Run("udp-echo", h.udpEcho)
			t.Run("supersede", func(t *testing.T) { h.supersede(t, first, c.name, c.interpreter) })
		})
	}
}

// conformance is one kvm side: mux, front door, the httptest listener the
// clients dial, and the templated identity they are served with.
type conformance struct {
	lan      netip.Addr
	slot     Slot
	cfg      Config
	origin   Origin
	mux      *Mux
	door     *FrontDoor
	srv      *httptest.Server
	sessions chan Backend
	closes   chan string
	dialer   proxy.ContextDialer
	authFail chan string
}

func newConformance(t *testing.T) *conformance {
	t.Helper()
	slot := MustSlot("0")
	cfg, err := DefaultConfig(slot)
	if err != nil {
		t.Fatal(err)
	}
	cfg.AllowPrivate = true
	h := &conformance{
		slot:     slot,
		cfg:      cfg,
		sessions: make(chan Backend, 16),
		closes:   make(chan string, 16),
		authFail: make(chan string, 16),
	}
	policy := func() Policy { return cfg.Policy() }
	counter := NewCounter()
	h.door = NewFrontDoor(slot, policy, counter)
	// What the manager's onSession/onClose do: point the door at the new
	// session, detach only when the current one closes.
	h.mux = NewMux(slot, MuxHooks{
		OnSession: func(s Backend) {
			h.door.SetNative(s)
			h.sessions <- s
		},
		OnClose: func(s Backend, reason string) {
			if cur := h.mux.Current(); cur == nil || cur == s {
				h.door.SetNative(nil)
			}
			h.closes <- reason
		},
		Policy: policy,
		Bytes:  counter,
	})
	// The door binds before any skip below so TestMain's wildcard check has a
	// listener to judge even when the suite cannot run.
	if err := h.door.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.door.Stop)

	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The gate is the manager's job; here the check only proves the
		// clients present the header the spec says they must (D10).
		if got := r.Header.Get("Authorization"); got != "Bearer "+cfg.Token {
			h.authFail <- fmt.Sprintf("%s %s: Authorization=%q", r.Method, r.URL.Path, got)
			writeNotFound(w)
			return
		}
		if r.URL.Path != "/exit/"+slot.ID+"/native" {
			h.authFail <- fmt.Sprintf("%s %s: unexpected path", r.Method, r.URL.Path)
			writeNotFound(w)
			return
		}
		h.mux.ServeNative(w, r)
	}))
	t.Cleanup(func() {
		h.mux.CloseAll("conformance over")
		h.srv.Close()
	})
	h.origin = Origin{Scheme: "http", Host: strings.TrimPrefix(h.srv.URL, "http://")}

	lan, ok := lanAddress()
	if !ok {
		t.Skip("no non-loopback IPv4 address on this host; the policy denies loopback destinations, so the echo servers have nowhere to listen")
	}
	h.lan = lan
	if !cfg.Policy().Allow(lan) {
		t.Skipf("the host address %s is denied by the destination policy even with allowPrivate", lan)
	}

	d, err := proxy.SOCKS5("tcp", h.door.Addr(), nil, proxy.Direct)
	if err != nil {
		t.Fatal(err)
	}
	h.dialer = d.(proxy.ContextDialer)
	return h
}

// lanAddress is the first IPv4 unicast address of an interface that is up
// and not loopback, excluding link-local.
func lanAddress() (netip.Addr, bool) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return netip.Addr{}, false
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			addr := netip.AddrFrom4([4]byte(ip4))
			if addr.IsLinkLocalUnicast() || addr.IsLoopback() || addr.IsUnspecified() {
				continue
			}
			return addr, true
		}
	}
	return netip.Addr{}, false
}

// clientProc is one running client with its log.
type clientProc struct {
	name    string
	cmd     *exec.Cmd
	log     *lockedBuffer
	session Backend
	exited  chan struct{}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// startClient templates name through RenderClient exactly as the gate does,
// runs it under interpreter and waits for the session it opens.
func (h *conformance) startClient(t *testing.T, name, interpreter string) *clientProc {
	t.Helper()
	body, ok := RenderClient(name, h.slot, h.cfg, h.origin, "")
	if !ok {
		t.Fatalf("%s is not embedded", name)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	p := &clientProc{name: name, log: &lockedBuffer{}, exited: make(chan struct{})}
	p.cmd = exec.Command(interpreter, path)
	p.cmd.Stdout = p.log
	p.cmd.Stderr = p.log
	p.cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	if err := p.cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", name, err)
	}
	go func() {
		_ = p.cmd.Wait()
		close(p.exited)
	}()
	t.Cleanup(func() {
		p.kill()
		if t.Failed() {
			t.Logf("--- %s log\n%s", name, p.log)
		}
	})

	select {
	case s := <-h.sessions:
		p.session = s
	case why := <-h.authFail:
		t.Fatalf("%s: %s\n--- client log\n%s", name, why, p.log)
	case <-p.exited:
		t.Fatalf("%s exited before connecting\n--- client log\n%s", name, p.log)
	case <-time.After(conformanceSessionWait):
		t.Fatalf("%s did not connect within %s\n--- client log\n%s", name, conformanceSessionWait, p.log)
	}
	if h.mux.Current() != p.session {
		t.Fatalf("%s: session is not the mux's current one", name)
	}
	peer := p.session.Peer()
	if peer.Hostname == "" || peer.OS == "" {
		t.Fatalf("%s: HELLO carried hostname=%q os=%q", name, peer.Hostname, peer.OS)
	}
	t.Logf("%s connected: hostname=%q os=%q", name, peer.Hostname, peer.OS)
	return p
}

func (p *clientProc) kill() {
	select {
	case <-p.exited:
		return
	default:
	}
	_ = p.cmd.Process.Signal(syscall.SIGKILL)
	<-p.exited
}

// tcpServer listens on the LAN address and runs handler per connection.
func (h *conformance) tcpServer(t *testing.T, handler func(net.Conn)) netip.AddrPort {
	t.Helper()
	ln, err := net.Listen("tcp4", net.JoinHostPort(h.lan.String(), "0"))
	if err != nil {
		t.Fatalf("listen on %s: %v", h.lan, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handler(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).AddrPort()
}

func echoHandler(c net.Conn) {
	defer c.Close()
	_, _ = io.Copy(c, c)
	closeWrite(c)
}

// dial goes through the front door with x/net/proxy as the SOCKS client. Its
// conn embeds the net.Conn interface, so CloseWrite is not reachable through
// it; tests that need a half close use dialRaw.
func (h *conformance) dial(t *testing.T, dst netip.AddrPort) net.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := h.dialer.DialContext(ctx, "tcp", dst.String())
	if err != nil {
		t.Fatalf("socks connect %s: %v", dst, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// dialRaw is the same CONNECT by hand, returning the TCP conn itself so the
// test can shut down its write side the way hev does.
func (h *conformance) dialRaw(t *testing.T, dst netip.AddrPort) *net.TCPConn {
	t.Helper()
	c, rep, _ := rawSocks(t, h.door.Addr(), socksCmdConnect, dst)
	if rep != RepSucceeded {
		_ = c.Close()
		t.Fatalf("socks connect %s: rep %#x", dst, rep)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c.(*net.TCPConn)
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// tcpEcho sends size random bytes through the echo and reads them back while
// still writing, so both directions carry the full payload at once.
func (h *conformance) tcpEcho(t *testing.T, size int) {
	dst := h.tcpServer(t, echoHandler)
	c := h.dial(t, dst)
	payload := randomBytes(t, size)
	_ = c.SetDeadline(time.Now().Add(time.Minute))

	writeErr := make(chan error, 1)
	go func() {
		_, err := c.Write(payload)
		writeErr <- err
	}()
	start := time.Now()
	got := make([]byte, size)
	if n, err := io.ReadFull(c, got); err != nil {
		t.Fatalf("read: %v (got %d of %d)", err, n, size)
	}
	if err := <-writeErr; err != nil {
		t.Fatalf("write: %v", err)
	}
	if sha(got) != sha(payload) {
		t.Fatal("echoed bytes differ from what was sent")
	}
	elapsed := time.Since(start)
	t.Logf("%d bytes each way in %s (%.1f MB/s)", size, elapsed.Round(time.Millisecond), float64(size)/elapsed.Seconds()/1e6)
}

// halfClose: the client shuts its write side, the server sees EOF only then,
// answers with everything reversed and closes; the client must still receive.
func (h *conformance) halfClose(t *testing.T) {
	dst := h.tcpServer(t, func(c net.Conn) {
		defer c.Close()
		data, err := io.ReadAll(c)
		if err != nil {
			return
		}
		for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
			data[i], data[j] = data[j], data[i]
		}
		_, _ = c.Write(data)
	})
	c := h.dialRaw(t, dst)
	_ = c.SetDeadline(time.Now().Add(time.Minute))
	payload := randomBytes(t, 300<<10)
	if _, err := c.Write(payload); err != nil {
		t.Fatal(err)
	}
	// Nothing comes back before the half close.
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if n, err := c.Read(make([]byte, 1)); err == nil {
		t.Fatalf("server answered %d byte(s) before EOF", n)
	}
	_ = c.SetReadDeadline(time.Now().Add(time.Minute))
	if err := c.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(c)
	if err != nil {
		t.Fatalf("read after half close: %v", err)
	}
	if len(got) != len(payload) {
		t.Fatalf("got %d bytes back, want %d", len(got), len(payload))
	}
	for i := range got {
		if got[i] != payload[len(payload)-1-i] {
			t.Fatalf("reply is not the reversed payload (first difference at %d)", i)
		}
	}
}

// openFailRefused: a port nothing listens on yields OPEN_FAIL 0x05, which the
// front door passes on as the SOCKS reply.
func (h *conformance) openFailRefused(t *testing.T) {
	ln, err := net.Listen("tcp4", net.JoinHostPort(h.lan.String(), "0"))
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().(*net.TCPAddr).AddrPort()
	_ = ln.Close()
	c, rep, _ := rawSocks(t, h.door.Addr(), socksCmdConnect, closed)
	_ = c.Close()
	if rep != RepConnRefused {
		t.Fatalf("SOCKS rep %#x for a refused port, want %#x", rep, RepConnRefused)
	}
}

// udpEcho drives UDP ASSOCIATE: datagrams to a LAN echo through the relay,
// replies carrying the echo's real source.
func (h *conformance) udpEcho(t *testing.T) {
	pc, err := net.ListenPacket("udp4", net.JoinHostPort(h.lan.String(), "0"))
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
	echo := pc.LocalAddr().(*net.UDPAddr).AddrPort()

	control, rep, bnd := rawSocks(t, h.door.Addr(), socksCmdUDPAssoc, netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
	t.Cleanup(func() { _ = control.Close() })
	if rep != RepSucceeded {
		t.Fatalf("UDP ASSOCIATE rep %#x", rep)
	}
	client, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	relay := net.UDPAddrFromAddrPort(bnd)
	buf := make([]byte, 64<<10)
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("ping-%d", i)
		if _, err := client.WriteTo(encodeSocksUDP(echo, []byte(msg)), relay); err != nil {
			t.Fatal(err)
		}
		_ = client.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, from, err := client.ReadFrom(buf)
		if err != nil {
			t.Fatalf("datagram %d: no reply: %v", i, err)
		}
		if from.String() != relay.String() {
			t.Fatalf("reply from %s, want the relay %s", from, relay)
		}
		src, payload, ok := decodeSocksUDP(buf[:n])
		if !ok || string(payload) != "echo:"+msg {
			t.Fatalf("reply %q ok=%v", buf[:n], ok)
		}
		if src != echo {
			t.Fatalf("reply source %s, want %s (full cone: the exit records the real source)", src, echo)
		}
	}
}

// supersede: a second client takes the slot, the first is closed with the
// reason the spec names and (killed before its 1 s backoff reconnects it)
// traffic flows through the second.
func (h *conformance) supersede(t *testing.T, first *clientProc, name, interpreter string) {
	second := h.startClient(t, name, interpreter)
	if second.session == first.session {
		t.Fatal("the second client did not open a new session")
	}
	select {
	case <-first.session.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("the first session was not closed by the supersede")
	}
	// The first client logs the close reason the kvm sent, then backs off
	// 1 s before it would reconnect and supersede the second in turn; wait
	// for the log line, then end it inside that window.
	waitFor(t, "first client logs the supersede", func() bool {
		return strings.Contains(first.log.String(), "superseded")
	})
	first.kill()
	if h.mux.Current() != second.session {
		t.Fatal("the mux's current session is not the second client")
	}
	reason := ""
	select {
	case reason = <-h.closes:
	case <-time.After(5 * time.Second):
		t.Fatal("OnClose did not fire for the superseded session")
	}
	if !strings.Contains(reason, "superseded by") {
		t.Fatalf("close reason %q", reason)
	}
	h.tcpEcho(t, 256<<10)
}
