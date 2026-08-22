package bridge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// One entry of "ip -j link show". Only the fields a restore needs are decoded;
// ip emits more and adds more between releases.
type Link struct {
	Index     int      `json:"ifindex"`
	Name      string   `json:"ifname"`
	Flags     []string `json:"flags,omitempty"`
	MTU       int      `json:"mtu,omitempty"`
	Master    string   `json:"master,omitempty"`
	OperState string   `json:"operstate,omitempty"`
	Address   string   `json:"address,omitempty"`
	LinkType  string   `json:"link_type,omitempty"`
}

// IFF_UP is administrative state and is set on an enslaved port too.
func (l Link) Up() bool { return l.hasFlag("UP") }

// A bridge does not report IFF_RUNNING until a port forwards, which is why STP
// is off: listening plus learning is roughly thirty seconds and udhcpc's
// -t 10 -T 1 has given up long before.
func (l Link) Running() bool { return l.hasFlag("LOWER_UP") || l.hasFlag("RUNNING") }

func (l Link) hasFlag(flag string) bool {
	for _, f := range l.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

type AddrInfo struct {
	Family    string `json:"family"`
	Local     string `json:"local"`
	PrefixLen int    `json:"prefixlen"`
	Scope     string `json:"scope,omitempty"`
	Broadcast string `json:"broadcast,omitempty"`
}

func (a AddrInfo) CIDR() string {
	return fmt.Sprintf("%s/%d", a.Local, a.PrefixLen)
}

type Addr struct {
	Index    int        `json:"ifindex"`
	Name     string     `json:"ifname"`
	AddrInfo []AddrInfo `json:"addr_info,omitempty"`
}

// A default route has Dst "default".
type Route struct {
	Dst      string `json:"dst"`
	Gateway  string `json:"gateway,omitempty"`
	Dev      string `json:"dev"`
	Protocol string `json:"protocol,omitempty"`
	PrefSrc  string `json:"prefsrc,omitempty"`
	Metric   int    `json:"metric,omitempty"`
}

// The full network state, written before the first mutation. It is the only
// thing standing between a failed apply and a device that needs physical
// access, so it is captured in one pass and fsynced before anything is touched.
type Snapshot struct {
	CapturedAt time.Time `json:"capturedAt"`

	Links  []Link  `json:"links"`
	Addrs  []Addr  `json:"addrs"`
	Routes []Route `json:"routes"`

	// Present distinguishes an absent file from an empty one: restoring an empty
	// file where there was none leaves resolv.conf shadowing nothing.
	ResolvConf        string `json:"resolvConf"`
	ResolvConfPresent bool   `json:"resolvConfPresent"`

	Gateway        string `json:"gateway"`
	GatewayPresent bool   `json:"gatewayPresent"`

	// UplinkPresent false is the stock eth0 state, and a restore removes the file
	// rather than writing "eth0" into it.
	Uplink        string `json:"uplink"`
	UplinkPresent bool   `json:"uplinkPresent"`

	// Whether /boot/eth.nodhcp existed, captured rather than re-tested at restore
	// time because a restore may run at boot from a different filesystem state.
	StaticPath bool `json:"staticPath"`
}

// The read half of every ip invocation, split from the write half so a test can
// drive capture and verification from fixtures without giving the fake any way
// to pretend a mutation succeeded.
type Inspector interface {
	Links(ctx context.Context) ([]Link, error)
	Addrs(ctx context.Context) ([]Addr, error)
	Routes(ctx context.Context) ([]Route, error)
}

// Step 1. All three ip reads happen before any file is read, so the L2 and L3
// halves come from as close to the same instant as separate processes allow.
func Capture(ctx context.Context, inspector Inspector) (*Snapshot, error) {
	links, err := inspector.Links(ctx)
	if err != nil {
		return nil, fmt.Errorf("capture links: %w", err)
	}
	addrs, err := inspector.Addrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("capture addresses: %w", err)
	}
	routes, err := inspector.Routes(ctx)
	if err != nil {
		return nil, fmt.Errorf("capture routes: %w", err)
	}

	snapshot := &Snapshot{
		CapturedAt: time.Now().UTC(),
		Links:      links,
		Addrs:      addrs,
		Routes:     routes,
		StaticPath: fileExists(noDHCPPath),
	}

	snapshot.ResolvConf, snapshot.ResolvConfPresent = readFile(resolvPath)
	snapshot.Gateway, snapshot.GatewayPresent = readFile(gatewayPath)

	if uplink, ok := readFile(uplinkPath); ok {
		snapshot.Uplink, snapshot.UplinkPresent = trimLine(uplink), true
	}

	return snapshot, nil
}

func (s *Snapshot) Link(dev string) (Link, bool) {
	for _, link := range s.Links {
		if link.Name == dev {
			return link, true
		}
	}
	return Link{}, false
}

func (s *Snapshot) Master(dev string) (string, bool) {
	link, ok := s.Link(dev)
	if !ok || link.Master == "" {
		return "", false
	}
	return link.Master, true
}

// inet6 is filtered here rather than by a grep: the latent bug at S30eth:59 is
// exactly a "grep inet" that also matches "inet6" and so reports an interface
// carrying only a link-local as addressed.
func (s *Snapshot) IPv4(dev string) []AddrInfo {
	var out []AddrInfo
	for _, addr := range s.Addrs {
		if addr.Name != dev {
			continue
		}
		for _, info := range addr.AddrInfo {
			if info.Family == "inet" && info.Local != "" {
				out = append(out, info)
			}
		}
	}
	return out
}

func (s *Snapshot) DefaultRoute() (Route, bool) {
	for _, route := range s.Routes {
		if route.Dst == "default" {
			return route, true
		}
	}
	return Route{}, false
}

// DefaultRouteVia is the captured default route through a specific device.
func (s *Snapshot) DefaultRouteVia(dev string) (Route, bool) {
	for _, route := range s.Routes {
		if route.Dst == "default" && route.Dev == dev {
			return route, true
		}
	}
	return Route{}, false
}

// UplinkName resolves the captured uplink the same way every reader does: an
// absent file means eth0.
func (s *Snapshot) UplinkName() string {
	if !s.UplinkPresent || s.Uplink == "" {
		return StockUplink
	}
	return s.Uplink
}

// Keeps the static replay in step 8 from adding a nameserver the device did not
// have, the other half of avoiding S30eth:55's non-idempotent append.
func (s *Snapshot) HasNameserver(gateway string) bool {
	for _, line := range strings.Split(s.ResolvConf, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" && fields[1] == gateway {
			return true
		}
	}
	return false
}

// Restore undoes an apply from the record on disk. It is called by the
// watchdog when the deadline expires, by a failed verification, and at boot
// when a pending marker survived a power cut.
//
// Every step is best-effort and the errors are joined rather than returned at
// the first failure: this runs when the device is already in an unknown state,
// and abandoning the remaining steps because one of them failed is how a
// recoverable device becomes one that needs physical access. The order is
// fixed, though, and two orderings are load-bearing:
//
//   - the files come first, because S30eth reads l2-uplink to decide which
//     device to address, so restoring the file after re-running the script
//     would address the wrong one;
//   - the ports are released before br0 is deleted, because deleting a bridge
//     that still has ports is the one operation here with no defined recovery.
func (m *Manager) Restore(ctx context.Context, snapshot *Snapshot) error {
	uplink := snapshot.UplinkName()
	var errs []error

	// 1. Files. l2-uplink first: every later step reads through it.
	if snapshot.UplinkPresent {
		errs = append(errs, wrap("restore l2-uplink", WriteUplink(snapshot.Uplink)))
	} else {
		errs = append(errs, wrap("remove l2-uplink", RemoveUplink()))
	}
	errs = append(errs, wrap("restore gateway",
		restoreFile(gatewayPath, snapshot.Gateway, snapshot.GatewayPresent, 0o644)))
	errs = append(errs, wrap("restore resolv.conf",
		restoreFile(resolvPath, snapshot.ResolvConf, snapshot.ResolvConfPresent, 0o644)))

	// 2. L2. The live state is read first so the restore issues only the
	// commands that are actually needed. Blindly releasing and deleting would
	// work too, but every no-op would fail with "Cannot find device", and a
	// rollback that reports failure because it had nothing to undo is
	// indistinguishable from one that genuinely could not put the device back.
	// That distinction is the difference between an operator reading the record
	// and reaching for the serial console or not.
	live, liveErr := m.ip.Links(ctx)
	if liveErr != nil {
		errs = append(errs, wrap("read live links", liveErr))
	}

	exists := make(map[string]bool, len(live))
	master := make(map[string]string, len(live))
	for _, link := range live {
		exists[link.Name] = true
		master[link.Name] = link.Master
	}

	// unknown is true when the live read failed, in which case the commands are
	// issued anyway and their errors are not escalated: acting blindly is
	// better than not acting, but it cannot be reported as a clean restore.
	unknown := liveErr != nil

	// Release every device the capture shows had no master, so a port enslaved
	// during the apply goes back to being a plain interface.
	for _, dev := range []string{GadgetName, StockUplink} {
		if _, wasEnslaved := snapshot.Master(dev); wasEnslaved {
			continue
		}
		if _, existed := snapshot.Link(dev); !existed {
			continue
		}
		if unknown {
			_ = m.ip.SetNoMaster(ctx, dev)
			_ = m.ip.SetUp(ctx, dev)
			continue
		}
		if !exists[dev] {
			continue
		}
		if master[dev] != "" {
			errs = append(errs, wrap("release "+dev, m.ip.SetNoMaster(ctx, dev)))
		}
		errs = append(errs, wrap("up "+dev, m.ip.SetUp(ctx, dev)))
	}

	// 3. The bridge itself, only when the capture shows it did not exist and
	// the device is actually there now. A restore of a failed disable leaves
	// br0 in place on purpose: verification runs against eth0 and the teardown
	// is step 13, after the disarm.
	if _, existed := snapshot.Link(BridgeName); !existed {
		switch {
		case unknown:
			_ = m.ip.SetDown(ctx, BridgeName)
			_ = m.ip.DeleteLink(ctx, BridgeName)
		case exists[BridgeName]:
			errs = append(errs, wrap("down "+BridgeName, m.ip.SetDown(ctx, BridgeName)))
			errs = append(errs, wrap("delete "+BridgeName, m.ip.DeleteLink(ctx, BridgeName)))
		}
	}

	// 4. Re-address. S30eth start flushes the uplink and re-runs whichever
	// branch the device is configured for.
	errs = append(errs, wrap("S30eth start", m.scripts.StartEth(ctx)))

	// 5. The DHCP branch at S30eth:68 has no fallback of any kind: udhcpc is
	// backgrounded, and if it never takes a lease the device sits with no IPv4
	// and nothing that retries. Replaying the captured address closes that hole
	// on the one path where it matters most, a rollback, and it is skipped
	// whenever the script already produced an address.
	if !m.hasIPv4(ctx, uplink) {
		errs = append(errs, wrap("replay address", m.replay(ctx, uplink, snapshot)))
	}

	// 6. The five firewall rules against whichever device is the uplink again.
	errs = append(errs, wrap("firewall", m.firewall.Install(ctx, uplink)))

	return errors.Join(errs...)
}

// replay puts the captured IPv4 addresses and default route back on a device in
// a single ip -batch, so a partially applied address list is not a state the
// device can end up in.
func (m *Manager) replay(ctx context.Context, dev string, snapshot *Snapshot) error {
	addrs := snapshot.IPv4(dev)
	if len(addrs) == 0 {
		// Nothing captured for this device: an apply that had already moved the
		// address off it, or a device that was never addressed. Either way there
		// is nothing to replay and reporting an error here would mask the real
		// failure in the joined result.
		return nil
	}

	route, hasRoute := snapshot.DefaultRouteVia(dev)
	if !hasRoute {
		route, hasRoute = snapshot.DefaultRoute()
	}

	batch := make([]string, 0, len(addrs)+1)
	for _, addr := range addrs {
		batch = append(batch, fmt.Sprintf("addr add %s brd + dev %s", addr.CIDR(), dev))
	}
	if hasRoute && isIPv4(route.Gateway) {
		batch = append(batch, fmt.Sprintf("route add default via %s dev %s", route.Gateway, dev))
	}

	return m.ip.Batch(ctx, batch)
}

func wrap(what string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", what, err)
}

func readFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func trimLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func isIPv4(s string) bool {
	ip := net.ParseIP(strings.TrimSpace(s))
	return ip != nil && ip.To4() != nil
}
