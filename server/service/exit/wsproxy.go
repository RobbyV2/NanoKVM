package exit

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/proto"

	log "github.com/sirupsen/logrus"
)

// Reverse-listener probe cadence (D13, plan: Connected() needs one successful
// TCP probe of the reverse listener since the first tracked upgrade), and the
// grace a proven peer keeps after its last connection (see maybeClearLocked).
var (
	reverseProbeInterval = 500 * time.Millisecond
	reverseProbeDial     = time.Second
	relayGrace           = 750 * time.Millisecond
	wstunnelServerAddr   = func(s Slot) string { return s.WstunnelAddr() }
)

// WSProxy is the Mode B reverse proxy in front of wstunnel server (D13). It
// forwards only WebSocket upgrades whose path ends in /events, rewrites
// /exit/<n>/<rest> to /exit<n>/<rest>, keeps Authorization, strips Cookie and
// any incoming Forwarded / X-Forwarded-* before setting its own, tracks the
// peer's connections by RemoteAddr and supersedes on a new source (D12).
// It implements RelayGate for the front door.
//
// wstunnel's reverse tunnel model shapes the tracking: the client keeps
// exactly one upgrade request open and the server answers it (101) only when
// a downstream connection has completed its SOCKS handshake on the reverse
// listener, at which point the client opens the next one. So an idle exit is
// one pending, un-upgraded request; a busy one is hijacked connections plus
// at most one pending request; and between a short relay ending and the next
// request landing (one client round trip) the exit holds nothing here at all.
// Pending requests are therefore tracked, closed and counted like hijacked
// connections, and a proven peer keeps its state for relayGrace after its
// last one goes away.
type WSProxy struct {
	slot         Slot
	bytes        ByteCounter
	onPeerChange func(prev, next proto.ExitPeer)
	proxy        *httputil.ReverseProxy

	mu          sync.Mutex
	conns       map[*trackedConn]struct{}
	pending     map[*pendingReq]struct{}
	peer        *proto.ExitPeer
	connectedAt *time.Time
	probed      bool
	probeCancel context.CancelFunc
	grace       *time.Timer
	pinPeer     bool
	pinnedHost  string
	wg          sync.WaitGroup
}

// pendingReq is an upgrade request wstunnel has not answered yet; cancel
// aborts the proxied round trip, which answers the client 502.
type pendingReq struct {
	host   string
	cancel context.CancelFunc
}

// NewWSProxy builds the proxy toward slot.WstunnelAddr(). onPeerChange may be nil.
func NewWSProxy(slot Slot, bytes ByteCounter, onPeerChange func(prev, next proto.ExitPeer)) *WSProxy {
	p := &WSProxy{
		slot:         slot,
		bytes:        bytes,
		onPeerChange: onPeerChange,
		conns:        make(map[*trackedConn]struct{}),
		pending:      make(map[*pendingReq]struct{}),
	}
	p.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			target := &url.URL{Scheme: "http", Host: wstunnelServerAddr(slot)}
			pr.SetURL(target)
			pr.Out.Host = target.Host
			for _, h := range []string{"Cookie", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-Ip"} {
				pr.Out.Header.Del(h)
			}
			pr.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Warnf("%s: wstunnel proxy: %s", slot.Name(), err)
			w.WriteHeader(http.StatusBadGateway)
		},
		ErrorLog: nil,
	}
	return p
}

// SetPinPeer arms or clears "pin to first peer".
func (p *WSProxy) SetPinPeer(pin bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pinPeer = pin
	if !pin {
		p.pinnedHost = ""
		return
	}
	if p.peer != nil {
		p.pinnedHost = p.peer.Addr
	}
}

// CloseAll drops every connection of the peer, upgraded or still pending,
// and forgets the peer at once: an operator's disconnect or a regenerated
// token must not leave the exit one more relay.
func (p *WSProxy) CloseAll(reason string) {
	p.mu.Lock()
	conns := p.snapshotLocked()
	pending := make([]*pendingReq, 0, len(p.pending))
	for q := range p.pending {
		pending = append(pending, q)
	}
	p.clearLocked()
	p.mu.Unlock()
	if len(conns)+len(pending) > 0 {
		log.Infof("%s: closing %d wstunnel connection(s) and %d pending upgrade(s): %s", p.slot.Name(), len(conns), len(pending), reason)
	}
	for _, q := range pending {
		q.cancel()
	}
	for _, c := range conns {
		_ = c.Close()
	}
	p.wg.Wait()
}

func (p *WSProxy) snapshotLocked() []*trackedConn {
	out := make([]*trackedConn, 0, len(p.conns))
	for c := range p.conns {
		out = append(out, c)
	}
	return out
}

// RelayGate.

// Connected is true once the reverse listener has been proven and the peer
// still holds a connection here, upgraded or pending, or left within the
// grace. Counting only hijacked connections would leave the front door
// waiting for a relay that waits for the front door.
func (p *WSProxy) Connected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.probed && (len(p.conns) > 0 || len(p.pending) > 0 || p.grace != nil)
}

func (p *WSProxy) Peer() *proto.ExitPeer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.peer == nil {
		return nil
	}
	cp := *p.peer
	return &cp
}

func (p *WSProxy) ConnectedAt() *time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connectedAt == nil {
		return nil
	}
	t := *p.connectedAt
	return &t
}

// Tracked is the number of hijacked connections alive.
func (p *WSProxy) Tracked() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns)
}

// ServeHTTP handles an authenticated request under /exit/<n>/.
func (p *WSProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := p.rest(r.URL.Path)
	if !ok || !isWebSocketUpgrade(r) || !strings.HasSuffix(rest, "/events") && rest != "events" {
		writeNotFound(w)
		return
	}
	host := hostOf(r.RemoteAddr)

	p.mu.Lock()
	if p.pinPeer && p.pinnedHost != "" && host != p.pinnedHost {
		p.mu.Unlock()
		log.Warnf("%s: refusing wstunnel from %s, pinned to %s", p.slot.Name(), r.RemoteAddr, p.pinnedHost)
		writeNotFound(w)
		return
	}
	var stale []*trackedConn
	for c := range p.conns {
		if c.host != host {
			stale = append(stale, c)
		}
	}
	var stalePending []*pendingReq
	for q := range p.pending {
		if q.host != host {
			stalePending = append(stalePending, q)
		}
	}
	var prev *proto.ExitPeer
	next := proto.ExitPeer{Addr: host, Transport: proto.ExitModeWstunnel}
	newPeer := false
	if p.peer == nil || p.peer.Addr != host {
		newPeer = true
		if p.peer != nil {
			cp := *p.peer
			prev = &cp
		}
		p.peer = &next
		now := time.Now()
		p.connectedAt = &now
		p.probed = false
		p.startProbeLocked()
	}
	if p.grace != nil {
		// The peer is back within its grace; it never left.
		p.grace.Stop()
		p.grace = nil
	}
	if p.pinPeer {
		p.pinnedHost = host
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	pr := &pendingReq{host: host, cancel: cancel}
	p.pending[pr] = struct{}{}
	p.mu.Unlock()

	for _, c := range stale {
		log.Warnf("%s: wstunnel peer %s superseded by %s", p.slot.Name(), c.RemoteAddr(), r.RemoteAddr)
		_ = c.Close()
	}
	for _, q := range stalePending {
		log.Warnf("%s: wstunnel peer %s (pending upgrade) superseded by %s", p.slot.Name(), q.host, r.RemoteAddr)
		q.cancel()
	}
	// Fire on the first peer too (prev is the zero peer) so the manager can
	// record its source for the token-regenerate limiter reset (D11); a real
	// supersede carries the dropped peer as prev (D12). This used to fire only
	// when a previous peer existed, so peer A's first connection never reached
	// the manager (MGR-5).
	if newPeer && p.onPeerChange != nil {
		var pv proto.ExitPeer
		if prev != nil {
			pv = *prev
		}
		p.onPeerChange(pv, next)
	}

	r2 := r.Clone(ctx)
	r2.URL.Path = "/" + p.slot.ServerPathPrefix() + "/" + rest
	r2.URL.RawPath = ""
	hw := &hijackWriter{ResponseWriter: w, p: p, req: pr}
	p.proxy.ServeHTTP(hw, r2)

	// A hijacked request handed itself over to the tracked connection; only
	// a request that never upgraded is still pending here.
	if !hw.hijacked.Load() {
		p.mu.Lock()
		delete(p.pending, pr)
		p.maybeClearLocked()
		p.mu.Unlock()
	}
}

// rest strips /exit/<id>/ and returns what follows.
func (p *WSProxy) rest(path string) (string, bool) {
	prefix := "/exit/" + p.slot.ID + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" || strings.Contains(rest, "..") {
		return "", false
	}
	return rest, true
}

func isWebSocketUpgrade(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	for _, v := range r.Header.Values("Connection") {
		for _, tok := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(tok), "upgrade") {
				return true
			}
		}
	}
	return false
}

// startProbeLocked dials the reverse listener until it accepts once, marking
// the gate connected. It keeps trying for as long as the peer is tracked:
// the probe is cancelled when the peer changes or its last connection
// closes. p.mu must be held.
func (p *WSProxy) startProbeLocked() {
	if p.probeCancel != nil {
		p.probeCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.probeCancel = cancel
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(reverseProbeInterval)
		defer ticker.Stop()
		for {
			if p.probeOnce(ctx) {
				p.mu.Lock()
				if ctx.Err() == nil {
					p.probed = true
				}
				p.mu.Unlock()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (p *WSProxy) probeOnce(ctx context.Context) bool {
	dctx, cancel := context.WithTimeout(ctx, reverseProbeDial)
	defer cancel()
	var d net.Dialer
	c, err := d.DialContext(dctx, "tcp", wstunnelReverseAddr(p.slot))
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// maybeClearLocked runs when a connection or pending request goes away. A
// peer that never proved its listener is forgotten at once. A proven peer
// whose last connection just ended is most likely between a relay and its
// next upgrade request (one round trip away), so it keeps its state for
// relayGrace; if nothing arrives by then it is gone. p.mu must be held.
func (p *WSProxy) maybeClearLocked() {
	if len(p.conns) > 0 || len(p.pending) > 0 || p.peer == nil {
		return
	}
	if !p.probed {
		p.clearLocked()
		return
	}
	if p.grace == nil {
		p.grace = time.AfterFunc(relayGrace, p.expireGrace)
	}
}

func (p *WSProxy) expireGrace() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.grace == nil || len(p.conns) > 0 || len(p.pending) > 0 {
		return
	}
	p.clearLocked()
}

// clearLocked forgets the peer. p.mu must be held.
func (p *WSProxy) clearLocked() {
	p.peer = nil
	p.connectedAt = nil
	p.probed = false
	if p.probeCancel != nil {
		p.probeCancel()
		p.probeCancel = nil
	}
	if p.grace != nil {
		p.grace.Stop()
		p.grace = nil
	}
}

// track registers a hijacked connection in place of the pending request
// that produced it.
func (p *WSProxy) track(c net.Conn, pr *pendingReq) *trackedConn {
	tc := &trackedConn{Conn: c, p: p, host: pr.host}
	p.mu.Lock()
	p.conns[tc] = struct{}{}
	delete(p.pending, pr)
	p.mu.Unlock()
	return tc
}

func (p *WSProxy) untrack(tc *trackedConn) {
	p.mu.Lock()
	delete(p.conns, tc)
	p.maybeClearLocked()
	p.mu.Unlock()
}

// hijackWriter wraps the response so the hijacked connection is tracked.
type hijackWriter struct {
	http.ResponseWriter
	p        *WSProxy
	req      *pendingReq
	hijacked atomic.Bool
}

func (h *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, rw, err := http.NewResponseController(h.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	h.hijacked.Store(true)
	return h.p.track(c, h.req), rw, nil
}

func (h *hijackWriter) Flush() {
	_ = http.NewResponseController(h.ResponseWriter).Flush()
}

func (h *hijackWriter) Unwrap() http.ResponseWriter { return h.ResponseWriter }

// trackedConn counts transport bytes (exit->kvm as Down, kvm->exit as Up;
// WebSocket framing included, so this is a transport-level figure) and
// untracks itself on Close.
type trackedConn struct {
	net.Conn
	p    *WSProxy
	host string
	once sync.Once
}

func (c *trackedConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 && c.p.bytes != nil {
		c.p.bytes.AddDown(int64(n))
	}
	return n, err
}

func (c *trackedConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 && c.p.bytes != nil {
		c.p.bytes.AddUp(int64(n))
	}
	return n, err
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.p.untrack(c) })
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}
