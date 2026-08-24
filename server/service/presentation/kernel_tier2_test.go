//go:build linux && kernelint

package presentation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"NanoKVM-Server/service/kernelint"

	"golang.org/x/sys/unix"
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

func hidMinors(t *testing.T) map[string]string {
	t.Helper()

	minors := make(map[string]string, 3)
	for _, name := range []string{"hid.GS0", "hid.GS1", "hid.GS2"} {
		data, err := os.ReadFile(filepath.Join(GadgetRoot, "functions", name, "dev"))
		if err != nil {
			t.Fatalf("read %s minor: %v", name, err)
		}
		minors[name] = strings.TrimSpace(string(data))
	}
	return minors
}

func configLinks(t *testing.T) map[string]bool {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(GadgetRoot, configPrefix))
	if err != nil {
		t.Fatalf("read %s: %v", configPrefix, err)
	}
	links := make(map[string]bool, len(entries))
	for _, entry := range entries {
		info, err := os.Lstat(filepath.Join(GadgetRoot, configPrefix, entry.Name()))
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			links[entry.Name()] = true
		}
	}
	return links
}

// S03usbdev's gadget: three hid functions linked into c.1, then the UDC bound.
// configfs refuses a config change on a bound gadget and f_hid refuses an
// attribute store on a linked function, so the order is unbind, unlink, write,
// link, bind. One VM boot runs every test in this package against one configfs,
// so the hid links are taken back down rather than assumed to be down; what
// kernelint.BootstrapGadget links beside them is left alone, since a config
// change on a bound gadget is EINVAL and the next test to call it would find
// its own link gone.
func bootstrapHIDGadget(t *testing.T) {
	t.Helper()
	kernelint.RequireTier2(t)

	for _, dir := range []string{"", "strings/0x409", "configs/c.1", "configs/c.1/strings/0x409"} {
		if err := os.MkdirAll(filepath.Join(GadgetRoot, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	udcAttrPath := filepath.Join(GadgetRoot, udcAttr)
	if bound, err := os.ReadFile(udcAttrPath); err == nil && strings.TrimSpace(string(bound)) != "" {
		if err := os.WriteFile(udcAttrPath, []byte(emptyUDCName), 0o644); err != nil {
			t.Fatalf("unbind: %v", err)
		}
	}
	for name := range configLinks(t) {
		if !strings.HasPrefix(name, string(FunctionHID)+".") {
			continue
		}
		if err := os.Remove(filepath.Join(GadgetRoot, configPrefix, name)); err != nil {
			t.Fatalf("unlink %s: %v", name, err)
		}
	}
	for path, value := range map[string]string{
		"idVendor":                                "0x3346",
		"idProduct":                               "0x1009",
		"strings/0x409/manufacturer":              "sipeed",
		"strings/0x409/product":                   "NanoKVM",
		"strings/0x409/serialnumber":              "0123456789ABCDEF",
		"configs/c.1/strings/0x409/configuration": "NanoKVM",
	} {
		if err := os.WriteFile(filepath.Join(GadgetRoot, path), []byte(value+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	for _, name := range []string{"hid.GS0", "hid.GS1", "hid.GS2"} {
		dir := filepath.Join(GadgetRoot, "functions", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "report_desc"),
			[]byte{0x05, 0x01, 0x09, 0x06, 0xa1, 0x01, 0xc0}, 0o644); err != nil {
			t.Fatalf("write %s report_desc: %v", name, err)
		}
		if err := os.Symlink(dir, filepath.Join(GadgetRoot, "configs/c.1", name)); err != nil && !os.IsExist(err) {
			t.Fatalf("link %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(GadgetRoot, "UDC"), []byte(kernelint.DummyUDC+"\n"), 0o644); err != nil {
		t.Fatalf("bind UDC: %v", err)
	}
}

// f_hid takes its /dev/hidgN minor from an ida in hidg_alloc_inst, which runs at
// mkdir, and returns it in hidg_free_inst, which runs at rmdir. The refcnt that
// makes an attribute write -EBUSY is the one hidg_alloc and hidg_free move on
// link and unlink. So an attribute write needs the symlink gone, and dropping
// the symlink is not what renumbers the minors: only rmdir is.
func TestKernelTier2UnlinkKeepsHIDMinors(t *testing.T) {
	previousDir := presentationDir
	presentationDir = filepath.Join(t.TempDir(), "presentation")
	t.Cleanup(func() { presentationDir = previousDir })

	bootstrapHIDGadget(t)
	before := hidMinors(t)

	udc := filepath.Join(GadgetRoot, "UDC")
	link := filepath.Join(GadgetRoot, "configs/c.1/hid.GS0")
	subclass := filepath.Join(GadgetRoot, "functions/hid.GS0/subclass")

	if err := os.WriteFile(udc, []byte("\n"), 0o644); err != nil {
		t.Fatalf("unbind UDC: %v", err)
	}

	if err := os.WriteFile(subclass, []byte("1\n"), 0o644); err == nil {
		t.Fatal("writing subclass while linked should be EBUSY")
	} else if !errors.Is(err, unix.EBUSY) {
		t.Fatalf("writing subclass while linked: got %v, want EBUSY", err)
	}

	if err := os.Remove(link); err != nil {
		t.Fatalf("unlink hid.GS0: %v", err)
	}
	if err := os.WriteFile(subclass, []byte("1\n"), 0o644); err != nil {
		t.Fatalf("write subclass while unlinked: %v", err)
	}
	if err := os.Symlink(filepath.Join(GadgetRoot, "functions/hid.GS0"), link); err != nil {
		t.Fatalf("relink hid.GS0: %v", err)
	}
	if err := os.WriteFile(udc, []byte(kernelint.DummyUDC+"\n"), 0o644); err != nil {
		t.Fatalf("rebind UDC: %v", err)
	}

	if after := hidMinors(t); !maps.Equal(before, after) {
		t.Fatalf("unlink and relink renumbered the minors:\n before %v\n after  %v", before, after)
	}
}

// The failure the device reports, reproduced against a real kernel: a profile
// that changes the identity the host keys its driver binding off and a HID
// attribute at the same time, applied over the gadget S03usbdev built and
// bound. Every one of those stores is -EBUSY while the function holds
// opts->refcnt, the rollback and the hid-only fallback are refused for the same
// reason, and what the device was left with was three HID links in c.1 and six
// function directories the host could not see.
func TestKernelTier2ApplyChangesAttributesOnALinkedGadget(t *testing.T) {
	manager := kernelManager(t)
	bootstrapHIDGadget(t)

	minors := hidMinors(t)
	linked := configLinks(t)
	for _, instance := range hidInstances {
		if name := string(FunctionHID) + "." + instance; !linked[name] {
			t.Fatalf("configs/c.1 holds %v, want the %s link S03usbdev leaves", linked, name)
		}
	}

	profile := kernelProfile(standardProfile())
	profile.Name, profile.BuiltIn = "foreign", false
	profile.Device.VendorID, profile.Device.ProductID = "0x046d", "0xc52b"
	profile.Device.Manufacturer, profile.Device.Product = "Logitech, Inc.", "Unifying Receiver"
	profile.Device.Serial = ptr("KVM0000000000001")
	for i := range profile.Functions {
		hid := *profile.Functions[i].HID
		hid.SubClass = 1
		profile.Functions[i].HID = &hid
	}
	profile.Normalize()

	plan, err := Compile(profile, manager.caps)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if _, _, err := manager.applyPlan(context.Background(), profile, plan, false); err != nil {
		t.Fatalf("apply a changed identity over the boot linkage: %v", err)
	}

	waitFor(t, "the gadget to reach configured", func() bool { return udcState(t) == "configured" })

	// Eliding a redundant write is what makes the first apply after boot
	// possible; it must not be what makes this one pass. Every value here
	// differs from the one bootstrapHIDGadget wrote.
	for path, want := range map[string]string{
		"idVendor":                   "0x046d",
		"idProduct":                  "0xc52b",
		"strings/0x409/manufacturer": "Logitech, Inc.",
		"strings/0x409/product":      "Unifying Receiver",
		"strings/0x409/serialnumber": "KVM0000000000001",
		"functions/hid.GS0/subclass": "1",
	} {
		data, err := os.ReadFile(filepath.Join(GadgetRoot, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if got := strings.TrimSpace(string(data)); got != want {
			t.Fatalf("%s holds %q, want %q", path, got, want)
		}
	}

	bound, err := os.ReadFile(filepath.Join(GadgetRoot, udcAttr))
	if err != nil || strings.TrimSpace(string(bound)) != kernelint.DummyUDC {
		t.Fatalf("UDC holds %q err = %v, want %q", strings.TrimSpace(string(bound)), err, kernelint.DummyUDC)
	}
	for name := range linked {
		if !configLinks(t)[name] {
			t.Fatalf("%s was linked before the apply and is not linked after", name)
		}
	}
	if after := hidMinors(t); !maps.Equal(minors, after) {
		t.Fatalf("the apply renumbered the minors:\n before %v\n after  %v", minors, after)
	}
}

// Why service/hid still reboots after a mode change. hid-only writes bcdUSB and
// the standard profile leaves it alone, so the way back to normal keeps the USB
// 1.1 descriptor the host enumerated hid-only with. Nothing else survives the
// round trip: the descriptors go back and the minors hold.
func TestKernelTier2ModeRoundTripKeepsHIDOnlyBCDUSB(t *testing.T) {
	manager := kernelManager(t)
	bootstrapHIDGadget(t)

	minors := hidMinors(t)
	standard := kernelProfile(standardProfile())
	for _, profile := range []Profile{kernelProfile(hidOnlyProfile()), standard} {
		plan, err := Compile(profile, manager.caps)
		if err != nil {
			t.Fatalf("compile %s: %v", profile.Name, err)
		}
		if _, _, err := manager.applyPlan(context.Background(), profile, plan, false); err != nil {
			t.Fatalf("apply %s: %v", profile.Name, err)
		}
	}
	waitFor(t, "the gadget to reach configured", func() bool { return udcState(t) == "configured" })

	if after := hidMinors(t); !maps.Equal(minors, after) {
		t.Fatalf("the mode round trip renumbered the minors:\n before %v\n after  %v", minors, after)
	}
	for _, function := range standard.Functions {
		path := filepath.Join(GadgetRoot, functionsDir, string(function.Kind)+"."+function.Instance, "report_desc")
		stored, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if !bytes.Equal(stored, function.HID.ReportDesc) {
			t.Fatalf("%s did not go back to the standard descriptor", path)
		}
	}

	data, err := os.ReadFile(filepath.Join(GadgetRoot, "bcdUSB"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != "0x0101" {
		t.Fatalf("bcdUSB = %q: the round trip now restores it, so SetHidMode no longer needs its reboot", got)
	}
}

// A composite report descriptor is the one thing in the layout work that no
// pure test can settle: hid-generic parses the same bytes the attached host
// will, and an ill-formed Report ID shows up as a HID device that never
// appears rather than as a failed write. What this cannot show is the dwc2
// FIFO budget, because dummy_hcd has endpoints the SG2002 does not.
func TestKernelTier2CompositeHIDReportDescriptorBinds(t *testing.T) {
	kernelint.RequireTier2(t)
	manager := kernelManager(t)

	profile := kernelProfile(standardProfile())
	if err := SetHIDLayout(&profile, [][]HIDRole{{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}}); err != nil {
		t.Fatal(err)
	}
	profile = kernelProfile(profile)
	if got := len(profile.Functions); got != 1 {
		t.Fatalf("hid functions = %d, want 1", got)
	}
	if profile.Functions[0].HID.ReportLength != 9 {
		t.Fatalf("report length = %d, want 9", profile.Functions[0].HID.ReportLength)
	}

	plan, err := Compile(profile, manager.caps)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.applyPlan(context.Background(), profile, plan, false); err != nil {
		t.Fatalf("apply the composite plan against %s: %v", kernelint.DummyUDC, err)
	}
	waitFor(t, "the gadget to reach configured", func() bool { return udcState(t) == "configured" })

	path := filepath.Join(GadgetRoot, functionsDir, functionName(profile.Functions[0]), "report_desc")
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if !bytes.Equal(stored, profile.Functions[0].HID.ReportDesc) {
		t.Fatalf("%s round-tripped %d of %d bytes", path, len(stored), len(profile.Functions[0].HID.ReportDesc))
	}

	waitFor(t, "hid-generic to parse the composite report descriptor", func() bool {
		entries, err := os.ReadDir("/sys/bus/hid/devices")
		return err == nil && len(entries) >= 1
	})
}

// A host reads a camera's name out of the descriptors it enumerated, so the
// only place the writable function_name can be confirmed is the other side of
// an enumeration. Two cameras that differ in nothing else must read back as two
// different strings; identical ones are the "UVC Camera, UVC Camera" this
// exists to fix. dummy_hcd still refuses PIPE_ISOCHRONOUS traffic, so this says
// nothing about streaming and everything about descriptors.
func TestKernelTier2NamedCamerasEnumerateDistinctHostNames(t *testing.T) {
	kernelint.RequireTier2(t)
	manager := kernelManager(t)
	if !manager.caps.Functions[FunctionUVC].Attributes[UVCAttrFunctionName] {
		t.Fatalf("uvc has no writable %s on this kernel; the naming backport is not in it", UVCAttrFunctionName)
	}

	left, right := "Left Camera", "Right Camera"
	profile := standardProfile()
	profile.Name = "media-naming"
	profile.BuiltIn = false
	profile.Functions = []Function{
		kernelCamera("cam0", &left),
		kernelCamera("cam1", &right),
	}
	plan, err := Compile(profile, manager.caps)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.applyPlan(context.Background(), profile, plan, false); err != nil {
		t.Fatalf("apply two named cameras: %v", err)
	}
	for _, node := range videoNodes(t, len(profile.Functions)) {
		subscribeUVCSetup(t, node)
	}
	waitFor(t, "the gadget to reach configured", func() bool { return udcState(t) == "configured" })

	// The gadget the previous apply left behind is still enumerated for a
	// moment, so the names are polled rather than read once.
	deadline := time.Now().Add(5 * time.Second)
	for {
		names := videoFunctionNames(t, profile.Device.VendorID, profile.Device.ProductID)
		if slices.Contains(names, left) && slices.Contains(names, right) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("host-visible video function names = %q, want %q and %q", names, left, right)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// f_uvc binds deactivated and asks the composite to connect only once a
// userspace app subscribes to UVC_EVENT_SETUP on its V4L2 node, which is what
// service/media does on the device. Without it the gadget, though bound, never
// pulls up and the host sees nothing to enumerate.
func subscribeUVCSetup(t *testing.T, node string) {
	t.Helper()

	const (
		vidiocSubscribeEvent = 0x4020565A
		uvcEventSetup        = 0x08000000 + 4
	)
	file, err := os.OpenFile(node, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open %s: %v", node, err)
	}
	t.Cleanup(func() { _ = file.Close() })

	subscription := [8]uint32{uvcEventSetup}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, file.Fd(), vidiocSubscribeEvent, uintptr(unsafe.Pointer(&subscription))); errno != 0 {
		t.Fatalf("subscribe UVC_EVENT_SETUP on %s: %v", node, errno)
	}
}

func videoNodes(t *testing.T, want int) []string {
	t.Helper()

	var nodes []string
	waitFor(t, "f_uvc to register a V4L2 node per camera", func() bool {
		nodes, _ = filepath.Glob("/dev/video*")
		return len(nodes) >= want
	})
	return nodes[:want]
}

func kernelCamera(instance string, name *string) Function {
	return Function{Kind: FunctionUVC, Instance: instance, Video: &VideoFunction{
		FunctionName: "NanoKVM " + instance, HostName: name,
		StreamingMaxPacket: 768, StreamingInterval: 1,
		Formats: []VideoFormat{{Codec: "mjpeg", Frames: []VideoFrame{
			{Width: 640, Height: 480, Intervals: []uint32{333333}},
		}}},
	}}
}

// The gadget and its host controller are the same machine under dummy_hcd, so
// the descriptors it presented are readable back through usbfs. f_uvc puts the
// function name on the interface association descriptor's iFunction, which is
// the string a host shows for the whole camera; sysfs exposes no attribute for
// it, so the index comes out of the raw config descriptor and the string out of
// a GET_DESCRIPTOR the host controller answers.
func videoFunctionNames(t *testing.T, vendor, product string) []string {
	t.Helper()

	entries, err := os.ReadDir("/sys/bus/usb/devices")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		device := filepath.Join("/sys/bus/usb/devices", entry.Name())
		if !sysfsHas(device, "idVendor", strings.TrimPrefix(vendor, "0x")) || !sysfsHas(device, "idProduct", strings.TrimPrefix(product, "0x")) {
			continue
		}
		descriptors, err := os.ReadFile(filepath.Join(device, "descriptors"))
		if err != nil {
			continue
		}
		node := usbfsNode(device)
		if node == nil {
			continue
		}
		for _, index := range videoFunctionStringIndices(descriptors) {
			names = append(names, usbStringDescriptor(node, index))
		}
		_ = node.Close()
	}
	return names
}

func videoFunctionStringIndices(descriptors []byte) []uint8 {
	const associationDescriptor, videoClass = 0x0B, 0x0E

	var indices []uint8
	for i := 0; i+2 <= len(descriptors); {
		length := int(descriptors[i])
		if length < 2 || i+length > len(descriptors) {
			break
		}
		if descriptors[i+1] == associationDescriptor && length >= 8 && descriptors[i+4] == videoClass {
			indices = append(indices, descriptors[i+7])
		}
		i += length
	}
	return indices
}

// A device that is on its way out between the sysfs walk and the open is the
// gadget the previous apply left behind, so a failure here is polled through
// rather than reported.
func usbfsNode(device string) *os.File {
	bus, err := os.ReadFile(filepath.Join(device, "busnum"))
	if err != nil {
		return nil
	}
	dev, err := os.ReadFile(filepath.Join(device, "devnum"))
	if err != nil {
		return nil
	}
	busNumber, _ := strconv.Atoi(strings.TrimSpace(string(bus)))
	devNumber, _ := strconv.Atoi(strings.TrimSpace(string(dev)))
	file, err := os.OpenFile(fmt.Sprintf("/dev/bus/usb/%03d/%03d", busNumber, devNumber), os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	return file
}

func usbStringDescriptor(node *os.File, index uint8) string {
	const usbdevfsControl = 0xC0185500
	buffer := make([]byte, 255)
	request := struct {
		requestType, request uint8
		value, index, length uint16
		timeout              uint32
		data                 unsafe.Pointer
	}{
		requestType: 0x80, request: 0x06,
		value: 0x0300 | uint16(index), index: 0x0409, length: uint16(len(buffer)),
		timeout: 1000, data: unsafe.Pointer(&buffer[0]),
	}
	read, _, errno := unix.Syscall(unix.SYS_IOCTL, node.Fd(), usbdevfsControl, uintptr(unsafe.Pointer(&request)))
	if errno != 0 {
		return ""
	}
	var decoded []rune
	for i := 2; i+1 < int(read); i += 2 {
		decoded = append(decoded, rune(uint16(buffer[i])|uint16(buffer[i+1])<<8))
	}
	return strings.TrimSpace(string(decoded))
}

func sysfsHas(dir, attr, want string) bool {
	data, err := os.ReadFile(filepath.Join(dir, attr))
	return err == nil && strings.EqualFold(strings.TrimSpace(string(data)), want)
}
