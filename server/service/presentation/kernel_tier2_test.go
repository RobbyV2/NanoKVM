//go:build linux && kernelint

package presentation

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

// The state S03usbdev:99,114,129 leaves every device in before the server
// starts: the three HID functions built, written and linked into c.1, then the
// UDC bound last. f_hid takes opts->refcnt at link time, so this is what
// decides whether the first ApplyProfile after boot can write them at all.
func bootLinkedHID(t *testing.T) {
	t.Helper()

	udc := filepath.Join(GadgetRoot, udcAttr)
	if bound, err := os.ReadFile(udc); err == nil && strings.TrimSpace(string(bound)) != "" {
		if err := os.WriteFile(udc, []byte(emptyUDCName), 0o644); err != nil {
			t.Fatalf("unbind: %v", err)
		}
	}

	for _, function := range standardProfile().Functions {
		name := functionName(function)
		dir := filepath.Join(GadgetRoot, functionsDir, name)
		link := filepath.Join(GadgetRoot, configPrefix, name)
		attrs := map[string][]byte{
			"protocol":      []byte(strconv.Itoa(int(function.HID.Protocol)) + "\n"),
			"report_length": []byte(strconv.Itoa(int(function.HID.ReportLength)) + "\n"),
			"report_desc":   function.HID.ReportDesc,
		}

		if _, err := os.Lstat(link); err != nil {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", name, err)
			}
			for attr, value := range attrs {
				if err := os.WriteFile(filepath.Join(dir, attr), value, 0o644); err != nil {
					t.Fatalf("write %s/%s: %v", name, attr, err)
				}
			}
			if err := os.Symlink(dir, link); err != nil {
				t.Fatalf("link %s: %v", name, err)
			}
		}

		// f_hid guards its option stores with opts->refcnt and its shows with
		// nothing, so the values are still readable through the link. That is
		// what lets the transaction tell a redundant write from a real one.
		for attr, value := range attrs {
			stored, err := os.ReadFile(filepath.Join(link, attr))
			if err != nil {
				t.Fatalf("read %s/%s while linked: %v", name, attr, err)
			}
			if !bytes.Equal(stored, value) {
				t.Fatalf("%s/%s holds %q, want %q", name, attr, stored, value)
			}
		}
	}

	if err := os.WriteFile(udc, []byte(kernelint.DummyUDC+"\n"), 0o644); err != nil {
		t.Fatalf("bind %s: %v", kernelint.DummyUDC, err)
	}
}

// Every HID attribute the standard plan carries is one S03usbdev already wrote,
// and unlinkStale keeps a link the incoming plan also carries, so the refcnt is
// never released and the write has to be recognised as redundant rather than
// reissued.
func TestKernelTier2ApplyOverBootLinkedHID(t *testing.T) {
	kernelint.RequireTier2(t)
	manager := kernelManager(t)
	bootLinkedHID(t)

	profile := kernelProfile(standardProfile())
	plan, err := Compile(profile, manager.caps)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.applyPlan(context.Background(), profile, plan, false); err != nil {
		t.Fatalf("apply over the boot linkage: %v", err)
	}

	waitFor(t, "the gadget to reach configured", func() bool { return udcState(t) == "configured" })
}

// Writing an empty UDC is ENODEV unless the gadget is bound right now, so the
// unbind applyPlan opens with cannot be unconditional. On a device S03usbdev
// has always bound it first, which is the only reason this has never fired
// there; the rollback ladder reaches it with the controller already released.
func TestKernelTier2ApplyBindsAnUnboundGadget(t *testing.T) {
	kernelint.RequireTier2(t)
	manager := kernelManager(t)
	bootLinkedHID(t)
	if err := manager.ops.UnbindUDC(); err != nil {
		t.Fatalf("unbind: %v", err)
	}

	profile := kernelProfile(standardProfile())
	plan, err := Compile(profile, manager.caps)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.applyPlan(context.Background(), profile, plan, false); err != nil {
		t.Fatalf("apply to an unbound gadget: %v", err)
	}

	waitFor(t, "the gadget to reach configured", func() bool { return udcState(t) == "configured" })
}

func gadgetSnapshot(t *testing.T) string {
	t.Helper()

	var parts []string
	for _, dir := range []string{"functions", "configs/c.1"} {
		entries, err := os.ReadDir(filepath.Join(GadgetRoot, dir))
		if err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		parts = append(parts, dir+"="+strings.Join(names, ","))
	}
	bound, err := os.ReadFile(filepath.Join(GadgetRoot, "UDC"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(parts, " ") + " UDC=" + strings.TrimSpace(string(bound))
}

// The device reaches LoadCapabilities with S03usbdev's gadget already built and
// bound, and the probe then creates a second gadget and instantiates a real
// kernel object per function kind in it. No fake can say whether that disturbs
// the live one, and if it did the server would never reach its listener.
func TestKernelTier2CapabilityProbeSparesTheBoundGadget(t *testing.T) {
	kernelint.RequireTier2(t)

	previousDir := presentationDir
	presentationDir = filepath.Join(t.TempDir(), "presentation")
	t.Cleanup(func() { presentationDir = previousDir })

	kernelint.BootstrapGadget(t, GadgetRoot)
	before := gadgetSnapshot(t)

	started := time.Now()
	table := LoadCapabilities()
	elapsed := time.Since(started)

	if elapsed > probeBudget {
		t.Fatalf("probe took %s, past its %s budget", elapsed, probeBudget)
	}
	if table.Source != SourceProbeV1 {
		t.Fatalf("got capability source %q, want %q", table.Source, SourceProbeV1)
	}
	if _, err := os.Stat(probeGadgetDir); err == nil {
		t.Fatalf("%s survived the probe", probeGadgetDir)
	}
	if after := gadgetSnapshot(t); after != before {
		t.Fatalf("the probe changed the bound gadget:\n before %s\n after  %s", before, after)
	}
}
