package utils

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// The first TCP listen in a Go process reads /proc/sys/net/core/somaxconn,
// once, through net's line reader, whose buffer is 64 KB: net/parse.go opens
// the file with make([]byte, 0, 64*1024) and io.ReadFull fills the whole
// capacity. The 5.10 kernel answers a sysctl read by kzalloc(count+1), so a
// read of 65536 bytes asks for a 128 KB contiguous block (order 5), and on this
// board, 160 MB and fragmented within minutes of boot, that fails and prints a
// page allocation failure with a stack trace on every server start (the read
// itself then falls back to a default backlog, so nothing else goes wrong).
// The read lives in the standard library, so the fix is to own the listen:
// Listen builds the socket itself, reads somaxconn with a buffer the size of
// the number it holds, and hands the descriptor to net.FileListener, which
// never reaches the 64 KB path. Every TCP listener in the server comes through
// here for that reason; one plain net.Listen anywhere brings the trace back.
func Listen(addr string) (net.Listener, error) {
	laddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return listenTCP(laddr)
}

const (
	somaxconnPath = "/proc/sys/net/core/somaxconn"
	// The largest read this package makes of a /proc file. A sysctl integer is
	// at most eleven digits and a newline; 64 bytes leaves room and stays far
	// under one page, so the kernel's copy buffer is a small allocation that
	// cannot fail for want of contiguous memory.
	procReadLimit = 64
)

// listenBacklog is what net's maxListenerBacklog computes, with a small read.
func listenBacklog() int {
	file, err := os.Open(somaxconnPath)
	if err != nil {
		return syscall.SOMAXCONN
	}
	defer file.Close()
	return backlogFrom(file)
}

// backlogFrom parses one sysctl integer from r with a single bounded read. A
// value that is missing, unparsable or zero yields the platform default, as it
// does in the standard library.
func backlogFrom(r io.Reader) int {
	buf := make([]byte, procReadLimit)
	n, err := r.Read(buf)
	if n == 0 && err != nil {
		return syscall.SOMAXCONN
	}
	field, _, _ := strings.Cut(strings.TrimSpace(string(buf[:n])), " ")
	value, err := strconv.Atoi(field)
	if err != nil || value <= 0 {
		return syscall.SOMAXCONN
	}
	return value
}

func listenError(laddr *net.TCPAddr, op string, err error) error {
	return fmt.Errorf("listen tcp %s: %s: %w", laddr, op, err)
}
