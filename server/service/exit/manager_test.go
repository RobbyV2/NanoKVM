package exit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

// The fakes. Every external contact of the manager is recorded in one trace so
// a test asserts the step order of a transaction.
type fakeRunner struct {
	mu    sync.Mutex
	trace *[]string
	fail  map[string]error
	// status is what `S94exit status <n>` prints.
	status string
}

func (r *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	// exec.CommandContext refuses to start on a done context; so does the fake,
	// before the call is even traced.
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	call := filepath.Base(name) + " " + strings.Join(args, " ")
	*r.trace = append(*r.trace, call)
	if err, ok := r.fail[call]; ok {
		return "", err
	}
	if strings.HasPrefix(call, "S94exit status") {
		return r.status, nil
	}
	return "", nil
}

type fakeNIC struct {
	name, protocol string
	bound          bool
	err            error
}

func (n *fakeNIC) NIC(context.Context) (string, error)             { return n.name, n.err }
func (n *fakeNIC) NetworkProtocol(context.Context) (string, error) { return n.protocol, n.err }
func (n *fakeNIC) UDCBound() bool                                  { return n.bound }

type fakeRebinder struct {
	trace *[]string
	err   error
}

func (r *fakeRebinder) Rebind(context.Context) error {
	*r.trace = append(*r.trace, "rebind")
	return r.err
}

type fakeBridge struct {
	active bool
	why    string
}

func (b fakeBridge) BridgeActive() (bool, string) { return b.active, b.why }

type fakeDoor struct {
	trace   *[]string
	startEr error
	state   proto.ExitTunnelState
	peer    *proto.ExitPeer
	since   *time.Time
	native  Backend
	relay   RelayGate
}

func (d *fakeDoor) Start() error {
	*d.trace = append(*d.trace, "door.start")
	return d.startEr
}
func (d *fakeDoor) Stop()                { *d.trace = append(*d.trace, "door.stop") }
func (d *fakeDoor) SetNative(b Backend)  { d.native = b }
func (d *fakeDoor) SetRelay(g RelayGate) { d.relay = g }
func (d *fakeDoor) Attached() (proto.ExitTunnelState, *proto.ExitPeer, *time.Time) {
	if d.state == "" {
		return proto.ExitDisconnected, nil, nil
	}
	return d.state, d.peer, d.since
}

type fakeMux struct {
	trace   *[]string
	current Backend
	pin     bool
	served  int
}

func (m *fakeMux) ServeNative(w http.ResponseWriter, _ *http.Request) {
	m.served++
	w.WriteHeader(http.StatusSwitchingProtocols)
}
func (m *fakeMux) Current() Backend       { return m.current }
func (m *fakeMux) CloseAll(reason string) { *m.trace = append(*m.trace, "mux.closeall "+reason) }
func (m *fakeMux) SetPinPeer(pin bool)    { m.pin = pin }

type fakeProxy struct {
	trace  *[]string
	served int
	pin    bool
}

func (p *fakeProxy) Connected() bool         { return false }
func (p *fakeProxy) Peer() *proto.ExitPeer   { return nil }
func (p *fakeProxy) ConnectedAt() *time.Time { return nil }
func (p *fakeProxy) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	p.served++
	w.WriteHeader(http.StatusSwitchingProtocols)
}
func (p *fakeProxy) CloseAll(reason string) { *p.trace = append(*p.trace, "proxy.closeall "+reason) }
func (p *fakeProxy) SetPinPeer(pin bool)    { p.pin = pin }

type fakeDNS struct {
	trace  *[]string
	bindEr error
	bound  bool
	addr   netip.Addr
}

func (d *fakeDNS) Bind(addr netip.Addr) error {
	*d.trace = append(*d.trace, "dns.bind "+addr.String())
	if d.bindEr != nil {
		return d.bindEr
	}
	d.bound, d.addr = true, addr
	return nil
}
func (d *fakeDNS) Stop()                     { *d.trace = append(*d.trace, "dns.stop"); d.bound = false }
func (d *fakeDNS) Bound() bool               { return d.bound }
func (d *fakeDNS) Stats() proto.ExitDNSStats { return proto.ExitDNSStats{Queries: 7} }

type fakeProbe struct{ trace *[]string }

func (p *fakeProbe) Start() { *p.trace = append(*p.trace, "probe.start") }
func (p *fakeProbe) Stop()  { *p.trace = append(*p.trace, "probe.stop") }
func (p *fakeProbe) Result() proto.ExitUpstream {
	return proto.ExitUpstream{Reachable: true, LatencyMs: 41}
}

type fakeBackend struct {
	peer proto.ExitPeer
	done chan struct{}
}

func (b *fakeBackend) DialTCP(context.Context, netip.AddrPort) (net.Conn, error) { return nil, nil }
func (b *fakeBackend) OpenUDP(context.Context) (UDPStream, error)                { return nil, nil }
func (b *fakeBackend) Peer() proto.ExitPeer                                      { return b.peer }
func (b *fakeBackend) ConnectedAt() time.Time                                    { return time.Time{} }
func (b *fakeBackend) Done() <-chan struct{}                                     { return b.done }
func (b *fakeBackend) Close(string)                                              {}

type harness struct {
	t      *testing.T
	trace  []string
	runner *fakeRunner
	nic    *fakeNIC
	rebind *fakeRebinder
	bridge *fakeBridge
	door   *fakeDoor
	mux    *fakeMux
	proxy  *fakeProxy
	dns    *fakeDNS
	probe  *fakeProbe
	deps   componentDeps
	built  int
	now    time.Time
	mgr    *Manager
}

const healthy = "forward=1\nrouting=1\ntun=1\nhev=1\nwstunnel=1\nnat=1\nnic=usb0\ngw=10.1.2.1\ndns_redirected=3\n"

func newHarness(t *testing.T) *harness {
	t.Helper()
	useTempDirs(t)
	writeExec(t, filepath.Join(InitSeedDir, S94Script), "#!/bin/sh\n# seed\n")
	writeExec(t, filepath.Join(InitSeedDir, S30Script), "#!/bin/sh\n# seed\n")

	oldEnsure := ensureBinary
	ensureBinary = func(name string) (string, error) { return filepath.Join(BinDir, name), nil }
	t.Cleanup(func() { ensureBinary = oldEnsure })
	oldBudget := hevStartBudget
	hevStartBudget = 150 * time.Millisecond
	t.Cleanup(func() { hevStartBudget = oldBudget })

	h := &harness{t: t, now: time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)}
	h.runner = &fakeRunner{trace: &h.trace, fail: map[string]error{}, status: healthy}
	h.nic = &fakeNIC{name: "usb0", protocol: "ncm", bound: true}
	h.rebind = &fakeRebinder{trace: &h.trace}
	h.bridge = &fakeBridge{}
	h.door = &fakeDoor{trace: &h.trace}
	h.mux = &fakeMux{trace: &h.trace}
	h.proxy = &fakeProxy{trace: &h.trace}
	h.dns = &fakeDNS{trace: &h.trace}
	h.probe = &fakeProbe{trace: &h.trace}

	h.mgr = NewManager(Deps{
		NIC:    h.nic,
		Rebind: h.rebind,
		Bridge: h.bridge,
		Runner: h.runner,
		Now:    func() time.Time { return h.now },
		Components: func(slot Slot, d componentDeps) components {
			h.built++
			h.deps = d
			return components{Door: h.door, Mux: h.mux, Proxy: h.proxy, DNS: h.dns, Probe: h.probe}
		},
		NICInfo: func(ifname string) (bool, netip.Prefix) {
			if ifname != "usb0" && ifname != "usb1" {
				return false, netip.Prefix{}
			}
			return true, netip.MustParsePrefix("10.1.2.1/24")
		},
	})
	t.Cleanup(h.mgr.Close)
	return h
}

func (h *harness) reset() { h.trace = h.trace[:0] }

func (h *harness) index(substr string) int {
	for i, line := range h.trace {
		if strings.Contains(line, substr) {
			return i
		}
	}
	return -1
}

func (h *harness) mustContain(substrs ...string) {
	h.t.Helper()
	last := -1
	for _, s := range substrs {
		i := h.index(s)
		if i < 0 {
			h.t.Fatalf("trace lacks %q:\n%s", s, strings.Join(h.trace, "\n"))
		}
		if i < last {
			h.t.Fatalf("%q came before its predecessor in:\n%s", s, strings.Join(h.trace, "\n"))
		}
		last = i
	}
}

func (h *harness) mustNotContain(substr string) {
	h.t.Helper()
	if h.index(substr) >= 0 {
		h.t.Fatalf("trace unexpectedly has %q:\n%s", substr, strings.Join(h.trace, "\n"))
	}
}

func TestInitCreatesSlotZeroAndInstallsScripts(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()

	cfg, ok, err := LoadConfig(MustSlot("0"))
	if err != nil || !ok {
		t.Fatalf("slot 0 config: ok=%v err=%v", ok, err)
	}
	if cfg.Enabled || !ValidToken(cfg.Token) || cfg.Mode != proto.ExitModeNative {
		t.Fatalf("slot 0 = %+v", cfg)
	}
	if _, err := os.Stat(MustSlot("0").EnvPath()); err != nil {
		t.Fatalf("env not rendered: %v", err)
	}
	for _, name := range []string{S94Script, S30Script} {
		info, err := os.Stat(filepath.Join(InitDir, name))
		if err != nil || info.Mode()&0o111 == 0 {
			t.Fatalf("%s not installed executable: %v", name, err)
		}
	}
	if h.built != 0 {
		t.Fatal("a disabled slot built components")
	}
	h.mustNotContain("S94exit start")

	slots := h.mgr.Slots()
	if len(slots.Slots) != 1 || slots.Slots[0].Slot != "0" || slots.Slots[0].Tunnel != proto.ExitDisconnected {
		t.Fatalf("slots = %+v", slots)
	}
}

func TestInitDisarmsAPendingSlotAndStartsAnEnabledOne(t *testing.T) {
	h := newHarness(t)
	pending, _ := DefaultConfig(MustSlot("0"))
	pending.Pending = true
	if err := SaveConfig(pending); err != nil {
		t.Fatal(err)
	}
	enabled, _ := DefaultConfig(MustSlot("1"))
	enabled.Enabled = true
	if err := SaveConfig(enabled); err != nil {
		t.Fatal(err)
	}
	if err := WriteGadgetRoute(MustSlot("0")); err != nil {
		t.Fatal(err)
	}

	h.mgr.Init()

	cfg, _, _ := LoadConfig(MustSlot("0"))
	if cfg.Enabled || cfg.Pending {
		t.Fatalf("pending slot not disarmed: %+v", cfg)
	}
	if _, err := os.Stat(GadgetRoutePath()); !os.IsNotExist(err) {
		t.Fatal("pending slot's marker survived")
	}
	h.mustContain("S94exit stop 0", "S94exit start 1", "door.start", "dns.bind 10.1.2.1", "probe.start", "S94exit status 1")
	if h.built != 1 {
		t.Fatalf("built %d component sets, want 1", h.built)
	}
	status, _ := h.mgr.Status(MustSlot("1"))
	if !status.Enabled || !status.Downstream.Hev || !status.Downstream.DNS || status.NIC.Ifname != "usb0" || status.NIC.Address != "10.1.2.1/24" {
		t.Fatalf("status = %+v", status)
	}
}

func TestEnableStepOrder(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	h.reset()
	slot := MustSlot("0")

	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatalf("enable: %v", err)
	}
	h.mustContain(
		"S94exit start 0",
		"S94exit status 0",
		"door.start",
		"dns.bind 10.1.2.1",
		"probe.start",
		"S30rndis restart",
		"rebind",
	)
	// The marker and the flip come after the listeners are up and before the
	// udhcpd restart that reads the marker.
	cfg, _, _ := LoadConfig(slot)
	if !cfg.Enabled || cfg.Pending {
		t.Fatalf("config after enable = %+v", cfg)
	}
	data, err := os.ReadFile(GadgetRoutePath())
	if err != nil || strings.TrimSpace(string(data)) != "0" {
		t.Fatalf("marker = %q, %v", data, err)
	}
	env, _ := os.ReadFile(slot.EnvPath())
	if !strings.Contains(string(env), "ENABLED=1\nPENDING=0\n") || !strings.Contains(string(env), "NIC=usb0\n") {
		t.Fatalf("env = %s", env)
	}
	if h.door.relay != nil {
		t.Fatal("native mode set a relay")
	}
	status, _ := h.mgr.Status(slot)
	if !status.Enabled || status.Pending || status.Message != "" || !status.Downstream.DNS || status.DNS.Redirected != 3 || status.DNS.Queries != 7 || !status.Upstream.Reachable {
		t.Fatalf("status = %+v", status)
	}

	// Idempotent.
	h.reset()
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	if len(h.trace) != 0 {
		t.Fatalf("second enable did work:\n%s", strings.Join(h.trace, "\n"))
	}
}

func TestEnableRefusals(t *testing.T) {
	cases := []struct {
		name  string
		setup func(h *harness)
		want  string
	}{
		{"bridge", func(h *harness) { h.bridge.active, h.bridge.why = true, "br0 exists" }, "L2 bridge"},
		{"no network function", func(h *harness) { h.nic.protocol = "" }, "USB Network Adapter"},
		{"gadget unreadable", func(h *harness) { h.nic.err = errors.New("no gadget") }, "no gadget"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.mgr.Init()
			tc.setup(h)
			h.reset()
			err := h.mgr.Enable(context.Background(), MustSlot("0"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("enable error = %v, want %q", err, tc.want)
			}
			if len(h.trace) != 0 {
				t.Fatalf("a refusal touched the device:\n%s", strings.Join(h.trace, "\n"))
			}
			cfg, _, _ := LoadConfig(MustSlot("0"))
			if cfg.Enabled || cfg.Pending {
				t.Fatalf("config after refusal = %+v", cfg)
			}
			status, _ := h.mgr.Status(MustSlot("0"))
			if !strings.Contains(status.Message, tc.want) {
				t.Fatalf("status message = %q", status.Message)
			}
		})
	}
}

func TestEnableRollsBackOnEachFailingStep(t *testing.T) {
	cases := []struct {
		name  string
		setup func(h *harness)
		want  string
	}{
		{"S94exit start", func(h *harness) { h.runner.fail["S94exit start 0"] = errors.New("iptables: nft_compat missing") }, "S94exit start"},
		{"hev dead", func(h *harness) { h.runner.status = strings.Replace(healthy, "hev=1", "hev=0", 1) }, "hev-socks5-tunnel is not running"},
		{"front door", func(h *harness) { h.door.startEr = errors.New("address in use") }, "socks front door"},
		{"dns bind", func(h *harness) { h.dns.bindEr = errors.New("bind: address in use") }, "dns forwarder"},
		{"no address", func(h *harness) {
			h.mgr.deps.NICInfo = func(string) (bool, netip.Prefix) { return true, netip.Prefix{} }
		}, "no 10.x address"},
		{"S30rndis", func(h *harness) { h.runner.fail["S30rndis restart"] = errors.New("boom") }, "S30rndis restart"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.mgr.Init()
			tc.setup(h)
			h.reset()
			slot := MustSlot("0")

			err := h.mgr.Enable(context.Background(), slot)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("enable error = %v, want %q", err, tc.want)
			}
			h.mustContain("S94exit stop 0")
			h.mustNotContain("rebind")
			cfg, _, _ := LoadConfig(slot)
			if cfg.Enabled || cfg.Pending {
				t.Fatalf("config after rollback = %+v", cfg)
			}
			if _, err := os.Stat(GadgetRoutePath()); !os.IsNotExist(err) {
				t.Fatal("marker survived the rollback")
			}
			h.mgr.mu.Lock()
			comps := h.mgr.slots["0"].comps
			h.mgr.mu.Unlock()
			if comps != nil {
				t.Fatal("components survived the rollback")
			}
			// A front door that failed to listen has nothing to stop; every
			// other started component set is torn down.
			if h.built > 0 && tc.name != "front door" && h.index("door.stop") < 0 {
				t.Fatalf("started components were not stopped:\n%s", strings.Join(h.trace, "\n"))
			}
			status, _ := h.mgr.Status(slot)
			if !strings.Contains(status.Message, tc.want) || status.Enabled {
				t.Fatalf("status = %+v", status)
			}
			// A locked-out no-address enable ran S30rndis start once to try.
			if tc.name == "no address" {
				h.mustContain("S30rndis start")
			}
		})
	}
}

func TestEnableSurvivesARebindRefusal(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	h.rebind.err = errors.New("usb presentation has an active transient mode")

	if err := h.mgr.Enable(context.Background(), MustSlot("0")); err != nil {
		t.Fatalf("a rebind refusal failed the enable: %v", err)
	}
	status, _ := h.mgr.Status(MustSlot("0"))
	if !status.Enabled || !strings.Contains(status.Message, "re-plug the USB cable") {
		t.Fatalf("status = %+v", status)
	}
}

func TestDisableOrderAndConvergence(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	h.reset()

	if err := h.mgr.Disable(context.Background(), slot); err != nil {
		t.Fatalf("disable: %v", err)
	}
	h.mustContain("S30rndis restart", "probe.stop", "dns.stop", "mux.closeall", "proxy.closeall", "door.stop", "S94exit stop 0", "rebind")
	if _, err := os.Stat(GadgetRoutePath()); !os.IsNotExist(err) {
		t.Fatal("marker survived disable")
	}
	cfg, _, _ := LoadConfig(slot)
	if cfg.Enabled || cfg.Pending {
		t.Fatalf("config after disable = %+v", cfg)
	}
	env, _ := os.ReadFile(slot.EnvPath())
	if !strings.Contains(string(env), "ENABLED=0\n") {
		t.Fatalf("env after disable = %s", env)
	}
	// The marker and the config flip precede the first command, so a crash
	// after either converges to off.
	if h.index("S30rndis restart") != 0 {
		t.Fatalf("something ran before the udhcpd restart:\n%s", strings.Join(h.trace, "\n"))
	}
	status, _ := h.mgr.Status(slot)
	if status.Enabled || status.Tunnel != proto.ExitDisconnected || status.Downstream.DNS {
		t.Fatalf("status = %+v", status)
	}

	h.reset()
	if err := h.mgr.Disable(context.Background(), slot); err != nil || len(h.trace) != 0 {
		t.Fatalf("second disable: err=%v trace=%v", err, h.trace)
	}
}

func TestSessionHooksRecordPeersAndSupersede(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}

	first := &fakeBackend{peer: proto.ExitPeer{Addr: "203.0.113.7:5000", Hostname: "a", Transport: proto.ExitModeNative}}
	h.deps.OnSession(first)
	if h.door.native != first {
		t.Fatal("front door not pointed at the session")
	}
	h.mux.current = first
	st, _ := LoadState(slot)
	if st.LastConnectedAt == nil || !st.LastConnectedAt.Equal(h.now) || st.PreviousPeer != nil {
		t.Fatalf("state after first session = %+v", st)
	}

	// A new session from another address supersedes: previousPeer is set.
	h.now = h.now.Add(time.Minute)
	second := &fakeBackend{peer: proto.ExitPeer{Addr: "198.51.100.9:6000", Hostname: "b", Transport: proto.ExitModeNative}}
	h.deps.OnSession(second)
	h.mux.current = second
	// The old session closing must not detach the new one.
	h.deps.OnClose(first, "superseded")
	if h.door.native != second {
		t.Fatal("a superseded session's close detached the current one")
	}
	status, _ := h.mgr.Status(slot)
	if status.PreviousPeer == nil || status.PreviousPeer.Addr != "203.0.113.7:5000" || status.PeerChangedAt == nil {
		t.Fatalf("status after supersede = %+v", status)
	}

	// The current session closing detaches the door.
	h.mux.current = nil
	h.deps.OnClose(second, "ws closed")
	if h.door.native != nil {
		t.Fatal("current session's close left the door attached")
	}

	// Regenerating the token closes everything and forgets the peer's limiter entry.
	h.mgr.limiter.Fail("198.51.100.9")
	old := status.Token
	h.reset()
	token, err := h.mgr.RegenerateToken(context.Background(), slot)
	if err != nil || token == old || !ValidToken(token) {
		t.Fatalf("regenerate: %q %v", token, err)
	}
	h.mustContain("mux.closeall token regenerated", "proxy.closeall token regenerated")
	if h.mgr.limiter.Len() != 0 {
		t.Fatal("peer's limiter entry survived the regenerate")
	}
	restrict, _ := os.ReadFile(slot.RestrictPath())
	if !strings.Contains(string(restrict), token) || strings.Contains(string(restrict), old) {
		t.Fatalf("restrict yaml not rewritten: %s", restrict)
	}
}

func TestSetConfigModeSwitchRewiresTheFrontDoor(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	h.reset()

	mode := proto.ExitModeWstunnel
	pin := true
	if err := h.mgr.SetConfig(context.Background(), slot, proto.SetExitConfigReq{Mode: &mode, PinPeer: &pin}); err != nil {
		t.Fatal(err)
	}
	h.mustContain("door.stop", "S94exit start 0", "door.start")
	if h.door.relay == nil {
		t.Fatal("wstunnel mode did not set the relay gate")
	}
	if !h.mux.pin || !h.proxy.pin {
		t.Fatal("pinPeer not pushed")
	}
	env, _ := os.ReadFile(slot.EnvPath())
	if !strings.Contains(string(env), "MODE=wstunnel\n") {
		t.Fatalf("env = %s", env)
	}
	cfg, _ := h.mgr.Config(slot)
	if cfg.Mode != proto.ExitModeWstunnel || !cfg.PinPeer {
		t.Fatalf("config = %+v", cfg)
	}

	// A policy change reapplies the chain only.
	h.reset()
	allow := true
	if err := h.mgr.SetConfig(context.Background(), slot, proto.SetExitConfigReq{AllowPrivate: &allow}); err != nil {
		t.Fatal(err)
	}
	h.mustContain("S94exit start 0")
	h.mustNotContain("door.stop")
	if !h.deps.Policy().AllowPrivate {
		t.Fatal("policy closure does not read the live config")
	}
	env, _ = os.ReadFile(slot.EnvPath())
	if strings.Contains(string(env), "10.0.0.0/8") {
		t.Fatalf("private prefixes still rejected in env:\n%s", env)
	}

	// Bad values are refused by validation upstream; here they normalise.
	mtu := 9000
	if err := h.mgr.SetConfig(context.Background(), slot, proto.SetExitConfigReq{MTU: &mtu}); err != nil {
		t.Fatal(err)
	}
	cfg, _ = h.mgr.Config(slot)
	if cfg.MTU != DefaultMTU {
		t.Fatalf("mtu = %d", cfg.MTU)
	}
}

func TestRebindSubscriberConverges(t *testing.T) {
	h := newHarness(t)
	var hook func(context.Context)
	h.mgr.deps.Subscribe = func(fn func(context.Context)) { hook = fn }
	h.mgr.Init()
	slot := MustSlot("0")
	if hook == nil {
		t.Fatal("manager did not subscribe to rebinds")
	}
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}

	// The gadget came back as usb1 with no address yet.
	h.nic.name = "usb1"
	addressed := false
	h.mgr.deps.NICInfo = func(ifname string) (bool, netip.Prefix) {
		if ifname == "usb1" && addressed {
			return true, netip.MustParsePrefix("10.9.9.1/24")
		}
		return true, netip.Prefix{}
	}
	h.reset()
	hook(context.Background())

	// No address yet: the converge ran S30rndis start to put one there and
	// did not bind the forwarder to nothing.
	h.mustContain("S94exit start 0", "S30rndis start")
	h.mustNotContain("dns.bind")
	nicFile, _ := os.ReadFile(slot.NicPath())
	if strings.TrimSpace(string(nicFile)) != "usb1" {
		t.Fatalf("nic file = %q", nicFile)
	}
	status, _ := h.mgr.Status(slot)
	if status.NIC.Ifname != "usb1" {
		t.Fatalf("nic = %+v", status.NIC)
	}

	// Now the address is there: the next tick rebinds the forwarder.
	addressed = true
	h.reset()
	h.mgr.Tick(context.Background())
	h.mustContain("S94exit start 0", "dns.bind 10.9.9.1")
	if h.dns.addr.String() != "10.9.9.1" {
		t.Fatalf("forwarder bound to %s", h.dns.addr)
	}
	status, _ = h.mgr.Status(slot)
	if status.NIC.Address != "10.9.9.1/24" {
		t.Fatalf("nic = %+v", status.NIC)
	}
}

func TestUnboundUDCKeepsTheLastNIC(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	h.nic.bound = false
	h.nic.name = "(unnamed net_device)"
	h.mgr.Tick(context.Background())
	status, _ := h.mgr.Status(slot)
	if status.NIC.Ifname != "usb0" {
		t.Fatalf("nic after an unbound tick = %+v", status.NIC)
	}
}

func TestLogsRedactTokens(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	cfg, _, _ := LoadConfig(slot)
	if err := os.MkdirAll(LogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := "restriction matched Bearer " + cfg.Token + " from 203.0.113.7 path exit0 abcdefgh\n"
	if err := os.WriteFile(slot.LogPath("wstunnel"), []byte(strings.Repeat("noise\n", 300)+line), 0o644); err != nil {
		t.Fatal(err)
	}
	logs, err := h.mgr.Logs(slot)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Wstunnel) != logTailLines || len(logs.Hev) != 0 {
		t.Fatalf("tail lengths = %d/%d", len(logs.Wstunnel), len(logs.Hev))
	}
	last := logs.Wstunnel[len(logs.Wstunnel)-1]
	if strings.Contains(last, cfg.Token) || strings.Contains(last, "abcdefgh") || !strings.Contains(last, "********") {
		t.Fatalf("token not redacted: %q", last)
	}
}

// TestTransactionsOutliveTheRequest: gin cancels the request context the
// moment the client goes away, and exec.CommandContext then refuses to run
// anything. A disable that has already flipped enabled:false must still stop
// the daemons, and an enable that has been armed must still finish or roll
// back, whatever the caller's context does.
func TestTransactionsOutliveTheRequest(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	if err := h.mgr.Enable(cancelled, slot); err != nil {
		t.Fatalf("enable on a cancelled request context: %v", err)
	}
	h.mustContain("S94exit start 0", "door.start", "S30rndis restart", "rebind")
	cfg, _, _ := LoadConfig(slot)
	if !cfg.Enabled {
		t.Fatalf("config after enable = %+v", cfg)
	}

	h.reset()
	if err := h.mgr.Disable(cancelled, slot); err != nil {
		t.Fatalf("disable on a cancelled request context: %v", err)
	}
	h.mustContain("S30rndis restart", "door.stop", "S94exit stop 0", "rebind")
	cfg, _, _ = LoadConfig(slot)
	if cfg.Enabled {
		t.Fatalf("config after disable = %+v", cfg)
	}

	h.reset()
	mode := proto.ExitModeWstunnel
	if err := h.mgr.SetConfig(cancelled, slot, proto.SetExitConfigReq{Mode: &mode}); err != nil {
		t.Fatalf("set config on a cancelled request context: %v", err)
	}
	if _, err := h.mgr.RegenerateToken(cancelled, slot); err != nil {
		t.Fatalf("regenerate on a cancelled request context: %v", err)
	}
}

// TestWatchdogStopsAnOrphanedDisabledSlot: a slot recorded disabled whose
// downstream is still up (the server died between the config flip and the
// S94exit stop) is stopped by the watchdog rather than left running forever.
func TestWatchdogStopsAnOrphanedDisabledSlot(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init() // slot 0 disabled; the fake status still reports hev=1 routing=1
	h.reset()
	h.mgr.Tick(context.Background())
	h.mustContain("S94exit status 0", "S94exit stop 0")
	h.mustNotContain("S94exit start")

	// Once the downstream is down, a disabled slot costs a status call only.
	h.runner.status = "forward=0\nrouting=0\ntun=0\nhev=0\nwstunnel=1\nnat=0\nnic=\ngw=\ndns_redirected=0\n"
	h.reset()
	h.mgr.Tick(context.Background())
	h.mustContain("S94exit status 0")
	h.mustNotContain("S94exit stop")
}

// gatedRunner parks one named call until released, so a test can hold a
// converge mid-flight while a transaction arrives.
type gatedRunner struct {
	inner   Runner
	call    string
	parked  chan struct{} // closed when the call is parked
	release chan struct{}
	once    sync.Once
}

func (g *gatedRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if filepath.Base(name)+" "+strings.Join(args, " ") == g.call {
		g.once.Do(func() { close(g.parked) })
		<-g.release
	}
	return g.inner.Run(ctx, name, args...)
}

// TestWatchdogYieldsToDisable: a tick that has passed the enabled check and
// is inside S94exit start must not re-converge the slot after Disable's
// S94exit stop, or hev, the rules and the chains come back on a slot
// recorded disabled and nothing ever stops them again.
func TestWatchdogYieldsToDisable(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	h.reset()

	gate := &gatedRunner{inner: h.runner, call: "S94exit start 0", parked: make(chan struct{}), release: make(chan struct{})}
	h.mgr.deps.Runner = gate
	h.mgr.down = downstream{run: gate}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.mgr.Tick(context.Background())
	}()
	select {
	case <-gate.parked:
	case <-time.After(3 * time.Second):
		t.Fatal("tick never reached S94exit start")
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := h.mgr.Disable(context.Background(), slot); err != nil {
			t.Errorf("disable: %v", err)
		}
	}()
	time.Sleep(50 * time.Millisecond) // let a non-yielding Disable run ahead of the parked tick
	close(gate.release)
	wg.Wait()

	stop := h.index("S94exit stop 0")
	if stop < 0 {
		t.Fatalf("no stop in:\n%s", strings.Join(h.trace, "\n"))
	}
	for i, line := range h.trace {
		if i > stop && (strings.Contains(line, "S94exit start") || strings.Contains(line, "S30rndis start") || strings.Contains(line, "dns.bind")) {
			t.Fatalf("%q re-converged the slot after its stop:\n%s", line, strings.Join(h.trace, "\n"))
		}
	}
	cfg, _, _ := LoadConfig(slot)
	if cfg.Enabled {
		t.Fatalf("config after disable = %+v", cfg)
	}
	if h.dns.bound {
		t.Fatal("forwarder left bound on a disabled slot")
	}
}

// TestWatchdogHealsASlotWithoutComponents: an enabled slot whose listeners
// failed to start (the SOCKS port held by a stale process at boot, or a mode
// switch whose restart failed) gets them retried by the watchdog like every
// other part of the slot, instead of staying enabled with hev pointed at a
// closed port until an operator toggles it.
func TestWatchdogHealsASlotWithoutComponents(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	h.door.startEr = errors.New("address in use")
	mode := proto.ExitModeWstunnel
	if err := h.mgr.SetConfig(context.Background(), slot, proto.SetExitConfigReq{Mode: &mode}); err == nil {
		t.Fatal("mode switch with a dead front door succeeded")
	}
	cfg, _, _ := LoadConfig(slot)
	if !cfg.Enabled {
		t.Fatalf("config after a failed mode switch = %+v", cfg)
	}
	h.mgr.mu.Lock()
	comps := h.mgr.slots["0"].comps
	h.mgr.mu.Unlock()
	if comps != nil {
		t.Fatal("components survived the failed start")
	}
	status, _ := h.mgr.Status(slot)
	if !strings.Contains(status.Message, "address in use") {
		t.Fatalf("status message = %q", status.Message)
	}

	// The port is free again: the next tick starts the listeners.
	h.door.startEr = nil
	h.reset()
	h.mgr.Tick(context.Background())
	h.mustContain("S94exit start 0", "door.start", "dns.bind 10.1.2.1", "probe.start", "S94exit status 0")
	h.mgr.mu.Lock()
	comps = h.mgr.slots["0"].comps
	h.mgr.mu.Unlock()
	if comps == nil {
		t.Fatal("watchdog did not start the components")
	}
	if h.door.relay == nil {
		t.Fatal("healed wstunnel slot has no relay gate")
	}
	status, _ = h.mgr.Status(slot)
	if status.Message != "" || !status.Downstream.DNS {
		t.Fatalf("status after heal = %+v", status)
	}

	// A tick with the listeners up builds nothing new.
	built := h.built
	h.mgr.Tick(context.Background())
	if h.built != built {
		t.Fatal("a healthy tick rebuilt the components")
	}
}

// TestOnAttachOutlivesTheAttachBudget: the router calls OnAttach with the
// startup budget's context, which may already be past its deadline by the
// time the post-attach converge runs; the converge must still run its
// scripts, or the consumer gets a NIC with no address and no lease until the
// first watchdog tick.
func TestOnAttachOutlivesTheAttachBudget(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	h.reset()
	expired, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-expired.Done()
	h.mgr.OnAttach(expired)
	h.mustContain("S94exit start 0", "S94exit status 0")
	status, _ := h.mgr.Status(slot)
	if !status.Downstream.Hev || !status.Downstream.Routing {
		t.Fatalf("downstream after an over-budget attach = %+v", status.Downstream)
	}
}

// TestModeBFirstPeerAndSupersedeAreRecorded: the proxy fires onPeerChange on
// the first Mode B upgrade (a zero prev) so the connected peer's source is
// known to the token-regenerate limiter reset (D11), and on a supersede with
// the dropped peer as prev so previousPeer/peerChangedAt are recorded (D12),
// which the old lastPeer-comparison path missed for Mode B (MGR-5).
func TestModeBFirstPeerAndSupersedeAreRecorded(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}

	// First peer A: no supersede, but the source is recorded.
	a := proto.ExitPeer{Addr: "203.0.113.7", Transport: proto.ExitModeWstunnel}
	h.deps.OnPeerChange(proto.ExitPeer{}, a)
	status, _ := h.mgr.Status(slot)
	if status.PreviousPeer != nil || status.PeerChangedAt != nil {
		t.Fatalf("first peer recorded a supersede: %+v", status)
	}
	h.mgr.limiter.Fail(SourceKey("203.0.113.7"))
	if _, err := h.mgr.RegenerateToken(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	if h.mgr.limiter.Len() != 0 {
		t.Fatal("the connected Mode B peer's limiter entry survived RegenerateToken")
	}

	// Supersede A -> B: previousPeer is A.
	h.now = h.now.Add(time.Minute)
	b := proto.ExitPeer{Addr: "198.51.100.9", Transport: proto.ExitModeWstunnel}
	h.deps.OnPeerChange(a, b)
	status, _ = h.mgr.Status(slot)
	if status.PreviousPeer == nil || status.PreviousPeer.Addr != "203.0.113.7" || status.PeerChangedAt == nil {
		t.Fatalf("supersede not recorded: %+v", status)
	}
}

// TestReconnectAfterRebootIsNotASupersede: the last peer coming back after a
// reboot, when only the superseded previousPeer was persisted, must not be
// logged and stamped as a fresh supersede (MGR-6).
func TestReconnectAfterRebootIsNotASupersede(t *testing.T) {
	h := newHarness(t)
	slot := MustSlot("0")
	a := proto.ExitPeer{Addr: "203.0.113.7:5000", Transport: proto.ExitModeNative}
	if err := SaveState(slot, State{PreviousPeer: &a}); err != nil {
		t.Fatal(err)
	}
	// A config so Init loads the slot.
	cfg, _ := DefaultConfig(slot)
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	h.mgr.Init()
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	b := &fakeBackend{peer: proto.ExitPeer{Addr: "198.51.100.9:6000", Hostname: "b", Transport: proto.ExitModeNative}}
	h.deps.OnSession(b)
	status, _ := h.mgr.Status(slot)
	if status.PeerChangedAt != nil {
		t.Fatalf("a reconnect after reboot was recorded as a supersede: %+v", status)
	}
}
