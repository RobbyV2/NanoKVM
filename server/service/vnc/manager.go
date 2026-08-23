package vnc

import (
	"errors"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Manager owns the listener so VNC can be switched on and off while the server
// runs. Stopping also closes the accepted connections: a disabled VNC service
// must not keep streaming to clients that were already attached.
type Manager struct {
	mutex    sync.Mutex
	build    func() *Server
	listener *trackingListener
}

var manager Manager

func GetManager() *Manager {
	return &manager
}

// Configure records how to build a server. The frame source and the input gate
// link against libkvm, so they are injected once from main instead of imported.
func (m *Manager) Configure(build func() *Server) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.build = build
}

func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.listener != nil {
		return nil
	}
	if m.build == nil {
		return errors.New("vnc server not configured")
	}

	server := m.build()
	base, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}

	listener := &trackingListener{Listener: base, conns: map[net.Conn]struct{}{}}
	m.listener = listener
	log.Infof("vnc server listening on %s", server.Addr)

	go func() {
		if err := server.Serve(listener); err != nil && !listener.stopped() {
			log.Errorf("vnc server stopped: %s", err)
		}
	}()

	return nil
}

func (m *Manager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.listener == nil {
		return
	}

	_ = m.listener.Close()
	m.listener = nil
}

type trackingListener struct {
	net.Listener

	mutex  sync.Mutex
	conns  map[net.Conn]struct{}
	closed bool
}

func (l *trackingListener) Accept() (net.Conn, error) {
	netConn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	l.mutex.Lock()
	if l.closed {
		l.mutex.Unlock()
		_ = netConn.Close()
		return nil, net.ErrClosed
	}
	l.conns[netConn] = struct{}{}
	l.mutex.Unlock()

	return &trackedConn{Conn: netConn, listener: l}, nil
}

func (l *trackingListener) Close() error {
	l.mutex.Lock()
	l.closed = true
	conns := make([]net.Conn, 0, len(l.conns))
	for conn := range l.conns {
		conns = append(conns, conn)
	}
	l.conns = map[net.Conn]struct{}{}
	l.mutex.Unlock()

	err := l.Listener.Close()
	for _, conn := range conns {
		_ = conn.Close()
	}

	return err
}

func (l *trackingListener) stopped() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.closed
}

func (l *trackingListener) forget(conn net.Conn) {
	l.mutex.Lock()
	delete(l.conns, conn)
	l.mutex.Unlock()
}

type trackedConn struct {
	net.Conn

	listener *trackingListener
	once     sync.Once
}

func (c *trackedConn) Close() error {
	c.once.Do(func() { c.listener.forget(c.Conn) })

	return c.Conn.Close()
}
