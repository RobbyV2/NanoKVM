package exit

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

var udpIdleTimeout = UDPIdleTimeout

// serveUDPAssociate is the SOCKS UDP ASSOCIATE relay of the spec's "UDP"
// subsection: one loopback UDP socket per association, returned as
// BND.ADDR, bound to the source of hev's first datagram, mapped 1:1 onto one
// nexit UDP stream, torn down when the control TCP connection closes or after
// udpIdleTimeout without traffic.
func (f *FrontDoor) serveUDPAssociate(control net.Conn, stream UDPStream) {
	pc, err := listenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		_ = stream.Close()
		_ = writeSocksReply(control, RepGeneralFailure, netip.AddrPort{})
		return
	}
	if err := writeSocksReply(control, RepSucceeded, addrPortOf(pc.LocalAddr())); err != nil {
		_ = pc.Close()
		_ = stream.Close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	touch := func() { lastActivity.Store(time.Now().UnixNano()) }

	// hev's control connection goes before the stream: the stream's Close
	// queues an RST behind the session's writer, which can be stalled for as
	// long as the WS write timeout by an exit that stopped reading, and hev
	// must not keep its association (and its 60 s timer) for that long.
	var wg sync.WaitGroup
	teardown := sync.OnceFunc(func() {
		cancel()
		_ = pc.Close()
		_ = control.Close()
		_ = stream.Close()
	})

	// hev -> exit.
	var peer atomic.Pointer[net.UDPAddr]
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer teardown()
		buf := make([]byte, 64<<10)
		policy := f.policy
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			ua, ok := from.(*net.UDPAddr)
			if !ok {
				continue
			}
			if p := peer.Load(); p == nil {
				peer.Store(ua)
			} else if !p.IP.Equal(ua.IP) || p.Port != ua.Port {
				continue // the association is bound to the first source
			}
			dst, payload, ok := decodeSocksUDP(buf[:n])
			if !ok || !policy().Allow(dst.Addr()) {
				continue
			}
			touch()
			// Bytes are the mux's to count in Mode A (Mode B UDP goes
			// through serveRelay's counter), so nothing is counted here.
			if err := stream.Send(dst, payload); err != nil {
				return
			}
		}
	}()

	// exit -> hev.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer teardown()
		for {
			src, payload, err := stream.Recv(ctx)
			if err != nil {
				return
			}
			p := peer.Load()
			if p == nil {
				continue // nothing to deliver to yet
			}
			touch()
			if _, err := pc.WriteTo(encodeSocksUDP(src, payload), p); err != nil {
				return
			}
		}
	}()

	// Idle watchdog.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer teardown()
		tick := udpIdleTimeout / 4
		if tick < time.Millisecond {
			tick = time.Millisecond
		}
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if time.Since(time.Unix(0, lastActivity.Load())) >= udpIdleTimeout {
					return
				}
			}
		}
	}()

	// The control connection carries nothing more; its close ends the
	// association.
	buf := make([]byte, 64)
	for {
		if _, err := control.Read(buf); err != nil {
			break
		}
	}
	teardown()
	wg.Wait()
}

// decodeSocksUDP parses RSV(2) FRAG ATYP DST.ADDR DST.PORT DATA. Fragments
// and domain destinations are dropped (hev sends literal IPs).
func decodeSocksUDP(b []byte) (netip.AddrPort, []byte, bool) {
	if len(b) < 4 || b[0] != 0 || b[1] != 0 || b[2] != 0 {
		return netip.AddrPort{}, nil, false
	}
	dst, n, err := decodeAddrPort(b[3:])
	if err != nil {
		return netip.AddrPort{}, nil, false
	}
	return dst, b[3+n:], true
}

// encodeSocksUDP renders the reply header with the datagram's source.
func encodeSocksUDP(src netip.AddrPort, payload []byte) []byte {
	hdr := encodeAddrPort(src)
	out := make([]byte, 0, 3+len(hdr)+len(payload))
	out = append(out, 0, 0, 0)
	out = append(out, hdr...)
	return append(out, payload...)
}
