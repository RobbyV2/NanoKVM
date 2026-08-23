//go:build linux && kernelint

package presentation

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"NanoKVM-Server/service/kernelint"
)

// dummy_hcd has no OTG role switch and no dwc2 to rebind, so that write is
// pointed at a scratch file. Every other op in the plan reaches configfs.
func kernelManager(t *testing.T) *Manager {
	t.Helper()

	dir := t.TempDir()
	previousDir, previousOTG := presentationDir, otgRolePath
	presentationDir = filepath.Join(dir, "presentation")
	otgRolePath = filepath.Join(dir, "otg_role")
	t.Cleanup(func() { presentationDir, otgRolePath = previousDir, previousOTG })
	if err := os.MkdirAll(presentationDir, 0o755); err != nil {
		t.Fatal(err)
	}

	kernelint.BootstrapGadget(t, GadgetRoot)
	ops, err := NewConfigFSOps(GadgetRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ops.Close() })
	return NewManager(NewStore(), ops, LoadCapabilities())
}

// f_hid gained wakeup_on_write in the vendor 5.10 tree the device runs; stock
// 6.8 has no such attribute, and a write to it is ENOENT. It is the one op in
// either built-in plan this kernel cannot take, so it is compiled out rather
// than swallowed, which would hide a real ENOENT somewhere else.
func kernelProfile(base Profile) Profile {
	profile := base
	profile.Functions = append([]Function(nil), base.Functions...)
	for i := range profile.Functions {
		hid := *profile.Functions[i].HID
		hid.WakeupOnWrite = false
		profile.Functions[i].HID = &hid
	}
	return profile
}

func udcState(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("/sys/class/udc", kernelint.DummyUDC, "state"))
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.TrimSpace(data))
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// The twenty golden traces assert the plan matches a recorded byte sequence.
// This asserts the kernel accepts that same sequence: the traces say the
// compiler did not change, and this says what it emits still binds.
func TestKernelTier2StandardPlanBindsTheUDC(t *testing.T) {
	kernelint.RequireTier2(t)
	manager := kernelManager(t)

	profile := kernelProfile(standardProfile())
	plan, err := Compile(profile, manager.caps)
	if err != nil {
		t.Fatal(err)
	}

	_, udc, err := manager.applyPlan(context.Background(), profile, plan, false)
	if err != nil {
		t.Fatalf("apply the standard plan against %s: %v", kernelint.DummyUDC, err)
	}
	if udc != kernelint.DummyUDC {
		t.Fatalf("bound udc = %q", udc)
	}

	waitFor(t, "the gadget to reach configured", func() bool { return udcState(t) == "configured" })

	for index := range profile.Functions {
		node := filepath.Join("/dev", "hidg"+string(rune('0'+index)))
		if _, err := os.Stat(node); err != nil {
			t.Fatalf("%s: %v", node, err)
		}
	}

	// The report descriptors are the one part of the plan a fake cannot
	// validate. f_hid stores them verbatim, and the host side of dummy_hcd runs
	// hid-generic over the same bytes, so a descriptor the parser rejects shows
	// up as a missing /sys/bus/hid device rather than as a failed write.
	for _, function := range profile.Functions {
		path := filepath.Join(GadgetRoot, functionsDir, string(function.Kind)+"."+function.Instance, "report_desc")
		stored, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if !bytes.Equal(stored, function.HID.ReportDesc) {
			t.Fatalf("%s round-tripped %d of %d bytes", path, len(stored), len(function.HID.ReportDesc))
		}
	}

	waitFor(t, "hid-generic to parse all three report descriptors", func() bool {
		entries, err := os.ReadDir("/sys/bus/hid/devices")
		return err == nil && len(entries) >= len(profile.Functions)
	})
}
