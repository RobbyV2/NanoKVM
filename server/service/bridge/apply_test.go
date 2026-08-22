package bridge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
)

// armedAt records, for every command the transaction issues, whether the
// dead-man marker was on disk at that instant. It is how the two ordering
// properties that are not visible in the command trace get asserted: that the
// marker was armed before the first mutation, and that step 13 runs after the
// disarm.
type armedAt struct {
	lines []string
	armed []bool
	snap  []bool
}

func (a *armedAt) watch(h *harness) {
	h.net.onCall = func(line string) {
		pending, _ := h.store.Pending()
		a.lines = append(a.lines, line)
		a.armed = append(a.armed, pending != nil)
		a.snap = append(a.snap, fileExists(h.store.SnapshotPath()))
	}
}

func (a *armedAt) at(t *testing.T, substr string) (armed, snap bool) {
	t.Helper()
	for i, line := range a.lines {
		if strings.Contains(line, substr) {
			return a.armed[i], a.snap[i]
		}
	}
	t.Fatalf("no command containing %q was issued", substr)
	return false, false
}

// requireOrder asserts that each command appears after the one before it. It
// asserts relative order rather than an exact transcript, so an extra read
// added later does not break every step-order test in the file.
func requireOrder(t *testing.T, trace []string, steps ...string) {
	t.Helper()

	previous := -1
	for _, step := range steps {
		at := indexOf(t, trace, step)
		if at <= previous {
			t.Fatalf("%q ran at %d, out of order.\ntrace:\n%s",
				step, at, strings.Join(trace, "\n"))
		}
		previous = at
	}
}

// ---------------------------------------------------------------------------
// enable
// ---------------------------------------------------------------------------

// TestEnableStepOrder walks the enable transaction's numbered steps. Several of
// the orderings are the whole design rather than tidiness:
//
//   - the snapshot is on disk and the marker armed before step 3's first
//     mutation, which is what makes a power cut at step 4 recoverable;
//   - the MAC is read and pinned before eth0 is enslaved, so the bridge never
//     holds an identity the DHCP reservation was not made against;
//   - udhcpc is killed before the flush, so no lease renewal re-adds an address
//     to a device that is about to become a port;
//   - l2-uplink is written before S30eth start, which reads it to decide which
//     device to address;
//   - step 13 runs after the disarm.
func TestEnableStepOrder(t *testing.T) {
	h := newHarness(t)
	h.mgr.gadget = fakeGadget{nic: GadgetName}

	var killed []int
	swap(t, &killProcess, func(pid int) error {
		killed = append(killed, pid)
		return nil
	})
	writeFile(t, udhcpcPidPath, "1234\n")

	observed := &armedAt{}
	observed.watch(h)

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if rsp.State != proto.BridgeEnabled {
		t.Fatalf("Enable returned %q (%s), want enabled", rsp.State, rsp.Message)
	}

	requireOrder(t, h.net.trace(),
		// Step 1. Snapshot: all three ip reads, before anything is touched.
		"ip -j link show",
		"ip -j addr show",
		"ip -j route show",
		// Step 3. br0, STP off, MAC pinned, up.
		"ip link add name br0 type bridge stp_state 0 forward_delay 0",
		"ip link set dev br0 address "+testMAC,
		"ip link set dev br0 up",
		// Step 5. eth0's address and route come off.
		"ip addr flush dev eth0",
		"ip route del default dev eth0",
		// Step 6. Enslave.
		"ip link set dev eth0 master br0",
		"ip link set dev eth0 up",
		// Step 8. Address br0 through S30eth, which reads the file step 7 wrote.
		"/etc/init.d/S30eth start",
		// Step 10. The five rules against br0, then the five naming eth0 go.
		"iptables -A INPUT -i br0",
		"iptables -D INPUT -i eth0",
		// Step 11. Verify.
		"ip -4 -j addr show dev br0",
		"ping -I br0",
		// Step 13. usb0, after everything above.
		"ip link set dev usb0 master br0",
	)

	// Step 2 before step 3: the marker and the snapshot are both on disk at the
	// instant of the first mutation.
	armed, snap := observed.at(t, "ip link add name br0")
	if !armed {
		t.Error("br0 was created before the dead-man was armed")
	}
	if !snap {
		t.Error("br0 was created before the snapshot reached disk")
	}

	// Step 4 before step 5.
	if len(killed) != 1 || killed[0] != 1234 {
		t.Errorf("udhcpc kills = %v, want [1234]", killed)
	}
	if _, err := os.Stat(udhcpcPidPath); !errors.Is(err, os.ErrNotExist) {
		t.Error("the udhcpc pidfile survived step 4")
	}

	// Step 7 before step 8: the fake's S30eth addresses whichever device
	// l2-uplink names, so br0 holding the lease is proof the file was written
	// first.
	if got := ReadUplink(); got != BridgeName {
		t.Errorf("uplink = %q, want br0", got)
	}
	if len(h.net.addrs[BridgeName]) == 0 {
		t.Error("S30eth addressed something other than br0, so l2-uplink was written late")
	}

	// Step 13 after step 12: no marker was armed when usb0 was enslaved.
	armed, _ = observed.at(t, "ip link set dev usb0 master br0")
	if armed {
		t.Error("usb0 was enslaved while the dead-man was still armed")
	}

	// The MAC pin survived the enslavement of a port with a lower address.
	if got := h.net.links[BridgeName].Address; got != testMAC {
		t.Errorf("br0 MAC = %q, want eth0's permanent %q", got, testMAC)
	}
}

// Step 13 failing must not leave usb0 enslaved but down, which is a silent
// black hole for the attached host's frames.
func TestGadgetIsReleasedIfItCannotBeBroughtUp(t *testing.T) {
	h := newHarness(t)
	h.mgr.gadget = fakeGadget{nic: GadgetName}
	h.net.failOn("link set dev usb0 up", errors.New("RTNETLINK answers: No such device"))

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}

	// The twelve steps that hold the management address still succeeded.
	if rsp.State != proto.BridgeEnabled {
		t.Fatalf("state = %q (%s), want enabled", rsp.State, rsp.Message)
	}
	if !strings.Contains(rsp.Message, "gadget NIC not enslaved") {
		t.Errorf("message %q does not report the gadget failure", rsp.Message)
	}
	if master := h.net.links[GadgetName].Master; master != "" {
		t.Fatalf("usb0 is enslaved to %q but was never brought up", master)
	}
}

// A device whose profile has no gadget NIC runs the whole transaction and
// simply skips step 13.
func TestEnableWithNoGadgetNIC(t *testing.T) {
	h := newHarness(t)
	h.mgr.gadget = fakeGadget{nic: ""}

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if rsp.State != proto.BridgeEnabled {
		t.Fatalf("state = %q (%s), want enabled", rsp.State, rsp.Message)
	}
	notInTrace(t, h.net.trace(), "usb0")
}

func TestEnableRecordsTheOutcomeBeforeReturning(t *testing.T) {
	h := newHarness(t)

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}

	// The caller losing its connection partway through is the expected failure
	// mode, so the outcome has to be readable from GET afterwards.
	lkg, err := h.store.LastKnownGood()
	if err != nil || lkg == nil {
		t.Fatalf("LastKnownGood = %v, %v", lkg, err)
	}
	if !lkg.Enabled || lkg.Uplink != BridgeName || lkg.State != proto.BridgeEnabled {
		t.Fatalf("recorded %+v, want an enabled br0", lkg)
	}
	if !lkg.Checks.Passed() {
		t.Fatalf("recorded checks %+v, want all three", lkg.Checks)
	}
	if pending, _ := h.store.Pending(); pending != nil {
		t.Fatal("the marker survived a successful enable")
	}
	if rsp.Checks != lkg.Checks {
		t.Fatalf("response checks %+v differ from the record %+v", rsp.Checks, lkg.Checks)
	}

	// And /etc/kvm/gateway is what an unpatched kvm_system reads verbatim.
	gateway, ok := ReadGateway()
	if !ok || strings.TrimSpace(gateway) != "192.168.1.1" {
		t.Fatalf("gateway file = %q, %v, want 192.168.1.1", gateway, ok)
	}
}

// TestEnableStaticPathReplaysRatherThanRerunningS30eth covers the reason the
// static branch exists: S30eth:55 appends a nameserver line rather than
// replacing one, so re-running the script accumulates a duplicate on every
// apply.
func TestEnableStaticPathReplaysRatherThanRerunningS30eth(t *testing.T) {
	h := newHarness(t)
	writeFile(t, noDHCPPath, "192.168.1.50/24 192.168.1.1\n")

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	trace := h.net.trace()
	notInTrace(t, trace, "/etc/init.d/S30eth")

	batch := trace[indexOf(t, trace, "ip -batch -")]
	for _, want := range []string{
		"addr add 192.168.1.50/24 brd + dev br0",
		"route add default via 192.168.1.1 dev br0",
	} {
		if !strings.Contains(batch, want) {
			t.Errorf("batch %q is missing %q", batch, want)
		}
	}

	// The captured address and route are now on br0, and resolv.conf still has
	// exactly one nameserver line.
	resolv, _ := readFile(resolvPath)
	if got := strings.Count(resolv, "nameserver 192.168.1.1"); got != 1 {
		t.Fatalf("resolv.conf has %d nameserver lines, want 1:\n%s", got, resolv)
	}
}

// ---------------------------------------------------------------------------
// the three verification gates
// ---------------------------------------------------------------------------

// TestVerificationGatesFailIndependently is the core of the rollback contract.
// Each of the three gates is failed on its own, with the other two satisfiable,
// and each one alone must prevent the disarm and trigger a restore.
//
// The third gate is the one that is easy to argue away and must not be: the
// device runs a DROP OUTPUT tcp --sport 8000 rule whose whole purpose is to
// make a perfectly reachable host refuse to answer on a port. A gateway that
// answers ICMP is not evidence that the management plane answers anyone.
func TestVerificationGatesFailIndependently(t *testing.T) {
	tests := []struct {
		name   string
		break_ func(h *harness)
		want   proto.BridgeChecks
		reason string
	}{
		{
			name:   "no IPv4 address on br0",
			break_: func(h *harness) { h.net.dhcpFails = true },
			want:   proto.BridgeChecks{},
			reason: "no IPv4 address on br0",
		},
		{
			name:   "no default route through br0",
			break_: func(h *harness) { h.net.dhcpGateway = "" },
			want:   proto.BridgeChecks{Address: true},
			reason: "no default route through br0",
		},
		{
			name:   "the gateway does not answer",
			break_: func(h *harness) { h.net.pingOK = false },
			want:   proto.BridgeChecks{Address: true},
			reason: "does not answer",
		},
		{
			name: "no inbound liveness proof",
			break_: func(h *harness) {
				h.live.observed = false
				h.live.selfConnects = false
			},
			want:   proto.BridgeChecks{Address: true, Gateway: true},
			reason: "management plane not reachable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			tc.break_(h)

			rsp, err := h.mgr.Enable(context.Background())
			if err != nil {
				t.Fatalf("Enable returned an error rather than a rollback: %v", err)
			}

			if rsp.State != proto.BridgeRolledBack {
				t.Fatalf("state = %q (%s), want rolledBack", rsp.State, rsp.Message)
			}
			if rsp.Checks != tc.want {
				t.Fatalf("checks = %+v, want %+v", rsp.Checks, tc.want)
			}
			if !strings.Contains(rsp.Message, tc.reason) {
				t.Errorf("message %q does not name the failing gate (%q)", rsp.Message, tc.reason)
			}

			// The restore ran: the device is back on stock eth0 with no bridge,
			// no uplink file, and no armed marker.
			if got := ReadUplink(); got != StockUplink {
				t.Errorf("uplink = %q after rollback, want eth0", got)
			}
			if _, err := os.Stat(uplinkPath); !errors.Is(err, os.ErrNotExist) {
				t.Error("the l2-uplink file survived the rollback")
			}
			if _, exists := h.net.links[BridgeName]; exists {
				t.Error("br0 survived the rollback")
			}
			if master := h.net.links[StockUplink].Master; master != "" {
				t.Errorf("eth0 is still enslaved to %q after the rollback", master)
			}
			if pending, _ := h.store.Pending(); pending != nil {
				t.Error("the marker survived the rollback")
			}

			lkg, err := h.store.LastKnownGood()
			if err != nil || lkg == nil {
				t.Fatalf("LastKnownGood = %v, %v", lkg, err)
			}
			if lkg.Enabled {
				t.Error("a rolled-back enable recorded the bridge as enabled")
			}
			if lkg.State != proto.BridgeRolledBack {
				t.Errorf("recorded state %q, want rolledBack", lkg.State)
			}
			if lkg.Checks != tc.want {
				t.Errorf("recorded checks %+v, want %+v", lkg.Checks, tc.want)
			}
		})
	}
}

// The self-connect is the weaker proof and has to be recorded as such, so an
// operator reading the outcome knows whether a real client completed a round
// trip or only the local delivery path was exercised.
func TestInboundFallsBackToSelfConnect(t *testing.T) {
	h := newHarness(t)
	h.live.observed = false
	h.live.selfConnects = true

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if rsp.State != proto.BridgeEnabled {
		t.Fatalf("state = %q (%s), want enabled", rsp.State, rsp.Message)
	}
	if !rsp.Checks.Inbound || !rsp.Checks.InboundWeak {
		t.Fatalf("checks = %+v, want Inbound with InboundWeak", rsp.Checks)
	}
	if h.live.calls != 1 {
		t.Fatalf("SelfConnect ran %d times, want 1", h.live.calls)
	}
}

// A real inbound request must not be downgraded to the weak proof.
func TestObservedInboundIsNotWeak(t *testing.T) {
	h := newHarness(t)
	h.live.observed = true

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !rsp.Checks.Inbound || rsp.Checks.InboundWeak {
		t.Fatalf("checks = %+v, want a strong Inbound", rsp.Checks)
	}
	if h.live.calls != 0 {
		t.Fatalf("SelfConnect ran %d times despite a real observation", h.live.calls)
	}
}

// A gate that passes is not enough on its own: all three are required, so a
// device with an address and a live gateway but a dead management plane still
// rolls back. This is the case the DROP rule makes real.
func TestAllThreeGatesAreRequired(t *testing.T) {
	h := newHarness(t)
	h.live.observed = false
	h.live.selfConnects = false

	rsp, _ := h.mgr.Enable(context.Background())
	if rsp.Checks.Address != true || rsp.Checks.Gateway != true {
		t.Fatalf("checks = %+v, want the first two gates passing", rsp.Checks)
	}
	if rsp.State != proto.BridgeRolledBack {
		t.Fatalf("state = %q, want rolledBack with two of three gates passing", rsp.State)
	}
}

// ---------------------------------------------------------------------------
// mid-transaction failures
// ---------------------------------------------------------------------------

// Every mutating step must roll back, not just the verification. The table
// walks the steps that can plausibly fail on a device that has never run any of
// this before, which is R3.1 in the design's risk register.
func TestMidTransactionFailuresRollBack(t *testing.T) {
	tests := []struct {
		name string
		fail string
		want proto.BridgeState
	}{
		{"step 3, bridge creation", "link add name br0", proto.BridgeRolledBack},
		{"step 3, the MAC pin", "link set dev br0 address", proto.BridgeRolledBack},
		{"step 5, the flush", "addr flush dev eth0", proto.BridgeRolledBack},
		{"step 6, enslavement", "link set dev eth0 master br0", proto.BridgeRolledBack},

		// S30eth is the one step the restore itself depends on, so a device
		// where it cannot run is genuinely unrecoverable from here and the
		// outcome says so rather than claiming a clean rollback.
		{"step 8, addressing", "/etc/init.d/S30eth start", proto.BridgeFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.net.failOn(tc.fail, errors.New("RTNETLINK answers: Operation not supported"))

			rsp, err := h.mgr.Enable(context.Background())
			if err != nil {
				t.Fatalf("Enable returned an error rather than a rollback: %v", err)
			}
			if rsp.State != tc.want {
				t.Fatalf("state = %q (%s), want %q", rsp.State, rsp.Message, tc.want)
			}
			if got := ReadUplink(); got != StockUplink {
				t.Errorf("uplink = %q after rollback, want eth0", got)
			}
			if pending, _ := h.store.Pending(); pending != nil {
				t.Error("the marker survived the rollback")
			}
		})
	}
}

// A rollback that cannot put the device back must say so rather than reporting
// a clean restore: that distinction is what tells an operator to reach for the
// wlan0 AP or the serial console.
func TestFailedRestoreIsReportedAsFailedNotRolledBack(t *testing.T) {
	h := newHarness(t)
	h.net.failOn("link set dev eth0 nomaster", errors.New("Device or resource busy"))
	h.live.observed = false
	h.live.selfConnects = false

	rsp, _ := h.mgr.Enable(context.Background())
	if rsp.State != proto.BridgeFailed {
		t.Fatalf("state = %q (%s), want failed", rsp.State, rsp.Message)
	}
	if !strings.Contains(rsp.Message, "rollback incomplete") {
		t.Errorf("message %q does not say the rollback was incomplete", rsp.Message)
	}
}

// ---------------------------------------------------------------------------
// disable
// ---------------------------------------------------------------------------

// enabled puts the harness into the post-enable state, so the disable
// transaction has something to undo.
func enabled(t *testing.T, h *harness) {
	t.Helper()

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if err := h.mgr.ip.SetMaster(context.Background(), GadgetName, BridgeName); err != nil {
		t.Fatalf("enslave usb0: %v", err)
	}

	h.net.mu.Lock()
	h.net.calls = nil
	h.net.mu.Unlock()
}

// TestDisableStepOrder walks the disable transaction. Step 13's placement is
// the load-bearing one: br0 is torn down after the disarm, so a verification
// that fails restores onto a bridge that still exists.
func TestDisableStepOrder(t *testing.T) {
	h := newHarness(t)
	enabled(t, h)

	observed := &armedAt{}
	observed.watch(h)

	rsp, err := h.mgr.Disable(context.Background())
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if rsp.State != proto.BridgeDisabled {
		t.Fatalf("Disable returned %q (%s), want disabled", rsp.State, rsp.Message)
	}

	requireOrder(t, h.net.trace(),
		// Step 1. Snapshot.
		"ip -j link show",
		"ip -j addr show",
		"ip -j route show",
		// Step 3. usb0 first.
		"ip link set dev usb0 nomaster",
		// Step 5. br0's address and route come off.
		"ip addr flush dev br0",
		"ip route del default dev br0",
		// Step 6. Release eth0.
		"ip link set dev eth0 nomaster",
		"ip link set dev eth0 up",
		// Step 8. Address eth0, through the file step 7 removed.
		"/etc/init.d/S30eth start",
		// Step 10. The five rules against eth0, then the five naming br0 go.
		"iptables -A INPUT -i eth0",
		"iptables -D INPUT -i br0",
		// Step 11. Verify against eth0, not br0.
		"ip -4 -j addr show dev eth0",
		"ping -I eth0",
		// Step 13. Teardown, last.
		"ip link set dev br0 down",
		"ip link delete br0",
	)

	// Step 13 after step 12.
	armed, _ := observed.at(t, "ip link delete br0")
	if armed {
		t.Error("br0 was deleted while the dead-man was still armed")
	}

	// Step 7: the file is gone, so every reader falls back to eth0 and the
	// on-disk state matches a device that never had the bridge.
	if _, err := os.Stat(uplinkPath); !errors.Is(err, os.ErrNotExist) {
		t.Error("the l2-uplink file survived a disable")
	}
	if got := ReadUplink(); got != StockUplink {
		t.Errorf("uplink = %q, want eth0", got)
	}
	if _, exists := h.net.links[BridgeName]; exists {
		t.Error("br0 survived the disable")
	}

	lkg, _ := h.store.LastKnownGood()
	if lkg == nil || lkg.Enabled || lkg.State != proto.BridgeDisabled {
		t.Fatalf("recorded %+v, want a disabled eth0", lkg)
	}
}

// A disable whose verification fails must leave br0 in place, because the
// restore puts the address back onto it.
func TestDisableRollbackLeavesTheBridgeInPlace(t *testing.T) {
	h := newHarness(t)
	enabled(t, h)

	h.live.observed = false
	h.live.selfConnects = false

	rsp, err := h.mgr.Disable(context.Background())
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if rsp.State != proto.BridgeRolledBack {
		t.Fatalf("state = %q (%s), want rolledBack", rsp.State, rsp.Message)
	}

	if _, exists := h.net.links[BridgeName]; !exists {
		t.Fatal("br0 was torn down by a rollback, leaving nothing to restore onto")
	}
	if got := ReadUplink(); got != BridgeName {
		t.Fatalf("uplink = %q after a rolled-back disable, want br0", got)
	}
	notInTrace(t, h.net.trace(), "ip link delete br0")
}

// ---------------------------------------------------------------------------
// serialisation and revert
// ---------------------------------------------------------------------------

func TestTransactionsDoNotInterleave(t *testing.T) {
	h := newHarness(t)

	if err := h.mgr.lock(); err != nil {
		t.Fatalf("lock: %v", err)
	}
	defer h.mgr.unlock()

	if _, err := h.mgr.Enable(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatalf("Enable during another apply = %v, want ErrBusy", err)
	}
	if _, err := h.mgr.Disable(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatalf("Disable during another apply = %v, want ErrBusy", err)
	}
	if len(h.net.trace()) != 0 {
		t.Fatalf("a refused transaction still ran %v", h.net.trace())
	}
}

// Revert exists for the case where all three gates passed and the operator
// still cannot reach the device from somewhere the verification did not test.
func TestRevertRestoresTheLastSnapshot(t *testing.T) {
	h := newHarness(t)

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if got := ReadUplink(); got != BridgeName {
		t.Fatalf("uplink = %q after enable, want br0", got)
	}

	if err := h.mgr.Revert(context.Background()); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	if got := ReadUplink(); got != StockUplink {
		t.Fatalf("uplink = %q after revert, want eth0", got)
	}
	if _, exists := h.net.links[BridgeName]; exists {
		t.Fatal("br0 survived the revert")
	}
}

func TestRevertWithNoSnapshot(t *testing.T) {
	h := newHarness(t)

	if err := h.mgr.Revert(context.Background()); !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("Revert with no snapshot = %v, want ErrNoSnapshot", err)
	}
	_ = filepath.Join(h.root, "unused")
}

func TestStatusReportsTheLiveDevice(t *testing.T) {
	h := newHarness(t)

	status, err := h.mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Exists || status.Uplink != StockUplink || status.State != proto.BridgeDisabled {
		t.Fatalf("stock device reports %+v", status)
	}

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	status, err = h.mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Exists || status.Uplink != BridgeName || status.State != proto.BridgeEnabled {
		t.Fatalf("enabled device reports %+v", status)
	}
	if status.MAC != testMAC {
		t.Errorf("MAC = %q, want the pinned %q", status.MAC, testMAC)
	}
	if len(status.Ports) != 1 || status.Ports[0].Name != StockUplink {
		t.Errorf("ports = %+v, want just eth0", status.Ports)
	}
	if status.Gateway != "192.168.1.1" {
		t.Errorf("gateway = %q", status.Gateway)
	}
	if status.LastApply == nil || !status.LastApply.Enabled {
		t.Errorf("lastApply = %+v", status.LastApply)
	}
}
