package vnc

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func managerJPEG() []byte {
	jpeg := bytes.Repeat([]byte{0x41}, 512)
	jpeg[0], jpeg[1] = 0xff, 0xd8
	return jpeg
}

func freeAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	return addr
}

func TestManagerStartStop(t *testing.T) {
	addr := freeAddr(t)

	m := GetManager()
	m.Configure(func() *Server {
		return &Server{
			Addr:      addr,
			AllowNone: true,
			Screen:    func() (uint16, uint16, uint16, int) { return 640, 480, 80, 30 },
			ReadJPEG:  func(uint16, uint16, uint16) ([]byte, int) { return managerJPEG(), 0 },
		}
	})
	t.Cleanup(func() {
		m.Stop()
		m.Configure(nil)
	})

	if err := m.Start(); err != nil {
		t.Fatalf("start: %s", err)
	}
	if err := m.Start(); err != nil {
		t.Fatalf("second start: %s", err)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %s", err)
	}
	defer func() { _ = conn.Close() }()

	version := make([]byte, 12)
	if _, err := io.ReadFull(conn, version); err != nil {
		t.Fatalf("read version: %s", err)
	}

	m.Stop()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("connection still open after stop")
	}

	if again, err := net.Dial("tcp", addr); err == nil {
		_ = again.Close()
		t.Fatal("listener still accepting after stop")
	}

	if err := m.Start(); err != nil {
		t.Fatalf("restart: %s", err)
	}
	restarted, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial after restart: %s", err)
	}
	_ = restarted.Close()
}
