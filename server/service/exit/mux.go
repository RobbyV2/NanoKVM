package exit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Durations the tests shorten. Production values are the exported constants
// in types.go.
var (
	helloTimeout   = HelloTimeout
	openTimeout    = OpenTimeout
	pingInterval   = PingInterval
	pingMisses     = PingMisses
	wsWriteTimeout = 30 * time.Second
)

var (
	errSessionClosed = errors.New("nexit: session closed")
	errStreamReset   = errors.New("nexit: stream reset")
)

// MuxHooks is how the manager observes a Mux (plan, "Contracts").
type MuxHooks struct {
	// OnSession fires after WELCOME; the manager calls FrontDoor.SetNative(s)
	// and records the peer.
	OnSession func(s Backend)
	// OnClose fires once when a session ends for any reason.
	OnClose func(s Backend, reason string)
	// Policy is the destination policy applied at OPEN (D21).
	Policy func() Policy
	// Bytes counts DATA payload bytes: Up for kvm->exit, Down for exit->kvm.
	// The front door does not count in Mode A, so nothing is counted twice.
	Bytes ByteCounter
}

// Mux is the nexit/1 server side for one slot. It holds at most one session:
// a new valid connection supersedes the old one (D12).
type Mux struct {
	slot     Slot
	hooks    MuxHooks
	upgrader websocket.Upgrader

	mu         sync.Mutex
	current    *session
	connecting int
	pinPeer    bool
	pinnedHost string
}

// NewMux builds the mux with its dedicated upgrader: 32 KiB buffers and a
// permissive origin check, since the exit is a script, not a browser.
func NewMux(slot Slot, hooks MuxHooks) *Mux {
	return &Mux{
		slot:  slot,
		hooks: hooks,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  32 << 10,
			WriteBufferSize: 32 << 10,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
	}
}

// SetPinPeer arms or clears "pin to first peer" (D12). Arming with a live
// session pins that peer's host; arming without one pins the next host.
func (m *Mux) SetPinPeer(pin bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pinPeer = pin
	if !pin {
		m.pinnedHost = ""
		return
	}
	if m.current != nil {
		m.pinnedHost = m.current.host
	}
}

// Current is the attached session, nil when none.
func (m *Mux) Current() Backend {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return nil
	}
	return m.current
}

// Connecting reports whether an upgrade has been accepted whose HELLO has not
// arrived yet (status "connecting").
func (m *Mux) Connecting() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connecting > 0
}

// CloseAll ends the current session with reason.
func (m *Mux) CloseAll(reason string) {
	m.mu.Lock()
	s := m.current
	m.mu.Unlock()
	if s != nil {
		s.Close(reason)
		<-s.done
	}
}

// ServeNative upgrades r (already token-authenticated by the caller) and runs
// the session until it ends.
func (m *Mux) ServeNative(w http.ResponseWriter, r *http.Request) {
	host := hostOf(r.RemoteAddr)

	m.mu.Lock()
	if m.pinPeer && m.pinnedHost != "" && host != m.pinnedHost {
		m.mu.Unlock()
		log.Warnf("%s: refusing exit from %s, pinned to %s", m.slot.Name(), r.RemoteAddr, m.pinnedHost)
		writeNotFound(w)
		return
	}
	m.connecting++
	m.mu.Unlock()

	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.mu.Lock()
		m.connecting--
		m.mu.Unlock()
		log.Warnf("%s: websocket upgrade from %s failed: %s", m.slot.Name(), r.RemoteAddr, err)
		return
	}

	s := newSession(m, conn, r.RemoteAddr, host)
	h, err := s.awaitHello()
	m.mu.Lock()
	m.connecting--
	m.mu.Unlock()
	if err != nil {
		log.Warnf("%s: exit %s: %s", m.slot.Name(), r.RemoteAddr, err)
		s.Close(err.Error())
		s.finish()
		return
	}
	s.peer.Hostname = h.hostname
	s.peer.OS = h.os

	if !m.attach(s) {
		log.Warnf("%s: refusing exit from %s, pinned to %s", m.slot.Name(), r.RemoteAddr, m.pinnedHost)
		s.Close("pinned to another peer")
		s.finish()
		return
	}

	if err := s.sendWelcome(); err != nil {
		m.detach(s)
		s.Close(err.Error())
		s.finish()
		return
	}
	s.connectedAt = time.Now()
	log.Infof("%s: exit %s connected (%s, %s)", m.slot.Name(), r.RemoteAddr, h.hostname, h.os)
	if m.hooks.OnSession != nil {
		m.hooks.OnSession(s)
	}

	go s.pinger()
	reason := s.readLoop()
	s.Close(reason)
	m.detach(s)
	s.finish()
	log.Infof("%s: exit %s disconnected: %s", m.slot.Name(), r.RemoteAddr, s.reason())
	if m.hooks.OnClose != nil {
		m.hooks.OnClose(s, s.reason())
	}
}

// attach makes s the current session, closing the previous one (D12).
func (m *Mux) attach(s *session) bool {
	m.mu.Lock()
	if m.pinPeer && m.pinnedHost != "" && s.host != m.pinnedHost {
		m.mu.Unlock()
		return false
	}
	prev := m.current
	m.current = s
	if m.pinPeer {
		m.pinnedHost = s.host
	}
	m.mu.Unlock()
	if prev != nil {
		log.Warnf("%s: exit %s superseded by %s", m.slot.Name(), prev.remote, s.remote)
		prev.Close("superseded by " + s.remote)
	}
	return true
}

func (m *Mux) detach(s *session) {
	m.mu.Lock()
	if m.current == s {
		m.current = nil
	}
	m.mu.Unlock()
}

func (m *Mux) policy() Policy {
	if m.hooks.Policy == nil {
		return Policy{}
	}
	return m.hooks.Policy()
}

func hostOf(remote string) string {
	if h, _, err := net.SplitHostPort(remote); err == nil {
		return h
	}
	return remote
}

// writeNotFound mirrors gin's default 404 so a refusal is indistinguishable
// from an unknown route. The handler in front of the mux does the same.
func writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("404 page not found"))
}

// session is one attached exit. It implements Backend.
type session struct {
	mux    *Mux
	conn   *websocket.Conn
	remote string
	host   string
	peer   proto.ExitPeer

	connectedAt time.Time
	writeCh     chan []byte
	done        chan struct{}
	finished    chan struct{}
	closeOnce   sync.Once
	reasonMu    sync.Mutex
	closeReason string
	misses      atomic.Int32

	mu      sync.Mutex
	streams map[uint32]*stream
	nextID  uint32

	// Send-side credit (kvm -> exit) for the connection; per-stream credit
	// lives on the stream.
	connSend *credit
	// Receive-side accounting (exit -> kvm), guarded by flowMu.
	flowMu          sync.Mutex
	connOutstanding int64
	connConsumed    int64
}

func newSession(m *Mux, conn *websocket.Conn, remote, host string) *session {
	conn.SetReadLimit(int64(MaxFrame + 64))
	s := &session{
		mux:      m,
		conn:     conn,
		remote:   remote,
		host:     host,
		peer:     proto.ExitPeer{Addr: host, Transport: proto.ExitModeNative},
		writeCh:  make(chan []byte, 256),
		done:     make(chan struct{}),
		finished: make(chan struct{}),
		streams:  make(map[uint32]*stream),
		nextID:   1,
		connSend: newCredit(ConnWindow),
	}
	go s.writer()
	return s
}

// Backend.

func (s *session) Peer() proto.ExitPeer     { return s.peer }
func (s *session) ConnectedAt() time.Time   { return s.connectedAt }
func (s *session) Done() <-chan struct{}    { return s.done }
func (s *session) RemoteAddr() string       { return s.remote }
func (s *session) reason() string           { s.reasonMu.Lock(); defer s.reasonMu.Unlock(); return s.closeReason }
func (s *session) streamCount() (n int)     { s.mu.Lock(); defer s.mu.Unlock(); return len(s.streams) }
func (s *session) connSendAvailable() int64 { return s.connSend.available() }

// Close ends the session. The first reason wins; teardown of streams happens
// in finish, on the ServeNative goroutine, once the read loop has returned.
func (s *session) Close(reason string) {
	s.closeOnce.Do(func() {
		s.reasonMu.Lock()
		s.closeReason = reason
		s.reasonMu.Unlock()
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, truncate(reason, 120))
		_ = s.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
		close(s.done)
		_ = s.conn.Close()
	})
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// finish runs after the read loop: RST every stream toward hev (D12, "On WS
// loss the kvm immediately RSTs every stream toward hev").
func (s *session) finish() {
	s.mu.Lock()
	streams := make([]*stream, 0, len(s.streams))
	for _, st := range s.streams {
		streams = append(streams, st)
	}
	s.mu.Unlock()
	for _, st := range streams {
		s.releaseStream(st, errSessionClosed, false)
	}
	s.connSend.close(errSessionClosed)
	close(s.finished)
}

func (s *session) awaitHello() (hello, error) {
	_ = s.conn.SetReadDeadline(time.Now().Add(helloTimeout))
	mt, data, err := s.conn.ReadMessage()
	if err != nil {
		return hello{}, fmt.Errorf("no HELLO: %w", err)
	}
	if mt != websocket.BinaryMessage {
		return hello{}, errors.New("HELLO must be a binary message")
	}
	f, err := decodeFrame(data)
	if err != nil {
		return hello{}, err
	}
	if f.typ != frameHello || f.id != 0 {
		return hello{}, fmt.Errorf("expected HELLO, got type 0x%02x on stream %d", f.typ, f.id)
	}
	h, err := decodeHello(f.payload)
	if err != nil {
		return hello{}, fmt.Errorf("bad HELLO: %w", err)
	}
	if h.version != nexitVersion {
		return hello{}, fmt.Errorf("unsupported nexit version %d", h.version)
	}
	return h, nil
}

func (s *session) sendWelcome() error {
	s.conn.SetPongHandler(func(string) error {
		s.misses.Store(0)
		return s.conn.SetReadDeadline(time.Now().Add(s.readDeadline()))
	})
	_ = s.conn.SetReadDeadline(time.Now().Add(s.readDeadline()))
	return s.send(frameWelcome, 0, encodeWelcome(welcome{
		version:      nexitVersion,
		streamWindow: StreamWindow,
		connWindow:   ConnWindow,
		maxStreams:   MaxStreams,
	}))
}

// readDeadline is the backstop behind the pinger: one interval later than
// the third miss, so the pinger's reason wins when both would fire.
func (s *session) readDeadline() time.Duration {
	return pingInterval * time.Duration(pingMisses+2)
}

func (s *session) writer() {
	for {
		select {
		case msg := <-s.writeCh:
			_ = s.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := s.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				s.Close("write: " + err.Error())
				return
			}
		case <-s.done:
			return
		}
	}
}

func (s *session) pinger() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if int(s.misses.Add(1)) > pingMisses {
				s.Close("ping timeout")
				return
			}
			_ = s.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		case <-s.done:
			return
		}
	}
}

// send queues one frame for the writer. It blocks only on the writer's
// channel, never on a hev socket.
func (s *session) send(typ byte, id uint32, payload []byte) error {
	b := encodeFrame(typ, id, payload)
	select {
	case s.writeCh <- b:
		return nil
	case <-s.done:
		return errSessionClosed
	}
}

// readLoop parses frames until the connection fails or a protocol error
// occurs. It returns the close reason.
func (s *session) readLoop() string {
	for {
		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			select {
			case <-s.done:
				return s.reason()
			default:
			}
			return "read: " + err.Error()
		}
		if mt != websocket.BinaryMessage {
			return "protocol error: non-binary message"
		}
		f, err := decodeFrame(data)
		if err != nil {
			return "protocol error: " + err.Error()
		}
		if err := s.handle(f); err != nil {
			return "protocol error: " + err.Error()
		}
	}
}

func (s *session) lookup(id uint32) *stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streams[id]
}

func (s *session) handle(f frame) error {
	switch f.typ {
	case frameWindow:
		if len(f.payload) < 4 {
			return errBadFrame
		}
		inc := int64(uint32(f.payload[0])<<24 | uint32(f.payload[1])<<16 | uint32(f.payload[2])<<8 | uint32(f.payload[3]))
		if f.id == 0 {
			s.connSend.add(inc)
			return nil
		}
		if st := s.lookup(f.id); st != nil && st.proto == protoTCP {
			st.sendCredit.add(inc)
		}
		return nil

	case frameOpened:
		st := s.lookup(f.id)
		if st == nil {
			return nil
		}
		var bound netip.AddrPort
		if st.proto == protoTCP {
			ap, _, err := decodeAddrPort(f.payload)
			if err != nil {
				return fmt.Errorf("bad OPENED: %w", err)
			}
			bound = ap
		}
		st.deliverOpen(openResult{bound: bound})
		return nil

	case frameOpenFail:
		st := s.lookup(f.id)
		if st == nil {
			return nil
		}
		rep := RepGeneralFailure
		if len(f.payload) > 0 {
			rep = f.payload[0]
		}
		st.deliverOpen(openResult{err: &RepError{Rep: rep}})
		s.releaseStream(st, &RepError{Rep: rep}, false)
		return nil

	case frameData:
		st := s.lookup(f.id)
		if st == nil {
			return nil
		}
		return s.onData(st, f.payload)

	case frameEOF:
		st := s.lookup(f.id)
		if st == nil {
			return nil
		}
		if !st.isOpen() {
			return errors.New("EOF before OPENED")
		}
		if release := st.remoteEOFArrived(); release {
			s.releaseStream(st, nil, false)
		}
		return nil

	case frameRST:
		st := s.lookup(f.id)
		if st == nil {
			return nil
		}
		rep := RepGeneralFailure
		if len(f.payload) > 0 {
			rep = f.payload[0]
		}
		st.deliverOpen(openResult{err: &RepError{Rep: rep, Err: errStreamReset}})
		s.releaseStream(st, &RepError{Rep: rep, Err: errStreamReset}, false)
		return nil

	default:
		return fmt.Errorf("unexpected frame type 0x%02x", f.typ)
	}
}

func (s *session) onData(st *stream, payload []byte) error {
	if !st.isOpen() {
		return errors.New("DATA before OPENED")
	}
	if st.proto == protoUDP {
		src, n, err := decodeAddrPort(payload)
		if err != nil {
			return fmt.Errorf("bad UDP DATA: %w", err)
		}
		st.enqueueDatagram(src, payload[n:])
		s.countDown(len(payload) - n)
		return nil
	}
	s.flowMu.Lock()
	if s.connOutstanding+int64(len(payload)) > ConnWindow {
		s.flowMu.Unlock()
		return errors.New("connection window exceeded")
	}
	s.connOutstanding += int64(len(payload))
	s.flowMu.Unlock()

	switch st.enqueue(payload) {
	case enqueueOK:
		s.countDown(len(payload))
	case enqueueAfterEOF, enqueueOverWindow:
		s.flowMu.Lock()
		s.connOutstanding -= int64(len(payload))
		s.flowMu.Unlock()
		s.releaseStream(st, errStreamReset, true)
	}
	return nil
}

func (s *session) countDown(n int) {
	if s.mux.hooks.Bytes != nil && n > 0 {
		s.mux.hooks.Bytes.AddDown(int64(n))
	}
}

func (s *session) countUp(n int) {
	if s.mux.hooks.Bytes != nil && n > 0 {
		s.mux.hooks.Bytes.AddUp(int64(n))
	}
}

// consumed is called when n bytes of TCP DATA have been passed to hev; it
// emits the connection WINDOW once half of ConnWindow has been consumed.
func (s *session) consumed(n int) {
	if n <= 0 {
		return
	}
	s.flowMu.Lock()
	s.connOutstanding -= int64(n)
	s.connConsumed += int64(n)
	var inc int64
	if s.connConsumed >= ConnWindow/2 {
		inc = s.connConsumed
		s.connConsumed = 0
	}
	s.flowMu.Unlock()
	if inc > 0 {
		_ = s.send(frameWindow, 0, be32(uint32(inc)))
	}
}

func be32(v uint32) []byte {
	return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

// releaseStream drops st from the table. rst says whether to send RST to
// the exit (not after a clean two-way EOF, not after the exit's own RST or
// OPEN_FAIL, not once the session is gone).
func (s *session) releaseStream(st *stream, err error, rst bool) {
	s.mu.Lock()
	if s.streams[st.id] != st {
		s.mu.Unlock()
		return
	}
	delete(s.streams, st.id)
	s.mu.Unlock()

	queued := st.release(err)
	if queued > 0 {
		s.consumed(queued)
	}
	if rst {
		_ = s.send(frameRST, st.id, nil)
	}
}

// openStream allocates the next odd id and sends OPEN.
func (s *session) openStream(proto byte, extra []byte) (*stream, error) {
	s.mu.Lock()
	select {
	case <-s.done:
		s.mu.Unlock()
		return nil, errSessionClosed
	default:
	}
	if len(s.streams) >= MaxStreams {
		s.mu.Unlock()
		return nil, &RepError{Rep: RepGeneralFailure, Err: errors.New("max streams reached")}
	}
	id := s.nextID
	s.nextID += 2
	st := newStream(s, id, proto)
	s.streams[id] = st
	s.mu.Unlock()

	payload := append([]byte{proto}, extra...)
	if err := s.send(frameOpen, id, payload); err != nil {
		s.releaseStream(st, err, false)
		return nil, err
	}
	return st, nil
}

// awaitOpen waits for OPENED or OPEN_FAIL under the kvm-side OPEN timer.
func (s *session) awaitOpen(ctx context.Context, st *stream) error {
	timer := time.NewTimer(openTimeout)
	defer timer.Stop()
	select {
	case res := <-st.opened:
		if res.err != nil {
			return res.err
		}
		return nil
	case <-timer.C:
		s.releaseStream(st, &RepError{Rep: RepTTLExpired}, true)
		return &RepError{Rep: RepTTLExpired, Err: errors.New("OPEN timed out")}
	case <-ctx.Done():
		s.releaseStream(st, ctx.Err(), true)
		return ctx.Err()
	case <-s.done:
		return errSessionClosed
	}
}

// DialTCP opens a TCP stream through the exit. The front door has already
// applied the policy; the mux applies it again at OPEN per D21.
func (s *session) DialTCP(ctx context.Context, dst netip.AddrPort) (net.Conn, error) {
	if !s.mux.policy().Allow(dst.Addr()) {
		return nil, &RepError{Rep: RepNotAllowed, Err: fmt.Errorf("destination %s denied", dst)}
	}
	st, err := s.openStream(protoTCP, encodeAddrPort(dst))
	if err != nil {
		return nil, err
	}
	st.remote = dst
	if err := s.awaitOpen(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// OpenUDP opens one UDP association through the exit.
func (s *session) OpenUDP(ctx context.Context) (UDPStream, error) {
	st, err := s.openStream(protoUDP, nil)
	if err != nil {
		return nil, err
	}
	if err := s.awaitOpen(ctx, st); err != nil {
		return nil, err
	}
	go st.udpSender()
	return st, nil
}

// stream is one nexit/1 stream. For TCP it is the net.Conn the front door
// copies into; for UDP it is the UDPStream.
type stream struct {
	sess   *session
	id     uint32
	proto  byte
	remote netip.AddrPort

	opened   chan openResult
	openOnce sync.Once

	mu        sync.Mutex
	cond      *sync.Cond
	open      bool
	released  bool
	err       error
	remoteEOF bool
	localEOF  bool
	bound     netip.AddrPort
	recvQ     [][]byte
	recvBytes int // queued, not yet read by hev (stream-level outstanding)
	consumed  int // read by hev since the last WINDOW
	udpQ      []datagram
	readDL    *deadline
	writeDL   *deadline

	sendCredit *credit
	sendQ      chan []byte // UDP only: at most UDPQueue unsent datagrams
	ctx        context.Context
	cancel     context.CancelFunc
}

type openResult struct {
	bound netip.AddrPort
	err   error
}

type datagram struct {
	src     netip.AddrPort
	payload []byte
}

func newStream(s *session, id uint32, proto byte) *stream {
	ctx, cancel := context.WithCancel(context.Background())
	st := &stream{
		sess:       s,
		id:         id,
		proto:      proto,
		opened:     make(chan openResult, 1),
		sendCredit: newCredit(StreamWindow),
		readDL:     newDeadline(),
		writeDL:    newDeadline(),
		ctx:        ctx,
		cancel:     cancel,
	}
	st.cond = sync.NewCond(&st.mu)
	if proto == protoUDP {
		st.sendQ = make(chan []byte, UDPQueue)
	}
	st.readDL.onExpire = func() { st.mu.Lock(); st.mu.Unlock(); st.cond.Broadcast() } //nolint:staticcheck
	return st
}

func (st *stream) deliverOpen(res openResult) {
	st.openOnce.Do(func() {
		if res.err == nil {
			st.mu.Lock()
			st.open = true
			st.bound = res.bound
			st.mu.Unlock()
		}
		st.opened <- res
	})
}

func (st *stream) isOpen() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.open
}

type enqueueResult int

const (
	enqueueOK enqueueResult = iota
	enqueueAfterEOF
	enqueueOverWindow
	enqueueReleased
)

// enqueue appends exit->kvm DATA to the bounded queue. The reader never
// blocks here: the bound is the credit the exit was granted.
func (st *stream) enqueue(payload []byte) enqueueResult {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.released {
		return enqueueReleased
	}
	if st.remoteEOF {
		return enqueueAfterEOF
	}
	if st.recvBytes+len(payload) > StreamWindow {
		return enqueueOverWindow
	}
	buf := make([]byte, len(payload))
	copy(buf, payload)
	st.recvQ = append(st.recvQ, buf)
	st.recvBytes += len(buf)
	st.cond.Broadcast()
	return enqueueOK
}

func (st *stream) enqueueDatagram(src netip.AddrPort, payload []byte) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.released {
		return
	}
	buf := make([]byte, len(payload))
	copy(buf, payload)
	if len(st.udpQ) >= UDPQueue {
		st.udpQ = st.udpQ[1:]
	}
	st.udpQ = append(st.udpQ, datagram{src: src, payload: buf})
	st.cond.Broadcast()
}

// remoteEOFArrived records the exit's half close and reports whether the
// stream is now fully closed (both EOFs exchanged).
func (st *stream) remoteEOFArrived() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.remoteEOF = true
	st.cond.Broadcast()
	return st.localEOF
}

// release marks the stream dead. A clean release (err nil: both EOFs
// exchanged) keeps what the exit sent so hev still reads it; an abort drops
// it and returns the queued TCP bytes so the connection window can be
// credited back.
func (st *stream) release(err error) int {
	st.mu.Lock()
	if st.released {
		st.mu.Unlock()
		return 0
	}
	st.released = true
	queued := 0
	if err == nil {
		err = io.EOF
	} else {
		queued = st.recvBytes
		st.recvQ = nil
		st.recvBytes = 0
		st.udpQ = nil
	}
	st.err = err
	st.mu.Unlock()
	st.cond.Broadcast()
	st.cancel()
	st.sendCredit.close(err)
	st.readDL.stop()
	st.writeDL.stop()
	return queued
}

// net.Conn (TCP streams).

func (st *stream) Read(p []byte) (int, error) {
	st.mu.Lock()
	for len(st.recvQ) == 0 {
		if st.released {
			err := st.err
			st.mu.Unlock()
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			return 0, err
		}
		if st.remoteEOF {
			st.mu.Unlock()
			return 0, io.EOF
		}
		if st.readDL.expired() {
			st.mu.Unlock()
			return 0, os.ErrDeadlineExceeded
		}
		st.cond.Wait()
	}
	head := st.recvQ[0]
	n := copy(p, head)
	if n == len(head) {
		st.recvQ = st.recvQ[1:]
	} else {
		st.recvQ[0] = head[n:]
	}
	st.recvBytes -= n
	st.consumed += n
	var inc int
	if st.consumed >= StreamWindow/2 {
		inc = st.consumed
		st.consumed = 0
	}
	st.mu.Unlock()

	if inc > 0 {
		_ = st.sess.send(frameWindow, st.id, be32(uint32(inc)))
	}
	st.sess.consumed(n)
	return n, nil
}

func (st *stream) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		st.mu.Lock()
		if st.released {
			err := st.err
			st.mu.Unlock()
			return total, wrapWriteErr(err)
		}
		if st.localEOF {
			st.mu.Unlock()
			return total, net.ErrClosed
		}
		st.mu.Unlock()

		want := len(p)
		if want > MaxFrame {
			want = MaxFrame
		}
		ctx := st.ctx
		var cancel context.CancelFunc
		if dl, ok := st.writeDL.get(); ok {
			ctx, cancel = context.WithDeadline(ctx, dl)
		}
		n, err := takeBoth(ctx, st.sendCredit, st.sess.connSend, want)
		if cancel != nil {
			cancel()
		}
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return total, os.ErrDeadlineExceeded
			}
			return total, wrapWriteErr(err)
		}
		if err := st.sess.send(frameData, st.id, p[:n]); err != nil {
			return total, err
		}
		st.sess.countUp(n)
		p = p[n:]
		total += n
	}
	return total, nil
}

func wrapWriteErr(err error) error {
	if err == nil || errors.Is(err, io.EOF) {
		return net.ErrClosed
	}
	return err
}

// CloseWrite half-closes the kvm -> exit direction (EOF).
func (st *stream) CloseWrite() error {
	st.mu.Lock()
	if st.released || st.localEOF {
		st.mu.Unlock()
		return nil
	}
	st.localEOF = true
	both := st.remoteEOF
	st.mu.Unlock()
	if err := st.sess.send(frameEOF, st.id, nil); err != nil {
		return err
	}
	if both {
		st.sess.releaseStream(st, nil, false)
	}
	return nil
}

// Close aborts the stream with RST unless both EOFs were already exchanged.
func (st *stream) Close() error {
	st.mu.Lock()
	released := st.released
	st.mu.Unlock()
	if released {
		return nil
	}
	st.sess.releaseStream(st, net.ErrClosed, true)
	return nil
}

func (st *stream) LocalAddr() net.Addr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.bound.IsValid() {
		return net.TCPAddrFromAddrPort(st.bound)
	}
	return &net.TCPAddr{IP: net.IPv4zero}
}

func (st *stream) RemoteAddr() net.Addr {
	if st.remote.IsValid() {
		return net.TCPAddrFromAddrPort(st.remote)
	}
	return &net.TCPAddr{IP: net.IPv4zero}
}

func (st *stream) SetDeadline(t time.Time) error {
	st.readDL.set(t)
	st.writeDL.set(t)
	return nil
}

func (st *stream) SetReadDeadline(t time.Time) error  { st.readDL.set(t); return nil }
func (st *stream) SetWriteDeadline(t time.Time) error { st.writeDL.set(t); return nil }

// UDPStream (UDP streams).

func (st *stream) Send(dst netip.AddrPort, payload []byte) error {
	st.mu.Lock()
	released := st.released
	st.mu.Unlock()
	if released {
		return net.ErrClosed
	}
	hdr := encodeAddrPort(dst)
	if len(hdr)+len(payload) > MaxFrame {
		return nil // dropped by the sender, per spec
	}
	b := make([]byte, 0, len(hdr)+len(payload))
	b = append(b, hdr...)
	b = append(b, payload...)
	for {
		select {
		case st.sendQ <- b:
			return nil
		default:
		}
		// Full: drop the oldest and retry.
		select {
		case <-st.sendQ:
		default:
		}
	}
}

func (st *stream) udpSender() {
	for {
		select {
		case b := <-st.sendQ:
			if err := st.sess.send(frameData, st.id, b); err != nil {
				return
			}
			st.sess.countUp(udpPayloadLen(b))
		case <-st.ctx.Done():
			return
		}
	}
}

func udpPayloadLen(b []byte) int {
	if _, n, err := decodeAddrPort(b); err == nil {
		return len(b) - n
	}
	return 0
}

func (st *stream) Recv(ctx context.Context) (netip.AddrPort, []byte, error) {
	stop := context.AfterFunc(ctx, func() {
		st.mu.Lock()
		st.mu.Unlock() //nolint:staticcheck
		st.cond.Broadcast()
	})
	defer stop()
	st.mu.Lock()
	defer st.mu.Unlock()
	for len(st.udpQ) == 0 {
		if st.released {
			return netip.AddrPort{}, nil, st.err
		}
		if err := ctx.Err(); err != nil {
			return netip.AddrPort{}, nil, err
		}
		st.cond.Wait()
	}
	d := st.udpQ[0]
	st.udpQ = st.udpQ[1:]
	return d.src, d.payload, nil
}

// deadline is a settable point in time that wakes a waiter when it passes.
type deadline struct {
	mu       sync.Mutex
	t        time.Time
	timer    *time.Timer
	onExpire func()
}

func newDeadline() *deadline { return &deadline{} }

func (d *deadline) set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	d.t = t
	if t.IsZero() {
		return
	}
	dur := time.Until(t)
	if dur <= 0 {
		if d.onExpire != nil {
			go d.onExpire()
		}
		return
	}
	if d.onExpire != nil {
		d.timer = time.AfterFunc(dur, d.onExpire)
	}
}

func (d *deadline) get() (time.Time, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.t, !d.t.IsZero()
}

func (d *deadline) expired() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.t.IsZero() && !time.Now().Before(d.t)
}

func (d *deadline) stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
