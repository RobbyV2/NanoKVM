package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"NanoKVM-Server/service/presentation"
)

// The subscription is what makes a hold worth taking on this fork's kernel:
// f_uvc keeps the gadget on the bus from the bind and forwards every class
// request to whichever handle has subscribed, so a hold that only opened the
// node would leave the host's first request unanswered and ep0 wedged. It is
// armed in the same call as the open, on the very descriptor that is held,
// before any caller can see it.
func TestHoldArmsTheDescriptorAtTheOpen(t *testing.T) {
	previous := armHold
	defer func() { armHold = previous }()
	var armed []int
	armHold = func(fd int) error {
		armed = append(armed, fd)
		return nil
	}

	node := filepath.Join(t.TempDir(), "video0")
	if err := os.WriteFile(node, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := holdNode(node)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if len(armed) != 1 || armed[0] != holder.FD() {
		t.Fatalf("armed %v, want exactly the held descriptor %d, once", armed, holder.FD())
	}
}

// A subscription that fails is not a hold that fails. The descriptor still
// keeps the function activated on a kernel that arms bind_deactivated, and the
// worker subscribes again when it adopts the dup.
func TestAHoldSurvivesASubscriptionThatFails(t *testing.T) {
	previous := armHold
	defer func() { armHold = previous }()
	armHold = func(int) error { return errors.New("EINVAL") }

	node := filepath.Join(t.TempDir(), "video0")
	if err := os.WriteFile(node, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := holdNode(node)
	if err != nil {
		t.Fatalf("a failed subscription cost the hold: %v", err)
	}
	defer holder.Close()
	if holder.FD() < 0 {
		t.Fatal("the hold came back without a descriptor")
	}
}

// Bound is the presentation manager's call at the instant the controller
// binds, before the OTG role write, the HID verify and the rebind hooks. The
// hold it takes is the one Reconcile later finds: the node is opened once, and
// the worker dups the descriptor Bound opened rather than opening its own.
func TestBoundHoldsTheNodeOnceAndReconcileReusesIt(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	holds := newFakeHolds()
	manager := newManagerWith(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory, holds.open)
	defer manager.Suspend()

	manager.Bound(context.Background())
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 0 {
		t.Fatalf("at the bind the node was opened %d times and closed %d, want 1 and 0", opens, closes)
	}

	if err := manager.Reconcile(context.Background(), compositeProfile(), presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 0 {
		t.Fatalf("the reconcile after the bind opened the node again: %d opens and %d closes, want 1 and 0", opens, closes)
	}
	if fd := factory.spec("uvc.cam0").FD; fd == 0 {
		t.Fatal("the camera output was opened without the held descriptor")
	}
}
