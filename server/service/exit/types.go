package exit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"NanoKVM-Server/proto"
)

// This file pins the contracts between the components of the package so they
// can be built independently:
//
//   front door (socks.go)  --uses-->  Backend        (mux.go implements it for Mode A)
//   front door (socks.go)  --uses-->  RelayGate      (wsproxy.go implements it for Mode B)
//   dns.go / probe.go      --dial-->  the front door as ordinary SOCKS5 clients
//   manager.go             --owns-->  all of the above per Slot, plus downstream.go
//
// Nothing outside manager.go constructs more than one of these per slot.

// SOCKS5 reply codes (RFC 1928 §6), reused as nexit/1 OPEN_FAIL and RST reasons.
const (
	RepSucceeded       byte = 0x00
	RepGeneralFailure  byte = 0x01
	RepNotAllowed      byte = 0x02
	RepNetworkUnreach  byte = 0x03
	RepHostUnreach     byte = 0x04
	RepConnRefused     byte = 0x05
	RepTTLExpired      byte = 0x06
	RepCommandNotSupp  byte = 0x07
	RepAddrTypeNotSupp byte = 0x08
)

// RepError carries a SOCKS reply code from a Backend to the front door.
type RepError struct {
	Rep byte
	Err error
}

func (e *RepError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("socks rep 0x%02x: %v", e.Rep, e.Err)
	}
	return fmt.Sprintf("socks rep 0x%02x", e.Rep)
}

func (e *RepError) Unwrap() error { return e.Err }

// RepOf extracts the reply code from err, defaulting to RepGeneralFailure.
func RepOf(err error) byte {
	var re *RepError
	if errors.As(err, &re) {
		return re.Rep
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return RepTTLExpired
	}
	return RepGeneralFailure
}

// UDPStream is one SOCKS UDP association mapped 1:1 onto a nexit/1 UDP stream.
// Datagrams carry their own destination (send) or source (recv). Uncredited:
// implementations keep at most 32 unsent datagrams and drop the oldest.
type UDPStream interface {
	Send(dst netip.AddrPort, payload []byte) error
	// Recv blocks until a datagram arrives from the exit or ctx ends.
	Recv(ctx context.Context) (src netip.AddrPort, payload []byte, err error)
	Close() error
}

// Backend is an attached exit device as seen by the front door (Mode A). The
// front door has already applied the destination Policy before calling DialTCP.
type Backend interface {
	// DialTCP opens a TCP stream to dst through the exit. On failure the error
	// wraps a *RepError with the exit's OPEN_FAIL reason.
	DialTCP(ctx context.Context, dst netip.AddrPort) (net.Conn, error)
	// OpenUDP opens one UDP association through the exit.
	OpenUDP(ctx context.Context) (UDPStream, error)
	// Peer identifies the exit; Transport is always ExitModeNative here.
	Peer() proto.ExitPeer
	// ConnectedAt is when WELCOME was sent.
	ConnectedAt() time.Time
	// Done is closed when the session ends for any reason.
	Done() <-chan struct{}
	// Close ends the session; reason is logged and sent as the WS close text.
	Close(reason string)
}

// RelayGate is what the front door consults in Mode B before relaying a SOCKS
// connection byte-for-byte into the slot's WstunnelReverseAddr.
type RelayGate interface {
	// Connected reports whether at least one upgraded wstunnel connection is
	// tracked and the reverse listener accepts (probed on the first upgrade).
	Connected() bool
	Peer() *proto.ExitPeer
	ConnectedAt() *time.Time
}

// ByteCounter is shared by the front door and the relays so status can report
// consumer->exit (Up) and exit->consumer (Down) totals per slot.
type ByteCounter interface {
	AddUp(n int64)
	AddDown(n int64)
	Totals() proto.ExitBytes
}

// Runner executes a command; the manager uses it for S94exit, S30rndis and the
// presentation cycle so tests can substitute a fake. stdout is returned trimmed.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout string, err error)
}

// DownstreamStatus is what `S94exit status <slot>` prints, one `key=0|1` per
// line: forward routing tun hev wstunnel nat. DNS is filled in by the manager
// from the forwarder's own bind state.
type DownstreamStatus = proto.ExitDownstream

// NICResolver is the slice of the presentation manager the exit needs: the live
// gadget netdev name (empty when the profile links no network function or the
// UDC is not bound yet) and its protocol. Implemented by presentation.Manager.
type NICResolver interface {
	NIC(ctx context.Context) (string, error)
	NetworkProtocol(ctx context.Context) (string, error)
	UDCBound() bool
}

// Rebinder is the presentation manager's pull-up cycle (D5). ErrTransient and
// similar refusals are reported, never fatal to an enable.
type Rebinder interface {
	Rebind(ctx context.Context) error
}

// BridgeGate answers whether the L2 bridge owns the gadget NIC (D17). Names
// which evidence tripped so the refusal message can say so.
type BridgeGate interface {
	BridgeActive() (active bool, why string)
}

// Timeouts shared across components (seconds in the spec, durations here).
const (
	HelloTimeout       = 5 * time.Second  // HELLO must follow the upgrade within this
	OpenTimeout        = 8 * time.Second  // kvm-side OPEN timer -> RepTTLExpired
	PingInterval       = 20 * time.Second // kvm WS ping cadence
	PingMisses         = 3                // consecutive misses that close a session
	UDPIdleTimeout     = 60 * time.Second // association torn down after this idle
	ProbeInterval      = 30 * time.Second
	ProbeTimeout       = 5 * time.Second
	WatchdogInterval   = 30 * time.Second
	FrontDoorHandshake = 10 * time.Second // SOCKS5 greeting+request budget from hev
)

// nexit/1 sizing (D24).
const (
	StreamWindow = 128 << 10
	ConnWindow   = 4 << 20
	MaxStreams   = 256
	MaxFrame     = 16 << 10 // payload bytes; the WS read limit is MaxFrame+64
	UDPQueue     = 32       // unsent datagrams kept per UDP stream
)
