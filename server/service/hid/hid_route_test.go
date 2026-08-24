package hid

import (
	"os"
	"path/filepath"
	"testing"

	"NanoKVM-Server/service/presentation"
)

// Collapsing the HID layout moves absolute and relative onto the keyboard's
// node and deletes the nodes they came from. A handle opened before the change
// keeps pointing at a node that no longer carries an interface, and writing to
// it still succeeds - the role just goes quiet. That is indistinguishable from
// working, so the handle has to be dropped when the mapping changes.
func TestCollapsingTheLayoutDropsHandlesLeftOnRemovedNodes(t *testing.T) {
	dir := t.TempDir()
	keyboard := filepath.Join(dir, "hidg0")
	absolute := filepath.Join(dir, "hidg2")

	open := func(path string) *os.File {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		t.Cleanup(func() { _ = file.Close() })
		return file
	}

	h := &Hid{
		g0: open(keyboard), g0Path: keyboard,
		g2: open(absolute), g2Path: absolute,
	}

	h.SetHIDRoutes([]presentation.HIDRoute{
		{Role: presentation.HIDRoleKeyboard, Path: keyboard, ReportID: 1, Length: 8},
		{Role: presentation.HIDRoleAbsolute, Path: keyboard, ReportID: 3, Length: 6},
	})

	if h.g2 != nil {
		t.Fatalf("absolute kept its handle on %s after the role moved to %s", absolute, keyboard)
	}
	if h.g0 == nil {
		t.Fatal("the keyboard handle was dropped even though its node did not change")
	}
	if h.g0Path != keyboard {
		t.Fatalf("keyboard path = %q, want %q", h.g0Path, keyboard)
	}
}
