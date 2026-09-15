package bridge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
)

// D17 of the exit-tunnel design: the gadget NIC is either a bridge port or the
// exit's routed downstream. An enable while an exit slot is enabled is refused
// before the snapshot, the marker or any mutation, and the refusal names the
// slot so the operator knows what to turn off.
func TestEnableRefusesWhileAnExitSlotOwnsTheGadgetNIC(t *testing.T) {
	h := newHarness(t, &fakeGadget{nic: StockGadgetName})
	h.mgr.exitOwner = func() (string, bool) { return "0", true }
	// New's ReattachGadget catch-up already read the links once.
	before := len(h.net.trace())

	rsp, err := h.mgr.Enable(context.Background())
	if !errors.Is(err, ErrPreflight) {
		t.Fatalf("Enable error = %v, want ErrPreflight", err)
	}
	if rsp.State != proto.BridgeDisabled || !strings.Contains(rsp.Message, "exit tunnel slot 0") {
		t.Fatalf("Enable returned %q (%s)", rsp.State, rsp.Message)
	}
	if trace := h.net.trace(); len(trace) != before {
		t.Fatalf("a refused enable touched the device:\n%s", strings.Join(trace[before:], "\n"))
	}
	if pending, _ := h.store.Pending(); pending != nil {
		t.Fatal("a refused enable armed the dead-man")
	}
}

// The default owner check reads the exit config directory, which does not
// exist on a device that never enabled an exit, so it answers "not owned"
// and the enable proceeds as before.
func TestEnableProceedsWithNoExitConfig(t *testing.T) {
	h := newHarness(t, &fakeGadget{nic: StockGadgetName})
	if h.mgr.exitOwner == nil {
		t.Fatal("New left exitOwner nil")
	}
	h.mgr.exitOwner = func() (string, bool) { return "", false }

	if _, err := h.mgr.Enable(context.Background()); err != nil {
		t.Fatalf("Enable: %v", err)
	}
}
