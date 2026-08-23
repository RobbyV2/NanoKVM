package presentation

import (
	"crypto/sha512"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	tracesDir = "testdata/traces"

	goldenBaseUID = "nanokvm-golden-base-uid"
	goldenUDC     = "4340000.usb"
	goldenDisk    = "/dev/mmcblk0p3"
)

type flags struct {
	hidOnly    bool
	ncm        bool
	rndis      bool
	disk       bool
	diskRO     bool
	bios       bool
	noWakeup   bool
	disableHID bool
}

func goldenCases() map[string]flags {
	modes := []struct {
		name    string
		hidOnly bool
	}{{"normal", false}, {"hidonly", true}}
	nets := []struct {
		name       string
		ncm, rndis bool
	}{{"none", false, false}, {"ncm", true, false}, {"rndis", false, true}, {"ncmrndis", true, true}}
	disks := []struct {
		name string
		on   bool
	}{{"nodisk", false}, {"disk", true}}

	cases := map[string]flags{}
	for _, mode := range modes {
		for _, net := range nets {
			for _, disk := range disks {
				name := mode.name + "." + net.name + "." + disk.name
				cases[name] = flags{hidOnly: mode.hidOnly, ncm: net.ncm, rndis: net.rndis, disk: disk.on}
			}
		}
	}

	base := flags{rndis: true, disk: true}
	for name, mutate := range map[string]func(*flags){
		"normal.rndis.disk.bios":       func(f *flags) { f.bios = true },
		"normal.rndis.disk.notwakeup":  func(f *flags) { f.noWakeup = true },
		"normal.rndis.disk.disablehid": func(f *flags) { f.disableHID = true },
		"normal.rndis.disk.diskro":     func(f *flags) { f.diskRO = true },
	} {
		delta := base
		mutate(&delta)
		cases[name] = delta
	}
	return cases
}

func goldenMACs() (string, string) {
	sum := sha512.Sum512([]byte(goldenBaseUID))
	uid := hex.EncodeToString(sum[:])[:4]
	return "48:da:35:6e:" + uid[:2] + ":" + uid[2:], "48:da:35:6d:" + uid[:2] + ":" + uid[2:]
}

func profileForFlags(f flags) Profile {
	if f.hidOnly {
		profile := hidOnlyProfile()
		setWakeup(profile.Functions, !f.noWakeup)
		return profile
	}

	profile := standardProfile()
	hid := profile.Functions
	if f.disableHID {
		hid = nil
	} else {
		setWakeup(hid, !f.noWakeup)
		if f.bios {
			for _, function := range hid {
				function.HID.SubClass = 1
			}
		}
	}

	dev, host := goldenMACs()
	var functions []Function
	switch {
	case f.ncm:
		functions = append(functions, Function{Kind: FunctionNCM, Instance: "usb0", Net: &NetFunction{
			DevAddr: &dev, HostAddr: &host, CompatibleID: "WINNCM",
		}})
	case f.rndis:
		functions = append(functions, Function{Kind: FunctionRNDIS, Instance: "usb0", Net: &NetFunction{
			DevAddr:      &dev,
			HostAddr:     &host,
			SubClass:     ptr[uint8](0x01),
			Protocol:     ptr[uint8](0x03),
			CompatibleID: "RNDIS", SubCompatibleID: "5162001",
		}})
	}
	if len(functions) > 0 {
		profile.OSDesc = &OSDesc{VendorCode: "0xCD", QwSign: "MSFT100"}
	}

	functions = append(functions, hid...)
	if f.disk {
		functions = append(functions, Function{Kind: FunctionMassStorage, Instance: "disk0", Storage: &StorageFunction{
			Removable: true, ReadOnly: f.diskRO, InquiryString: InquiryString, File: goldenDisk,
		}})
	}
	profile.Functions = functions
	return profile
}

func setWakeup(functions []Function, on bool) {
	for _, function := range functions {
		if function.HID != nil {
			function.HID.WakeupOnWrite = on
		}
	}
}

// renderTrace speaks the format gen_traces.sh records. The leading mkdir is the
// gadget directory itself, which NewConfigFSOps creates and the plan therefore
// never contains, and the two trailing ops carry the bytes the applier writes:
// the single UDC name from ListUDC, and the otg role with the newline
// ConfigFSOps.SetOTGRole appends.
func renderTrace(plan Plan) []string {
	lines := []string{"mkdir\tg0"}

	for _, op := range plan.Ops {
		switch op.Kind {
		case OpMkdir:
			lines = append(lines, "mkdir\t"+op.Path)
		case OpWrite:
			lines = append(lines, "write\t"+op.Path+"\t"+hex.EncodeToString(op.Data))
		case OpSymlink:
			lines = append(lines, "symlink\t"+op.Path+"\t"+op.Target)
		case OpBind:
			lines = append(lines, "write\t"+op.Path+"\t"+hex.EncodeToString([]byte(goldenUDC+"\n")))
		case OpOTGRole:
			lines = append(lines, "otg\t"+hex.EncodeToString([]byte(string(op.Data)+"\n")))
		default:
			lines = append(lines, op.Kind.String()+"\t"+op.Path)
		}
	}
	return lines
}

func readTrace(t *testing.T, name string) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(tracesDir, name+".trace"))
	if err != nil {
		t.Fatalf("read golden trace: %v", err)
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func compileFlags(t *testing.T, f flags) Plan {
	t.Helper()

	plan, err := Compile(profileForFlags(f), staticV1)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return plan
}

func TestGoldenTraces(t *testing.T) {
	for name, f := range goldenCases() {
		t.Run(name, func(t *testing.T) {
			want := readTrace(t, name)
			got := renderTrace(compileFlags(t, f))

			for i := 0; i < len(want) || i < len(got); i++ {
				switch {
				case i >= len(got):
					t.Fatalf("op %d: missing, script has %q", i, want[i])
				case i >= len(want):
					t.Fatalf("op %d: compiled %q, script stops here", i, got[i])
				case want[i] != got[i]:
					t.Fatalf("op %d:\n script   %q\n compiled %q", i, want[i], got[i])
				}
			}
		})
	}
}

func TestGoldenTracesCoverEveryRecordedTrace(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join(tracesDir, "*.trace"))
	if err != nil {
		t.Fatalf("glob traces: %v", err)
	}

	cases := goldenCases()
	if len(entries) != len(cases) {
		t.Fatalf("%d recorded traces, %d compiled cases", len(entries), len(cases))
	}
	for _, entry := range entries {
		name := strings.TrimSuffix(filepath.Base(entry), ".trace")
		if _, ok := cases[name]; !ok {
			t.Fatalf("recorded trace %s has no compiled case", name)
		}
	}
}

func TestEveryTraceWritesItsModeMarkerExactlyOnce(t *testing.T) {
	for name, f := range goldenCases() {
		t.Run(name, func(t *testing.T) {
			marker, mode := BCDDeviceNormal, ModeNormal
			if f.hidOnly {
				marker, mode = BCDDeviceHIDOnly, ModeHIDOnly
			}
			want := "write\tbcdDevice\t" + hex.EncodeToString([]byte(marker+"\n"))

			for source, lines := range map[string][]string{"script": readTrace(t, name), "compiled": renderTrace(compileFlags(t, f))} {
				var found []string
				for _, line := range lines {
					if strings.HasPrefix(line, "write\tbcdDevice\t") {
						found = append(found, line)
					}
				}
				if len(found) != 1 || found[0] != want {
					t.Fatalf("%s writes %v, want exactly %q", source, found, want)
				}
			}

			got, err := modeFromBCDDevice(marker)
			if err != nil || got != mode {
				t.Fatalf("modeFromBCDDevice(%q) = %q %v, want %q", marker, got, err, mode)
			}
		})
	}
}

func TestHIDOnlyIgnoresTheNetworkAndDiskSentinels(t *testing.T) {
	want := renderTrace(compileFlags(t, flags{hidOnly: true}))

	for name, f := range goldenCases() {
		if !f.hidOnly {
			continue
		}
		got := renderTrace(compileFlags(t, f))
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%s diverges from the bare hid-only trace", name)
		}
	}
}

func TestNCMWinsOverRNDIS(t *testing.T) {
	both := renderTrace(compileFlags(t, flags{ncm: true, rndis: true, disk: true}))
	ncm := renderTrace(compileFlags(t, flags{ncm: true, disk: true}))

	if strings.Join(both, "\n") != strings.Join(ncm, "\n") {
		t.Fatal("ncm+rndis diverges from ncm alone, S03usbdev:53 gives ncm absolute priority")
	}
}

func TestLinkOrderIsTheInterfaceNumberOrder(t *testing.T) {
	plan := compileFlags(t, flags{rndis: true, disk: true})

	var links []string
	for _, op := range plan.Ops {
		if op.Kind == OpSymlink && strings.HasPrefix(op.Path, configPrefix+"/") {
			links = append(links, strings.TrimPrefix(op.Path, configPrefix+"/"))
		}
	}

	want := []string{"rndis.usb0", "hid.GS0", "hid.GS1", "hid.GS2", "mass_storage.disk0"}
	if strings.Join(links, ",") != strings.Join(want, ",") {
		t.Fatalf("link order = %v, want %v", links, want)
	}
}

// Compile plans a virgin gadget, so it links HID functions and never touches
// one that is already there. The unlink a live gadget needs to release
// opts->refcnt is Reconcile's, and it goes back in the same plan.
func TestPlanNeverRemovesAHIDFunction(t *testing.T) {
	for _, f := range []flags{{}, {hidOnly: true}, {}, {disk: true}} {
		for _, op := range compileFlags(t, f).Ops {
			if op.Kind == OpUnbind || (op.Kind == OpUnlink && strings.Contains(op.Path, string(FunctionHID)+".")) {
				t.Fatalf("compile emits %s %s against a gadget it is building from nothing", op.Kind, op.Path)
			}
		}
	}

	var order []string
	for _, op := range compileFlags(t, flags{}).Ops {
		if op.Kind == OpMkdir && strings.HasPrefix(op.Path, functionsDir+"/hid.") {
			order = append(order, strings.TrimPrefix(op.Path, functionsDir+"/hid."))
		}
	}
	if strings.Join(order, ",") != "GS0,GS1,GS2" {
		t.Fatalf("hid mkdir order = %v, want GS0,GS1,GS2", order)
	}
}

func TestCompiledPathsStayInsideTheGadget(t *testing.T) {
	for name, f := range goldenCases() {
		t.Run(name, func(t *testing.T) {
			for _, op := range compileFlags(t, f).Ops {
				if op.Kind == OpOTGRole {
					continue
				}
				if err := validateRel(op.Path); err != nil {
					t.Fatalf("%s %s: %v", op.Kind, op.Path, err)
				}
			}
		})
	}
}

func TestCompileRefusesAnOverBudgetProfile(t *testing.T) {
	profile := profileForFlags(flags{ncm: true, disk: true})
	dev, host := goldenMACs()
	profile.Functions = append(profile.Functions, Function{Kind: FunctionRNDIS, Instance: "usb1", Net: &NetFunction{
		DevAddr: &dev, HostAddr: &host, CompatibleID: "RNDIS",
	}})

	_, err := Compile(profile, staticV1)
	if err == nil {
		t.Fatal("compile accepted a profile over the static-v1 budget")
	}
	if !strings.Contains(err.Error(), "rejected by capability table static-v1") {
		t.Fatalf("err = %v, want the capability source carried", err)
	}
}

// Two HID functions is a layout now. A gap in the instances is not, and never
// can be: f_hid hands out /dev/hidgN in mkdir order.
func TestCompileRefusesAGapInTheHIDInstances(t *testing.T) {
	profile := standardProfile()
	profile.Functions = []Function{profile.Functions[0], profile.Functions[2]}
	profile.Normalize()

	_, err := Compile(profile, staticV1)
	if err == nil {
		t.Fatal("compile accepted hid.GS0 followed by hid.GS2")
	}
	if !strings.Contains(err.Error(), `hid function 1 is "GS2", want "GS1"`) {
		t.Fatalf("err = %v, want the instance mismatch named", err)
	}

	prefix := standardProfile()
	prefix.Functions = prefix.Functions[:2]
	prefix.Normalize()
	if _, err := Compile(prefix, staticV1); err != nil {
		t.Fatalf("compile refused a two interface layout: %v", err)
	}
}

func TestPlanNamesItsProfile(t *testing.T) {
	plan := compileFlags(t, flags{rndis: true, disk: true})

	if plan.Profile != ProfileStandard {
		t.Fatalf("profile = %q, want %q", plan.Profile, ProfileStandard)
	}
}

func TestOSDescFollowsTheFunctionsInThePlan(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags flags
		want  []string
	}{
		{
			name:  "rndis sets it",
			flags: flags{rndis: true, disk: true},
			want: []string{
				"write\tos_desc/use\t" + hex.EncodeToString([]byte("1\n")),
				"write\tos_desc/b_vendor_code\t" + hex.EncodeToString([]byte(osDescVendorCode+"\n")),
				"write\tos_desc/qw_sign\t" + hex.EncodeToString([]byte(osDescQwSign+"\n")),
				"symlink\tos_desc/c.1\t" + configPrefix,
			},
		},
		{
			name:  "ncm sets it",
			flags: flags{ncm: true},
			want: []string{
				"write\tos_desc/use\t" + hex.EncodeToString([]byte("1\n")),
				"write\tos_desc/b_vendor_code\t" + hex.EncodeToString([]byte(osDescVendorCode+"\n")),
				"write\tos_desc/qw_sign\t" + hex.EncodeToString([]byte(osDescQwSign+"\n")),
				"symlink\tos_desc/c.1\t" + configPrefix,
			},
		},
		{
			name:  "no network function clears it",
			flags: flags{disk: true},
			want:  []string{"write\tos_desc/use\t" + hex.EncodeToString([]byte("0\n")), "unlink\tos_desc/c.1"},
		},
		{
			name:  "hid-only clears it",
			flags: flags{hidOnly: true},
			want:  []string{"write\tos_desc/use\t" + hex.EncodeToString([]byte("0\n")), "unlink\tos_desc/c.1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, line := range renderTrace(compileFlags(t, tc.flags)) {
				if strings.Contains(line, osDescDir+"/") && !strings.Contains(line, functionsDir+"/") {
					got = append(got, line)
				}
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("os_desc ops =\n%v\nwant\n%v", got, tc.want)
			}
		})
	}
}

func TestOSDescIgnoresAStaleProfileField(t *testing.T) {
	profile := profileForFlags(flags{disk: true})
	profile.OSDesc = MSOSDesc()

	plan, err := Compile(profile, staticV1)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, op := range plan.Ops {
		if op.Kind == OpWrite && op.Path == osDescDir+"/use" && string(op.Data) != "0\n" {
			t.Fatalf("os_desc/use = %q, want it cleared when no function needs it", op.Data)
		}
	}
}

func TestRNDISDropsTheDeadClassWriteAndPrefixesTheRest(t *testing.T) {
	got := map[string]string{}
	for _, op := range compileFlags(t, flags{rndis: true, disk: true}).Ops {
		prefix := functionsDir + "/" + string(FunctionRNDIS) + "." + netInstance + "/"
		if op.Kind == OpWrite && strings.HasPrefix(op.Path, prefix) {
			got[strings.TrimPrefix(op.Path, prefix)] = string(op.Data)
		}
	}

	if class, ok := got["class"]; ok {
		t.Fatalf("plan writes class=%q, but kstrtou8(page, 0, ...) never accepted the script's e0 and the IAD keeps the kernel default (H8)", class)
	}
	for attr, want := range map[string]string{"subclass": "0x01\n", "protocol": "0x03\n"} {
		if got[attr] != want {
			t.Fatalf("%s = %q, want %q", attr, got[attr], want)
		}
	}
}

// The plan is compiled before the apply and unlinkStale runs at apply time, so
// the operator can only be told what an apply takes away if the plan can answer
// it against the linkage that is up. Removes is that answer and the transaction
// unlinks exactly it.
func TestPlanOutcomeNamesWhatTheApplyWillRemove(t *testing.T) {
	tests := []struct {
		name     string
		flags    flags
		before   []string
		removes  []string
		hid      bool
		recovery string
	}{
		{
			name:     "a function leaving the composite strands the host driver",
			before:   []string{"rndis.usb0", "hid.GS0", "uvc.cam0", "mass_storage.disk0"},
			removes:  []string{"rndis.usb0", "uvc.cam0", "mass_storage.disk0"},
			hid:      true,
			recovery: RecoveryReboot,
		},
		{
			name:     "a camera is unlinked and rebuilt with the pipeline behind it",
			before:   []string{"hid.GS0", "uac2.mic0"},
			removes:  []string{"uac2.mic0"},
			hid:      true,
			recovery: RecoveryHDMIReset,
		},
		{
			name:     "the same linkage still costs a rebind",
			before:   []string{"hid.GS0", "hid.GS1", "hid.GS2"},
			removes:  []string{},
			hid:      true,
			recovery: RecoveryReconnect,
		},
		{
			name:     "nothing is left to drive the host with",
			flags:    flags{rndis: true, disk: true, disableHID: true},
			before:   []string{},
			removes:  []string{},
			hid:      false,
			recovery: RecoveryPowerCycle,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outcome := compileFlags(t, test.flags).Outcome(Snapshot{Linked: test.before})

			if strings.Join(outcome.Removes, ",") != strings.Join(test.removes, ",") {
				t.Fatalf("removes = %v, want %v", outcome.Removes, test.removes)
			}
			if outcome.HID != test.hid {
				t.Fatalf("hid = %v, want %v", outcome.HID, test.hid)
			}
			if outcome.Recovery != test.recovery {
				t.Fatalf("recovery = %q, want %q", outcome.Recovery, test.recovery)
			}
		})
	}
}

// The four fields a host keys its driver binding off are spread across three
// optional pointers in the profile. The plan resolves them, so a preview can
// say which device the target will see rather than which one was typed in.
func TestPlanCarriesTheResolvedDeviceIdentity(t *testing.T) {
	want := DeviceIdentity{
		VendorID:     "0x3346",
		ProductID:    "0x1009",
		BCDDevice:    BCDDeviceHIDOnly,
		Manufacturer: "sipeed",
		Product:      "NanoKVM",
	}
	if got := compileFlags(t, flags{hidOnly: true}).Device; got != want {
		t.Fatalf("identity = %+v, want %+v", got, want)
	}
	if got := compileFlags(t, flags{}).Device; got.Serial != "0123456789ABCDEF" || got.BCDDevice != BCDDeviceNormal {
		t.Fatalf("standard identity = %+v, want the normal marker and its serial", got)
	}
}

func linkedSnapshot(plan Plan) Snapshot {
	var snapshot Snapshot
	for _, op := range plan.Ops {
		if op.Kind != OpSymlink {
			continue
		}
		if name, ok := strings.CutPrefix(op.Path, configPrefix+"/"); ok {
			snapshot.Linked = append(snapshot.Linked, name)
		}
	}
	return snapshot
}

func opIndex(plan Plan, kind OpKind, path string) int {
	for i, op := range plan.Ops {
		if op.Kind == kind && op.Path == path {
			return i
		}
	}
	return -1
}

func firstWrite(plan Plan, name string) int {
	for i, op := range plan.Ops {
		if op.Kind == OpWrite && writtenFunction(op.Path) == name {
			return i
		}
	}
	return -1
}

// The state the device is in when the operator presses apply: S03usbdev and
// vm/virtual-device.go have linked every function already, so opts->refcnt is
// held and every attribute store the profile changes is -EBUSY.
func TestReconcileUnlinksALinkedFunctionBeforeItsAttributes(t *testing.T) {
	plan := compileFlags(t, flags{rndis: true, disk: true})
	got := Reconcile(linkedSnapshot(plan), plan)

	for _, name := range []string{"rndis.usb0", "hid.GS0", "hid.GS1", "hid.GS2"} {
		unlink := opIndex(got, OpUnlink, configPrefix+"/"+name)
		write := firstWrite(got, name)
		link := opIndex(got, OpSymlink, configPrefix+"/"+name)
		if unlink < 0 {
			t.Fatalf("%s keeps its link across its attribute writes", name)
		}
		if unlink > write || write > link {
			t.Fatalf("%s: unlink at %d, first write at %d, link at %d", name, unlink, write, link)
		}
	}

	// Ordering constraint 3: the LUN attributes carry no refcnt check and the
	// link precedes them, so an unlink in front of them would have nothing left
	// in the plan to put the link back.
	if i := opIndex(got, OpUnlink, configPrefix+"/"+diskFunctionName); i >= 0 {
		t.Fatalf("op %d unlinks %s, whose link precedes its attributes", i, diskFunctionName)
	}
}

func TestReconcileChangesNothingOnAVirginGadget(t *testing.T) {
	for name, f := range goldenCases() {
		t.Run(name, func(t *testing.T) {
			plan := compileFlags(t, f)
			got := strings.Join(renderTrace(Reconcile(Snapshot{}, plan)), "\n")
			if want := strings.Join(renderTrace(plan), "\n"); got != want {
				t.Fatal("reconcile inserted an unlink for a function nothing has linked")
			}
		})
	}
}

// What dropRedundantWrites leaves behind for a function the boot script already
// wrote the plan's own values into. Nothing is refused, so nothing is unlinked.
func TestReconcileSkipsAFunctionWhoseWritesWereDropped(t *testing.T) {
	plan := compileFlags(t, flags{})
	kept := plan
	kept.Ops = nil
	for _, op := range plan.Ops {
		if op.Kind == OpWrite && writtenFunction(op.Path) == "hid.GS0" {
			continue
		}
		kept.Ops = append(kept.Ops, op)
	}

	got := Reconcile(linkedSnapshot(kept), kept)
	if i := opIndex(got, OpUnlink, configPrefix+"/hid.GS0"); i >= 0 {
		t.Fatalf("op %d unlinks hid.GS0 for a plan that writes none of its attributes", i)
	}
	if opIndex(got, OpUnlink, configPrefix+"/hid.GS1") < 0 {
		t.Fatal("hid.GS1 still has writes and keeps its link")
	}
}

// A media function is rebuilt on every apply, so unlinkStale removes it before
// the first op runs. A second unlink here would be a second copy of that rule.
func TestReconcileLeavesAMediaFunctionToUnlinkStale(t *testing.T) {
	profile := standardProfile()
	profile.Functions = append(profile.Functions, defaultCamera(0, "cam"))
	profile.Normalize()

	plan, err := Compile(profile, staticV1)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	current := linkedSnapshot(plan)
	if i := opIndex(Reconcile(current, plan), OpUnlink, configPrefix+"/uvc.cam0"); i >= 0 {
		t.Fatalf("op %d unlinks uvc.cam0, which unlinkStale has already removed", i)
	}

	var removed bool
	for _, name := range plan.Outcome(current).Removes {
		removed = removed || name == "uvc.cam0"
	}
	if !removed {
		t.Fatal("unlinkStale no longer removes uvc.cam0, so reconcile has to")
	}
}

// R1.1 restated against what the kernel actually does: the /dev/hidgN minor is
// taken from an ida at mkdir and returned at rmdir, so only a removal under
// functions/hid.* renumbers. A config symlink is free to go as long as the plan
// puts it back, and every unlink reconcile inserts is followed by that link.
func TestReconcileNeverLeavesAFunctionUnlinked(t *testing.T) {
	for name, f := range goldenCases() {
		t.Run(name, func(t *testing.T) {
			plan := compileFlags(t, f)
			got := Reconcile(linkedSnapshot(plan), plan)

			for i, op := range got.Ops {
				if (op.Kind == OpUnlink || op.Kind == OpRmdir) && strings.HasPrefix(op.Path, functionsDir+"/hid.") {
					t.Fatalf("op %d %s %s renumbers /dev/hidgN", i, op.Kind, op.Path)
				}
				if op.Kind != OpUnlink || !strings.HasPrefix(op.Path, configPrefix+"/") {
					continue
				}
				if opIndex(got, OpSymlink, op.Path) < i {
					t.Fatalf("op %d unlinks %s and no later op links it again", i, op.Path)
				}
			}
		})
	}
}
