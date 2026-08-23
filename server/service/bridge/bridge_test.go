package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

// ---------------------------------------------------------------------------
// harness
// ---------------------------------------------------------------------------

// fakeNet is a Commander over an in-memory network. It records every argv it is
// handed and interprets enough of ip, ping, iptables and S30eth for both
// transactions to run end to end with no device.
//
// It records rather than mocks: the step order of a transaction whose steps are
// side effects on a kernel is only observable as the sequence of commands it
// issues, so the trace is the assertion.
type fakeNet struct {
	mu sync.Mutex

	calls []string

	links  map[string]*Link
	addrs  map[string][]AddrInfo
	routes []Route

	// dhcpAddr is what S30eth start hands the uplink, so the DHCP branch has an
	// outcome a test can assert against.
	dhcpAddr    AddrInfo
	dhcpGateway string
	dhcpFails   bool

	pingOK bool

	// onCall runs before every command is interpreted. It is how a test
	// observes on-disk state at the instant of a particular step, which is the
	// only way to assert that the dead-man was armed before the first mutation
	// and disarmed before step 13.
	onCall func(line string)

	// failures are substring matches against the recorded command line, so a
	// test can make exactly one step fail without stubbing the rest.
	failures []failure
}

type failure struct {
	contains string
	err      error
}

func newFakeNet() *fakeNet {
	return &fakeNet{
		links: map[string]*Link{
			"lo":   {Index: 1, Name: "lo", Flags: []string{"UP"}, Address: "00:00:00:00:00:00"},
			"eth0": {Index: 2, Name: "eth0", Flags: []string{"UP", "LOWER_UP"}, Address: testMAC},
			"usb0": {Index: 3, Name: "usb0", Flags: []string{"UP"}, Address: "48:da:35:6e:11:22"},
			"wlan0": {
				Index: 4, Name: "wlan0", Flags: []string{"UP", "LOWER_UP"},
				Address: "aa:bb:cc:dd:ee:ff",
			},
		},
		addrs: map[string][]AddrInfo{
			"lo":   {{Family: "inet", Local: "127.0.0.1", PrefixLen: 8, Scope: "host"}},
			"eth0": {{Family: "inet", Local: "192.168.1.50", PrefixLen: 24, Scope: "global"}},
		},
		routes: []Route{
			{Dst: "default", Gateway: "192.168.1.1", Dev: "eth0", Protocol: "static"},
			{Dst: "192.168.1.0/24", Dev: "eth0", Protocol: "kernel", PrefSrc: "192.168.1.50"},
		},
		dhcpAddr:    AddrInfo{Family: "inet", Local: "192.168.1.50", PrefixLen: 24, Scope: "global"},
		dhcpGateway: "192.168.1.1",
		pingOK:      true,
	}
}

const testMAC = "3e:7c:1a:2b:3c:4d"

func (f *fakeNet) failOn(contains string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failures = append(f.failures, failure{contains: contains, err: err})
}

func (f *fakeNet) trace() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func (f *fakeNet) Run(_ context.Context, argv []string, stdin []byte) ([]byte, error) {
	line := strings.Join(argv, " ")
	if len(stdin) > 0 {
		line += " << " + strings.ReplaceAll(strings.TrimSpace(string(stdin)), "\n", " ; ")
	}

	f.mu.Lock()
	f.calls = append(f.calls, line)
	onCall := f.onCall
	for _, fail := range f.failures {
		if strings.Contains(line, fail.contains) {
			f.mu.Unlock()
			return nil, fail.err
		}
	}
	f.mu.Unlock()

	if onCall != nil {
		onCall(line)
	}

	if !allowedBinaries[argv[0]] {
		return nil, fmt.Errorf("fakeNet: %q is not allowed", argv[0])
	}

	switch argv[0] {
	case IPBinary:
		return f.ip(argv[1:], stdin)
	case PingBinary:
		if f.pingOK {
			return nil, nil
		}
		return nil, errors.New("no answer")
	case IptablesBinary:
		// -C reports "no such rule" so Install always falls through to -A,
		// which is the state a boot with a different uplink leaves behind.
		if len(argv) > 1 && argv[1] == "-C" {
			return nil, errors.New("no such rule")
		}
		return nil, nil
	case S30ethScript:
		return nil, f.startEth()
	}
	return nil, fmt.Errorf("fakeNet: unhandled %q", line)
}

// startEth models the DHCP branch: it flushes the uplink it reads from
// l2-uplink and hands it a lease.
func (f *fakeNet) startEth() error {
	uplink := ReadUplink()

	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.addrs, uplink)
	f.dropRoutes(uplink)

	if f.dhcpFails {
		return nil // backgrounded udhcpc that never takes a lease: not an error
	}

	f.addrs[uplink] = []AddrInfo{f.dhcpAddr}
	f.routes = append(f.routes, Route{Dst: "default", Gateway: f.dhcpGateway, Dev: uplink})
	return nil
}

func (f *fakeNet) dropRoutes(dev string) {
	kept := f.routes[:0]
	for _, route := range f.routes {
		if route.Dev != dev {
			kept = append(kept, route)
		}
	}
	f.routes = kept
}

func (f *fakeNet) ip(args []string, stdin []byte) ([]byte, error) {
	joined := strings.Join(args, " ")

	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case joined == "-j link show":
		return f.encodeLinks()
	case joined == "-j addr show":
		return f.encodeAddrs("", false)
	case joined == "-j route show":
		return json.Marshal(f.routes)
	case strings.HasPrefix(joined, "-4 -j addr show dev "):
		return f.encodeAddrs(args[len(args)-1], true)
	case strings.HasPrefix(joined, "link add name "):
		name := args[3]
		if _, ok := f.links[name]; ok {
			return nil, errors.New("File exists")
		}
		f.links[name] = &Link{Index: 90 + len(f.links), Name: name, Address: "00:00:00:00:00:00"}
		return nil, nil
	case strings.HasPrefix(joined, "link delete "):
		delete(f.links, args[2])
		return nil, nil
	case strings.HasPrefix(joined, "link set dev "):
		return f.linkSet(args[4:], args[3])
	case strings.HasPrefix(joined, "addr flush dev "):
		delete(f.addrs, args[3])
		return nil, nil
	case strings.HasPrefix(joined, "route del default dev "):
		dev := args[4]
		before := len(f.routes)
		kept := f.routes[:0]
		for _, route := range f.routes {
			if !(route.Dst == "default" && route.Dev == dev) {
				kept = append(kept, route)
			}
		}
		f.routes = kept
		if before == len(f.routes) {
			return nil, errors.New("RTNETLINK answers: No such process")
		}
		return nil, nil
	case joined == "-batch -":
		return nil, f.batch(string(stdin))
	}
	return nil, fmt.Errorf("fakeNet: unhandled ip %q", joined)
}

func (f *fakeNet) linkSet(rest []string, dev string) ([]byte, error) {
	link, ok := f.links[dev]
	if !ok {
		return nil, fmt.Errorf("Cannot find device %q", dev)
	}

	switch rest[0] {
	case "up":
		link.Flags = []string{"UP", "LOWER_UP"}
	case "down":
		link.Flags = nil
	case "address":
		link.Address = rest[1]
	case "master":
		link.Master = rest[1]
	case "nomaster":
		link.Master = ""
	default:
		return nil, fmt.Errorf("fakeNet: unhandled link set %v", rest)
	}
	return nil, nil
}

func (f *fakeNet) batch(script string) error {
	for _, line := range strings.Split(strings.TrimSpace(script), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch {
		case fields[0] == "addr" && fields[1] == "add":
			cidr, dev := fields[2], fields[len(fields)-1]
			local, prefix, ok := strings.Cut(cidr, "/")
			if !ok {
				return fmt.Errorf("fakeNet: bad batch address %q", cidr)
			}
			var length int
			if _, err := fmt.Sscanf(prefix, "%d", &length); err != nil {
				return err
			}
			f.addrs[dev] = append(f.addrs[dev],
				AddrInfo{Family: "inet", Local: local, PrefixLen: length, Scope: "global"})
		case fields[0] == "route" && fields[1] == "add":
			f.routes = append(f.routes,
				Route{Dst: "default", Gateway: fields[4], Dev: fields[len(fields)-1]})
		default:
			return fmt.Errorf("fakeNet: unhandled batch line %q", line)
		}
	}
	return nil
}

func (f *fakeNet) encodeLinks() ([]byte, error) {
	names := make([]string, 0, len(f.links))
	for name := range f.links {
		names = append(names, name)
	}
	sort.Strings(names)

	links := make([]Link, 0, len(names))
	for _, name := range names {
		links = append(links, *f.links[name])
	}
	return json.Marshal(links)
}

func (f *fakeNet) encodeAddrs(dev string, v4Only bool) ([]byte, error) {
	names := make([]string, 0, len(f.addrs))
	for name := range f.addrs {
		if dev == "" || name == dev {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	out := make([]Addr, 0, len(names))
	for _, name := range names {
		infos := f.addrs[name]
		if v4Only {
			var kept []AddrInfo
			for _, info := range infos {
				if info.Family == "inet" {
					kept = append(kept, info)
				}
			}
			infos = kept
		}
		out = append(out, Addr{Name: name, AddrInfo: infos})
	}
	return json.Marshal(out)
}

// fakeLiveness drives the third gate.
type fakeLiveness struct {
	observed     bool
	selfConnects bool
	calls        int
}

func (l *fakeLiveness) Observed(string, time.Time) bool { return l.observed }

func (l *fakeLiveness) SelfConnect(context.Context, string) error {
	l.calls++
	if l.selfConnects {
		return nil
	}
	return errors.New("connection refused")
}

type fakeGadget struct {
	nic      string
	protocol string
	err      error

	// rebound is the callback the bridge registers, held so a test can fire it
	// the way a presentation apply does.
	rebound func(context.Context)
}

func (g *fakeGadget) NIC(context.Context) (string, error) { return g.nic, g.err }

func (g *fakeGadget) NetworkProtocol(context.Context) (string, error) { return g.protocol, g.err }

func (g *fakeGadget) OnRebind(fn func(context.Context)) { g.rebound = fn }

type harness struct {
	t     *testing.T
	net   *fakeNet
	live  *fakeLiveness
	mgr   *Manager
	store *Store
	root  string
	clock time.Time
}

// newHarness points every path at a temp directory and builds a Manager whose
// only external contact is the fake.
func newHarness(t *testing.T, gadget ...Gadget) *harness {
	t.Helper()

	root := t.TempDir()
	swap(t, &stateDir, filepath.Join(root, "etc/kvm/presentation/network"))
	swap(t, &uplinkPath, filepath.Join(root, "etc/kvm/network/l2-uplink"))
	swap(t, &gatewayPath, filepath.Join(root, "etc/kvm/gateway"))
	swap(t, &resolvPath, filepath.Join(root, "etc/resolv.conf"))
	swap(t, &udhcpcPidPath, filepath.Join(root, "run/udhcpc.eth0.pid"))
	swap(t, &noDHCPPath, filepath.Join(root, "boot/eth.nodhcp"))
	swap(t, &sysClassNet, filepath.Join(root, "sys/class/net"))

	writeFile(t, filepath.Join(root, "sys/class/net/eth0/address"), testMAC+"\n")
	writeFile(t, filepath.Join(root, "etc/resolv.conf"), "nameserver 192.168.1.1\n")

	net := newFakeNet()
	live := &fakeLiveness{observed: true}
	store := NewStore()

	h := &harness{
		t: t, net: net, live: live, store: store, root: root,
		clock: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	}
	config := Config{
		Commander: net,
		Liveness:  live,
		Store:     store,
		Window:    DefaultWindow,
		Now:       func() time.Time { return h.clock },
	}
	if len(gadget) == 1 {
		config.Gadget = gadget[0]
	}
	h.mgr = New(config)
	return h
}

func swap[T any](t *testing.T, target *T, value T) {
	t.Helper()
	old := *target
	*target = value
	t.Cleanup(func() { *target = old })
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// indexOf is the step-order primitive: it returns where a command appears in
// the trace, so a test asserts relative order rather than an exact transcript
// that any harmless extra read would break.
func indexOf(t *testing.T, trace []string, substr string) int {
	t.Helper()
	for i, line := range trace {
		if strings.Contains(line, substr) {
			return i
		}
	}
	t.Fatalf("command containing %q never ran.\ntrace:\n%s", substr, strings.Join(trace, "\n"))
	return -1
}

func notInTrace(t *testing.T, trace []string, substr string) {
	t.Helper()
	for _, line := range trace {
		if strings.Contains(line, substr) {
			t.Fatalf("command containing %q ran and must not have: %q", substr, line)
		}
	}
}

// ---------------------------------------------------------------------------
// wlan0
// ---------------------------------------------------------------------------

// TestWlan0IsNeverEnslaved covers every layer that could reach an enslavement.
// wlan0 is the out-of-band recovery path: enslaving it puts the way back into
// the device inside the thing being recovered from, so it is refused by the
// data (the closed enslavable set), by the validator, and by the ip wrapper,
// rather than by a convention a later caller has to remember.
func TestWlan0IsNeverEnslaved(t *testing.T) {
	t.Run("not in the enslavable set", func(t *testing.T) {
		if enslavable[RecoveryName] {
			t.Fatal("wlan0 is in the enslavable set")
		}
		if !enslavable[StockUplink] || !enslavable[GadgetName] {
			t.Fatal("the enslavable set must contain exactly eth0 and usb0")
		}
		if len(enslavable) != 2 {
			t.Fatalf("enslavable set has %d entries, want exactly eth0 and usb0", len(enslavable))
		}
	})

	t.Run("checkEnslavable reports it distinctly", func(t *testing.T) {
		err := checkEnslavable(RecoveryName)
		if !errors.Is(err, ErrRecoveryInterface) {
			t.Fatalf("checkEnslavable(wlan0) = %v, want ErrRecoveryInterface", err)
		}
	})

	t.Run("SetMaster refuses it and issues no command", func(t *testing.T) {
		net := newFakeNet()
		ip := NewIPTool(net)

		err := ip.SetMaster(context.Background(), RecoveryName, BridgeName)
		if !errors.Is(err, ErrRecoveryInterface) {
			t.Fatalf("SetMaster(wlan0) = %v, want ErrRecoveryInterface", err)
		}
		if len(net.trace()) != 0 {
			t.Fatalf("a refused enslavement still ran %v", net.trace())
		}
	})

	t.Run("every other name is refused too", func(t *testing.T) {
		net := newFakeNet()
		ip := NewIPTool(net)

		for _, name := range []string{"br0", "lo", "eth1", "wlan1", "", "eth0 wlan0"} {
			if err := ip.SetMaster(context.Background(), name, BridgeName); err == nil {
				t.Fatalf("SetMaster(%q) was allowed", name)
			}
		}
	})

	t.Run("step 13 refuses a profile that names it", func(t *testing.T) {
		h := newHarness(t)
		h.mgr.gadget = &fakeGadget{nic: RecoveryName}

		err := h.mgr.enslaveGadget(context.Background())
		if !errors.Is(err, ErrRecoveryInterface) {
			t.Fatalf("enslaveGadget(wlan0) = %v, want ErrRecoveryInterface", err)
		}
		notInTrace(t, h.net.trace(), "master")
	})

	t.Run("a full enable never touches it", func(t *testing.T) {
		h := newHarness(t)

		if _, err := h.mgr.Enable(context.Background()); err != nil {
			t.Fatalf("Enable: %v", err)
		}
		notInTrace(t, h.net.trace(), RecoveryName)

		if master := h.net.links[RecoveryName].Master; master != "" {
			t.Fatalf("wlan0 ended up enslaved to %q", master)
		}
	})
}

// ---------------------------------------------------------------------------
// the command boundary
// ---------------------------------------------------------------------------

func TestCommanderAllowlist(t *testing.T) {
	cmd := NewCommander()

	for _, argv := range [][]string{
		{},
		{"sh", "-c", "true"},
		{"/bin/sh"},
		{"brctl", "addbr", "br0"},
		{"ip; rm -rf /"},
	} {
		if _, err := cmd.Run(context.Background(), argv, nil); err == nil {
			t.Fatalf("Run(%v) was allowed", argv)
		}
	}
}

func TestBridgeCreationHasSTPOff(t *testing.T) {
	net := newFakeNet()
	ip := NewIPTool(net)

	if err := ip.AddBridge(context.Background(), BridgeName); err != nil {
		t.Fatalf("AddBridge: %v", err)
	}

	got := net.trace()[0]
	want := "ip link add name br0 type bridge stp_state 0 forward_delay 0"
	if got != want {
		t.Fatalf("AddBridge ran %q, want %q", got, want)
	}
}

func TestBridgeCreationRefusesAnyOtherName(t *testing.T) {
	net := newFakeNet()
	ip := NewIPTool(net)

	if err := ip.AddBridge(context.Background(), "br1"); err == nil {
		t.Fatal("AddBridge(br1) was allowed")
	}
	if err := ip.DeleteLink(context.Background(), "eth0"); err == nil {
		t.Fatal("DeleteLink(eth0) was allowed")
	}
}

func TestSetMACValidatesTheAddress(t *testing.T) {
	net := newFakeNet()
	ip := NewIPTool(net)

	if err := ip.SetMAC(context.Background(), BridgeName, "not-a-mac"); err == nil {
		t.Fatal("SetMAC accepted a non-MAC")
	}
	if len(net.trace()) != 0 {
		t.Fatalf("a rejected MAC still ran %v", net.trace())
	}
}

func TestBatchRefusesInjectedLines(t *testing.T) {
	net := newFakeNet()
	ip := NewIPTool(net)

	for _, line := range []string{
		"addr add 1.2.3.4/24 dev eth0; reboot",
		"addr add 1.2.3.4/24 dev eth0\nlink delete br0",
		"addr add $(whoami) dev eth0",
		"addr add `id` dev eth0",
	} {
		if err := ip.Batch(context.Background(), []string{line}); err == nil {
			t.Fatalf("Batch accepted %q", line)
		}
	}
	if len(net.trace()) != 0 {
		t.Fatalf("a rejected batch still ran %v", net.trace())
	}
}

func TestAddrsV4IgnoresIPv6(t *testing.T) {
	// The bug this guards is S30eth:59's "ip a show | grep inet", which matches
	// inet6 and so reports a device carrying only a link-local as addressed.
	net := newFakeNet()
	net.addrs["br0"] = []AddrInfo{
		{Family: "inet6", Local: "fe80::1", PrefixLen: 64, Scope: "link"},
	}

	addrs, err := NewIPTool(net).AddrsV4(context.Background(), "br0")
	if err != nil {
		t.Fatalf("AddrsV4: %v", err)
	}
	if len(addrs) != 0 {
		t.Fatalf("AddrsV4 returned %v for a device with only a link-local", addrs)
	}
}

func TestFirewallRulesMatchTheScript(t *testing.T) {
	rules := firewallRules("br0")
	if len(rules) != 5 {
		t.Fatalf("got %d rules, want the 5 at S95nanokvm:92-105", len(rules))
	}

	want := []string{
		"INPUT -i br0 -p tcp --dport 80 -m state --state NEW,ESTABLISHED -j ACCEPT",
		"OUTPUT -o br0 -p tcp --sport 80 -m state --state ESTABLISHED -j ACCEPT",
		"INPUT -i br0 -p tcp --sport 22 -m state --state NEW,ESTABLISHED -j ACCEPT",
		"OUTPUT -o br0 -p tcp --dport 22 -m state --state ESTABLISHED -j ACCEPT",
		"OUTPUT -o br0 -p tcp --sport 8000 -m state --state ESTABLISHED -j DROP",
	}
	for i, rule := range rules {
		if got := strings.Join(rule, " "); got != want[i] {
			t.Errorf("rule %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestFirewallInstallFallsBackToAppend(t *testing.T) {
	net := newFakeNet()

	if err := NewFirewall(net).Install(context.Background(), "br0"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	trace := net.trace()
	if len(trace) != 10 {
		t.Fatalf("Install ran %d commands, want 5 checks and 5 appends:\n%s",
			len(trace), strings.Join(trace, "\n"))
	}
	for i := 0; i < len(trace); i += 2 {
		if !strings.HasPrefix(trace[i], "iptables -C ") {
			t.Errorf("trace[%d] = %q, want a -C check", i, trace[i])
		}
		if !strings.HasPrefix(trace[i+1], "iptables -A ") {
			t.Errorf("trace[%d] = %q, want an -A append", i+1, trace[i+1])
		}
	}
}

func TestPermanentMACIsReadFromSysfs(t *testing.T) {
	h := newHarness(t)

	mac, err := permanentMAC(StockUplink)
	if err != nil {
		t.Fatalf("permanentMAC: %v", err)
	}
	if mac != testMAC {
		t.Fatalf("permanentMAC = %q, want %q", mac, testMAC)
	}

	writeFile(t, filepath.Join(h.root, "sys/class/net/eth0/address"), "garbage\n")
	if _, err := permanentMAC(StockUplink); err == nil {
		t.Fatal("permanentMAC accepted a MAC that does not parse")
	}
}

// ---------------------------------------------------------------------------
// the dead-man itself
// ---------------------------------------------------------------------------

func TestPendingDeadlineArithmetic(t *testing.T) {
	armed := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	pending := Pending{ArmedAt: armed, Deadline: armed.Add(DefaultWindow)}

	if DefaultWindow != 60*time.Second {
		t.Fatalf("window is %s, the design says sixty seconds", DefaultWindow)
	}
	if pending.Expired(armed) {
		t.Fatal("a freshly armed marker reports expired")
	}
	if pending.Expired(armed.Add(59 * time.Second)) {
		t.Fatal("expired one second early")
	}
	if !pending.Expired(armed.Add(DefaultWindow)) {
		t.Fatal("not expired at the deadline itself")
	}
	if got := pending.Remaining(armed.Add(45 * time.Second)); got != 15*time.Second {
		t.Fatalf("Remaining = %s, want 15s", got)
	}
	if got := pending.Remaining(armed.Add(2 * DefaultWindow)); got != 0 {
		t.Fatalf("Remaining past the deadline = %s, want 0", got)
	}
}

// TestDeadmanFiresAndRestores drives the watchdog off a manual timer, so the
// deadline path is exercised without waiting for it.
func TestDeadmanFiresAndRestores(t *testing.T) {
	h := newHarness(t)

	fire := make(chan time.Time, 1)
	swap(t, &newTimer, func(time.Duration) (<-chan time.Time, func()) {
		return fire, func() {}
	})

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	path, err := h.store.WriteSnapshot(snapshot)
	if err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	dm, err := h.mgr.arm(operationEnable, path)
	if err != nil {
		t.Fatalf("arm: %v", err)
	}

	// A mutation the watchdog is expected to undo.
	if err := WriteUplink(BridgeName); err != nil {
		t.Fatalf("WriteUplink: %v", err)
	}

	fire <- time.Now()

	// Wait for the watchdog goroutine to finish before disarming. A disarm that
	// races the fire is a genuine tie, and the tie goes to the disarm by design:
	// a transaction only reaches its disarm once all three gates have passed, so
	// restoring at that instant would undo a state that was just verified good.
	<-dm.done

	// disarm reports that it did not win.
	if dm.disarm() {
		t.Fatal("disarm claimed the outcome after the deadline fired")
	}
	if !dm.Expired() {
		t.Fatal("deadman does not report having expired")
	}

	if got := ReadUplink(); got != StockUplink {
		t.Fatalf("uplink is %q after the watchdog fired, want the file removed", got)
	}
	if pending, _ := h.store.Pending(); pending != nil {
		t.Fatal("the marker survived the watchdog restore")
	}

	lkg, err := h.store.LastKnownGood()
	if err != nil || lkg == nil {
		t.Fatalf("LastKnownGood = %v, %v", lkg, err)
	}
	if lkg.State != proto.BridgeRolledBack {
		t.Fatalf("recorded state %q, want %q", lkg.State, proto.BridgeRolledBack)
	}
	if lkg.Enabled {
		t.Fatal("a rolled-back enable recorded the bridge as enabled")
	}
}

// TestDeadmanOnlyOnePathRestores is the race the compare-and-swap exists for: a
// verification failing at the same instant the deadline expires must not run
// two concurrent restores over the same interfaces.
func TestDeadmanOnlyOnePathRestores(t *testing.T) {
	dm := &deadman{stop: make(chan struct{}), done: make(chan struct{})}
	close(dm.done)

	var wins int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if dm.take() {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if wins != 1 {
		t.Fatalf("%d callers claimed the restore, want exactly 1", wins)
	}
}

// ---------------------------------------------------------------------------
// the boot-time check
// ---------------------------------------------------------------------------

// TestRecoverPendingAtBoot is the power-cut case. A marker that survived a
// reboot means the process that armed it never reached its disarm, so the boot
// check restores unconditionally rather than consulting a deadline measured
// against a clock the device did not keep across the cut.
func TestRecoverPendingAtBoot(t *testing.T) {
	tests := []struct {
		name     string
		deadline time.Duration
	}{
		{"deadline long expired", -10 * time.Minute},
		{"deadline still in the future", +10 * time.Minute},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)

			// The pre-apply state: stock eth0, no bridge, no uplink file.
			snapshot, err := Capture(context.Background(), h.mgr.ip)
			if err != nil {
				t.Fatalf("Capture: %v", err)
			}
			path, err := h.store.WriteSnapshot(snapshot)
			if err != nil {
				t.Fatalf("WriteSnapshot: %v", err)
			}

			armed := h.clock
			if err := h.store.Arm(Pending{
				Operation:    operationEnable,
				SnapshotPath: path,
				ArmedAt:      armed,
				Deadline:     armed.Add(tc.deadline),
			}); err != nil {
				t.Fatalf("Arm: %v", err)
			}

			// Now simulate the half-applied device the power cut left behind.
			if err := WriteUplink(BridgeName); err != nil {
				t.Fatalf("WriteUplink: %v", err)
			}
			h.net.links[BridgeName] = &Link{Index: 90, Name: BridgeName, Address: testMAC}
			h.net.links[StockUplink].Master = BridgeName

			recovered, err := h.mgr.RecoverPending(context.Background())
			if err != nil {
				t.Fatalf("RecoverPending: %v", err)
			}
			if !recovered {
				t.Fatal("RecoverPending found no marker")
			}

			if got := ReadUplink(); got != StockUplink {
				t.Fatalf("uplink is %q after recovery, want eth0 through the absent-file fallback", got)
			}
			if _, err := os.Stat(uplinkPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("the l2-uplink file still exists after recovery")
			}
			if master := h.net.links[StockUplink].Master; master != "" {
				t.Fatalf("eth0 is still enslaved to %q after recovery", master)
			}
			if _, exists := h.net.links[BridgeName]; exists {
				t.Fatal("br0 still exists after recovery from a snapshot that predates it")
			}
			if pending, _ := h.store.Pending(); pending != nil {
				t.Fatal("the marker survived recovery")
			}
		})
	}
}

func TestRecoverPendingIsANoOpWithNoMarker(t *testing.T) {
	h := newHarness(t)

	recovered, err := h.mgr.RecoverPending(context.Background())
	if err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}
	if recovered {
		t.Fatal("RecoverPending restored with no marker armed")
	}
	if len(h.net.trace()) != 0 {
		t.Fatalf("RecoverPending ran %v with no marker armed", h.net.trace())
	}
}

// A marker naming a snapshot that is gone must be an error rather than a silent
// success: reporting "nothing to do" would leave a half-applied device.
func TestRecoverPendingWithNoSnapshot(t *testing.T) {
	h := newHarness(t)

	if err := h.store.Arm(Pending{
		Operation:    operationEnable,
		SnapshotPath: filepath.Join(h.root, "gone.json"),
		ArmedAt:      h.clock,
		Deadline:     h.clock.Add(DefaultWindow),
	}); err != nil {
		t.Fatalf("Arm: %v", err)
	}

	recovered, err := h.mgr.RecoverPending(context.Background())
	if !recovered {
		t.Fatal("RecoverPending ignored a marker whose snapshot is missing")
	}
	if !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("RecoverPending = %v, want ErrNoSnapshot", err)
	}
}

// bootRestore is what S29bridge does to the store: it restored the device
// itself, because it has to run before S30eth addresses anything, and moves the
// armed marker aside rather than deleting it.
func bootRestore(t *testing.T, h *harness) {
	t.Helper()

	if err := os.Remove(uplinkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove uplink: %v", err)
	}
	if err := os.Remove(h.store.lastKnownGoodPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove last-known-good: %v", err)
	}
	if err := os.Rename(h.store.pendingPath(), h.store.recoveredPath()); err != nil {
		t.Fatalf("move the marker aside: %v", err)
	}
}

// The record the shell leaves has to survive to the server, which starts sixty
// scripts later. Deleting it loses the only account of why the device is back
// on eth0, and a GET after a power cut mid-apply reports a bare disabled that
// an operator cannot tell from a bridge that was never enabled.
func TestTheBootRestoreLeavesAnOutcomeTheServerReports(t *testing.T) {
	h := newHarness(t)

	armed := Pending{
		Operation:    operationEnable,
		SnapshotPath: h.store.SnapshotPath(),
		ArmedAt:      h.clock,
		Deadline:     h.clock.Add(DefaultWindow),
	}
	if err := h.store.Arm(armed); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	if err := WriteUplink(BridgeName); err != nil {
		t.Fatalf("WriteUplink: %v", err)
	}

	bootRestore(t, h)

	// The moved marker is the armed one verbatim, which is what lets the shell
	// leave a record without authoring a file.
	recovered, err := h.store.Recovered()
	if err != nil || recovered == nil {
		t.Fatalf("Recovered = %v, %v, want the marker S29bridge moved aside", recovered, err)
	}
	if recovered.Operation != armed.Operation || !recovered.ArmedAt.Equal(armed.ArmedAt) {
		t.Fatalf("recovered marker = %+v, want %+v", *recovered, armed)
	}

	adopted, err := h.mgr.RecoverPending(context.Background())
	if err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}
	if !adopted {
		t.Fatal("the server ignored the record S29bridge left")
	}

	// The shell already put the interfaces back; the server touches none.
	if len(h.net.trace()) != 0 {
		t.Fatalf("adopting the record ran %v", h.net.trace())
	}

	status, err := h.mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.LastApply == nil {
		t.Fatal("GET reports no lastApply after a power cut mid-apply")
	}
	if status.State != proto.BridgeRolledBack || status.LastApply.Enabled {
		t.Fatalf("GET reports %+v, want a rolled-back disable", status.LastApply)
	}
	if !strings.Contains(status.LastApply.Message, operationEnable) {
		t.Errorf("message %q does not name the interrupted operation", status.LastApply.Message)
	}
	if status.Uplink != StockUplink {
		t.Fatalf("uplink = %q, want eth0", status.Uplink)
	}
}

// The second boot after the cut. The marker is gone, so S29bridge restores
// nothing, and the outcome the server adopted says disabled, so the script's
// enabled check does not re-create br0 either.
func TestASecondBootNeitherRestoresNorAdoptsAgain(t *testing.T) {
	h := newHarness(t)

	if err := h.store.Arm(Pending{
		Operation: operationDisable, SnapshotPath: h.store.SnapshotPath(),
		ArmedAt: h.clock, Deadline: h.clock.Add(DefaultWindow),
	}); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	bootRestore(t, h)

	if _, err := h.mgr.RecoverPending(context.Background()); err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}

	first, err := h.store.LastKnownGood()
	if err != nil || first == nil {
		t.Fatalf("LastKnownGood = %v, %v", first, err)
	}
	if first.Enabled {
		t.Fatal("the adopted outcome says enabled, so S29bridge re-creates br0 on the next boot")
	}

	// Nothing is left for the next boot to act on.
	if pending, _ := h.store.Pending(); pending != nil {
		t.Fatal("an armed marker survived the recovery")
	}
	if recovered, _ := h.store.Recovered(); recovered != nil {
		t.Fatal("the record survived being adopted, so every later boot re-reports it")
	}

	h.clock = h.clock.Add(time.Hour)
	again, err := h.mgr.RecoverPending(context.Background())
	if err != nil {
		t.Fatalf("second RecoverPending: %v", err)
	}
	if again {
		t.Fatal("the second boot recovered again")
	}

	second, _ := h.store.LastKnownGood()
	if second == nil || !second.AppliedAt.Equal(first.AppliedAt) {
		t.Fatalf("the second boot rewrote the outcome: %+v then %+v", first, second)
	}
	if len(h.net.trace()) != 0 {
		t.Fatalf("the second boot ran %v", h.net.trace())
	}
}

// Both files present means the server died inside an apply after a boot that
// had already recovered an earlier one. The armed marker is the live one and
// takes the restore.
func TestAnArmedMarkerOutranksAnAdoptedRecord(t *testing.T) {
	h := newHarness(t)

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	path, err := h.store.WriteSnapshot(snapshot)
	if err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	if err := h.store.Arm(Pending{
		Operation: operationDisable, SnapshotPath: path,
		ArmedAt: h.clock, Deadline: h.clock.Add(DefaultWindow),
	}); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	if err := utils.WriteFileAtomic(h.store.recoveredPath(), []byte(`{"operation":"enable"}`), fileMode); err != nil {
		t.Fatalf("seed recovered.json: %v", err)
	}

	if _, err := h.mgr.RecoverPending(context.Background()); err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}

	lkg, _ := h.store.LastKnownGood()
	if lkg == nil || !strings.Contains(lkg.Message, operationDisable) {
		t.Fatalf("outcome = %+v, want the armed disable rather than the adopted enable", lkg)
	}
	if len(h.net.trace()) == 0 {
		t.Fatal("an armed marker was adopted rather than restored")
	}
}

func TestKillUdhcpcRemovesThePidfile(t *testing.T) {
	newHarness(t)

	var killed []int
	swap(t, &killProcess, func(pid int) error {
		killed = append(killed, pid)
		return nil
	})

	// No pidfile at all is the static-device case and is not an error.
	if err := killUdhcpc(); err != nil {
		t.Fatalf("killUdhcpc with no pidfile: %v", err)
	}

	writeFile(t, udhcpcPidPath, "4242\n")
	if err := killUdhcpc(); err != nil {
		t.Fatalf("killUdhcpc: %v", err)
	}
	if len(killed) != 1 || killed[0] != 4242 {
		t.Fatalf("killed %v, want [4242]", killed)
	}
	if _, err := os.Stat(udhcpcPidPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("the pidfile survived killUdhcpc")
	}
}

// The boot half of the same durability. S29bridge is the only thing that builds
// br0 after a reboot, and it runs before S30eth, so a script that enslaves eth0
// alone brings a two-port transparent bridge up with one port and leaves it
// there until someone re-applies a profile.
//
// The script is run for real under a PATH shim that records ip. Only the two
// absolute roots it reaches outside its own logic are relocated into a sandbox;
// every command, argument and branch below is the script's own.
func TestBridgeBootScriptEnslavesBothPorts(t *testing.T) {
	tests := []struct {
		name   string
		gadget bool
		want   []string
	}{
		{
			name:   "gadget NIC present",
			gadget: true,
			want: []string{
				"link add name br0 type bridge stp_state 0 forward_delay 0",
				"link set dev br0 address " + testMAC,
				"link set dev eth0 master br0",
				"link set dev eth0 up",
				"link set dev br0 up",
				"link set dev usb0 master br0",
				"link set dev usb0 up",
			},
		},
		// A profile with no network function leaves no usb0 to enslave, and a
		// bridge with one port is the legitimate state there rather than a
		// failed boot.
		{
			name:   "no gadget NIC",
			gadget: false,
			want: []string{
				"link add name br0 type bridge stp_state 0 forward_delay 0",
				"link set dev eth0 master br0",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "sys/class/net/eth0/address"), testMAC+"\n")
			if test.gadget {
				writeFile(t, filepath.Join(root, "sys/class/net/usb0/address"), "48:da:35:6e:11:22\n")
			}
			writeFile(t, filepath.Join(root, "etc/kvm/presentation/network/last-known-good.json"),
				`{"state":"enabled","enabled":true}`)

			trace := filepath.Join(root, "ip.trace")
			shim := filepath.Join(root, "bin", "ip")
			writeFile(t, shim, "#!/bin/sh\necho \"$@\" >> "+trace+"\n")
			if err := os.Chmod(shim, 0o755); err != nil {
				t.Fatalf("chmod ip shim: %v", err)
			}

			source, err := os.ReadFile(filepath.Join("..", "..", "..", "kvmapp", "system", "init.d", "S29bridge"))
			if err != nil {
				t.Fatalf("read S29bridge: %v", err)
			}
			body := strings.ReplaceAll(string(source), "/etc/kvm", filepath.Join(root, "etc/kvm"))
			body = strings.ReplaceAll(body, "/sys/class/net", filepath.Join(root, "sys/class/net"))
			script := filepath.Join(root, "S29bridge")
			writeFile(t, script, body)

			cmd := exec.Command("/bin/sh", script, "start")
			cmd.Env = append(os.Environ(),
				"PATH="+filepath.Dir(shim)+string(os.PathListSeparator)+os.Getenv("PATH"))
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("S29bridge start: %v\n%s", err, out)
			}

			recorded, err := os.ReadFile(trace)
			if err != nil {
				t.Fatalf("read ip trace: %v", err)
			}
			lines := strings.Split(strings.TrimSpace(string(recorded)), "\n")
			requireOrder(t, lines, test.want...)
			if !test.gadget {
				notInTrace(t, lines, "usb0")
			}

			// create() returned success, so start never fell through to the
			// teardown that removes the file, and S30eth will address br0.
			uplink, err := os.ReadFile(filepath.Join(root, "etc/kvm/network/l2-uplink"))
			if err != nil || strings.TrimSpace(string(uplink)) != BridgeName {
				t.Fatalf("l2-uplink = %q, %v, want br0", uplink, err)
			}
		})
	}
}

func TestBridgeBootScriptIsInstalledBeforeEthernet(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	seed := filepath.Join(root, "kvmapp", "system", "init.d", "S29bridge")
	info, err := os.Stat(seed)
	if err != nil {
		t.Fatalf("stat bridge init seed: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("bridge init seed mode = %o, want executable", info.Mode().Perm())
	}

	source, err := os.ReadFile(filepath.Join(root, "support", "sg2002", "kvm_system", "main", "lib", "system_init", "system_init.cpp"))
	if err != nil {
		t.Fatalf("read system installer: %v", err)
	}
	bridgeCopy := `cp -f /kvmapp/system/init.d/S29bridge /etc/init.d/`
	ethernetCopy := `cp -f /kvmapp/system/init.d/S30eth /etc/init.d/`
	bridgeAt, ethernetAt := strings.Index(string(source), bridgeCopy), strings.Index(string(source), ethernetCopy)
	if bridgeAt < 0 {
		t.Fatal("system installer does not refresh S29bridge")
	}
	if ethernetAt < 0 || bridgeAt > ethernetAt {
		t.Fatal("S29bridge is not installed before S30eth")
	}
}
