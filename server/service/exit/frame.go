package exit

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"sync"
)

// nexit/1 wire format (spec, "The nexit/1 protocol"). Every frame is
// type:u8, stream:u32be, payload; exactly one frame per WebSocket binary
// message.
const (
	frameHello    byte = 0x01
	frameWelcome  byte = 0x02
	frameOpen     byte = 0x10
	frameOpened   byte = 0x11
	frameOpenFail byte = 0x12
	frameData     byte = 0x20
	frameEOF      byte = 0x21
	frameRST      byte = 0x22
	frameWindow   byte = 0x30

	frameHeaderLen = 5
	nexitVersion   = 1

	protoTCP byte = 1
	protoUDP byte = 2

	atypIPv4   byte = 1
	atypDomain byte = 3
	atypIPv6   byte = 4

	// helloFlagIPv4Default is bit 0 of HELLO.flags: the exit has an IPv4
	// default route.
	helloFlagIPv4Default byte = 0x01
)

var errBadFrame = errors.New("nexit: malformed frame")

type frame struct {
	typ     byte
	id      uint32
	payload []byte
}

// encodeFrame allocates a fresh message so callers may reuse payload.
func encodeFrame(typ byte, id uint32, payload []byte) []byte {
	b := make([]byte, frameHeaderLen+len(payload))
	b[0] = typ
	binary.BigEndian.PutUint32(b[1:5], id)
	copy(b[5:], payload)
	return b
}

func decodeFrame(b []byte) (frame, error) {
	if len(b) < frameHeaderLen {
		return frame{}, errBadFrame
	}
	if len(b)-frameHeaderLen > MaxFrame {
		return frame{}, fmt.Errorf("nexit: payload %d exceeds %d", len(b)-frameHeaderLen, MaxFrame)
	}
	return frame{typ: b[0], id: binary.BigEndian.Uint32(b[1:5]), payload: b[5:]}, nil
}

// encodeAddrPort renders atyp, addr, port:u16be. IPv4-mapped v6 is sent as v4.
func encodeAddrPort(ap netip.AddrPort) []byte {
	a := ap.Addr().Unmap()
	if a.Is4() {
		v := a.As4()
		out := make([]byte, 1+4+2)
		out[0] = atypIPv4
		copy(out[1:5], v[:])
		binary.BigEndian.PutUint16(out[5:], ap.Port())
		return out
	}
	v := a.As16()
	out := make([]byte, 1+16+2)
	out[0] = atypIPv6
	copy(out[1:17], v[:])
	binary.BigEndian.PutUint16(out[17:], ap.Port())
	return out
}

// decodeAddrPort parses atyp, addr, port and returns how many bytes it used.
// Domain names (atyp 3) are rejected: they never appear on this wire.
func decodeAddrPort(b []byte) (netip.AddrPort, int, error) {
	if len(b) < 1 {
		return netip.AddrPort{}, 0, errBadFrame
	}
	switch b[0] {
	case atypIPv4:
		if len(b) < 7 {
			return netip.AddrPort{}, 0, errBadFrame
		}
		var v [4]byte
		copy(v[:], b[1:5])
		return netip.AddrPortFrom(netip.AddrFrom4(v), binary.BigEndian.Uint16(b[5:7])), 7, nil
	case atypIPv6:
		if len(b) < 19 {
			return netip.AddrPort{}, 0, errBadFrame
		}
		var v [16]byte
		copy(v[:], b[1:17])
		return netip.AddrPortFrom(netip.AddrFrom16(v), binary.BigEndian.Uint16(b[17:19])), 19, nil
	default:
		return netip.AddrPort{}, 0, fmt.Errorf("nexit: address type %d not supported", b[0])
	}
}

// hello is the decoded HELLO payload. Strings are prefixed by one length
// byte; hostname and os are therefore at most 255 bytes each.
type hello struct {
	version  byte
	flags    byte
	hostname string
	os       string
}

func decodeHello(b []byte) (hello, error) {
	if len(b) < 2 {
		return hello{}, errBadFrame
	}
	h := hello{version: b[0], flags: b[1]}
	rest := b[2:]
	var err error
	if h.hostname, rest, err = readLenString(rest); err != nil {
		return hello{}, err
	}
	if h.os, _, err = readLenString(rest); err != nil {
		return hello{}, err
	}
	return h, nil
}

func encodeHello(h hello) []byte {
	out := []byte{h.version, h.flags}
	out = appendLenString(out, h.hostname)
	out = appendLenString(out, h.os)
	return out
}

func readLenString(b []byte) (string, []byte, error) {
	if len(b) < 1 {
		return "", nil, errBadFrame
	}
	n := int(b[0])
	if len(b) < 1+n {
		return "", nil, errBadFrame
	}
	return string(b[1 : 1+n]), b[1+n:], nil
}

func appendLenString(b []byte, s string) []byte {
	if len(s) > 255 {
		s = s[:255]
	}
	b = append(b, byte(len(s)))
	return append(b, s...)
}

// welcome is the WELCOME payload.
type welcome struct {
	version      byte
	streamWindow uint32
	connWindow   uint32
	maxStreams   uint16
}

func encodeWelcome(w welcome) []byte {
	out := make([]byte, 1+4+4+2)
	out[0] = w.version
	binary.BigEndian.PutUint32(out[1:5], w.streamWindow)
	binary.BigEndian.PutUint32(out[5:9], w.connWindow)
	binary.BigEndian.PutUint16(out[9:11], w.maxStreams)
	return out
}

func decodeWelcome(b []byte) (welcome, error) {
	if len(b) < 11 {
		return welcome{}, errBadFrame
	}
	return welcome{
		version:      b[0],
		streamWindow: binary.BigEndian.Uint32(b[1:5]),
		connWindow:   binary.BigEndian.Uint32(b[5:9]),
		maxStreams:   binary.BigEndian.Uint16(b[9:11]),
	}, nil
}

// credit is one additive flow-control window (HTTP/2 WINDOW_UPDATE
// semantics). take blocks while the window is empty.
type credit struct {
	mu     sync.Mutex
	cond   *sync.Cond
	avail  int64
	closed bool
	err    error
}

func newCredit(n int64) *credit {
	c := &credit{avail: n}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *credit) add(n int64) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	c.avail += n
	c.mu.Unlock()
	c.cond.Broadcast()
}

// take returns min(want, available) once at least one byte is available, or
// the error the window was closed with, or ctx's error.
func (c *credit) take(ctx context.Context, want int) (int, error) {
	if want <= 0 {
		return 0, nil
	}
	stop := context.AfterFunc(ctx, func() {
		c.mu.Lock()
		c.mu.Unlock() //nolint:staticcheck // the empty critical section orders the broadcast after a waiter's check
		c.cond.Broadcast()
	})
	defer stop()
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		if c.closed {
			return 0, c.err
		}
		if c.avail > 0 {
			n := int64(want)
			if n > c.avail {
				n = c.avail
			}
			c.avail -= n
			return int(n), nil
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		c.cond.Wait()
	}
}

func (c *credit) available() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.avail
}

func (c *credit) close(err error) {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		c.err = err
	}
	c.mu.Unlock()
	c.cond.Broadcast()
}

// takeBoth acquires credit from a stream window and the connection window
// for one DATA frame, returning what the connection window did not grant.
func takeBoth(ctx context.Context, stream, conn *credit, want int) (int, error) {
	n, err := stream.take(ctx, want)
	if err != nil {
		return 0, err
	}
	m, err := conn.take(ctx, n)
	if err != nil {
		stream.add(int64(n))
		return 0, err
	}
	if m < n {
		stream.add(int64(n - m))
	}
	return m, nil
}
