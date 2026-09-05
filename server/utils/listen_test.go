package utils

import (
	"net"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A reader that remembers how much each Read asked for. The kernel allocates a
// copy buffer of the size asked, so the size asked is the property under test.
type sizedReader struct {
	data  string
	sizes []int
}

func (r *sizedReader) Read(p []byte) (int, error) {
	r.sizes = append(r.sizes, len(p))
	return strings.NewReader(r.data).Read(p)
}

func TestProcReadsStaySmall(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		want int
	}{
		{"kernel default", "4096\n", 4096},
		{"raised", "65535\n", 65535},
		{"two fields", "128 0\n", 128},
		{"empty", "", syscall.SOMAXCONN},
		{"zero", "0\n", syscall.SOMAXCONN},
		{"garbage", "many\n", syscall.SOMAXCONN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &sizedReader{data: tc.data}
			if got := backlogFrom(reader); got != tc.want {
				t.Fatalf("backlogFrom(%q) = %d, want %d", tc.data, got, tc.want)
			}
			if len(reader.sizes) != 1 {
				t.Fatalf("read %d times, want one bounded read", len(reader.sizes))
			}
			// The 64 KB the standard library asks for is what the kernel could
			// not allocate; anything under a page is a small allocation.
			if size := reader.sizes[0]; size != procReadLimit || size > 4096 {
				t.Fatalf("read asked for %d bytes, want %d", size, procReadLimit)
			}
		})
	}
}

func TestListenAcceptsOnLoopbackAndWildcard(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:0", ":0"} {
		t.Run(addr, func(t *testing.T) {
			listener, err := Listen(addr)
			if err != nil {
				t.Fatalf("Listen(%q) = %v", addr, err)
			}
			defer listener.Close()

			port := listener.Addr().(*net.TCPAddr).Port
			if port == 0 {
				t.Fatal("the listener reports no port")
			}
			accepted := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err == nil {
					buf := make([]byte, 1)
					_, err = conn.Read(buf)
					conn.Close()
				}
				accepted <- err
			}()
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			if _, err := conn.Write([]byte{1}); err != nil {
				t.Fatalf("write: %v", err)
			}
			conn.Close()
			select {
			case err := <-accepted:
				if err != nil {
					t.Fatalf("accept: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("the listener never accepted the connection")
			}
		})
	}
}

func TestListenRefusesABusyPort(t *testing.T) {
	first, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := Listen(first.Addr().String()); err == nil {
		t.Fatal("a second listener on the same port did not fail")
	}
}
