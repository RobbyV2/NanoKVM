package exit

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// FakeExit is an in-process nexit/1 exit device: the client side of the
// protocol, implemented literally from the spec so it doubles as the
// conformance oracle for the Go mux (this package's tests) and for the
// native clients (W5). It carries no build tag so other packages can use it.
//
// Zero values are the faithful client. The knobs make it misbehave on
// purpose: NoHello never sends HELLO, NoPong ignores the kvm's pings,
// ManualWindow stops the automatic WINDOW frames so a test can starve and
// then refill the kvm's credit with SendWindow.
type FakeExit struct {
	// Connect dials dst for a TCP OPEN. A non-zero reason is answered with
	// OPEN_FAIL. Nil means net.DialTimeout with the errno mapping of the spec.
	Connect func(dst netip.AddrPort) (net.Conn, byte)
	// OnOpen observes every OPEN before it is acted on; dst is invalid for UDP.
	OnOpen func(id uint32, proto byte, dst netip.AddrPort)
	// OnFrame observes every frame the fake receives after WELCOME.
	OnFrame func(typ byte, id uint32, payload []byte)
	// ListenUDP allocates the per-stream UDP socket. Nil means 127.0.0.1:0.
	ListenUDP func() (net.PacketConn, error)
	// Policy mirrors the destination policy when non-nil (OPEN_FAIL 0x02).
	Policy *Policy

	Hostname string
	OS       string
	Flags    byte

	NoHello      bool
	NoPong       bool
	ManualWindow bool

	conn     *websocket.Conn
	welcome  welcome
	writeCh  chan wsMessage
	done     chan struct{}
	doneOnce sync.Once
	errMu    sync.Mutex
	err      error

	mu       sync.Mutex
	streams  map[uint32]*fakeStream
	connSend *credit

	flowMu       sync.Mutex
	connConsumed int64
}

// NewFakeExit returns a fake with the spec's default identity.
func NewFakeExit() *FakeExit {
	return &FakeExit{
		Hostname: "fake-exit",
		OS:       "go",
		Flags:    helloFlagIPv4Default,
	}
}

// Dial upgrades url (ws:// or wss://) with header, sends HELLO and waits
// for WELCOME. On success the session runs until Close or the kvm ends it.
func (f *FakeExit) Dial(ctx context.Context, url string, header http.Header) error {
	dialer := websocket.Dialer{
		ReadBufferSize:   32 << 10,
		WriteBufferSize:  32 << 10,
		HandshakeTimeout: 10 * time.Second,
	}
	conn, resp, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("dial: %w (status %d)", err, resp.StatusCode)
		}
		return fmt.Errorf("dial: %w", err)
	}
	conn.SetReadLimit(int64(MaxFrame + 64))
	if f.NoPong {
		conn.SetPingHandler(func(string) error { return nil })
	}
	f.conn = conn
	f.writeCh = make(chan wsMessage, 256)
	f.done = make(chan struct{})
	f.streams = make(map[uint32]*fakeStream)
	go f.writer()

	if f.NoHello {
		go f.readLoop()
		return nil
	}
	if err := f.send(frameHello, 0, encodeHello(hello{
		version:  nexitVersion,
		flags:    f.Flags,
		hostname: f.Hostname,
		os:       f.OS,
	})); err != nil {
		return err
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	mt, data, err := conn.ReadMessage()
	if err != nil {
		f.Close()
		return fmt.Errorf("no WELCOME: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})
	if mt != websocket.BinaryMessage {
		f.Close()
		return errors.New("WELCOME not binary")
	}
	fr, err := decodeFrame(data)
	if err != nil {
		f.Close()
		return err
	}
	if fr.typ != frameWelcome || fr.id != 0 {
		f.Close()
		return fmt.Errorf("expected WELCOME, got 0x%02x", fr.typ)
	}
	w, err := decodeWelcome(fr.payload)
	if err != nil {
		f.Close()
		return err
	}
	if w.version != nexitVersion {
		f.Close()
		return fmt.Errorf("WELCOME version %d", w.version)
	}
	f.welcome = w
	f.connSend = newCredit(int64(w.connWindow))
	go f.readLoop()
	return nil
}

// Welcome returns the negotiated windows and stream cap.
func (f *FakeExit) Welcome() (streamWindow, connWindow uint32, maxStreams uint16) {
	return f.welcome.streamWindow, f.welcome.connWindow, f.welcome.maxStreams
}

// Done is closed when the session has ended.
func (f *FakeExit) Done() <-chan struct{} { return f.done }

// Err is why the session ended (nil while running or after a local Close).
func (f *FakeExit) Err() error {
	f.errMu.Lock()
	defer f.errMu.Unlock()
	return f.err
}

// Streams is the number of live streams the fake tracks.
func (f *FakeExit) Streams() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.streams)
}

// StreamIDs lists the live stream ids.
func (f *FakeExit) StreamIDs() []uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]uint32, 0, len(f.streams))
	for id := range f.streams {
		out = append(out, id)
	}
	return out
}

// SendWindow grants inc bytes of credit on stream id (0 for the connection).
func (f *FakeExit) SendWindow(id uint32, inc uint32) error {
	return f.send(frameWindow, id, be32(inc))
}

// SendRaw sends an arbitrary frame, for protocol-error tests.
func (f *FakeExit) SendRaw(typ byte, id uint32, payload []byte) error {
	return f.send(typ, id, payload)
}

// SendText sends a WebSocket text message, which the kvm must treat as a
// session error.
func (f *FakeExit) SendText(s string) error {
	select {
	case f.writeCh <- wsMessage{typ: websocket.TextMessage, data: []byte(s)}:
		return nil
	case <-f.done:
		return errSessionClosed
	}
}

// wsMessage is one queued WebSocket message for the single writer.
type wsMessage struct {
	typ  int
	data []byte
}

// Close ends the session from the exit side.
func (f *FakeExit) Close() {
	f.finish(nil)
}

// Wait blocks until the session has ended or d elapses.
func (f *FakeExit) Wait(d time.Duration) bool {
	select {
	case <-f.done:
		return true
	case <-time.After(d):
		return false
	}
}

func (f *FakeExit) finish(err error) {
	f.doneOnce.Do(func() {
		f.errMu.Lock()
		f.err = err
		f.errMu.Unlock()
		close(f.done)
		_ = f.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		_ = f.conn.Close()
		f.mu.Lock()
		streams := make([]*fakeStream, 0, len(f.streams))
		for _, st := range f.streams {
			streams = append(streams, st)
		}
		f.streams = map[uint32]*fakeStream{}
		f.mu.Unlock()
		for _, st := range streams {
			st.abort()
		}
		if f.connSend != nil {
			f.connSend.close(errSessionClosed)
		}
	})
}

func (f *FakeExit) writer() {
	for {
		select {
		case msg := <-f.writeCh:
			_ = f.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := f.conn.WriteMessage(msg.typ, msg.data); err != nil {
				f.finish(err)
				return
			}
		case <-f.done:
			return
		}
	}
}

func (f *FakeExit) send(typ byte, id uint32, payload []byte) error {
	select {
	case f.writeCh <- wsMessage{typ: websocket.BinaryMessage, data: encodeFrame(typ, id, payload)}:
		return nil
	case <-f.done:
		return errSessionClosed
	}
}

func (f *FakeExit) readLoop() {
	for {
		mt, data, err := f.conn.ReadMessage()
		if err != nil {
			f.finish(err)
			return
		}
		if mt != websocket.BinaryMessage {
			f.finish(errors.New("non-binary message"))
			return
		}
		fr, err := decodeFrame(data)
		if err != nil {
			f.finish(err)
			return
		}
		if f.OnFrame != nil {
			f.OnFrame(fr.typ, fr.id, fr.payload)
		}
		if err := f.handle(fr); err != nil {
			f.finish(err)
			return
		}
	}
}

func (f *FakeExit) lookup(id uint32) *fakeStream {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.streams[id]
}

func (f *FakeExit) drop(st *fakeStream) {
	f.mu.Lock()
	if f.streams[st.id] == st {
		delete(f.streams, st.id)
	}
	f.mu.Unlock()
	st.abort()
}

func (f *FakeExit) handle(fr frame) error {
	switch fr.typ {
	case frameOpen:
		return f.onOpen(fr)
	case frameData:
		st := f.lookup(fr.id)
		if st == nil {
			return nil
		}
		if st.proto == protoUDP {
			dst, n, err := decodeAddrPort(fr.payload)
			if err != nil {
				return err
			}
			st.queue.push(queued{dst: dst, data: append([]byte(nil), fr.payload[n:]...)})
			return nil
		}
		if st.remoteEOFSeen() {
			f.drop(st)
			return f.send(frameRST, st.id, []byte{RepGeneralFailure})
		}
		st.queue.push(queued{data: append([]byte(nil), fr.payload...)})
		return nil
	case frameEOF:
		st := f.lookup(fr.id)
		if st == nil {
			return nil
		}
		st.markRemoteEOF()
		st.queue.push(queued{eof: true})
		return nil
	case frameRST:
		if st := f.lookup(fr.id); st != nil {
			f.drop(st)
		}
		return nil
	case frameWindow:
		if len(fr.payload) < 4 {
			return errBadFrame
		}
		inc := int64(binary.BigEndian.Uint32(fr.payload))
		if fr.id == 0 {
			f.connSend.add(inc)
		} else if st := f.lookup(fr.id); st != nil && st.proto == protoTCP {
			st.sendCredit.add(inc)
		}
		return nil
	default:
		return fmt.Errorf("unexpected frame 0x%02x", fr.typ)
	}
}

func (f *FakeExit) onOpen(fr frame) error {
	if fr.id == 0 || fr.id%2 == 0 {
		return fmt.Errorf("OPEN on even stream %d", fr.id)
	}
	if f.lookup(fr.id) != nil {
		return fmt.Errorf("OPEN on live stream %d", fr.id)
	}
	if len(fr.payload) < 1 {
		return errBadFrame
	}
	proto := fr.payload[0]
	st := &fakeStream{
		id:         fr.id,
		proto:      proto,
		sendCredit: newCredit(int64(f.welcome.streamWindow)),
		queue:      newQueue(),
		ctx:        context.Background(),
	}
	st.ctx, st.cancel = context.WithCancel(context.Background())

	switch proto {
	case protoTCP:
		dst, _, err := decodeAddrPort(fr.payload[1:])
		if err != nil {
			if f.OnOpen != nil {
				f.OnOpen(fr.id, proto, netip.AddrPort{})
			}
			return f.send(frameOpenFail, fr.id, []byte{RepAddrTypeNotSupp})
		}
		if f.OnOpen != nil {
			f.OnOpen(fr.id, proto, dst)
		}
		if f.Policy != nil && !f.Policy.Allow(dst.Addr()) {
			return f.send(frameOpenFail, fr.id, []byte{RepNotAllowed})
		}
		go f.connect(st, dst)
		return nil
	case protoUDP:
		if f.OnOpen != nil {
			f.OnOpen(fr.id, proto, netip.AddrPort{})
		}
		listen := f.ListenUDP
		if listen == nil {
			listen = func() (net.PacketConn, error) { return net.ListenPacket("udp4", "127.0.0.1:0") }
		}
		pc, err := listen()
		if err != nil {
			return f.send(frameOpenFail, fr.id, []byte{RepGeneralFailure})
		}
		st.pc = pc
		f.mu.Lock()
		f.streams[fr.id] = st
		f.mu.Unlock()
		if err := f.send(frameOpened, fr.id, nil); err != nil {
			return err
		}
		go f.udpRecv(st)
		go f.drainUDP(st)
		return nil
	default:
		return f.send(frameOpenFail, fr.id, []byte{RepCommandNotSupp})
	}
}

// connect runs the non-blocking connect of the spec (6 s budget) off the
// reader goroutine so a slow destination never stalls the session.
func (f *FakeExit) connect(st *fakeStream, dst netip.AddrPort) {
	dial := f.Connect
	if dial == nil {
		dial = defaultConnect
	}
	conn, reason := dial(dst)
	if reason != 0 || conn == nil {
		if reason == 0 {
			reason = RepGeneralFailure
		}
		_ = f.send(frameOpenFail, st.id, []byte{reason})
		return
	}
	st.conn = conn
	f.mu.Lock()
	select {
	case <-f.done:
		f.mu.Unlock()
		_ = conn.Close()
		return
	default:
	}
	f.streams[st.id] = st
	f.mu.Unlock()

	bound := netip.AddrPort{}
	if tcp, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		bound = tcp.AddrPort()
	}
	if !bound.IsValid() {
		bound = netip.AddrPortFrom(netip.IPv4Unspecified(), 0)
	}
	if err := f.send(frameOpened, st.id, encodeAddrPort(bound)); err != nil {
		return
	}
	go f.pump(st)
	go f.drainTCP(st)
}

func defaultConnect(dst netip.AddrPort) (net.Conn, byte) {
	conn, err := net.DialTimeout("tcp", dst.String(), 6*time.Second)
	if err != nil {
		return nil, reasonOf(err)
	}
	return conn, 0
}

// reasonOf is the errno mapping of the spec's "Reasons and timeouts".
func reasonOf(err error) byte {
	switch {
	case errors.Is(err, syscall.ECONNREFUSED):
		return RepConnRefused
	case errors.Is(err, syscall.EHOSTUNREACH):
		return RepHostUnreach
	case errors.Is(err, syscall.ENETUNREACH):
		return RepNetworkUnreach
	case errors.Is(err, syscall.ETIMEDOUT):
		return RepTTLExpired
	case errors.Is(err, syscall.EADDRNOTAVAIL), errors.Is(err, syscall.EAFNOSUPPORT):
		return RepAddrTypeNotSupp
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return RepTTLExpired
	}
	return RepGeneralFailure
}

// pump moves socket bytes to DATA frames within credit, then EOF.
func (f *FakeExit) pump(st *fakeStream) {
	buf := make([]byte, MaxFrame)
	for {
		n, err := st.conn.Read(buf)
		if n > 0 {
			p := buf[:n]
			for len(p) > 0 {
				m, cerr := takeBoth(st.ctx, st.sendCredit, f.connSend, len(p))
				if cerr != nil {
					return
				}
				if serr := f.send(frameData, st.id, p[:m]); serr != nil {
					return
				}
				p = p[m:]
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				_ = f.send(frameEOF, st.id, nil)
				if st.markLocalEOF() {
					f.drop(st)
				}
				return
			}
			if st.isAborted() {
				return
			}
			f.drop(st)
			_ = f.send(frameRST, st.id, []byte{RepGeneralFailure})
			return
		}
	}
}

// drainTCP writes queued DATA to the socket and emits WINDOW at half.
func (f *FakeExit) drainTCP(st *fakeStream) {
	for {
		q, ok := st.queue.pop(st.ctx)
		if !ok {
			return
		}
		if q.eof {
			if cw, ok := st.conn.(interface{ CloseWrite() error }); ok {
				_ = cw.CloseWrite()
			}
			if st.markRemoteEOF() {
				f.drop(st)
			}
			continue
		}
		if _, err := st.conn.Write(q.data); err != nil {
			f.drop(st)
			_ = f.send(frameRST, st.id, []byte{RepGeneralFailure})
			return
		}
		if !f.ManualWindow {
			if inc := st.consume(len(q.data), int(f.welcome.streamWindow)); inc > 0 {
				_ = f.send(frameWindow, st.id, be32(uint32(inc)))
			}
			f.flowMu.Lock()
			f.connConsumed += int64(len(q.data))
			var cinc int64
			if f.connConsumed >= int64(f.welcome.connWindow)/2 {
				cinc = f.connConsumed
				f.connConsumed = 0
			}
			f.flowMu.Unlock()
			if cinc > 0 {
				_ = f.send(frameWindow, 0, be32(uint32(cinc)))
			}
		}
	}
}

func (f *FakeExit) udpRecv(st *fakeStream) {
	buf := make([]byte, 64<<10)
	for {
		n, addr, err := st.pc.ReadFrom(buf)
		if err != nil {
			return
		}
		ua, ok := addr.(*net.UDPAddr)
		if !ok {
			continue
		}
		hdr := encodeAddrPort(ua.AddrPort())
		if len(hdr)+n > MaxFrame {
			continue
		}
		payload := make([]byte, 0, len(hdr)+n)
		payload = append(payload, hdr...)
		payload = append(payload, buf[:n]...)
		_ = f.send(frameData, st.id, payload)
	}
}

func (f *FakeExit) drainUDP(st *fakeStream) {
	for {
		q, ok := st.queue.pop(st.ctx)
		if !ok {
			return
		}
		if f.Policy != nil && !f.Policy.Allow(q.dst.Addr()) {
			continue
		}
		_, _ = st.pc.WriteTo(q.data, net.UDPAddrFromAddrPort(q.dst))
	}
}

type fakeStream struct {
	id    uint32
	proto byte
	conn  net.Conn
	pc    net.PacketConn

	sendCredit *credit
	queue      *queue
	ctx        context.Context
	cancel     context.CancelFunc

	mu        sync.Mutex
	remoteEOF bool
	localEOF  bool
	aborted   bool
	consumed  int
}

func (st *fakeStream) markRemoteEOF() (both bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.remoteEOF = true
	return st.localEOF
}

func (st *fakeStream) markLocalEOF() (both bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.localEOF = true
	return st.remoteEOF
}

func (st *fakeStream) remoteEOFSeen() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.remoteEOF
}

func (st *fakeStream) isAborted() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.aborted
}

func (st *fakeStream) consume(n, window int) int {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.consumed += n
	if st.consumed >= window/2 {
		inc := st.consumed
		st.consumed = 0
		return inc
	}
	return 0
}

func (st *fakeStream) abort() {
	st.mu.Lock()
	st.aborted = true
	st.mu.Unlock()
	st.cancel()
	st.sendCredit.close(errStreamReset)
	if st.conn != nil {
		_ = st.conn.Close()
	}
	if st.pc != nil {
		_ = st.pc.Close()
	}
}

type queued struct {
	data []byte
	dst  netip.AddrPort
	eof  bool
}

// queue is an unbounded FIFO fed by the reader; the credit the fake granted
// bounds it in practice.
type queue struct {
	mu    sync.Mutex
	cond  *sync.Cond
	items []queued
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *queue) push(item queued) {
	q.mu.Lock()
	q.items = append(q.items, item)
	q.mu.Unlock()
	q.cond.Broadcast()
}

func (q *queue) pop(ctx context.Context) (queued, bool) {
	stop := context.AfterFunc(ctx, func() {
		q.mu.Lock()
		q.mu.Unlock() //nolint:staticcheck
		q.cond.Broadcast()
	})
	defer stop()
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 {
		if ctx.Err() != nil {
			return queued{}, false
		}
		q.cond.Wait()
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}
