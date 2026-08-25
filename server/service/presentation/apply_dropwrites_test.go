package presentation

import (
	"testing"
)

// dropRedundantWrites reads the attribute to decide a write is unnecessary, but
// f_uvc's rebuild removes and recreates the directory that attribute lives in,
// and configfs hands the recreated one the function's compiled-in default. A
// write dropped on the strength of a value the same plan is about to destroy
// leaves that default standing.
//
// Measured on hardware before the fix: every second apply published UVC frames
// holding f_uvc's defaults - 640x360 with an empty dwFrameInterval - which
// renders as bFrameIntervalType 0 without the interval fields continuous layout
// requires, and Windows refused the camera with CM_PROB_FAILED_START.
func TestRedundantWritesSurviveADirectoryTheirPlanRecreates(t *testing.T) {
	const frame = "functions/uvc.cam0/streaming/mjpeg/m/1280x720"

	ops := NewRecordOps()
	// The state a previous good apply left: the values the plan wants.
	ops.files[frame+"/wWidth"] = []byte("1280\n")
	ops.files[frame+"/dwFrameInterval"] = []byte("333333\n")
	manager := &Manager{ops: ops}

	plan := Plan{Ops: []Op{
		{Kind: OpRmdir, Path: frame},
		{Kind: OpMkdir, Path: frame},
		{Kind: OpWrite, Path: frame + "/wWidth", Data: []byte("1280\n")},
		{Kind: OpWrite, Path: frame + "/dwFrameInterval", Data: []byte("333333\n")},
	}}
	before := Snapshot{Linked: []string{"uvc.cam0"}}

	got := manager.dropRedundantWrites(before, plan)
	for _, attr := range []string{"/wWidth", "/dwFrameInterval"} {
		if opIndex(got, OpWrite, frame+attr) < 0 {
			t.Fatalf("dropped the write to %s, which the plan's own rmdir resets to the kernel default", frame+attr)
		}
	}
}

// The case the dropping exists for: S03usbdev wrote these values at boot, the
// function is linked so configfs would refuse the store, and nothing in the
// plan removes the attribute. That write must still go.
func TestRedundantWritesAreStillDroppedWhenNothingRemovesThem(t *testing.T) {
	const attr = "functions/hid.GS0/protocol"

	ops := NewRecordOps()
	ops.files[attr] = []byte("0\n")
	manager := &Manager{ops: ops}

	plan := Plan{Ops: []Op{{Kind: OpWrite, Path: attr, Data: []byte("0\n")}}}
	got := manager.dropRedundantWrites(Snapshot{Linked: []string{"hid.GS0"}}, plan)
	if len(got.Ops) != 0 {
		t.Fatalf("kept %d ops, want the write dropped: it would take -EBUSY on a linked function", len(got.Ops))
	}
}
