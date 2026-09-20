package main

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

var (
	errSessionClosed = errors.New("session closed")
	errStreamReset   = errors.New("stream reset")
)

// session is one nexit/1 exit session: it holds a WebSocket to the NanoKVM
// and originates every TCP connection and UDP datagram the consumer behind
// that NanoKVM asks for. It is the Go restatement of what client.py does, and
// of FakeExit in the server's tests, which is the conformance oracle.
type session struct {
	pol      policy
	hostname string
	os       string
	log      func(string, ...any)

	conn    *websocket.Conn
	welcome welcome
	writeCh chan []byte

	done     chan struct{}
	doneOnce sync.Once
	errMu    sync.Mutex
	err      error

	mu       sync.Mutex
	streams  map[uint32]*stream
	released map[uint32]byte // proto of ids that left the table, for late DATA
	connSend *credit

	flowMu       sync.Mutex
	connConsumed int64
}

// dial upgrades url, sends HELLO and waits for WELCOME. On success the session
// runs until the kvm ends it or close is called.
func (s *session) dial(ctx context.Context, url string, header http.Header, tlsSkipVerify bool) error {
	dialer := websocket.Dialer{
		ReadBufferSize:   32 << 10,
		WriteBufferSize:  32 << 10,
		HandshakeTimeout: 15 * time.Second,
	}
	if tlsSkipVerify {
		dialer.TLSClientConfig = insecureTLS()
	}
	conn, resp, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return errors.New("rejected (404): wrong passcode or slot, or this source is rate limited")
		}
		if resp != nil {
			return fmt.Errorf("%w (status %d)", err, resp.StatusCode)
		}
		return err
	}
	conn.SetReadLimit(int64(maxFrame + 64))

	s.conn = conn
	s.writeCh = make(chan []byte, 256)
	s.done = make(chan struct{})
	s.streams = make(map[uint32]*stream)
	s.released = make(map[uint32]byte)
	go s.writer()

	flags := byte(0)
	if hasDefaultRoute() {
		flags |= helloFlagIPv4Default
	}
	if err := s.send(frameHello, 0, encodeHello(flags, s.hostname, s.os)); err != nil {
		return err
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	mt, data, err := conn.ReadMessage()
	if err != nil {
		s.close()
		return fmt.Errorf("no WELCOME: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})
	if mt != websocket.BinaryMessage {
		s.close()
		return errors.New("WELCOME was not a binary message")
	}
	fr, err := decodeFrame(data)
	if err != nil {
		s.close()
		return err
	}
	if fr.typ != frameWelcome || fr.id != 0 {
		s.close()
		return fmt.Errorf("expected WELCOME, got 0x%02x", fr.typ)
	}
	w, err := decodeWelcome(fr.payload)
	if err != nil {
		s.close()
		return err
	}
	if w.version != nexitVersion {
		s.close()
		return fmt.Errorf("unsupported protocol version %d", w.version)
	}
	s.welcome = w
	s.connSend = newCredit(int64(w.connWindow))
	go s.readLoop()
	return nil
}

func (s *session) wait() error {
	<-s.done
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

func (s *session) close() { s.finish(errSessionClosed) }

func (s *session) finish(err error) {
	s.doneOnce.Do(func() {
		s.errMu.Lock()
		s.err = err
		s.errMu.Unlock()
		close(s.done)
		if s.conn != nil {
			_ = s.conn.Close()
		}
		s.mu.Lock()
		live := make([]*stream, 0, len(s.streams))
		for _, st := range s.streams {
			live = append(live, st)
		}
		s.streams = map[uint32]*stream{}
		s.mu.Unlock()
		for _, st := range live {
			st.abort()
		}
		if s.connSend != nil {
			s.connSend.close(errSessionClosed)
		}
	})
}

func (s *session) writer() {
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()
	for {
		select {
		case msg := <-s.writeCh:
			_ = s.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if err := s.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				s.finish(err)
				return
			}
		case <-ping.C:
			_ = s.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.finish(err)
				return
			}
		case <-s.done:
			return
		}
	}
}

func (s *session) send(typ byte, id uint32, payload []byte) error {
	select {
	case s.writeCh <- encodeFrame(typ, id, payload):
		return nil
	case <-s.done:
		return errSessionClosed
	}
}

func (s *session) readLoop() {
	for {
		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			s.finish(err)
			return
		}
		if mt != websocket.BinaryMessage {
			s.finish(errors.New("non-binary message from the NanoKVM"))
			return
		}
		fr, err := decodeFrame(data)
		if err != nil {
			s.finish(err)
			return
		}
		if err := s.handle(fr); err != nil {
			s.finish(err)
			return
		}
	}
}

func (s *session) lookup(id uint32) *stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streams[id]
}

func (s *session) drop(st *stream) {
	s.mu.Lock()
	if s.streams[st.id] == st {
		delete(s.streams, st.id)
		s.released[st.id] = st.proto
	}
	s.mu.Unlock()
	st.abort()
}

// creditConn accounts n bytes of TCP DATA against the connection window and
// sends the stream-0 WINDOW once half of it is spent.
func (s *session) creditConn(n int) {
	if n <= 0 {
		return
	}
	s.flowMu.Lock()
	s.connConsumed += int64(n)
	var inc int64
	if s.connConsumed >= int64(s.welcome.connWindow)/2 {
		inc = s.connConsumed
		s.connConsumed = 0
	}
	s.flowMu.Unlock()
	if inc > 0 {
		_ = s.send(frameWindow, 0, be32(uint32(inc)))
	}
}

func (s *session) handle(fr frame) error {
	switch fr.typ {
	case frameOpen:
		return s.onOpen(fr)
	case frameData:
		st := s.lookup(fr.id)
		if st == nil {
			// Frames for unknown ids are ignored, but TCP DATA that raced our
			// RST still cost the kvm connection credit, so it is given back.
			s.mu.Lock()
			proto, released := s.released[fr.id]
			s.mu.Unlock()
			if released && proto == protoTCP {
				s.creditConn(len(fr.payload))
			}
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
			s.drop(st)
			return s.send(frameRST, st.id, []byte{reasonGeneral})
		}
		st.queue.push(queued{data: append([]byte(nil), fr.payload...)})
		return nil
	case frameEOF:
		if st := s.lookup(fr.id); st != nil {
			st.markRemoteEOF()
			st.queue.push(queued{eof: true})
		}
		return nil
	case frameRST:
		if st := s.lookup(fr.id); st != nil {
			s.drop(st)
		}
		return nil
	case frameWindow:
		if len(fr.payload) < 4 {
			return errBadFrame
		}
		inc := int64(binary.BigEndian.Uint32(fr.payload))
		if fr.id == 0 {
			s.connSend.add(inc)
		} else if st := s.lookup(fr.id); st != nil && st.proto == protoTCP {
			st.sendCredit.add(inc)
		}
		return nil
	default:
		return fmt.Errorf("unexpected frame 0x%02x", fr.typ)
	}
}

func (s *session) onOpen(fr frame) error {
	if fr.id == 0 || fr.id%2 == 0 {
		return fmt.Errorf("OPEN on invalid stream id %d", fr.id)
	}
	if s.lookup(fr.id) != nil {
		return fmt.Errorf("OPEN on live stream %d", fr.id)
	}
	if len(fr.payload) < 1 {
		return errBadFrame
	}
	proto := fr.payload[0]
	st := &stream{
		id:         fr.id,
		proto:      proto,
		sendCredit: newCredit(int64(s.welcome.streamWindow)),
		queue:      newQueue(),
	}
	st.ctx, st.cancel = context.WithCancel(context.Background())

	switch proto {
	case protoTCP:
		dst, _, err := decodeAddrPort(fr.payload[1:])
		if err != nil {
			return s.send(frameOpenFail, fr.id, []byte{reasonAtypUnsupp})
		}
		if !s.pol.allow(dst.Addr()) {
			return s.send(frameOpenFail, fr.id, []byte{reasonNotAllowed})
		}
		go s.connect(st, dst)
		return nil
	case protoUDP:
		pc, err := net.ListenPacket("udp4", "0.0.0.0:0")
		if err != nil {
			return s.send(frameOpenFail, fr.id, []byte{reasonGeneral})
		}
		st.pc = pc
		s.mu.Lock()
		s.streams[fr.id] = st
		s.mu.Unlock()
		if err := s.send(frameOpened, fr.id, nil); err != nil {
			return err
		}
		go s.udpRecv(st)
		go s.drainUDP(st)
		return nil
	default:
		return s.send(frameOpenFail, fr.id, []byte{reasonGeneral})
	}
}

// connect dials dst with the spec's 6 s budget, off the reader goroutine so a
// slow destination never stalls the session.
func (s *session) connect(st *stream, dst netip.AddrPort) {
	conn, err := net.DialTimeout("tcp", dst.String(), 6*time.Second)
	if err != nil {
		_ = s.send(frameOpenFail, st.id, []byte{reasonOf(err)})
		return
	}
	st.conn = conn
	s.mu.Lock()
	select {
	case <-s.done:
		s.mu.Unlock()
		_ = conn.Close()
		return
	default:
	}
	s.streams[st.id] = st
	s.mu.Unlock()

	bound := netip.AddrPort{}
	if tcp, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		bound = tcp.AddrPort()
	}
	if !bound.IsValid() {
		bound = netip.AddrPortFrom(netip.IPv4Unspecified(), 0)
	}
	if err := s.send(frameOpened, st.id, encodeAddrPort(bound)); err != nil {
		return
	}
	go s.pump(st)
	go s.drainTCP(st)
}

// reasonOf is the errno mapping of the spec's "Reasons and timeouts".
func reasonOf(err error) byte {
	switch {
	case errors.Is(err, syscall.ECONNREFUSED):
		return reasonRefused
	case errors.Is(err, syscall.EHOSTUNREACH):
		return reasonHostUnreach
	case errors.Is(err, syscall.ENETUNREACH):
		return reasonNetUnreach
	case errors.Is(err, syscall.ETIMEDOUT):
		return reasonTimeout
	case errors.Is(err, syscall.EADDRNOTAVAIL), errors.Is(err, syscall.EAFNOSUPPORT):
		return reasonAtypUnsupp
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return reasonTimeout
	}
	return reasonGeneral
}

// pump moves socket bytes into DATA frames within credit, then EOF.
func (s *session) pump(st *stream) {
	buf := make([]byte, maxFrame)
	for {
		n, err := st.conn.Read(buf)
		if n > 0 {
			p := buf[:n]
			for len(p) > 0 {
				m, cerr := takeBoth(st.ctx, st.sendCredit, s.connSend, len(p))
				if cerr != nil {
					return
				}
				if serr := s.send(frameData, st.id, p[:m]); serr != nil {
					return
				}
				p = p[m:]
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				_ = s.send(frameEOF, st.id, nil)
				if st.markLocalEOF() {
					s.drop(st)
				}
				return
			}
			if st.isAborted() {
				return
			}
			s.drop(st)
			_ = s.send(frameRST, st.id, []byte{reasonGeneral})
			return
		}
	}
}

// drainTCP writes queued DATA to the socket and emits WINDOW at half.
func (s *session) drainTCP(st *stream) {
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
				s.drop(st)
			}
			continue
		}
		if _, err := st.conn.Write(q.data); err != nil {
			s.drop(st)
			_ = s.send(frameRST, st.id, []byte{reasonGeneral})
			return
		}
		if inc := st.consume(len(q.data), int(s.welcome.streamWindow)); inc > 0 {
			_ = s.send(frameWindow, st.id, be32(uint32(inc)))
		}
		s.creditConn(len(q.data))
	}
}

func (s *session) udpRecv(st *stream) {
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
		if len(hdr)+n > maxFrame {
			continue
		}
		payload := make([]byte, 0, len(hdr)+n)
		payload = append(payload, hdr...)
		payload = append(payload, buf[:n]...)
		_ = s.send(frameData, st.id, payload)
	}
}

func (s *session) drainUDP(st *stream) {
	for {
		q, ok := st.queue.pop(st.ctx)
		if !ok {
			return
		}
		if !s.pol.allow(q.dst.Addr()) {
			continue
		}
		_, _ = st.pc.WriteTo(q.data, net.UDPAddrFromAddrPort(q.dst))
	}
}

func be32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}
