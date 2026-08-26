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
	h.mgr.gadget = &fakeGadget{nic: StockGadgetName}

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
	h.mgr.gadget = &fakeGadget{nic: StockGadgetName}
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
	if master := h.net.links[StockGadgetName].Master; master != "" {
		t.Fatalf("usb0 is enslaved to %q but was never brought up", master)
	}
}

// A device whose profile has no gadget NIC runs the whole transaction and
// simply skips step 13.
func TestEnableWithNoGadgetNIC(t *testing.T) {
	h := newHarness(t)
	h.mgr.gadget = &fakeGadget{nic: ""}

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if rsp.State != proto.BridgeEnabled {
		t.Fatalf("state = %q (%s), want enabled", rsp.State, rsp.Message)
	}
	notInTrace(t, h.net.trace(), "usb0")
}

// A presentation apply unbinds the UDC and binds it again, and the kernel hands
// back a usb0 that has never heard of br0. Step 13 runs once, inside Enable, so
// without the rebind hook a two-port transparent bridge silently drops to one
// port on the first profile change after it was enabled, and stays there.
func TestAPresentationApplyReEnslavesTheGadget(t *testing.T) {
	gadget := &fakeGadget{nic: StockGadgetName}
	h := newHarness(t, gadget)

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if gadget.rebound == nil {
		t.Fatal("the bridge registered no rebind hook, so nothing tells it usb0 was rebuilt")
	}

	// What an apply leaves behind: a fresh interface, no master, down.
	h.net.links[StockGadgetName].Master = ""
	h.net.links[StockGadgetName].Flags = nil

	gadget.rebound(context.Background())

	if master := h.net.links[StockGadgetName].Master; master != BridgeName {
		t.Fatalf("usb0 master = %q after a presentation apply, want br0", master)
	}
	if !h.net.links[StockGadgetName].Up() {
		t.Fatal("usb0 is enslaved but down, which is a black hole for the host's frames")
	}

	// The uplink half is untouched: re-enslaving a port must never reach the
	// device carrying the management address.
	if master := h.net.links[StockUplink].Master; master != BridgeName {
		t.Fatalf("eth0 master = %q, want br0", master)
	}
	if got := ReadUplink(); got != BridgeName {
		t.Fatalf("uplink = %q, want br0", got)
	}
	if got := h.net.links[BridgeName].Address; got != testMAC {
		t.Fatalf("br0 MAC = %q, want eth0's permanent %q", got, testMAC)
	}
	if len(h.net.addrs[BridgeName]) == 0 {
		t.Fatal("br0 lost its address")
	}
}

// Firing twice is what a second apply does, and the second one must be as
// harmless as the first.
func TestReattachingTheGadgetIsIdempotent(t *testing.T) {
	gadget := &fakeGadget{nic: StockGadgetName}
	h := newHarness(t, gadget)

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	h.mgr.ReattachGadget(context.Background())
	h.mgr.ReattachGadget(context.Background())

	if master := h.net.links[StockGadgetName].Master; master != BridgeName {
		t.Fatalf("usb0 master = %q, want br0", master)
	}
}

// Bridge disabled, or never enabled, which is every stock device: the hook fires
// on every apply there too and must not create a port on a bridge that does not
// exist. Nor must a profile with no network function turn into an enslavement.
func TestReattachingTheGadgetWithoutABridgeDoesNothing(t *testing.T) {
	t.Run("no br0", func(t *testing.T) {
		gadget := &fakeGadget{nic: StockGadgetName}
		h := newHarness(t, gadget)

		h.mgr.ReattachGadget(context.Background())
		notInTrace(t, h.net.trace(), "master br0")
	})

	t.Run("no gadget NIC", func(t *testing.T) {
		gadget := &fakeGadget{nic: ""}
		h := newHarness(t, gadget)

		if _, err := h.mgr.Enable(context.Background()); err != nil {
			t.Fatalf("Enable: %v", err)
		}
		h.mgr.ReattachGadget(context.Background())
		notInTrace(t, h.net.trace(), "usb0")
	})
}

// The gadget NIC is not always usb0, and nothing on this side may assume it is.
//
// Applying a profile unlinks the outgoing net function from configs/c.1, but
// unlinking does not destroy an f_ncm or f_rndis: only rmdir does, and
// gether_setup holds the netdev until then. So an orphaned rndis.usb0 keeps the
// name usb0 with NO-CARRIER, and the kernel's "usb%d" allocation hands the new
// ncm.usb0 the next free name, usb1. The presentation manager reads the linked
// function's configfs ifname and reports usb1; a bridge that enslaves the
// literal usb0 puts a dead interface into br0 and the attached host's frames go
// nowhere, with nothing in the status output to say so.
func TestGadgetPortFollowsTheRenamedNIC(t *testing.T) {
	// The device this exists for: an orphan still holding usb0 and the live
	// function up as usb1.
	orphaned := func(t *testing.T) *harness {
		t.Helper()

		h := newHarness(t, &fakeGadget{nic: "usb1", protocol: "ncm"})
		h.net.links["usb0"].Flags = nil
		h.net.links["usb1"] = &Link{
			Index: 5, Name: "usb1", Flags: []string{"UP"}, Address: "48:da:35:6e:11:22",
		}
		return h
	}

	t.Run("step 13 enslaves the live NIC and not the orphan", func(t *testing.T) {
		h := orphaned(t)

		rsp, err := h.mgr.Enable(context.Background())
		if err != nil {
			t.Fatalf("Enable: %v", err)
		}
		if master := h.net.links["usb1"].Master; master != BridgeName {
			t.Fatalf("usb1 master = %q, want br0 (%s)", master, rsp.Message)
		}
		if !h.net.links["usb1"].Up() {
			t.Fatal("usb1 is enslaved but down, a black hole for the host's frames")
		}
		if master := h.net.links["usb0"].Master; master != "" {
			t.Fatalf("the orphaned usb0 was enslaved to %q", master)
		}
	})

	t.Run("a rebind re-enslaves the live NIC", func(t *testing.T) {
		gadget := &fakeGadget{nic: "usb1", protocol: "ncm"}
		h := newHarness(t, gadget)
		h.net.links["usb0"].Flags = nil
		h.net.links["usb1"] = &Link{
			Index: 5, Name: "usb1", Flags: []string{"UP"}, Address: "48:da:35:6e:11:22",
		}

		if _, err := h.mgr.Enable(context.Background()); err != nil {
			t.Fatalf("Enable: %v", err)
		}
		h.net.links["usb1"].Master = ""
		h.net.links["usb1"].Flags = nil

		if gadget.rebound == nil {
			t.Fatal("the bridge registered no rebind hook")
		}
		gadget.rebound(context.Background())

		if master := h.net.links["usb1"].Master; master != BridgeName {
			t.Fatalf("usb1 master = %q after a presentation apply, want br0", master)
		}
	})

	// The release half. A disable that only ever names usb0 leaves the real
	// port enslaved to a bridge it is about to delete.
	t.Run("disable releases the gadget port the capture names", func(t *testing.T) {
		h := orphaned(t)

		if _, err := h.mgr.Enable(context.Background()); err != nil {
			t.Fatalf("Enable: %v", err)
		}
		// Set directly, so this subtest fails for the release half rather than
		// inheriting the enslave half's outcome.
		h.net.links["usb1"].Master = BridgeName

		if _, err := h.mgr.Disable(context.Background()); err != nil {
			t.Fatalf("Disable: %v", err)
		}
		if master := h.net.links["usb1"].Master; master != "" {
			t.Fatalf("usb1 is still enslaved to %q after a disable", master)
		}
	})

	// A restore runs against a device whose gadget NIC may have been renamed
	// since the capture, so it has to release the port br0 actually holds.
	t.Run("a rollback releases the gadget port the capture never saw", func(t *testing.T) {
		h := orphaned(t)
		h.live.observed = false
		h.live.selfConnects = false

		// The rollback runs after step 13 has not yet happened, so arrange the
		// port the way a dead-man restore finds it: enslaved under a name the
		// snapshot has no record of.
		h.net.onCall = func(line string) {
			if strings.Contains(line, "link set dev eth0 master br0") {
				h.net.links["usb1"].Master = BridgeName
			}
		}

		rsp, _ := h.mgr.Enable(context.Background())
		if rsp.State != proto.BridgeRolledBack {
			t.Fatalf("state = %q (%s), want rolled back", rsp.State, rsp.Message)
		}
		if master := h.net.links["usb1"].Master; master != "" {
			t.Fatalf("usb1 is still enslaved to %q after a rollback", master)
		}
	})
}

// The presentation manager being unable to say what the gadget NIC is says
// nothing about the management address, which is already up and verified by the
// time step 13 asks. The transaction reports the failure and stands.
func TestEnableSurvivesAGadgetThatCannotReportItsNIC(t *testing.T) {
	h := newHarness(t)
	h.mgr.gadget = &fakeGadget{err: errors.New("usb gadget unavailable")}

	rsp, err := h.mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if rsp.State != proto.BridgeEnabled {
		t.Fatalf("state = %q (%s), want enabled", rsp.State, rsp.Message)
	}
	if !strings.Contains(rsp.Message, "usb gadget unavailable") {
		t.Errorf("message %q does not report the gadget failure", rsp.Message)
	}

	lkg, err := h.store.LastKnownGood()
	if err != nil || lkg == nil || lkg.State != proto.BridgeEnabled {
		t.Fatalf("recorded %+v, %v, want an enabled br0", lkg, err)
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

// TestEnableRefusesAnUplinkWithNoCarrier is the preflight. The dead-man makes a
// doomed enable survivable; it does not make it free. Between step 5's flush and
// the rollback the device holds no address at all, and an uplink with no cable
// is the one case where that is known in advance. The refusal has to happen
// before the first write, so a refused enable leaves no snapshot, no marker and
// no br0 rather than a rollback that has to put all three back.
func TestEnableRefusesAnUplinkWithNoCarrier(t *testing.T) {
	tests := []struct {
		name   string
		break_ func(h *harness)
		reason string
	}{
		{
			name:   "no carrier",
			break_: func(h *harness) { h.net.links[StockUplink].Flags = []string{"UP"} },
			reason: "eth0 has no carrier",
		},
		{
			name:   "no eth0 at all",
			break_: func(h *harness) { delete(h.net.links, StockUplink) },
			reason: "eth0 is not present",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness(t)
			test.break_(h)

			rsp, err := h.mgr.Enable(context.Background())
			if !errors.Is(err, ErrPreflight) {
				t.Fatalf("Enable = %v, want ErrPreflight", err)
			}
			if !strings.Contains(rsp.Message, test.reason) {
				t.Errorf("message = %q, want it to name %q", rsp.Message, test.reason)
			}
			if rsp.State != proto.BridgeDisabled {
				t.Errorf("state = %q, want disabled", rsp.State)
			}

			// Nothing was written and nothing was changed: the read that
			// refused is the only command that ran.
			trace := h.net.trace()
			if len(trace) != 1 || trace[0] != "ip -j link show" {
				t.Fatalf("a refused enable ran %v", trace)
			}
			if fileExists(h.store.SnapshotPath()) {
				t.Error("a refused enable wrote a snapshot")
			}
			pending, err := h.store.Pending()
			if err != nil || pending != nil {
				t.Errorf("a refused enable left a marker: %+v, %v", pending, err)
			}
		})
	}
}

// TestEnableKeepsALeaseUnderTheStaticBranch is the other half of the static
// replay. /boot/eth.nodhcp says which branch S30eth took, not that the branch
// produced the address: its static assignment gives up on an arping conflict at
// S30eth:55 and falls through to udhcpc at :63. Replaying that lease onto br0 as
// a static address, with step 4 having killed the only client that renews it,
// leaves the device holding an address the server hands to somebody else.
func TestEnableKeepsALeaseUnderTheStaticBranch(t *testing.T) {
	h := newHarness(t)
	writeFile(t, noDHCPPath, "192.168.1.50/24 192.168.1.1\n")
	writeFile(t, udhcpcPidPath, "1234\n")
	swap(t, &processAlive, func(int) bool { return true })
	swap(t, &killProcess, func(int) error { return nil })

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	trace := h.net.trace()
	indexOf(t, trace, "/etc/init.d/S30eth start")
	notInTrace(t, trace, "ip -batch -")

	// The fall-through never reaches the nameserver append at S30eth:58, so the
	// duplicate the static replay exists to avoid is still not there.
	resolv, _ := readFile(resolvPath)
	if got := strings.Count(resolv, "nameserver 192.168.1.1"); got != 1 {
		t.Fatalf("resolv.conf has %d nameserver lines, want 1:\n%s", got, resolv)
	}
}

// A pidfile a dead client left behind is not a lease. Reading one as a lease
// would send a genuinely static device through S30eth, whose nameserver append
// is exactly what the static replay exists to avoid.
func TestEnableStaticPathIgnoresAStalePidfile(t *testing.T) {
	h := newHarness(t)
	writeFile(t, noDHCPPath, "192.168.1.50/24 192.168.1.1\n")
	writeFile(t, udhcpcPidPath, "1234\n")
	swap(t, &processAlive, func(int) bool { return false })

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	trace := h.net.trace()
	notInTrace(t, trace, "/etc/init.d/S30eth")
	indexOf(t, trace, "ip -batch -")
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
	if err := h.mgr.ip.SetMaster(context.Background(), StockGadgetName, BridgeName); err != nil {
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

// The panel names the protocol the gadget is presenting and offers no control
// for it, because the choice decides what the gadget looks like whether or not
// a bridge exists and therefore belongs to the USB profile. Reporting nothing
// leaves an operator with no way to tell an NCM host from an RNDIS one.
func TestStatusReportsTheActiveGadgetProtocol(t *testing.T) {
	tests := []struct {
		name   string
		gadget Gadget
		want   string
	}{
		{name: "ncm", gadget: &fakeGadget{nic: StockGadgetName, protocol: "ncm"}, want: "ncm"},
		{name: "rndis", gadget: &fakeGadget{nic: StockGadgetName, protocol: "rndis"}, want: "rndis"},
		{name: "no network function", gadget: &fakeGadget{}, want: ""},
		{name: "no gadget at all", gadget: nil, want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness(t)
			h.mgr.gadget = test.gadget

			status, err := h.mgr.Status(context.Background())
			if err != nil {
				t.Fatalf("Status: %v", err)
			}
			if status.Protocol != test.want {
				t.Fatalf("protocol = %q, want %q", status.Protocol, test.want)
			}
		})
	}
}

// TestStatusReportsLinkState is the reporting half of the preflight. A bridge
// whose uplink lost its cable looks identical in every other field to one that
// is working, and an operator reading "enabled, br0, 192.168.1.50" has no way
// to tell the difference. Carrier is per port as well, because a two-port
// bridge with a dead gadget port is a working device with a host that cannot
// reach the network.
func TestStatusReportsLinkState(t *testing.T) {
	h := newHarness(t, &fakeGadget{nic: StockGadgetName})

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	status, err := h.mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Carrier {
		t.Error("a working bridge reports no carrier on its uplink")
	}
	for _, port := range status.Ports {
		if !port.Carrier {
			t.Errorf("port %s reports no carrier", port.Name)
		}
	}

	// The cable comes out of both.
	h.net.links[BridgeName].Flags = []string{"UP"}
	h.net.links[StockUplink].Flags = []string{"UP"}

	status, err = h.mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Carrier {
		t.Error("br0 reports carrier with no LOWER_UP")
	}
	if len(status.Ports) != 2 {
		t.Fatalf("ports = %+v, want eth0 and usb0", status.Ports)
	}
	for _, port := range status.Ports {
		if port.Carrier != (port.Name == StockGadgetName) {
			t.Errorf("port %+v carrier is wrong", port)
		}
		if !port.Up {
			t.Errorf("port %+v is administratively up and reported down", port)
		}
	}
}

// TestStatusWarnsAboutASecondPathToTheSegment is the compensating control for
// STP being off. Nothing in the kernel breaks a loop here, so the one thing the
// device can do is say that it sees one: the gateway is on the LAN by
// definition, and its address being learned on the gadget port means the
// attached host is a second way onto the same segment.
func TestStatusWarnsAboutASecondPathToTheSegment(t *testing.T) {
	const gatewayMAC = "aa:bb:cc:00:11:22"

	tests := []struct {
		name string
		fdb  []FDBEntry
		want string
	}{
		{
			name: "the gateway is learned on the gadget port",
			fdb: []FDBEntry{
				{MAC: gatewayMAC, Ifname: StockGadgetName, Master: BridgeName},
			},
			want: StockGadgetName,
		},
		{
			// Where it belongs. This is every correctly wired device, and a
			// warning here would be one nobody could act on.
			name: "the gateway is learned on the uplink port",
			fdb: []FDBEntry{
				{MAC: gatewayMAC, Ifname: StockUplink, Master: BridgeName},
			},
		},
		{
			// The host's own address on the gadget port, which is what a
			// working gadget NIC looks like.
			name: "some other address on the gadget port",
			fdb: []FDBEntry{
				{MAC: "48:da:35:6e:11:33", Ifname: StockGadgetName, Master: BridgeName},
			},
		},
		{
			// The permanent entry the kernel writes for the port's own address
			// says nothing about where a frame came from.
			name: "a permanent entry is not a learned one",
			fdb: []FDBEntry{
				{MAC: gatewayMAC, Ifname: StockGadgetName, Master: BridgeName, State: "permanent"},
			},
		},
		{
			name: "nothing learned at all",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness(t, &fakeGadget{nic: StockGadgetName})
			h.net.neighbours = []Neigh{{Dst: "192.168.1.1", Dev: BridgeName, LLAddr: gatewayMAC}}
			h.net.fdb = test.fdb

			if _, err := h.mgr.Enable(context.Background()); err != nil {
				t.Fatalf("Enable: %v", err)
			}

			status, err := h.mgr.Status(context.Background())
			if err != nil {
				t.Fatalf("Status: %v", err)
			}

			if test.want == "" {
				if status.Loop != nil {
					t.Fatalf("loop reported where there is none: %+v", status.Loop)
				}
				return
			}
			if status.Loop == nil {
				t.Fatal("a second path to the uplink segment was not reported")
			}
			if status.Loop.Port != test.want || status.Loop.MAC != gatewayMAC {
				t.Fatalf("loop = %+v, want the gateway on %s", status.Loop, test.want)
			}
			if !strings.Contains(status.Loop.Reason, "STP is off") {
				t.Errorf("reason = %q, want it to name the missing spanning tree", status.Loop.Reason)
			}
		})
	}
}

// A one-port bridge cannot have a second path, and reading the neighbour table
// and the forwarding database on every poll to prove it is work for nothing.
func TestStatusSkipsTheLoopCheckWithOnePort(t *testing.T) {
	h := newHarness(t)

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	status, err := h.mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Loop != nil {
		t.Fatalf("loop reported on a one-port bridge: %+v", status.Loop)
	}
	notInTrace(t, h.net.trace(), "neigh show")
	notInTrace(t, h.net.trace(), "fdb show")
}

// At boot the presentation manager's startup refresh is the only rebind
// notification, and it fires while the routers are still being constructed -
// before this manager exists, so nothing is listening for it. S29bridge does
// not cover the gap either: it builds br0 about seven seconds before the gadget
// NIC exists and treats its absence as success. The result was a bridge with no
// gadget port on every boot, and an attached host whose USB network adapter
// enumerated but carried no traffic. Constructing the manager must therefore
// reattach once on its own, not merely register for rebinds still to come.
func TestConstructingTheManagerEnslavesAGadgetAlreadyPresent(t *testing.T) {
	gadget := &fakeGadget{nic: StockGadgetName}
	h := newHarness(t, gadget)

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	// The state S29bridge leaves behind: br0 up, gadget NIC outside it.
	h.net.mu.Lock()
	h.net.links[StockGadgetName].Master = ""
	h.net.mu.Unlock()

	// A second manager over the same network is this constructor running after
	// the notification has already been missed.
	New(Config{
		Commander: h.net,
		Liveness:  h.live,
		Store:     h.store,
		Window:    DefaultWindow,
		Gadget:    gadget,
	})

	h.net.mu.Lock()
	master := h.net.links[StockGadgetName].Master
	h.net.mu.Unlock()
	if master != BridgeName {
		t.Fatalf("usb0 master = %q after constructing the manager, want %q", master, BridgeName)
	}
}
