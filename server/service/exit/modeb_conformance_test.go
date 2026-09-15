//go:build conformance

package exit

// Mode B end to end against a real wstunnel binary: the slot's restriction
// yaml as render.go writes it, `wstunnel server --restrict-config` on it, the
// real WSProxy behind a token gate shaped like Manager.Gate, the real
// FrontDoor in relay mode, and `wstunnel client -R socks5://…` as the
// operator's one-liner runs it. Traffic is driven through the front door with
// golang.org/x/net/proxy the way hev would.
//
//	go test -race -tags conformance -run ModeB ./service/exit/ -v
//
// The binary comes from $WSTUNNEL_BIN or third_party/wstunnel/target/release/
// wstunnel; the suite skips when neither exists. Like the nexit/1 suite it
// needs one non-loopback IPv4 address on the host, because the policy denies
// loopback destinations. The reverse listener port is the slot's fixed
// 10820+n (the rendered yaml pins it), so the suite picks the first slot whose
// port is free and skips if none is.

import (
	"context"
	"crypto/subtle"
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

	"NanoKVM-Server/proto"

	"golang.org/x/net/proxy"
)

const (
	modeBConnectWait = 15 * time.Second
	modeBBulkSize    = 20 << 20
)

func TestModeB(t *testing.T) {
	bin := findWstunnel(t)
	h := newModeB(t, bin)

	first := h.startClient(t, "first", h.token(), h.srv.URL, "-R", "socks5://"+h.slot.WstunnelReverseAddr())
	h.waitConnected(t, first)

	t.Run("tcp-echo", func(t *testing.T) { h.tcpEcho(t, 1<<20) })
	t.Run("bulk-20MB-both-ways", func(t *testing.T) { h.tcpEcho(t, modeBBulkSize) })
	t.Run("connect-refused-port-is-success-then-eof", h.connectRefused)
	t.Run("no-flap-after-short-relay", h.noFlapAfterShortRelay)
	t.Run("udp-associate-echo", h.udpEcho)
	t.Run("wrong-token-404-and-not-logged", h.wrongToken)
	t.Run("forward-tunnel-refused", h.forwardTunnelRefused)
	t.Run("reverse-tcp-refused", h.reverseTCPRefused)
	t.Run("second-client-same-address", func(t *testing.T) { h.secondClientSameAddress(t, first) })
	t.Run("kill-client-fails-fast", func(t *testing.T) { h.killClientFailsFast(t, first) })
	t.Run("token-regenerate", h.tokenRegenerate)
	t.Run("token-never-in-wstunnel-log", func(t *testing.T) {
		for _, tok := range h.allTokens() {
			if strings.Contains(h.server.log.String(), tok) {
				t.Fatalf("wstunnel server log contains token %q:\n%s", tok, h.server.log)
			}
		}
	})
}

func findWstunnel(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("WSTUNNEL_BIN"); p != "" {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("WSTUNNEL_BIN=%s: %v", p, err)
		}
		return p
	}
	p := filepath.Join("..", "..", "..", "third_party", "wstunnel", "target", "release", "wstunnel")
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Skip(err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Skipf("no wstunnel binary at %s and WSTUNNEL_BIN unset (cd third_party/wstunnel && cargo build --release --bin wstunnel)", abs)
	}
	return abs
}

// modeB is one kvm side plus the wstunnel server it fronts.
type modeB struct {
	bin    string
	lan    netip.Addr
	slot   Slot
	door   *FrontDoor
	proxy  *WSProxy
	srv    *httptest.Server
	server *procLog
	bytes  *Counter
	limit  *Limiter
	dialer proxy.ContextDialer
	wsAddr string

	mu      sync.Mutex
	cfg     Config
	tokens  []string
	changes []string
}

func (h *modeB) token() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cfg.Token
}

func (h *modeB) allTokens() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.tokens...)
}

// freeSlot returns the first slot whose fixed reverse port (TCP and UDP) is
// free on loopback: the rendered yaml pins it, so it cannot be ephemeral.
func freeSlot(t *testing.T) (Slot, bool) {
	t.Helper()
	for n := 0; n <= maxSlot; n++ {
		s := MustSlot(fmt.Sprint(n))
		ln, err := net.Listen("tcp", s.WstunnelReverseAddr())
		if err != nil {
			continue
		}
		pc, err := net.ListenPacket("udp", s.WstunnelReverseAddr())
		_ = ln.Close()
		if err != nil {
			continue
		}
		_ = pc.Close()
		return s, true
	}
	return Slot{}, false
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func newModeB(t *testing.T, bin string) *modeB {
	t.Helper()
	lan, ok := lanAddress()
	if !ok {
		t.Skip("no non-loopback IPv4 address on this host; the policy denies loopback destinations")
	}
	slot, ok := freeSlot(t)
	if !ok {
		t.Skip("every slot's reverse port 10820..10829 is in use on this host")
	}
	cfg, err := DefaultConfig(slot)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Mode = proto.ExitModeWstunnel
	cfg.Enabled = true
	cfg.AllowPrivate = true
	if !cfg.Policy().Allow(lan) {
		t.Skipf("host address %s is denied by the policy even with allowPrivate", lan)
	}

	h := &modeB{bin: bin, lan: lan, slot: slot, cfg: cfg, tokens: []string{cfg.Token}, bytes: NewCounter(), limit: NewLimiter()}

	// The slot's files exactly as the manager writes them (writeSlotFiles,
	// atomic rename), under a temporary ConfigDir.
	oldDir := ConfigDir
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ConfigDir = dir
	t.Cleanup(func() { ConfigDir = oldDir })
	if err := writeSlotFiles(slot, cfg, ""); err != nil {
		t.Fatal(err)
	}

	// wstunnel server as S94exit starts it, on an ephemeral port.
	h.wsAddr = fmt.Sprintf("127.0.0.1:%d", freePort(t))
	h.server = startProc(t, "wstunnel-server", bin, "server", "--restrict-config", slot.RestrictPath(), "ws://"+h.wsAddr)
	waitFor(t, "wstunnel server listening", func() bool {
		c, err := net.DialTimeout("tcp", h.wsAddr, 200*time.Millisecond)
		if err != nil {
			return false
		}
		_ = c.Close()
		return true
	})
	setTestWSServerAddr(slot, h.wsAddr)
	setTestReverseAddr(slot, slot.WstunnelReverseAddr())

	// The dataplane as wire.go builds it for Mode B: proxy without a
	// counter, front door with the slot's counter, proxy as the relay gate.
	policy := func() Policy { h.mu.Lock(); defer h.mu.Unlock(); return h.cfg.Policy() }
	h.proxy = NewWSProxy(slot, nil, func(prev, next proto.ExitPeer) {
		h.mu.Lock()
		h.changes = append(h.changes, prev.Addr+"->"+next.Addr)
		h.mu.Unlock()
	})
	h.door = NewFrontDoor(slot, policy, h.bytes)
	if err := h.door.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.door.Stop)
	setTestSocksAddr(slot, h.door.Addr())
	h.door.SetRelay(h.proxy)

	// The token gate as Manager.Gate runs it for the wstunnel branch:
	// limiter, constant-time bearer compare, enabled, mode, then the proxy.
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := SourceKey(r.RemoteAddr)
		if h.limit.Locked(source) {
			writeNotFound(w)
			return
		}
		presented := bearerToken(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare([]byte(presented), []byte(h.token())) != 1 {
			h.limit.Fail(source)
			writeNotFound(w)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/exit/"+slot.ID+"/") {
			writeNotFound(w)
			return
		}
		h.proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(func() {
		h.proxy.CloseAll("mode b suite over")
		h.srv.Close()
	})

	d, err := proxy.SOCKS5("tcp", h.door.Addr(), nil, proxy.Direct)
	if err != nil {
		t.Fatal(err)
	}
	h.dialer = d.(proxy.ContextDialer)
	return h
}

// procLog is one wstunnel process with its combined output.
type procLog struct {
	name   string
	cmd    *exec.Cmd
	log    *lockedBuffer
	exited chan struct{}
}

func startProc(t *testing.T, name, bin string, args ...string) *procLog {
	t.Helper()
	p := &procLog{name: name, log: &lockedBuffer{}, exited: make(chan struct{})}
	p.cmd = exec.Command(bin, args...)
	p.cmd.Stdout = p.log
	p.cmd.Stderr = p.log
	p.cmd.Env = append(os.Environ(), "NO_COLOR=true")
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
	return p
}

func (p *procLog) kill() {
	select {
	case <-p.exited:
		return
	default:
	}
	_ = p.cmd.Process.Signal(syscall.SIGKILL)
	<-p.exited
}

func (p *procLog) alive() bool {
	select {
	case <-p.exited:
		return false
	default:
		return true
	}
}

// startClient runs `wstunnel client -P exit/<n> -H "Authorization: Bearer
// <token>" <tunnel args> <url>`, the shape of commands.go's one-liner.
func (h *modeB) startClient(t *testing.T, name, token, url string, tunnel ...string) *procLog {
	t.Helper()
	args := []string{"client", "-P", h.slot.ClientPathPrefix(), "-H", "Authorization: Bearer " + token}
	args = append(args, tunnel...)
	args = append(args, "ws://"+strings.TrimPrefix(url, "http://"))
	return startProc(t, name, h.bin, args...)
}

func (h *modeB) waitConnected(t *testing.T, c *procLog) {
	t.Helper()
	deadline := time.Now().Add(modeBConnectWait)
	for time.Now().Before(deadline) {
		if !c.alive() {
			t.Fatalf("%s exited before the tunnel came up\n--- client log\n%s\n--- server log\n%s", c.name, c.log, h.server.log)
		}
		if state, peer, _ := h.door.Attached(); state == proto.ExitConnected {
			if peer == nil || peer.Addr != "127.0.0.1" || peer.Transport != proto.ExitModeWstunnel {
				t.Fatalf("connected with peer %+v", peer)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	state, _, _ := h.door.Attached()
	t.Fatalf("%s: front door is %q after %s (tracked=%d)\n--- client log\n%s\n--- server log\n%s",
		c.name, state, modeBConnectWait, h.proxy.Tracked(), c.log, h.server.log)
}

func (h *modeB) tcpServer(t *testing.T, handler func(net.Conn)) netip.AddrPort {
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

func (h *modeB) dial(t *testing.T, dst netip.AddrPort) net.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := h.dialer.DialContext(ctx, "tcp", dst.String())
	if err != nil {
		t.Fatalf("socks connect %s: %v\n--- server log\n%s", dst, err, h.server.log)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func (h *modeB) tcpEcho(t *testing.T, size int) {
	dst := h.tcpServer(t, echoHandler)
	before := h.bytes.Totals()
	c := h.dial(t, dst)
	payload := randomBytes(t, size)
	_ = c.SetDeadline(time.Now().Add(2 * time.Minute))

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
	after := h.bytes.Totals()
	if after.Up-before.Up < uint64(size) || after.Down-before.Down < uint64(size) {
		t.Fatalf("front door counted up=%d down=%d for a %d byte echo (D12: the front door counts relayed payload in Mode B)",
			after.Up-before.Up, after.Down-before.Down, size)
	}
}

// connectRefused: on the reverse SOCKS path wstunnel acknowledges the CONNECT
// (0x00) as soon as its client picks the connection up, before the client has
// dialled the destination (server.rs, "the server cannot know whether the
// target was reached"), so a refused port is a success reply followed by EOF
// with no data. The front door relays the reply verbatim and cannot turn it
// into 0x05: hev, and so the consumer, sees an established TCP connection that
// closes at once instead of a refused SYN. Mode A answers 0x05 here; this is a
// Mode B caveat the spec should carry next to its UDP policy gap.
func (h *modeB) connectRefused(t *testing.T) {
	ln, err := net.Listen("tcp4", net.JoinHostPort(h.lan.String(), "0"))
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().(*net.TCPAddr).AddrPort()
	_ = ln.Close()
	c, rep, _ := rawSocks(t, h.door.Addr(), socksCmdConnect, closed)
	defer c.Close()
	switch rep {
	case RepSucceeded:
		t.Logf("refused port %s answered SOCKS rep 0x00 (wstunnel reverse SOCKS cannot report the far end's connect failure)", closed)
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := c.Read(make([]byte, 16))
		if err == nil || n != 0 {
			t.Fatalf("refused port: %d byte(s) and err=%v after the success reply; want an immediate EOF", n, err)
		}
	case RepConnRefused, RepHostUnreach, RepGeneralFailure:
		t.Logf("refused port answered SOCKS rep %#x", rep)
	default:
		t.Fatalf("SOCKS rep %#x for a refused port", rep)
	}
}

// udpEcho: UDP ASSOCIATE through the front door's verbatim relay; wstunnel's
// own UDP server address comes back as BND.ADDR and datagrams flow through it.
func (h *modeB) udpEcho(t *testing.T) {
	h.settle(t)
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

	t.Logf("before UDP ASSOCIATE: connected=%v tracked=%d accepted=%d", h.proxy.Connected(), h.proxy.Tracked(), h.acceptedTunnels())
	control, rep, bnd := rawSocks(t, h.door.Addr(), socksCmdUDPAssoc, netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
	t.Cleanup(func() { _ = control.Close() })
	if rep != RepSucceeded {
		t.Fatalf("UDP ASSOCIATE rep %#x (connected=%v tracked=%d)\n--- server log\n%s", rep, h.proxy.Connected(), h.proxy.Tracked(), h.server.log)
	}
	t.Logf("UDP ASSOCIATE BND.ADDR %s", bnd)
	if bnd.Addr().IsUnspecified() || bnd.Port() == 0 {
		t.Fatalf("UDP ASSOCIATE returned an unusable BND.ADDR %s", bnd)
	}
	if !bnd.Addr().IsLoopback() {
		t.Fatalf("UDP BND.ADDR %s is not loopback; hev on the kvm cannot reach it", bnd)
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
			t.Fatalf("datagram %d: no reply: %v\n--- server log\n%s", i, err, h.server.log)
		}
		if from.String() != relay.String() {
			t.Fatalf("reply from %s, want the relay %s", from, relay)
		}
		src, payload, ok := decodeSocksUDP(buf[:n])
		if !ok || string(payload) != "echo:"+msg {
			t.Fatalf("reply %q ok=%v", buf[:n], ok)
		}
		if src != echo {
			t.Logf("note: reply source %s, want %s (wstunnel does not report the real source)", src, echo)
		}
	}
}

// wrongToken: the gate answers gin's 404 and nothing reaches wstunnel, so its
// log never sees the attempt, let alone the presented token.
func (h *modeB) wrongToken(t *testing.T) {
	bad := "zzzzzzzz"
	before := h.server.log.String()
	req, _ := http.NewRequest(http.MethodGet, h.srv.URL+"/exit/"+h.slot.ID+"/events", nil)
	req.Header.Set("Authorization", "Bearer "+bad)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound || string(body) != default404Body || resp.Header.Get("Content-Type") != "text/plain" {
		t.Fatalf("wrong token: %d %q %q", resp.StatusCode, body, resp.Header.Get("Content-Type"))
	}
	// A real wstunnel client with the wrong token keeps retrying and never
	// comes up; while it does, the good client keeps serving.
	c := h.startClient(t, "wrong-token", bad, h.srv.URL, "-R", "socks5://"+h.slot.WstunnelReverseAddr())
	waitFor(t, "wrong-token client logs a failure", func() bool {
		return strings.Contains(c.log.String(), "Retrying") || !c.alive()
	})
	c.kill()
	time.Sleep(100 * time.Millisecond)
	added := strings.TrimPrefix(h.server.log.String(), before)
	if strings.Contains(added, bad) {
		t.Fatalf("wstunnel log saw the rejected token:\n%s", added)
	}
	if strings.Contains(added, "Rejecting") {
		t.Fatalf("a wrong-token request reached wstunnel:\n%s", added)
	}
	// The gate's limiter has now counted failures for 127.0.0.1, the same
	// source as the good client. Reset as RegenerateToken does so the rest
	// of the suite is not locked out.
	h.limit.Reset(SourceKey("127.0.0.1:0"))
	h.settle(t)
	h.tcpEcho(t, 64<<10)
}

// forwardTunnelRefused: `-L` is not in the yaml's allow list; the local
// listener accepts but every connection through it dies without data.
func (h *modeB) forwardTunnelRefused(t *testing.T) {
	dst := h.tcpServer(t, echoHandler)
	local := freePort(t)
	c := h.startClient(t, "forward-L", h.token(), h.srv.URL, "-L", fmt.Sprintf("tcp://127.0.0.1:%d:%s", local, dst))
	var conn net.Conn
	waitFor(t, "-L client listening", func() bool {
		var err error
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", local), 200*time.Millisecond)
		return err == nil
	})
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write([]byte("hello through a forward tunnel")); err != nil {
		t.Fatal(err)
	}
	n, err := conn.Read(make([]byte, 64))
	if err == nil {
		t.Fatalf("forward tunnel relayed %d byte(s); the restriction yaml must refuse !Tunnel\n--- server log\n%s", n, h.server.log)
	}
	waitFor(t, "server logs the refusal", func() bool {
		return strings.Contains(h.server.log.String(), "Rejecting connection")
	})
	c.kill()
	if state, _, _ := h.door.Attached(); state != proto.ExitConnected {
		t.Fatalf("good client dropped by a refused -L client: %s", state)
	}
}

// reverseTCPRefused: `-R tcp://0.0.0.0:<p>:127.0.0.1:22` asks for a protocol
// and a bind address the yaml does not allow; nothing must listen.
func (h *modeB) reverseTCPRefused(t *testing.T) {
	port := freePort(t)
	c := h.startClient(t, "reverse-tcp", h.token(), h.srv.URL, "-R", fmt.Sprintf("tcp://0.0.0.0:%d:127.0.0.1:22", port))
	waitFor(t, "server logs the refusal", func() bool {
		return strings.Contains(c.log.String(), "Retrying") || !c.alive()
	})
	for _, addr := range []string{fmt.Sprintf("127.0.0.1:%d", port), fmt.Sprintf("%s:%d", h.lan, port)} {
		if cc, err := net.DialTimeout("tcp", addr, 300*time.Millisecond); err == nil {
			_ = cc.Close()
			t.Fatalf("a refused reverse tcp tunnel is listening on %s\n--- server log\n%s", addr, h.server.log)
		}
	}
	c.kill()
	if state, _, _ := h.door.Attached(); state != proto.ExitConnected {
		t.Fatalf("good client dropped by a refused -R tcp client: %s", state)
	}
}

// secondClientSameAddress: D12 supersedes by source address, and here both
// clients are 127.0.0.1, so the implemented rule keeps both. This proves the
// rule as implemented (no supersede, no peer change, traffic still flows) and
// that the second client's exit leaves the first serving. Supersede from a
// different address needs a second loopback alias (127.0.0.2), which macOS
// does not configure; TestWSProxy* covers that path with a fake peer.
func (h *modeB) secondClientSameAddress(t *testing.T, first *procLog) {
	h.mu.Lock()
	changesBefore := len(h.changes)
	h.mu.Unlock()
	acceptedBefore := h.acceptedTunnels()

	second := h.startClient(t, "second", h.token(), h.srv.URL, "-R", "socks5://"+h.slot.WstunnelReverseAddr())
	waitFor(t, "second client's tunnel accepted by wstunnel", func() bool { return h.acceptedTunnels() > acceptedBefore })
	time.Sleep(300 * time.Millisecond)
	if !first.alive() {
		t.Fatalf("first client exited when a second one from the same address connected\n%s", first.log)
	}
	if strings.Contains(first.log.String(), "Retrying") {
		t.Fatalf("first client was disconnected by a second client from the same address:\n%s", first.log)
	}
	h.mu.Lock()
	changes := h.changes[changesBefore:]
	h.mu.Unlock()
	if len(changes) != 0 {
		t.Fatalf("peer change reported for a same-address client: %v", changes)
	}
	if state, _, _ := h.door.Attached(); state != proto.ExitConnected {
		t.Fatalf("state %s with two clients", state)
	}
	h.tcpEcho(t, 256<<10)
	h.tcpEcho(t, 256<<10)

	second.kill()
	time.Sleep(300 * time.Millisecond)
	if state, _, _ := h.door.Attached(); state != proto.ExitConnected {
		t.Fatalf("state %s after the second client left while the first is alive", state)
	}
	h.tcpEcho(t, 256<<10)
}

// killClientFailsFast: with the exit gone, the front door must answer 0x03
// in well under a second instead of dialling a reverse listener wstunnel
// keeps bound for its idle timeout.
func (h *modeB) killClientFailsFast(t *testing.T, c *procLog) {
	killed := time.Now()
	c.kill()
	deadline := killed.Add(time.Second)
	for h.proxy.Connected() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if h.proxy.Connected() {
		t.Fatalf("RelayGate.Connected still true %s after the client was killed (tracked=%d)", time.Since(killed), h.proxy.Tracked())
	}
	t.Logf("Connected() false %s after kill", time.Since(killed).Round(time.Millisecond))
	dst := h.tcpServer(t, echoHandler)
	start := time.Now()
	conn, rep, _ := rawSocks(t, h.door.Addr(), socksCmdConnect, dst)
	_ = conn.Close()
	took := time.Since(start)
	if rep != RepNetworkUnreach {
		t.Fatalf("rep %#x after the exit left, want 0x03", rep)
	}
	if took > time.Second {
		t.Fatalf("0x03 took %s", took)
	}
	if state, peer, _ := h.door.Attached(); state != proto.ExitDisconnected || peer != nil {
		t.Fatalf("Attached = %s %+v after the client left", state, peer)
	}
	// wstunnel keeps the reverse listener bound for its idle timeout; the
	// front door must not have used it.
	if rc, err := net.DialTimeout("tcp", h.slot.WstunnelReverseAddr(), 200*time.Millisecond); err == nil {
		_ = rc.Close()
		t.Logf("note: wstunnel still has %s bound after the client left (its idle timeout is 3m)", h.slot.WstunnelReverseAddr())
	}
}

// tokenRegenerate: RegenerateToken rewrites the yaml atomically and closes
// every tracked connection; wstunnel must reload it so a client with the old
// token is refused even when it reaches wstunnel directly, and one with the
// new token comes up through the gate.
func (h *modeB) tokenRegenerate(t *testing.T) {
	old := h.token()
	c := h.startClient(t, "pre-regenerate", old, h.srv.URL, "-R", "socks5://"+h.slot.WstunnelReverseAddr())
	h.waitConnected(t, c)

	next, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	h.mu.Lock()
	h.cfg.Token = next
	cfg := h.cfg
	h.tokens = append(h.tokens, next)
	h.mu.Unlock()
	logBefore := len(h.server.log.String())
	if err := writeSlotFiles(h.slot, cfg, ""); err != nil {
		t.Fatal(err)
	}
	h.proxy.CloseAll("token regenerated")
	h.limit.Reset(SourceKey("127.0.0.1:0"))

	waitForLong(t, 10*time.Second, "wstunnel reloads the restriction file", func() bool {
		return strings.Contains(h.server.log.String()[logBefore:], "reloaded")
	})
	waitForLong(t, 5*time.Second, "old client falls off", func() bool { return !h.proxy.Connected() })
	// The old client retries through the gate with the old token: 404, never up.
	time.Sleep(2500 * time.Millisecond)
	if h.proxy.Connected() || h.proxy.Tracked() != 0 {
		t.Fatalf("old-token client is still tracked after regenerate (tracked=%d)", h.proxy.Tracked())
	}
	c.kill()

	// Straight at wstunnel, bypassing the gate, with the old token and the
	// server-side prefix: the reloaded yaml alone must refuse it.
	direct := startProc(t, "direct-old-token", h.bin, "client", "-P", h.slot.ServerPathPrefix(),
		"-H", "Authorization: Bearer "+old, "-R", "socks5://"+h.slot.WstunnelReverseAddr(), "ws://"+h.wsAddr)
	waitForLong(t, 10*time.Second, "direct old-token client is refused", func() bool {
		return strings.Contains(direct.log.String(), "Retrying") || !direct.alive()
	})
	direct.kill()
	if !strings.Contains(h.server.log.String()[logBefore:], "Rejecting connection") {
		t.Fatalf("wstunnel did not log a rejection for the old token after reload:\n%s", h.server.log.String()[logBefore:])
	}

	fresh := h.startClient(t, "post-regenerate", next, h.srv.URL, "-R", "socks5://"+h.slot.WstunnelReverseAddr())
	h.waitConnected(t, fresh)
	h.tcpEcho(t, 256<<10)
}

// acceptedTunnels counts the reverse tunnel requests wstunnel has accepted;
// an idle reverse client is one accepted-but-unanswered upgrade, which the
// proxy holds as pending, not as a hijacked connection.
func (h *modeB) acceptedTunnels() int {
	return strings.Count(h.server.log.String(), "Tunnel accepted due to matched restriction")
}

// settle waits for the gate to be connected again after a relay ended.
func (h *modeB) settle(t *testing.T) {
	t.Helper()
	waitForLong(t, 5*time.Second, "gate connected again", h.proxy.Connected)
}

// noFlapAfterShortRelay: a relay that ends before the wstunnel client has
// opened its next pending upgrade must not make the front door answer 0x03
// to the very next CONNECT; the exit never went away. Repeated, because the
// window is one client round trip wide.
func (h *modeB) noFlapAfterShortRelay(t *testing.T) {
	h.settle(t)
	ln, err := net.Listen("tcp4", net.JoinHostPort(h.lan.String(), "0"))
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().(*net.TCPAddr).AddrPort()
	_ = ln.Close()
	echo := h.tcpServer(t, echoHandler)
	connectedAt := h.proxy.ConnectedAt()

	for i := 0; i < 20; i++ {
		c, _, _ := rawSocks(t, h.door.Addr(), socksCmdConnect, closed)
		// wstunnel answers 0x00 and then closes; wait for that close.
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, _ = c.Read(make([]byte, 1))
		_ = c.Close()

		start := time.Now()
		c2, rep, _ := rawSocks(t, h.door.Addr(), socksCmdConnect, echo)
		_ = c2.Close()
		if rep != RepSucceeded {
			for !h.proxy.Connected() && time.Since(start) < 5*time.Second {
				time.Sleep(time.Millisecond)
			}
			state, peer, _ := h.door.Attached()
			t.Fatalf("iteration %d: CONNECT right after a short relay: rep %#x (state %s peer %v now); gate connected again after %s",
				i, rep, state, peer, time.Since(start))
		}
	}
	if now := h.proxy.ConnectedAt(); connectedAt == nil || now == nil || !now.Equal(*connectedAt) {
		t.Fatalf("connectedAt moved from %v to %v across short relays: status uptime flapped", connectedAt, now)
	}
}

func waitForLong(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for %s", d, what)
}
