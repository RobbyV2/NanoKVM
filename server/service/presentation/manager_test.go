package presentation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// NIC is the presentation half of the bridge's step 13. Without it the bridge's
// Gadget is nil in production, step 13 never runs, and br0 comes up with eth0 as
// its only port.
func TestNICReportsTheGadgetInterfaceOnlyWhenOneIsLinked(t *testing.T) {
	tests := []struct {
		name   string
		linked string
		want   string
	}{
		{name: "ncm linked", linked: "ncm.usb0", want: GadgetNIC},
		{name: "rndis linked", linked: "rndis.usb0", want: GadgetNIC},
		{name: "no network function", linked: "", want: ""},
		// The function exists but is not in configs/c.1, so the kernel never
		// bound it and there is no usb0 to enslave.
		{name: "present but unlinked", linked: "-", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, ops := newTestManager(t)

			switch test.linked {
			case "":
			case "-":
				seed(t, ops, functionsDir+"/ncm.usb0/dev_addr")
			default:
				seed(t, ops, functionsDir+"/"+test.linked+"/dev_addr")
				seed(t, ops, configPrefix+"/"+test.linked+"/dev_addr")
			}

			nic, err := manager.NIC(context.Background())
			if err != nil {
				t.Fatalf("NIC: %v", err)
			}
			if nic != test.want {
				t.Fatalf("NIC = %q, want %q", nic, test.want)
			}
		})
	}
}

// Which of the two the gadget presents is read back from the linkage, so the
// bridge panel and the Settings, Device selector agree with the gadget rather
// than with a /boot sentinel.
func TestNetworkProtocolNamesTheLinkedFunction(t *testing.T) {
	tests := []struct {
		name   string
		linked string
		want   string
	}{
		{name: "ncm", linked: "ncm.usb0", want: string(FunctionNCM)},
		{name: "rndis", linked: "rndis.usb0", want: string(FunctionRNDIS)},
		{name: "none", linked: "", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, ops := newTestManager(t)
			if test.linked != "" {
				seed(t, ops, functionsDir+"/"+test.linked+"/dev_addr")
				seed(t, ops, configPrefix+"/"+test.linked+"/dev_addr")
			}

			protocol, err := manager.NetworkProtocol(context.Background())
			if err != nil {
				t.Fatalf("NetworkProtocol: %v", err)
			}
			if protocol != test.want {
				t.Fatalf("NetworkProtocol = %q, want %q", protocol, test.want)
			}
		})
	}
}

func TestNICPropagatesAnUnreadableGadget(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.ops = nil

	if _, err := manager.NIC(context.Background()); err == nil {
		t.Fatal("NIC on a manager with no gadget returned no error")
	}
}

// NIC alone is not enough. The bridge enslaves usb0 once, and every apply after
// that unbinds the UDC and binds it again, which destroys and recreates the
// interface with no memory of br0. Without this notification a two-port
// transparent bridge quietly drops to one port on the next profile change.
func TestEveryGadgetRebindNotifiesTheHook(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	var calls int
	manager.OnRebind(func(context.Context) { calls++ })

	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}
	if calls != 1 {
		t.Fatalf("one apply fired the rebind hook %d times, want 1", calls)
	}

	if err := manager.Rebind(ctx); err != nil {
		t.Fatalf("rebind: %v", err)
	}
	if calls != 2 {
		t.Fatalf("a bare rebind fired the hook %d times in total, want 2", calls)
	}
}

func seed(t *testing.T, ops *RecordOps, rel string) {
	t.Helper()
	if err := ops.Seed(rel, []byte("48:da:35:6e:11:22\n")); err != nil {
		t.Fatalf("seed %s: %v", rel, err)
	}
}

// The endpoint budget is what stops another function being added, so the
// accounting the compiler does to reject a profile is reported rather than
// thrown away with the plan it rejected.
func TestSnapshotReportsTheEndpointBudget(t *testing.T) {
	manager, _ := newTestManager(t)

	// A built-in is reconstructed from code on every load, so the edited one is
	// stored under the name an edit lands on.
	profile := profileForFlags(flags{rndis: true, disk: true})
	profile.Name, profile.BuiltIn = ProfileCurrent, false

	if err := manager.store.SaveProfile(profile); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if err := manager.store.SetActive(profile.Name); err != nil {
		t.Fatalf("set active: %v", err)
	}

	snapshot, err := manager.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.Endpoints != (EndpointUse{In: 6, Out: 5}) {
		t.Fatalf("endpoints = %+v, want 6 IN 5 OUT", snapshot.Endpoints)
	}
	if snapshot.Headroom != (EndpointUse{}) {
		t.Fatalf("headroom = %+v, want none left", snapshot.Headroom)
	}
}

// SurrenderUDC indexes udcs[0] the way apply and bind do, so ListUDC refusing
// anything but exactly one entry is the only thing keeping that index in range.
// The check runs before the gadget is touched, so a failure leaves it bound.
func TestSurrenderUDCFailsWhenTheUDCIsGone(t *testing.T) {
	manager, ops := newTestManager(t)
	hid := &fakeHID{}
	manager.SetHID(hid)

	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("reclaim udc: %v", err)
	}

	quiesced := len(hid.Events())

	ops.SetUDC()
	udc, err := manager.SurrenderUDC()
	if !errors.Is(err, ErrUDCCount) {
		t.Fatalf("err = %v, want the udc count error", err)
	}
	if udc != "" {
		t.Fatalf("udc = %q, want empty", udc)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}
	if events := hid.Events(); len(events) != quiesced {
		t.Fatalf("hid events = %v, want the %d from the bind", events, quiesced)
	}
}

// The close happens before the unbind, so an unbind that fails has taken the
// keyboard away for a session that never starts. The devices have to come back.
func TestSurrenderUDCReopensHIDWhenTheUnbindFails(t *testing.T) {
	manager, ops := newTestManager(t)
	hid := &fakeHID{}
	manager.SetHID(hid)

	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("reclaim udc: %v", err)
	}

	quiesced := len(hid.Events())
	refused := errors.New("device or resource busy")
	ops.FailUnbind(refused)

	if _, err := manager.SurrenderUDC(); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want %v", err, refused)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}

	events := hid.Events()
	closed := slices.Index(events[quiesced:], "close")
	reopened := slices.IndexFunc(events[quiesced:], func(event string) bool {
		return strings.HasPrefix(event, "open ")
	})
	if closed < 0 || reopened < closed {
		t.Fatalf("hid events = %v, want a close then a reopen after the bind", events)
	}
}

func TestUDCBoundTracksSurrenderAndReclaim(t *testing.T) {
	manager, _ := newTestManager(t)

	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if bound, err := manager.UDCBound(); err != nil || !bound {
		t.Fatalf("bound after reclaim = %t, %v, want true", bound, err)
	}
	if _, err := manager.SurrenderUDC(); err != nil {
		t.Fatalf("surrender: %v", err)
	}
	if bound, err := manager.UDCBound(); err != nil || bound {
		t.Fatalf("bound after surrender = %t, %v, want false", bound, err)
	}
}

// An Exact passthrough session hands udc->driver to usb-proxy, and until the
// loan was recorded nothing here knew. Every mutator below unbinds and binds
// the UDC, so each one pulls the controller out from under the running proxy:
// POST /api/storage/image/mount alone is enough to do it mid-session.
func TestGadgetMutatorsAreRefusedWhileTheUDCIsOnLoan(t *testing.T) {
	ctx := context.Background()
	mutators := []struct {
		name string
		run  func(*Manager) error
	}{
		{name: "apply", run: func(m *Manager) error { return m.Apply(ctx, ProfileStandard) }},
		{name: "set mode", run: func(m *Manager) error { return m.SetMode(ctx, ModeNormal) }},
		{name: "set lun", run: func(m *Manager) error { return m.SetLUN(ctx, LUN{File: "/data/disk.img"}) }},
		{name: "rebind", run: func(m *Manager) error { return m.Rebind(ctx) }},
		{name: "reset phy", run: func(m *Manager) error { return m.ResetPHY(ctx) }},
		{name: "set media slots", run: func(m *Manager) error { return m.SetMediaSlots(ctx, []string{"cam"}, nil) }},
		{name: "create functionfs", run: func(m *Manager) error { return m.CreateFunctionFS(ctx) }},
		{name: "start functionfs", run: func(m *Manager) error {
			_, err := m.StartFunctionFS(ctx, testFunctionFS())
			return err
		}},
		{name: "recover functionfs", run: func(m *Manager) error { return m.RecoverFunctionFS(ctx) }},
		{name: "second surrender", run: func(m *Manager) error {
			_, err := m.SurrenderUDC()
			return err
		}},
	}

	for _, mutator := range mutators {
		t.Run(mutator.name, func(t *testing.T) {
			manager, ops := newTestManager(t)
			if err := manager.ReclaimUDC(); err != nil {
				t.Fatalf("bind: %v", err)
			}
			if _, err := manager.SurrenderUDC(); err != nil {
				t.Fatalf("surrender: %v", err)
			}

			err := mutator.run(manager)
			if !errors.Is(err, ErrUDCLoaned) {
				t.Fatalf("%s during a passthrough session: err = %v, want %v", mutator.name, err, ErrUDCLoaned)
			}
			if !strings.Contains(err.Error(), "usb-proxy") {
				t.Fatalf("refusal %q does not name what holds the udc", err)
			}
			if bound := ops.Bound(); bound != "" {
				t.Fatalf("%s bound the udc to %q while usb-proxy had it", mutator.name, bound)
			}
		})
	}
}

func TestReclaimingTheUDCEndsTheLoan(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if _, err := manager.SurrenderUDC(); err != nil {
		t.Fatalf("surrender: %v", err)
	}
	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("reclaim: %v", err)
	}

	if err := manager.SetLUN(ctx, LUN{File: "/data/disk.img"}); err != nil {
		t.Fatalf("set lun after the session ended: %v", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}
}

// The watchdog reclaims the UDC when usb-proxy dies, and that bind can itself
// fail: the controller is gone, or the phy is still resetting. The loan is over
// either way. Holding it would refuse every gadget change until the next reboot,
// which is a worse failure than the rebind the loan prevents.
func TestAFailedReclaimDoesNotWedgeTheGadget(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if _, err := manager.SurrenderUDC(); err != nil {
		t.Fatalf("surrender: %v", err)
	}

	ops.SetUDC()
	if err := manager.ReclaimUDC(); err == nil {
		t.Fatal("reclaim with no udc returned no error")
	}
	ops.SetUDC(dwc2Device)

	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply after a reclaim that failed: %v", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}
}

// The loan is a claim about another process, so it is checked against the
// kernel rather than believed: the borrower holds the controller only while
// this gadget's UDC attribute is empty. Nothing writes the loan to disk either,
// so a restart cannot inherit one. What reconciles a reboot against reality is
// passthrough.Recover, which kills orphan usb-proxy processes and rebinds
// before the router is up.
func TestALoanIsDroppedOnceTheGadgetIsBoundAgain(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if _, err := manager.SurrenderUDC(); err != nil {
		t.Fatalf("surrender: %v", err)
	}
	if err := ops.BindUDC(dwc2Device); err != nil {
		t.Fatalf("bind behind the manager: %v", err)
	}

	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply against a gadget that is bound again: %v", err)
	}

	restarted := NewManager(NewStore(), ops, staticV1)
	if err := restarted.SetLUN(ctx, LUN{File: "/data/disk.img"}); err != nil {
		t.Fatalf("set lun after a restart: %v", err)
	}
}

// Suspend has no matching resume of its own: the observer only comes back when
// Applied runs. Every other refusal inside SurrenderUDC reaches that through
// refreshObserver, so a refusal that suspends before it checks and then returns
// early strands the media pipeline with no path back until the next apply.
func TestARefusedSurrenderLeavesTheObserverRunning(t *testing.T) {
	manager, _ := newTestManager(t)
	if err := manager.ReclaimUDC(); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if _, err := manager.SurrenderUDC(); err != nil {
		t.Fatalf("surrender: %v", err)
	}

	// Attached after the loan is taken, so the count below is the refusal's.
	observer := &fakeGadgetObserver{}
	manager.SetObserver(observer)

	if _, err := manager.SurrenderUDC(); !errors.Is(err, ErrUDCLoaned) {
		t.Fatalf("second surrender: err = %v, want %v", err, ErrUDCLoaned)
	}
	if observer.suspend != 0 {
		t.Fatalf("a refused surrender suspended the observer %d times and never resumed it", observer.suspend)
	}
}

func useTestUDCDir(t *testing.T, state, speed string) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), dwc2Device)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create udc dir: %v", err)
	}
	writeUDCAttr(t, dir, "state", state)
	writeUDCAttr(t, dir, "current_speed", speed)

	previous := udcDir
	udcDir = filepath.Dir(dir)
	t.Cleanup(func() { udcDir = previous })
	return dir
}

func writeUDCAttr(t *testing.T, dir, attr, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, attr), []byte(value+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", attr, err)
	}
}

// Every mutator reads the UDC attribute to prove its bind took and drops the
// answer. A caller asking for status gets the linkage but no way to tell a
// gadget nothing has plugged into from one a host has configured.
func TestSnapshotReportsTheControllerAndWhatTheHostDidWithIt(t *testing.T) {
	manager, ops := newTestManager(t)
	useTestUDCDir(t, udcConfigured, "high-speed")

	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}
	snapshot, err := manager.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	want := UDCStatus{Name: dwc2Device, Bound: true, State: udcConfigured, Speed: "high-speed"}
	if snapshot.UDC != want {
		t.Fatalf("udc = %+v, want %+v", snapshot.UDC, want)
	}

	if err := ops.UnbindUDC(); err != nil {
		t.Fatalf("unbind: %v", err)
	}
	snapshot, err = manager.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.UDC.Bound || snapshot.UDC.Name != dwc2Device {
		t.Fatalf("udc = %+v, want the controller named and unbound", snapshot.UDC)
	}
}

// The rollback puts the previous linkage back, so once the HTTP response is
// gone nothing on the device says an apply was ever attempted, let alone which
// one failed and why.
func TestAFailedApplyIsStillReadableAfterTheResponse(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}

	target := standardProfile()
	target.Name, target.BuiltIn = "desk", false
	wantErr := errors.New("configfs write failed")
	ops.FailWriteOnce(functionsDir+"/hid.GS1/report_length", wantErr)
	if err := manager.ApplyProfile(ctx, target); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want the configfs failure", err)
	}

	snapshot, err := manager.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.LastError == nil {
		t.Fatal("a failed apply left no trace in the status")
	}
	if snapshot.LastError.Profile != "desk" || !strings.Contains(snapshot.LastError.Message, wantErr.Error()) {
		t.Fatalf("last error = %+v, want the desk failure", snapshot.LastError)
	}

	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("reapply standard: %v", err)
	}
	if snapshot, err = manager.Snapshot(); err != nil || snapshot.LastError != nil {
		t.Fatalf("last error = %+v err = %v, want a successful apply to clear it", snapshot.LastError, err)
	}
}

// The dwc2 unbind takes the controller away from a host that has already
// enumerated it, and no rebind on this side puts that back. Nothing here can
// perform the power cycle, so the flag stands until a host has enumerated the
// gadget again.
func TestResetPHYLeavesAPendingPowerCycleUntilAHostReturns(t *testing.T) {
	manager, _ := newTestManager(t)
	dir := useTestUDCDir(t, "not attached", "UNKNOWN")

	if err := manager.ResetPHY(context.Background()); err != nil {
		t.Fatalf("reset phy: %v", err)
	}
	snapshot, err := manager.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !snapshot.PendingPowerCycle {
		t.Fatal("a phy reset left no pending power cycle")
	}

	writeUDCAttr(t, dir, "state", udcConfigured)
	if snapshot, err = manager.Snapshot(); err != nil || snapshot.PendingPowerCycle {
		t.Fatalf("pending = %v err = %v, want a configured controller to clear it", snapshot.PendingPowerCycle, err)
	}
}
