package exit

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gorilla/websocket"
)

// fakeWstunnel stands in for wstunnel server: it records every request and
// upgrades /exit0/events to a WebSocket echo.
type fakeWstunnel struct {
	srv *httptest.Server
	mu  sync.Mutex
	got []*http.Request
}

func newFakeWstunnel(t *testing.T, slot Slot) *fakeWstunnel {
	t.Helper()
	f := &fakeWstunnel{}
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.got = append(f.got, r.Clone(context.Background()))
		f.mu.Unlock()
		if !isWebSocketUpgrade(r) {
			w.WriteHeader(http.StatusOK)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			if err := c.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}))
	t.Cleanup(f.srv.Close)
	setTestWSServerAddr(slot, strings.TrimPrefix(f.srv.URL, "http://"))
	return f
}

func (f *fakeWstunnel) requests() []*http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*http.Request(nil), f.got...)
}

// proxyHarness is a WSProxy behind an httptest server whose handler takes
// the peer address from X-Test-Remote.
type proxyHarness struct {
	slot    Slot
	p       *WSProxy
	srv     *httptest.Server
	up      *fakeWstunnel
	rev     net.Listener
	bytes   *testCounter
	changes chan [2]proto.ExitPeer
}

func newProxyHarness(t *testing.T, withReverse bool) *proxyHarness {
	t.Helper()
	h := &proxyHarness{slot: MustSlot("0"), bytes: &testCounter{}, changes: make(chan [2]proto.ExitPeer, 8)}
	h.up = newFakeWstunnel(t, h.slot)
	if withReverse {
		h.openReverse(t)
	} else {
		setTestReverseAddr(h.slot, "127.0.0.1:1") // nothing listens there
	}
	h.p = NewWSProxy(h.slot, h.bytes, func(prev, next proto.ExitPeer) { h.changes <- [2]proto.ExitPeer{prev, next} })
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if remote := r.Header.Get("X-Test-Remote"); remote != "" {
			r.RemoteAddr = remote
		}
		h.p.ServeHTTP(w, r)
	}))
	t.Cleanup(func() {
		h.p.CloseAll("test over")
		h.srv.Close()
	})
	return h
}

// openReverse binds a plain TCP listener standing in for the reverse-SOCKS
// listener wstunnel opens once the exit's client has connected.
func (h *proxyHarness) openReverse(t *testing.T) {
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
			_ = c.Close()
		}
	}()
	h.rev = ln
	setTestReverseAddr(h.slot, ln.Addr().String())
}

func (h *proxyHarness) wsURL(rest string) string {
	return "ws" + strings.TrimPrefix(h.srv.URL, "http") + "/exit/" + h.slot.ID + "/" + rest
}

func (h *proxyHarness) dialWS(t *testing.T, rest, remote string, extra http.Header) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	hdr := http.Header{}
	for k, v := range extra {
		hdr[k] = v
	}
	if remote != "" {
		hdr.Set("X-Test-Remote", remote)
	}
	d := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	return d.Dial(h.wsURL(rest), hdr)
}

func TestWSProxyRefusesNonUpgradeAndOtherPaths(t *testing.T) {
	h := newProxyHarness(t, true)
	client := h.srv.Client()

	for _, tc := range []struct {
		name string
		path string
		ws   bool
	}{
		{"plain GET on events", "/exit/0/events", false},
		{"upgrade elsewhere", "/exit/0/other", true},
		{"upgrade on prefix only", "/exit/0/", true},
		{"upgrade with traversal", "/exit/0/../1/events", true},
		{"wrong slot", "/exit/1/events", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, h.srv.URL+tc.path, nil)
			if tc.ws {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Sec-WebSocket-Version", "13")
				req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound || string(body) != "404 page not found" {
				t.Fatalf("%s: %d %q", tc.path, resp.StatusCode, body)
			}
		})
	}
	if n := len(h.up.requests()); n != 0 {
		t.Fatalf("%d request(s) reached wstunnel", n)
	}
	if h.p.Connected() || h.p.Peer() != nil {
		t.Fatal("gate reports a peer after refusals")
	}
}

func TestWSProxyForwardsEventsWithHeaderHygiene(t *testing.T) {
	h := newProxyHarness(t, true)
	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer k7m2p9vx")
	hdr.Set("Cookie", "session=secret")
	hdr.Set("X-Forwarded-For", "6.6.6.6")
	hdr.Set("X-Forwarded-Proto", "https")
	hdr.Set("X-Forwarded-Host", "evil.example")
	hdr.Set("Forwarded", "for=6.6.6.6")
	c, resp, err := h.dialWS(t, "events", "203.0.113.5:5555", hdr)
	if err != nil {
		t.Fatalf("dial through proxy: %v (%v)", err, resp)
	}
	defer c.Close()

	// Echo through the pipe.
	if err := c.WriteMessage(websocket.BinaryMessage, []byte("through")); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := c.ReadMessage()
	if err != nil || string(data) != "through" {
		t.Fatalf("echo %q %v", data, err)
	}

	reqs := h.up.requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests at wstunnel", len(reqs))
	}
	r := reqs[0]
	if r.URL.Path != "/exit0/events" {
		t.Fatalf("path %q", r.URL.Path)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer k7m2p9vx" {
		t.Fatalf("Authorization %q", got)
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("Forwarded") != "" {
		t.Fatalf("Cookie/Forwarded leaked: %v", r.Header)
	}
	// The incoming X-Forwarded-For was dropped and replaced by the peer the
	// proxy actually saw (the handler set RemoteAddr to 203.0.113.5).
	if got := r.Header.Get("X-Forwarded-For"); got != "203.0.113.5" {
		t.Fatalf("X-Forwarded-For %q", got)
	}
	if got := r.Header.Get("X-Forwarded-Proto"); got != "http" {
		t.Fatalf("X-Forwarded-Proto %q", got)
	}
	if got := r.Header.Get("X-Forwarded-Host"); got == "evil.example" || got == "" {
		t.Fatalf("X-Forwarded-Host %q", got)
	}

	// Gate state: tracked, peer known, listener probed.
	if h.p.Tracked() != 1 {
		t.Fatalf("tracked %d", h.p.Tracked())
	}
	waitFor(t, "Connected", h.p.Connected)
	peer := h.p.Peer()
	if peer == nil || peer.Addr != "203.0.113.5" || peer.Transport != proto.ExitModeWstunnel {
		t.Fatalf("peer %+v", peer)
	}
	if h.p.ConnectedAt() == nil {
		t.Fatal("ConnectedAt nil")
	}
	if tot := h.bytes.Totals(); tot.Up == 0 || tot.Down == 0 {
		t.Fatalf("bytes %+v", tot)
	}

	_ = c.Close()
	waitFor(t, "untracked", func() bool { return h.p.Tracked() == 0 && !h.p.Connected() && h.p.Peer() == nil })
}

func TestWSProxyConnectedNeedsReverseListener(t *testing.T) {
	h := newProxyHarness(t, false)
	c, _, err := h.dialWS(t, "events", "203.0.113.5:5555", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if h.p.Tracked() != 1 || h.p.Peer() == nil {
		t.Fatal("upgrade not tracked")
	}
	time.Sleep(5 * reverseProbeInterval)
	if h.p.Connected() {
		t.Fatal("Connected without a reverse listener")
	}
	// The listener appears (the exit's client finished its handshake).
	h.openReverse(t)
	waitFor(t, "Connected once the listener accepts", h.p.Connected)
}

func TestWSProxySupersede(t *testing.T) {
	h := newProxyHarness(t, true)
	c1, _, err := h.dialWS(t, "events", "198.51.100.1:1000", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	c1b, _, err := h.dialWS(t, "events", "198.51.100.1:1001", nil) // same source, second connection
	if err != nil {
		t.Fatal(err)
	}
	defer c1b.Close()
	if h.p.Tracked() != 2 {
		t.Fatalf("tracked %d", h.p.Tracked())
	}
	select {
	case ch := <-h.changes:
		t.Fatalf("peer change fired for the same host: %+v", ch)
	default:
	}

	c2, _, err := h.dialWS(t, "events", "198.51.100.2:1000", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	select {
	case ch := <-h.changes:
		if ch[0].Addr != "198.51.100.1" || ch[1].Addr != "198.51.100.2" {
			t.Fatalf("peer change %+v", ch)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("onPeerChange did not fire")
	}
	_ = c1.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := c1.ReadMessage(); err == nil {
		t.Fatal("superseded connection still alive")
	}
	_ = c1b.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := c1b.ReadMessage(); err == nil {
		t.Fatal("second superseded connection still alive")
	}
	waitFor(t, "only the new peer tracked", func() bool { return h.p.Tracked() == 1 })
	if p := h.p.Peer(); p == nil || p.Addr != "198.51.100.2" {
		t.Fatalf("peer %+v", p)
	}
	if err := c2.WriteMessage(websocket.BinaryMessage, []byte("ok")); err != nil {
		t.Fatal(err)
	}
	_ = c2.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, data, err := c2.ReadMessage(); err != nil || string(data) != "ok" {
		t.Fatalf("new peer echo %q %v", data, err)
	}
}

func TestWSProxyPinPeer(t *testing.T) {
	h := newProxyHarness(t, true)
	c1, _, err := h.dialWS(t, "events", "198.51.100.1:1000", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	h.p.SetPinPeer(true)
	_, resp, err := h.dialWS(t, "events", "198.51.100.2:1000", nil)
	if err == nil || resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("pinned: other host got %v %v", err, resp)
	}
	if h.p.Tracked() != 1 {
		t.Fatal("pinned connection was dropped by a refused peer")
	}
	c1b, _, err := h.dialWS(t, "events", "198.51.100.1:2000", nil)
	if err != nil {
		t.Fatalf("same host refused: %v", err)
	}
	defer c1b.Close()
	h.p.SetPinPeer(false)
	c2, _, err := h.dialWS(t, "events", "198.51.100.9:1000", nil)
	if err != nil {
		t.Fatalf("after unpin: %v", err)
	}
	defer c2.Close()
}

func TestWSProxyCloseAll(t *testing.T) {
	h := newProxyHarness(t, true)
	c, _, err := h.dialWS(t, "events", "198.51.100.1:1000", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	waitFor(t, "connected", h.p.Connected)
	h.p.CloseAll("token regenerated")
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("connection survived CloseAll")
	}
	if h.p.Tracked() != 0 || h.p.Connected() || h.p.Peer() != nil {
		t.Fatal("state left behind after CloseAll")
	}
}

func TestWSProxyUpstreamDown(t *testing.T) {
	h := newProxyHarness(t, true)
	h.up.srv.Close()
	_, resp, err := h.dialWS(t, "events", "198.51.100.1:1000", nil)
	if err == nil || resp == nil || resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("dead upstream: %v %v", err, resp)
	}
	waitFor(t, "no peer without a hijack", func() bool { return h.p.Peer() == nil && h.p.Tracked() == 0 })
}
