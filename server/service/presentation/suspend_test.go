package presentation

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

// An observer that cannot give its device nodes back. On the live device this
// is a UVC video node left open by a worker that did not stop: configfs then
// blocks the unlink of that function in the kernel, where the apply's own
// context cannot reach it, and the gadget is left mid-transaction with nothing
// linked - no HID, no NIC, no disk - until the board is rebooted.
type stuckObserver struct {
	mu       sync.Mutex
	err      error
	suspends int
	applied  int
}

func (o *stuckObserver) Suspend() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.suspends++
	return o.err
}

func (o *stuckObserver) Applied(context.Context, Profile, Plan) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.applied++
	return nil
}

func (o *stuckObserver) counts() (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.suspends, o.applied
}

func TestApplyRefusesWhenMediaCannotReleaseItsNodes(t *testing.T) {
	manager, ops := newTestManager(t)
	manager.caps = staticV1
	observer := &stuckObserver{err: errors.New("/dev/video0 is still open")}
	manager.SetObserver(observer)

	profile := mediaProfile(testCamera("cam0", 768))
	err := manager.ApplyProfile(context.Background(), profile)
	if !errors.Is(err, ErrMediaBusy) {
		t.Fatalf("ApplyProfile() = %v, want ErrMediaBusy", err)
	}
	if !strings.Contains(err.Error(), "/dev/video0") {
		t.Fatalf("ApplyProfile() = %v, want the node named", err)
	}
	if suspends, _ := observer.counts(); suspends != 1 {
		t.Fatalf("suspends = %d, want 1", suspends)
	}
	// Nothing may have been taken apart: the refusal exists so the gadget is
	// never left mid-transaction.
	for _, op := range ops.Trace() {
		if op.Kind == OpUnlink || op.Kind == OpRmdir {
			t.Fatalf("a refused apply still unlinked %s", op.Path)
		}
	}
}

func TestApplyProceedsWhenMediaReleasesItsNodes(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.caps = staticV1
	observer := &stuckObserver{}
	manager.SetObserver(observer)

	profile := mediaProfile(testCamera("cam0", 768))
	if err := manager.ApplyProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	suspends, applied := observer.counts()
	if suspends != 1 || applied != 1 {
		t.Fatalf("suspends = %d applied = %d, want 1 and 1", suspends, applied)
	}
}

// On the way out the process gives the video node back first and reports what
// it could not release, without unlinking anything and without waiting on the
// gadget: the report is for the log, the exit happens either way.
func TestSuspendMediaReportsWhatTheObserverCouldNotRelease(t *testing.T) {
	manager, ops := newTestManager(t)
	if err := manager.SuspendMedia(); err != nil {
		t.Fatalf("SuspendMedia() with no observer = %v, want nil", err)
	}

	observer := &stuckObserver{err: errors.New("/dev/video0 is still open")}
	manager.SetObserver(observer)
	err := manager.SuspendMedia()
	if err == nil || !strings.Contains(err.Error(), "/dev/video0") {
		t.Fatalf("SuspendMedia() = %v, want the node named", err)
	}
	if suspends, applied := observer.counts(); suspends != 1 || applied != 0 {
		t.Fatalf("suspends = %d applied = %d, want 1 and 0: shutdown never rebuilds the pipeline", suspends, applied)
	}
	if trace := ops.Trace(); len(trace) != 0 {
		t.Fatalf("SuspendMedia touched the gadget: %+v", trace)
	}
}

// Mounting an image rebinds the controller, and a rebind takes the camera's
// video node away: f_uvc registers a new device, the media manager's hold is on
// the old one, and the new function keeps the gadget deactivated until its new
// node is opened. Every other rebind stands the pipeline down first and
// rebuilds it after; this one has to as well, or the mount costs the operator
// HID, the NIC and the disk it was mounting.
func TestSetLUNStandsTheMediaPipelineDownAndRebuildsIt(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.caps = staticV1
	observer := &stuckObserver{}
	manager.SetObserver(observer)
	if err := manager.ApplyProfile(context.Background(), mediaProfile(testCamera("cam0", 768))); err != nil {
		t.Fatal(err)
	}
	suspends, applied := observer.counts()
	if suspends != 1 || applied != 1 {
		t.Fatalf("after apply suspends = %d applied = %d, want 1 and 1", suspends, applied)
	}

	if err := manager.SetLUN(context.Background(), LUN{File: "/data/boot.iso", CDROM: true}); err != nil {
		t.Fatalf("set lun: %v", err)
	}
	suspends, applied = observer.counts()
	if suspends != 2 || applied != 2 {
		t.Fatalf("after set lun suspends = %d applied = %d, want 2 and 2: the rebind orphans the camera's node otherwise", suspends, applied)
	}
}
