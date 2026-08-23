package presentation

import (
	"context"
	"errors"
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
