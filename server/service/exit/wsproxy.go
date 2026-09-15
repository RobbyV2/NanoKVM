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
// TCP probe of the reverse listener since the first tracked upgrade).
var (
	reverseProbeInterval = 500 * time.Millisecond
	reverseProbeDial     = time.Second
	wstunnelServerAddr   = func(s Slot) string { return s.WstunnelAddr() }
)

// WSProxy is the Mode B reverse proxy in front of wstunnel server (D13). It
// forwards only WebSocket upgrades whose path ends in /events, rewrites
// /exit/<n>/<rest> to /exit<n>/<rest>, keeps Authorization, strips Cookie and
// any incoming Forwarded / X-Forwarded-* before setting its own, tracks the
// hijacked connections by RemoteAddr and supersedes on a new source (D12).
// It implements RelayGate for the front door.
type WSProxy struct {
	slot         Slot
	bytes        ByteCounter
	onPeerChange func(prev, next proto.ExitPeer)
	proxy        *httputil.ReverseProxy

	mu          sync.Mutex
	conns       map[*trackedConn]struct{}
	pending     int
	peer        *proto.ExitPeer
	connectedAt *time.Time
	probed      bool
	probeCancel context.CancelFunc
	pinPeer     bool
	pinnedHost  string
	wg          sync.WaitGroup
}

// NewWSProxy builds the proxy toward slot.WstunnelAddr(). onPeerChange may be nil.
func NewWSProxy(slot Slot, bytes ByteCounter, onPeerChange func(prev, next proto.ExitPeer)) *WSProxy {
	p := &WSProxy{
		slot:         slot,
		bytes:        bytes,
		onPeerChange: onPeerChange,
		conns:        make(map[*trackedConn]struct{}),
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

// CloseAll drops every tracked connection.
func (p *WSProxy) CloseAll(reason string) {
	p.mu.Lock()
	conns := p.snapshotLocked()
	p.mu.Unlock()
	if len(conns) > 0 {
		log.Infof("%s: closing %d wstunnel connection(s): %s", p.slot.Name(), len(conns), reason)
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

func (p *WSProxy) Connected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns) > 0 && p.probed
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
	var prev *proto.ExitPeer
	next := proto.ExitPeer{Addr: host, Transport: proto.ExitModeWstunnel}
	if p.peer == nil || p.peer.Addr != host {
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
	if p.pinPeer {
		p.pinnedHost = host
	}
	p.pending++
	p.mu.Unlock()

	for _, c := range stale {
		log.Warnf("%s: wstunnel peer %s superseded by %s", p.slot.Name(), c.RemoteAddr(), r.RemoteAddr)
		_ = c.Close()
	}
	if prev != nil && p.onPeerChange != nil {
		p.onPeerChange(*prev, next)
	}

	r2 := r.Clone(r.Context())
	r2.URL.Path = "/" + p.slot.ServerPathPrefix() + "/" + rest
	r2.URL.RawPath = ""
	hw := &hijackWriter{ResponseWriter: w, p: p, host: host}
	p.proxy.ServeHTTP(hw, r2)

	// A hijacked request handed its pending count over to the tracked
	// connection; only a request that never upgraded still holds one.
	if !hw.hijacked.Load() {
		p.mu.Lock()
		p.pending--
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

func (p *WSProxy) maybeClearLocked() {
	if len(p.conns) == 0 && p.pending == 0 {
		p.peer = nil
		p.connectedAt = nil
		p.probed = false
		if p.probeCancel != nil {
			p.probeCancel()
			p.probeCancel = nil
		}
	}
}

// track registers a hijacked connection and retires the pending count of
// the request that produced it.
func (p *WSProxy) track(c net.Conn, host string) *trackedConn {
	tc := &trackedConn{Conn: c, p: p, host: host}
	p.mu.Lock()
	p.conns[tc] = struct{}{}
	p.pending--
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
	host     string
	hijacked atomic.Bool
}

func (h *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, rw, err := http.NewResponseController(h.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	h.hijacked.Store(true)
	return h.p.track(c, h.host), rw, nil
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
