package presentation

import (
	"errors"
	"testing"
)

// A pull-up cycle is how the gadget is presented again to a host that gave up
// on it during boot. It must not touch the gadget itself: no unbind, no relink,
// no attribute store - only the controller's connect state.
func TestReattachCyclesTheBoundControllerWithoutChangingTheGadget(t *testing.T) {
	ops := NewRecordOps()
	ops.files[udcAttr] = []byte(dwc2Device + "\n")
	manager := &Manager{ops: ops}

	if err := manager.Reattach(); err != nil {
		t.Fatalf("Reattach() = %v", err)
	}
	if ops.reattached != 1 {
		t.Fatalf("reattached %d times, want 1", ops.reattached)
	}
	for _, op := range ops.Trace() {
		t.Fatalf("reattach recorded %s %s: it must not change the gadget", op.Kind, op.Path)
	}
}

// Nothing bound is not a failure: a board with no controller to present should
// finish starting up, not report an error every boot.
func TestReattachIsANoOpWhenNothingIsBound(t *testing.T) {
	ops := NewRecordOps()
	ops.files[udcAttr] = []byte("\n")
	manager := &Manager{ops: ops}

	if err := manager.Reattach(); err != nil {
		t.Fatalf("Reattach() = %v, want nil when the UDC attribute is empty", err)
	}
	if ops.reattached != 0 {
		t.Fatal("cycled a controller that nothing is bound to")
	}
}

// On the way out the gadget leaves the bus: one unbind, so the host sees a
// disconnect at once instead of a camera that stays enumerated and answers
// nothing until the next server has rebound it. Nothing is unlinked, because
// an unlink is what blocks in the kernel behind an open video node.
func TestDetachUnbindsTheControllerAndNothingElse(t *testing.T) {
	ops := NewRecordOps()
	ops.files[udcAttr] = []byte(dwc2Device + "\n")
	manager := &Manager{ops: ops}

	if err := manager.Detach(); err != nil {
		t.Fatalf("Detach() = %v", err)
	}
	if ops.Bound() != "" {
		t.Fatalf("controller still bound to %q after Detach", ops.Bound())
	}
	trace := ops.Trace()
	if len(trace) != 1 || trace[0].Kind != OpUnbind {
		t.Fatalf("Detach recorded %+v, want exactly one unbind", trace)
	}
	if ops.reattached != 0 {
		t.Fatal("Detach cycled the pull-up; the host has to see the device go, not come back")
	}
}

// A controller nothing is bound to, or one on loan to a passthrough session,
// is not this process's to take down.
func TestDetachLeavesAnUnboundOrLoanedControllerAlone(t *testing.T) {
	ops := NewRecordOps()
	ops.files[udcAttr] = []byte("\n")
	if err := (&Manager{ops: ops}).Detach(); err != nil {
		t.Fatalf("Detach() = %v, want nil when the UDC attribute is empty", err)
	}
	if err := (&Manager{ops: ops, loan: "usb-proxy"}).Detach(); !errors.Is(err, ErrUDCLoaned) {
		t.Fatalf("Detach() = %v, want ErrUDCLoaned", err)
	}
	if trace := ops.Trace(); len(trace) != 0 {
		t.Fatalf("Detach touched a controller that was not its own: %+v", trace)
	}
}

// A passthrough session that has borrowed the controller owns the bus. Startup
// must not yank it out from under that session.
func TestReattachRefusesALoanedController(t *testing.T) {
	ops := NewRecordOps()
	ops.files[udcAttr] = []byte("\n")
	manager := &Manager{ops: ops, loan: "usb-proxy"}

	if err := manager.Reattach(); !errors.Is(err, ErrUDCLoaned) {
		t.Fatalf("Reattach() = %v, want ErrUDCLoaned", err)
	}
	if ops.reattached != 0 {
		t.Fatal("cycled a controller on loan to a passthrough session")
	}
}
