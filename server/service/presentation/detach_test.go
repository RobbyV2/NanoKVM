package presentation

import (
	"errors"
	"testing"
)

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
