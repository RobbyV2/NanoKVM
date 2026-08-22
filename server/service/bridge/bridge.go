package bridge

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"NanoKVM-Server/proto"

	log "github.com/sirupsen/logrus"
)

// Commander rejects an argv whose first element is not one of these, so a bug
// that let a request-supplied string reach the runner still cannot spawn
// anything the design did not intend.
const (
	IPBinary       = "ip"
	PingBinary     = "ping"
	IptablesBinary = "iptables"
	S30ethScript   = "/etc/init.d/S30eth"
)

var allowedBinaries = map[string]bool{
	IPBinary:       true,
	PingBinary:     true,
	IptablesBinary: true,
	S30ethScript:   true,
}

// The last gate before an "ip -batch" line reaches a process: no shell
// metacharacter, no newline, nothing but the tokens ip's own parser expects.
var batchLine = regexp.MustCompile(`^[A-Za-z0-9 ./:+-]+$`)

// The single choke point for every external command. It takes an already-split
// argv rather than a string, so no layer of this package holds a command as
// text something could be interpolated into.
type Commander interface {
	Run(ctx context.Context, argv []string, stdin []byte) ([]byte, error)
}

type execCommander struct{}

func NewCommander() Commander { return execCommander{} }

func (execCommander) Run(ctx context.Context, argv []string, stdin []byte) ([]byte, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("bridge: empty command")
	}
	if !allowedBinaries[argv[0]] {
		return nil, fmt.Errorf("bridge: refusing to run %q", argv[0])
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s %s: %w: %s",
			argv[0], strings.Join(argv[1:], " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// Every ip invocation the package makes, one typed method each, so a test can
// swap the Commander for a recorder and assert the exact argv sequence. That is
// the only way to check a step order whose steps are side effects on a kernel.
type IPTool struct {
	cmd Commander
}

func NewIPTool(cmd Commander) *IPTool { return &IPTool{cmd: cmd} }

func (t *IPTool) run(ctx context.Context, args ...string) error {
	_, err := t.cmd.Run(ctx, append([]string{IPBinary}, args...), nil)
	return err
}

func (t *IPTool) decode(ctx context.Context, out any, args ...string) error {
	data, err := t.cmd.Run(ctx, append([]string{IPBinary}, args...), nil)
	if err != nil {
		return err
	}
	// ip emits "[]" for an empty result and, on some builds, nothing at all.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// Links is "ip -j link show".
func (t *IPTool) Links(ctx context.Context) ([]Link, error) {
	var links []Link
	err := t.decode(ctx, &links, "-j", "link", "show")
	return links, err
}

// Addrs is "ip -j addr show".
func (t *IPTool) Addrs(ctx context.Context) ([]Addr, error) {
	var addrs []Addr
	err := t.decode(ctx, &addrs, "-j", "addr", "show")
	return addrs, err
}

// Routes is "ip -j route show".
func (t *IPTool) Routes(ctx context.Context) ([]Route, error) {
	var routes []Route
	err := t.decode(ctx, &routes, "-j", "route", "show")
	return routes, err
}

// AddrsV4 is the first verification gate's primitive: "ip -4 -j addr show dev
// <dev>". The -4 is load-bearing. The bug this exists to avoid is the one at
// S30eth:59, where "ip a show | grep inet" also matches inet6 and so calls an
// interface carrying nothing but a link-local addressed.
func (t *IPTool) AddrsV4(ctx context.Context, dev string) ([]AddrInfo, error) {
	if !safeDevice(dev) {
		return nil, fmt.Errorf("bridge: bad device %q", dev)
	}

	var addrs []Addr
	if err := t.decode(ctx, &addrs, "-4", "-j", "addr", "show", "dev", dev); err != nil {
		return nil, err
	}

	var out []AddrInfo
	for _, addr := range addrs {
		for _, info := range addr.AddrInfo {
			if info.Family == "inet" && info.Local != "" {
				out = append(out, info)
			}
		}
	}
	return out, nil
}

// AddBridge creates br0 with STP off.
//
// stp_state 0 is not a tuning preference. With STP on, br_make_forwarding()
// takes a port through listening and learning first, roughly thirty seconds
// before the bridge raises carrier and reports IFF_RUNNING, and the DHCP branch
// at S30eth:68 is "udhcpc -t 10 -T 1" backgrounded with no fallback address of
// any kind. Ten one-second tries against a thirty-second delay is a device that
// boots with no IPv4 and nothing that will retry. forward_delay 0 is set as
// well and is redundant while STP is off; it is here so that a later reader
// does not mistake the redundant setting for the load-bearing one.
func (t *IPTool) AddBridge(ctx context.Context, name string) error {
	if name != BridgeName {
		return fmt.Errorf("bridge: refusing to create %q", name)
	}
	return t.run(ctx, "link", "add", "name", name, "type", "bridge",
		"stp_state", "0", "forward_delay", "0")
}

// SetMAC pins the bridge's hardware address.
//
// An explicit address marks the bridge address static, so
// br_stp_recalculate_bridge_id() stops re-electing the numerically lowest port
// address every time a port is added or removed. Without it, enslaving usb0 at
// its deterministic 48:da:35:6e:xx:xx can win that election, the bridge's L2
// identity changes under a live DHCP lease, and the reservation the operator
// configured against the old address stops matching.
func (t *IPTool) SetMAC(ctx context.Context, dev, mac string) error {
	if !safeDevice(dev) {
		return fmt.Errorf("bridge: bad device %q", dev)
	}
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("bridge: bad MAC %q: %w", mac, err)
	}
	return t.run(ctx, "link", "set", "dev", dev, "address", hw.String())
}

// SetUp is "ip link set dev <dev> up".
func (t *IPTool) SetUp(ctx context.Context, dev string) error {
	if !safeDevice(dev) {
		return fmt.Errorf("bridge: bad device %q", dev)
	}
	return t.run(ctx, "link", "set", "dev", dev, "up")
}

// SetDown is "ip link set dev <dev> down".
func (t *IPTool) SetDown(ctx context.Context, dev string) error {
	if !safeDevice(dev) {
		return fmt.Errorf("bridge: bad device %q", dev)
	}
	return t.run(ctx, "link", "set", "dev", dev, "down")
}

// SetMaster enslaves a device. It is the only path to "ip link set ... master",
// and checkEnslavable is the reason wlan0 can never reach one: the set is
// closed to eth0 and usb0, and wlan0 is rejected by name with its own error so
// the refusal is legible rather than a generic validation failure.
func (t *IPTool) SetMaster(ctx context.Context, dev, master string) error {
	if err := checkEnslavable(dev); err != nil {
		return err
	}
	if master != BridgeName {
		return fmt.Errorf("bridge: refusing to enslave to %q", master)
	}
	return t.run(ctx, "link", "set", "dev", dev, "master", master)
}

// SetNoMaster releases a device from whatever bridge holds it.
func (t *IPTool) SetNoMaster(ctx context.Context, dev string) error {
	if !safeDevice(dev) {
		return fmt.Errorf("bridge: bad device %q", dev)
	}
	return t.run(ctx, "link", "set", "dev", dev, "nomaster")
}

// DeleteLink removes br0.
func (t *IPTool) DeleteLink(ctx context.Context, name string) error {
	if name != BridgeName {
		return fmt.Errorf("bridge: refusing to delete %q", name)
	}
	return t.run(ctx, "link", "delete", name)
}

// FlushAddr is "ip addr flush dev <dev>".
func (t *IPTool) FlushAddr(ctx context.Context, dev string) error {
	if !safeDevice(dev) {
		return fmt.Errorf("bridge: bad device %q", dev)
	}
	return t.run(ctx, "addr", "flush", "dev", dev)
}

// DeleteDefaultRoute removes the default route through a device. A device with
// no default route makes this fail, and the caller treats that as fine: the
// step exists to guarantee absence, not to report a change.
func (t *IPTool) DeleteDefaultRoute(ctx context.Context, dev string) error {
	if !safeDevice(dev) {
		return fmt.Errorf("bridge: bad device %q", dev)
	}
	return t.run(ctx, "route", "del", "default", "dev", dev)
}

// Batch feeds several commands to one "ip -batch -". Without -force, ip stops
// at the first failure and exits nonzero, which is what makes this a
// transaction: replaying an address and its route either both happen or the
// caller learns that neither can be relied on.
func (t *IPTool) Batch(ctx context.Context, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	for _, line := range lines {
		if !batchLine.MatchString(line) {
			return fmt.Errorf("bridge: refusing batch line %q", line)
		}
	}

	script := strings.Join(lines, "\n") + "\n"
	_, err := t.cmd.Run(ctx, []string{IPBinary, "-batch", "-"}, []byte(script))
	return err
}

// Scripts is the init-script half. Only S30eth is ever invoked, and only with
// "start": the transactions do their own teardown, and S30eth stop hard-exits 1
// when the pidfile is absent, which on a static device is always.
type Scripts interface {
	StartEth(ctx context.Context) error
}

type initScripts struct{ cmd Commander }

// NewScripts returns the real init-script runner.
func NewScripts(cmd Commander) Scripts { return initScripts{cmd: cmd} }

func (s initScripts) StartEth(ctx context.Context) error {
	_, err := s.cmd.Run(ctx, []string{S30ethScript, "start"}, nil)
	return err
}

// Firewall installs and removes the five S95nanokvm rules against an
// interface. The boot-time block is idempotent through "-C || -A" but never
// removes anything, so a live apply has to delete the copies naming the old
// uplink itself or the device keeps a ruleset for a device that is now a port.
type Firewall interface {
	Install(ctx context.Context, iface string) error
	Remove(ctx context.Context, iface string) error
}

// S95nanokvm:92-105 verbatim with the ten interface matches parameterised, in
// the script's order so a diff of "iptables -S" against a stock boot reads
// straight. The last rule is why the third verification gate exists: it keeps
// the raw stream port off the wire, so a gateway that answers ICMP proves
// nothing about whether the management plane can reply.
func firewallRules(iface string) [][]string {
	return [][]string{
		{"INPUT", "-i", iface, "-p", "tcp", "--dport", "80", "-m", "state", "--state", "NEW,ESTABLISHED", "-j", "ACCEPT"},
		{"OUTPUT", "-o", iface, "-p", "tcp", "--sport", "80", "-m", "state", "--state", "ESTABLISHED", "-j", "ACCEPT"},
		{"INPUT", "-i", iface, "-p", "tcp", "--sport", "22", "-m", "state", "--state", "NEW,ESTABLISHED", "-j", "ACCEPT"},
		{"OUTPUT", "-o", iface, "-p", "tcp", "--dport", "22", "-m", "state", "--state", "ESTABLISHED", "-j", "ACCEPT"},
		{"OUTPUT", "-o", iface, "-p", "tcp", "--sport", "8000", "-m", "state", "--state", "ESTABLISHED", "-j", "DROP"},
	}
}

type iptablesFirewall struct{ cmd Commander }

func NewFirewall(cmd Commander) Firewall { return iptablesFirewall{cmd: cmd} }

func (f iptablesFirewall) Install(ctx context.Context, iface string) error {
	if !safeDevice(iface) {
		return fmt.Errorf("bridge: bad interface %q", iface)
	}

	for _, rule := range firewallRules(iface) {
		chain, body := rule[0], rule[1:]
		if _, err := f.cmd.Run(ctx, argv(IptablesBinary, "-C", chain, body), nil); err == nil {
			continue
		}
		if _, err := f.cmd.Run(ctx, argv(IptablesBinary, "-A", chain, body), nil); err != nil {
			return err
		}
	}
	return nil
}

// A rule that is not there is not an error: Remove guarantees absence, and on a
// device where the boot block never ran there is nothing to delete.
func (f iptablesFirewall) Remove(ctx context.Context, iface string) error {
	if !safeDevice(iface) {
		return fmt.Errorf("bridge: bad interface %q", iface)
	}

	for _, rule := range firewallRules(iface) {
		chain, body := rule[0], rule[1:]
		_, _ = f.cmd.Run(ctx, argv(IptablesBinary, "-D", chain, body), nil)
	}
	return nil
}

func argv(binary, action, chain string, body []string) []string {
	out := make([]string, 0, len(body)+3)
	out = append(out, binary, action, chain)
	return append(out, body...)
}

// Pinger is the second verification gate's primitive.
type Pinger interface {
	Ping(ctx context.Context, dev, gateway string) bool
}

type commandPinger struct{ cmd Commander }

// NewPinger returns the real pinger.
func NewPinger(cmd Commander) Pinger { return commandPinger{cmd: cmd} }

// Ping is "ping -I <dev> -w 1 <gw>". Binding the source to the device matters:
// pinging from an enslaved port bypasses the bridge path entirely and the
// gateway never answers, which is the same symptom system_state.cpp:189 shows
// once eth0 becomes a port.
func (p commandPinger) Ping(ctx context.Context, dev, gateway string) bool {
	if !safeDevice(dev) || !isIPv4(gateway) {
		return false
	}
	_, err := p.cmd.Run(ctx, []string{PingBinary, "-I", dev, "-w", "1", "-c", "1", gateway}, nil)
	return err == nil
}

// Liveness is the third verification gate: proof that the management plane, not
// merely the network layer, is reachable at the new address.
type Liveness interface {
	// Observed reports whether the HTTP listener accepted a request whose local
	// address was addr at or after since. This is the strong form, because a
	// real client completed a round trip over the wire.
	Observed(addr string, since time.Time) bool

	// SelfConnect dials the listener from a socket bound to addr. It proves the
	// listener and the local delivery path rather than the wire, and is
	// recorded as the weaker of the two.
	SelfConnect(ctx context.Context, addr string) error
}

// Records the local address of accepted requests, fed by RecordListener. With
// no client watching nothing is recorded, Observed stays false and verification
// falls through to the self-connect, which is the safe direction.
type ListenerWitness struct {
	mu     sync.Mutex
	seen   map[string]time.Time
	scheme string
	port   int
}

// scheme and port name the listener the server actually serves: http on a
// device left at the default proto, https once a certificate is configured.
func NewListenerWitness(scheme string, port int) *ListenerWitness {
	return &ListenerWitness{seen: make(map[string]time.Time), scheme: scheme, port: port}
}

func (w *ListenerWitness) Record(localAddr string) {
	host, _, err := net.SplitHostPort(localAddr)
	if err != nil {
		host = localAddr
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.seen[host] = time.Now()
}

func (w *ListenerWitness) Observed(addr string, since time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	at, ok := w.seen[addr]
	return ok && !at.Before(since)
}

// Binds the source to the address under test, so a reply proves delivery to
// that address rather than over the loopback the listener also answers on. Any
// HTTP status counts: a 401 is still the listener answering.
func (w *ListenerWitness) SelfConnect(ctx context.Context, addr string) error {
	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("bridge: bad address %q", addr)
	}

	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{IP: ip},
		Timeout:   5 * time.Second,
	}
	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
			// The device serves a self-signed certificate it generates itself,
			// so verification would fail on every device and prove nothing
			// about reachability either way.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}

	target := fmt.Sprintf("%s://%s/api/vm/info", w.scheme, net.JoinHostPort(addr, strconv.Itoa(w.port)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}

	rsp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	return nil
}

// var so the deadline arithmetic is testable without waiting sixty seconds.
var newTimer = func(d time.Duration) (<-chan time.Time, func()) {
	timer := time.NewTimer(d)
	return timer.C, func() { timer.Stop() }
}

// var for the same reason: a test observes that udhcpc was killed without
// there being a udhcpc.
var killProcess = func(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }

// One armed rollback. Exactly one of the watchdog and the transaction performs
// the restore, decided by a compare-and-swap: without it a verification failing
// at the instant the deadline expires runs two restores over the same links.
type deadman struct {
	pending Pending
	taken   atomic.Bool
	expired atomic.Bool
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
}

// Expired reports whether the deadline fired.
func (d *deadman) Expired() bool { return d.expired.Load() }

// take claims the right to restore. It succeeds for exactly one caller.
func (d *deadman) take() bool { return d.taken.CompareAndSwap(false, true) }

// stopWatch ends the watchdog goroutine and waits for it. The wait is
// deliberate: if the watchdog is mid-restore, the transaction must not proceed
// to mutate the interfaces it is putting back.
func (d *deadman) stopWatch() {
	d.once.Do(func() { close(d.stop) })
	<-d.done
}

// disarm stops the watchdog and reports whether the caller owns the outcome.
// False means the deadline beat it and a restore has already run.
func (d *deadman) disarm() bool {
	owned := d.take()
	d.stopWatch()
	return owned
}

// Every external dependency is an interface, so the two transactions run in a
// test with no device, no network and no clock.
type Manager struct {
	ip       *IPTool
	scripts  Scripts
	firewall Firewall
	pinger   Pinger
	live     Liveness
	gadget   Gadget
	store    *Store

	window time.Duration
	now    func() time.Time

	// mu serialises transactions in this process. Two applies interleaving
	// their mutations would leave a snapshot that describes neither.
	mu       sync.Mutex
	inflight bool
}

// A zero field takes the real implementation.
type Config struct {
	Commander Commander
	IP        *IPTool
	Scripts   Scripts
	Firewall  Firewall
	Pinger    Pinger
	Liveness  Liveness

	// Gadget may be nil. The twelve steps that hold the management address
	// never touch it, so a device with no gadget NIC runs the whole
	// transaction unchanged and simply skips step 13.
	Gadget Gadget

	Store  *Store
	Window time.Duration
	Now    func() time.Time
}

func New(cfg Config) *Manager {
	cmd := cfg.Commander
	if cmd == nil {
		cmd = NewCommander()
	}

	m := &Manager{
		ip:       cfg.IP,
		scripts:  cfg.Scripts,
		firewall: cfg.Firewall,
		pinger:   cfg.Pinger,
		live:     cfg.Liveness,
		gadget:   cfg.Gadget,
		store:    cfg.Store,
		window:   cfg.Window,
		now:      cfg.Now,
	}

	if m.ip == nil {
		m.ip = NewIPTool(cmd)
	}
	if m.scripts == nil {
		m.scripts = NewScripts(cmd)
	}
	if m.firewall == nil {
		m.firewall = NewFirewall(cmd)
	}
	if m.pinger == nil {
		m.pinger = NewPinger(cmd)
	}
	if m.live == nil {
		m.live = NewListenerWitness("https", 443)
	}
	if m.store == nil {
		m.store = NewStore()
	}
	if m.window <= 0 {
		m.window = DefaultWindow
	}
	if m.now == nil {
		m.now = time.Now
	}
	return m
}

func (m *Manager) Store() *Store { return m.store }

// Step 2. The marker is durable before this returns and the watchdog is running
// before the caller's first mutation, so there is no window in which the device
// has been changed and nothing is watching.
func (m *Manager) arm(operation, snapshotPath string) (*deadman, error) {
	now := m.now()
	pending := Pending{
		Operation:    operation,
		SnapshotPath: snapshotPath,
		ArmedAt:      now,
		Deadline:     now.Add(m.window),
	}
	if err := m.store.Arm(pending); err != nil {
		return nil, fmt.Errorf("arm dead-man: %w", err)
	}

	d := &deadman{
		pending: pending,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}

	fire, stopTimer := newTimer(pending.Remaining(now))
	go func() {
		defer close(d.done)
		select {
		case <-d.stop:
			stopTimer()
		case <-fire:
			d.expired.Store(true)
			if d.take() {
				m.expire(pending)
			}
		}
	}()

	return d, nil
}

// The watchdog firing. It runs on its own context: the transaction's is very
// likely the one that just expired, and a restore that inherits a cancelled
// context restores nothing.
func (m *Manager) expire(pending Pending) {
	log.Warnf("bridge: dead-man expired for %s, restoring %s", pending.Operation, pending.SnapshotPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := m.restoreFrom(ctx, pending); err != nil {
		log.Errorf("bridge: dead-man restore failed: %s", err)
	}
}

// The marker is cleared last, so an interruption anywhere inside leaves one the
// boot-time check acts on again.
func (m *Manager) restoreFrom(ctx context.Context, pending Pending) error {
	snapshot, err := m.store.ReadSnapshot(pending.SnapshotPath)
	if err != nil {
		return err
	}

	restoreErr := m.Restore(ctx, snapshot)

	lkg := LastKnownGood{
		Enabled:   snapshot.UplinkName() == BridgeName,
		Uplink:    snapshot.UplinkName(),
		State:     proto.BridgeRolledBack,
		Message:   fmt.Sprintf("%s rolled back", pending.Operation),
		AppliedAt: m.now().UTC(),
	}
	if restoreErr != nil {
		lkg.State = proto.BridgeFailed
		lkg.Message = fmt.Sprintf("%s rollback incomplete: %s", pending.Operation, restoreErr)
	}

	if err := m.store.Commit(lkg); err != nil {
		return fmt.Errorf("record rollback: %w", err)
	}
	return restoreErr
}

// The boot-time half of the dead-man. It restores unconditionally when a marker
// exists, without consulting the deadline: a marker that survived a reboot means
// the process that armed it never disarmed, and the deadline is measured against
// a clock this device likely did not keep across the power cut.
func (m *Manager) RecoverPending(ctx context.Context) (bool, error) {
	pending, err := m.store.Pending()
	if err != nil {
		return false, err
	}
	if pending == nil {
		return false, nil
	}

	log.Warnf("bridge: found armed pending marker from %s, restoring", pending.ArmedAt)
	return true, m.restoreFrom(ctx, *pending)
}

func (m *Manager) hasIPv4(ctx context.Context, dev string) bool {
	addrs, err := m.ip.AddrsV4(ctx, dev)
	return err == nil && len(addrs) > 0
}

// Read before anything is enslaved: once eth0 is a port, the value that matters
// for the pin is still eth0's own and not whatever the bridge settled on.
func permanentMAC(dev string) (string, error) {
	if !safeDevice(dev) {
		return "", fmt.Errorf("bridge: bad device %q", dev)
	}

	data, err := os.ReadFile(filepath.Join(sysClassNet, dev, "address"))
	if err != nil {
		return "", fmt.Errorf("read %s MAC: %w", dev, err)
	}

	mac := trimLine(string(data))
	if _, err := net.ParseMAC(mac); err != nil {
		return "", fmt.Errorf("bridge: %s reports MAC %q: %w", dev, mac, err)
	}
	return mac, nil
}

// Step 4. A lease renewal landing after the flush would re-add an address to a
// device about to become a port, leaving the address on a port while the route
// points at a bridge.
func killUdhcpc() error {
	data, err := os.ReadFile(udhcpcPidPath)
	if err != nil {
		return nil
	}

	pid, err := strconv.Atoi(trimLine(string(data)))
	if err == nil && pid > 0 {
		if err := killProcess(pid); err != nil {
			log.Warnf("bridge: kill udhcpc %d: %s", pid, err)
		}
	}

	if err := os.Remove(udhcpcPidPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
