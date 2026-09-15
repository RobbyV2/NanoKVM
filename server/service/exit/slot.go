// Package exit implements reverse-tunnel exit provisioning with downstream NIC
// bridging: a loopback SOCKS5 front door per slot whose egress is an exit
// device that dialled in over a WebSocket, a tun2socks translator behind it,
// and the routing, NAT, DHCP options and DNS that make the USB gadget NIC a
// transparent internet interface for the attached consumer.
//
// See docs/superpowers/specs/2026-09-15-exit-tunnel-design.md. Everything is
// scoped by a Slot so a second tunnel is additive, even though only slot 0
// exists today.
package exit

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
)

// Filesystem roots, variables so tests can redirect them.
var (
	// ConfigDir holds <n>.json, <n>.state.json, <n>/ and gadget.route.
	ConfigDir = "/etc/kvm/exit"
	// BinDir holds the extracted daemons, shared with the tunnel package.
	BinDir = "/etc/kvm/bin"
	// SeedDir holds the gzipped seeds shipped in the OTA package.
	SeedDir = "/kvmapp/exit"
	// InitSeedDir is where the package ships init scripts from.
	InitSeedDir = "/kvmapp/system/init.d"
	// InitDir is where rcS runs them.
	InitDir = "/etc/init.d"
	// RunDir holds pidfiles and the per-slot lock.
	RunDir = "/var/run"
	// LogDir holds the daemons' log tails (tmpfs).
	LogDir = "/tmp"
)

// Well-known names.
const (
	HevBinary      = "hev-socks5-tunnel"
	WstunnelBinary = "wstunnel"
	S94Script      = "S94exit"
	S30Script      = "S30rndis"
	// GadgetRouteMarker, relative to ConfigDir, tells S30rndis to hand out the
	// gateway and DNS options (D4). Content is the slot id that owns the NIC.
	GadgetRouteMarker = "gadget.route"

	// Base numbers every slot derives from (D9).
	baseTable     = 100
	baseRulePref  = 1000
	baseSocksPort = 10800
	baseWsPort    = 10810
	baseRevPort   = 10820
	maxSlot       = 9
)

// Slot is one exit tunnel instance. ID is the operator-visible id ("0"), N its
// numeric form.
type Slot struct {
	ID string
	N  int
}

// ErrBadSlot is returned for ids that are not a single digit 0-9.
var ErrBadSlot = errors.New("invalid slot id")

// ParseSlot accepts "0".."9".
func ParseSlot(id string) (Slot, error) {
	if len(id) != 1 || id[0] < '0' || id[0] > '9' {
		return Slot{}, ErrBadSlot
	}
	n, _ := strconv.Atoi(id)
	if n > maxSlot {
		return Slot{}, ErrBadSlot
	}
	return Slot{ID: id, N: n}, nil
}

// MustSlot is ParseSlot for literals in tests and defaults.
func MustSlot(id string) Slot {
	s, err := ParseSlot(id)
	if err != nil {
		panic(err)
	}
	return s
}

// Name is the slot's short name, used for the tun, chains, locks and logs: exit0.
func (s Slot) Name() string { return "exit" + s.ID }

// Tun is the persistent tun device S94exit owns.
func (s Slot) Tun() string { return s.Name() }

// TunAddr is the tun's /32 address (hev hardcodes the prefix length).
func (s Slot) TunAddr() string { return fmt.Sprintf("198.18.%d.1", s.N) }

// Table is the policy routing table holding the default via the tun and the fence.
func (s Slot) Table() int { return baseTable + s.N }

// RulePrefs returns the two ip rule preferences: lookup, then unreachable fence.
func (s Slot) RulePrefs() (lookup int, fence int) {
	return baseRulePref + 2*s.N, baseRulePref + 2*s.N + 1
}

// Chain and NATChain are the per-slot iptables chains.
func (s Slot) Chain() string    { return "EXIT" + s.ID }
func (s Slot) NATChain() string { return "EXIT" + s.ID + "_NAT" }

// SocksAddr is the front door the tun2socks translator dials (the contract).
func (s Slot) SocksAddr() string { return fmt.Sprintf("127.0.0.1:%d", baseSocksPort+s.N) }

// SocksPort is the numeric form of SocksAddr for config rendering.
func (s Slot) SocksPort() int { return baseSocksPort + s.N }

// WstunnelAddr is where wstunnel server listens for the proxied upgrades (Mode B).
func (s Slot) WstunnelAddr() string { return fmt.Sprintf("127.0.0.1:%d", baseWsPort+s.N) }

// WstunnelReverseAddr is the reverse-SOCKS listener the exit's wstunnel client
// asks the server to bind; the front door relays SOCKS bytes into it (Mode B).
func (s Slot) WstunnelReverseAddr() string { return fmt.Sprintf("127.0.0.1:%d", baseRevPort+s.N) }

// WstunnelReversePort is the numeric form for the restriction yaml and the client command.
func (s Slot) WstunnelReversePort() int { return baseRevPort + s.N }

// PathPrefix is the wstunnel http upgrade path prefix the client is told
// (`-P exit/<n>`) and what the proxy rewrites it to on the way in (`/exit<n>/`).
func (s Slot) ClientPathPrefix() string { return "exit/" + s.ID }
func (s Slot) ServerPathPrefix() string { return s.Name() }

// Paths.
func (s Slot) ConfigPath() string    { return filepath.Join(ConfigDir, s.ID+".json") }
func (s Slot) StatePath() string     { return filepath.Join(ConfigDir, s.ID+".state.json") }
func (s Slot) Dir() string           { return filepath.Join(ConfigDir, s.ID) }
func (s Slot) HevConfigPath() string { return filepath.Join(s.Dir(), "hev.yml") }
func (s Slot) RestrictPath() string  { return filepath.Join(s.Dir(), "wstunnel-restrict.yml") }
func (s Slot) NicPath() string       { return filepath.Join(s.Dir(), "nic") }
func (s Slot) LockPath() string      { return filepath.Join(RunDir, s.Name()+".lock") }
func (s Slot) PidPath(daemon string) string {
	return filepath.Join(RunDir, s.Name()+"-"+daemon+".pid")
}
func (s Slot) LogPath(daemon string) string {
	return filepath.Join(LogDir, s.Name()+"-"+daemon+".log")
}

// GadgetRoutePath is the marker S30rndis reads (D4).
func GadgetRoutePath() string { return filepath.Join(ConfigDir, GadgetRouteMarker) }
