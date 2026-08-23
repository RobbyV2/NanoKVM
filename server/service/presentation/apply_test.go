package presentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestManager(t *testing.T, flags ...string) (*Manager, *RecordOps) {
	t.Helper()

	useTestPresentationDir(t)
	useTestBootDir(t, flags...)
	ops := NewRecordOps()
	return NewManager(NewStore(), ops, staticV1), ops
}

type fakeHID struct {
	mu       sync.Mutex
	events   []string
	openErr  error
	openErrs []error
	writeErr error
}

func (f *fakeHID) Lock()        { f.append("lock") }
func (f *fakeHID) Unlock()      { f.append("unlock") }
func (f *fakeHID) CloseNoLock() { f.append("close") }

func (f *fakeHID) OpenNoLockWithRetry(timeout, delay time.Duration) error {
	f.append(fmt.Sprintf("open %s %s", timeout, delay))

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.openErrs) != 0 {
		err := f.openErrs[0]
		f.openErrs = f.openErrs[1:]
		return err
	}
	return f.openErr
}

func (f *fakeHID) WriteKeyboardReport(data []byte) error      { return f.write("keyboard", data) }
func (f *fakeHID) WriteRelativeMouseReport(data []byte) error { return f.write("relative", data) }
func (f *fakeHID) WriteAbsoluteMouseReport(data []byte) error { return f.write("absolute", data) }

func (f *fakeHID) write(device string, data []byte) error {
	f.append(fmt.Sprintf("%s %x", device, data))

	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writeErr
}

func (f *fakeHID) failOpens(count int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := 0; i < count; i++ {
		f.openErrs = append(f.openErrs, err)
	}
}

func (f *fakeHID) append(event string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeHID) Events() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.events...)
}

func TestApplyCycleNeverRemovesAHIDFunction(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	for _, name := range []string{ProfileStandard, ProfileHIDOnly, ProfileStandard} {
		if err := manager.Apply(ctx, name); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}

	if err := os.WriteFile(filepath.Join(bootDir, sentinelDisk), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.ApplyProfile(ctx, derivedProfile()); err != nil {
		t.Fatalf("apply standard plus disk: %v", err)
	}

	var order []string
	for _, op := range ops.Trace() {
		switch {
		case op.Kind == OpUnlink && strings.Contains(op.Path, string(FunctionHID)+"."):
			t.Fatalf("apply emits unlink %s, which can renumber /dev/hidgN", op.Path)
		case op.Kind == OpMkdir && strings.HasPrefix(op.Path, functionsDir+"/hid."):
			order = append(order, strings.TrimPrefix(op.Path, functionsDir+"/hid."))
		}
	}
	if want := "GS0,GS1,GS2,GS0,GS1,GS2,GS0,GS1,GS2,GS0,GS1,GS2"; strings.Join(order, ",") != want {
		t.Fatalf("hid mkdir order = %v, want %v", order, want)
	}

	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}
	if role := ops.Role(); role != OTGRoleDevice {
		t.Fatalf("otg role = %q, want %q", role, OTGRoleDevice)
	}

	store := NewStore()
	active, err := store.Active()
	if err != nil || active != ProfileCurrent {
		t.Fatalf("active = %q err = %v, want %q", active, err, ProfileCurrent)
	}
	lastKnownGood, err := store.LastKnownGood()
	if err != nil || lastKnownGood != ProfileCurrent {
		t.Fatalf("last known good = %q err = %v, want %q", lastKnownGood, err, ProfileCurrent)
	}
}

func TestUnlinkStaleSparesTheHIDLinks(t *testing.T) {
	manager, ops := newTestManager(t)

	plan, err := Compile(standardProfile(), staticV1)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	before := Snapshot{Linked: []string{"rndis.usb0", "hid.GS0", "mass_storage.disk0"}}
	if err := manager.unlinkStale(before, plan); err != nil {
		t.Fatalf("unlink stale: %v", err)
	}

	var unlinked []string
	for _, op := range ops.Trace() {
		if op.Kind == OpUnlink {
			unlinked = append(unlinked, op.Path)
		}
	}
	want := []string{configPrefix + "/rndis.usb0", configPrefix + "/mass_storage.disk0"}
	if strings.Join(unlinked, ",") != strings.Join(want, ",") {
		t.Fatalf("unlinked = %v, want %v", unlinked, want)
	}
}

func TestApplyQuiescesHIDAroundTheTransaction(t *testing.T) {
	manager, _ := newTestManager(t)
	h := &fakeHID{}
	manager.SetHID(h)

	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatalf("apply: %v", err)
	}

	want := []string{
		"lock", "close",
		"open 2s 100ms", "close",
		"open 2s 100ms", "unlock",
		"keyboard 0000000000000000", "relative 00000000", "absolute 000000000000",
	}
	if strings.Join(h.Events(), ",") != strings.Join(want, ",") {
		t.Fatalf("hid bracket = %v, want %v", h.Events(), want)
	}
}

// A host that has not enumerated the gadget refuses every report, which says
// nothing about the gadget. Failing the apply on it would refuse every profile
// change made while the attached machine is powered off.
func TestApplySurvivesAReleaseReportTheHostRefuses(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.SetHID(&fakeHID{writeErr: errors.New("cannot send after transport endpoint shutdown")})

	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

// The UDC attribute reads the same whether or not the kernel rebuilt
// /dev/hidgN, so the node check has to sit inside the transaction, where the
// rollback ladder is still available to it: the rung below the target is the
// profile that was running, and the rung below that is a keyboard.
func TestApplyLaddersDownWhenTheHIDNodesDoNotComeBack(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lost   int
		want   string
		active string
	}{
		{name: "the rollback gets them back", lost: 1, want: "rolled back to mutable", active: "mutable"},
		{name: "the rollback loses them too", lost: 2, want: "fell back to " + ProfileHIDOnly, active: ProfileHIDOnly},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager, ops := newTestManager(t)
			ctx := context.Background()
			h := &fakeHID{}
			manager.SetHID(h)
			previous := standardProfile()
			previous.Name = "mutable"
			previous.BuiltIn = false
			previous.Normalize()
			if err := manager.ApplyProfile(ctx, previous); err != nil {
				t.Fatalf("apply previous profile: %v", err)
			}

			target := previous
			target.Name = "target"
			target.Device.Product = "Failed update"
			target.Normalize()

			missing := errors.New("no such device")
			h.failOpens(tc.lost, missing)
			err := manager.ApplyProfile(ctx, target)
			if !errors.Is(err, missing) {
				t.Fatalf("err = %v, want the missing hid nodes", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if bound := ops.Bound(); bound != dwc2Device {
				t.Fatalf("bound = %q, want %q", bound, dwc2Device)
			}
			active, err := manager.store.Active()
			if err != nil || active != tc.active {
				t.Fatalf("active = %q err = %v, want %q", active, err, tc.active)
			}
		})
	}
}

// The hybrid profile drops GS2 on purpose, so the node behind it is gone by
// design. The devices open as a set, so probing them here would read a bind
// that did exactly what it was asked as a failed one.
func TestTransientStartDoesNotProbeTheHIDNodeItRemoved(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()
	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}
	h := &fakeHID{}
	manager.SetHID(h)

	if _, err := manager.StartFunctionFS(ctx, testFunctionFS()); err != nil {
		t.Fatalf("start functionfs: %v", err)
	}
	want := []string{
		"lock", "close", "open 2s 100ms", "unlock",
		"keyboard 0000000000000000", "relative 00000000", "absolute 000000000000",
	}
	if strings.Join(h.Events(), ",") != strings.Join(want, ",") {
		t.Fatalf("hid bracket = %v, want %v", h.Events(), want)
	}
}

// The rung below the rollback. Both the target and the profile it rolls back to
// failed, so the choice is a half-built gadget or the smallest one that still
// carries a keyboard.
func TestRollbackFailureFallsBackToHIDOnly(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	previous := standardProfile()
	previous.Name = "mutable"
	previous.BuiltIn = false
	previous.Functions = append([]Function{NetworkFunction(FunctionRNDIS)}, previous.Functions...)
	previous.OSDesc = MSOSDesc()
	previous.Normalize()
	if err := manager.ApplyProfile(ctx, previous); err != nil {
		t.Fatalf("apply previous profile: %v", err)
	}

	target := previous
	target.Name = "target"
	target.Device.Product = "Failed update"
	target.Normalize()
	path := functionsDir + "/hid.GS0/report_length"
	applyErr := errors.New("target write failed")
	rollbackErr := errors.New("rollback write failed")
	ops.FailWriteOnce(path, applyErr)
	ops.FailWriteOnce(path, rollbackErr)

	err := manager.ApplyProfile(ctx, target)
	if !errors.Is(err, applyErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("err = %v, want the target and rollback failures", err)
	}
	if !strings.Contains(err.Error(), "fell back to "+ProfileHIDOnly) {
		t.Fatalf("err = %v, want the hid-only fallback detail", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}

	links := ops.Links()
	for _, instance := range hidInstances {
		if _, ok := links[configPrefix+"/hid."+instance]; !ok {
			t.Fatalf("fallback dropped hid.%s, the operator has no keyboard", instance)
		}
	}
	if target, ok := links[configPrefix+"/rndis.usb0"]; ok {
		t.Fatalf("fallback kept the network link %q", target)
	}
	active, err := manager.store.Active()
	if err != nil || active != ProfileHIDOnly {
		t.Fatalf("active = %q err = %v, want %q", active, err, ProfileHIDOnly)
	}
}

// hid-only is the rung below the rollback, so a rollback that was already
// hid-only has none left under it. Running the same restore a second time is
// another unbind and bind of a gadget that just refused one.
func TestHIDOnlyRollbackFailureDoesNotRepeatItself(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	if err := manager.Apply(ctx, ProfileHIDOnly); err != nil {
		t.Fatalf("apply hid-only: %v", err)
	}

	target := standardProfile()
	target.Name = "target"
	target.BuiltIn = false
	target.Normalize()
	path := functionsDir + "/hid.GS0/report_length"
	ops.FailWriteOnce(path, errors.New("target write failed"))
	ops.FailWriteOnce(path, errors.New("rollback write failed"))

	err := manager.ApplyProfile(ctx, target)
	if strings.Contains(err.Error(), "fell back") || strings.Contains(err.Error(), "fall back") {
		t.Fatalf("err = %v, want no fallback below the rollback it already is", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}
}

// The fallback is the last rung, so a failure in it has to leave the controller
// bound rather than become the thing that took the gadget away.
func TestHIDOnlyFallbackFailureStillLeavesTheUDCBound(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}

	target := standardProfile()
	target.Name = "target"
	target.BuiltIn = false
	target.Normalize()
	path := functionsDir + "/hid.GS0/report_length"
	for _, err := range []error{errors.New("target write failed"), errors.New("rollback write failed"), errors.New("fallback write failed")} {
		ops.FailWriteOnce(path, err)
	}

	err := manager.ApplyProfile(ctx, target)
	if !strings.Contains(err.Error(), "fall back to "+ProfileHIDOnly) {
		t.Fatalf("err = %v, want the failed fallback named", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want emergency bind to %q", bound, dwc2Device)
	}
}

func TestApplyFailsWhenTheHIDNodesDoNotComeBack(t *testing.T) {
	manager, _ := newTestManager(t)
	missing := errors.New("no such device")
	manager.SetHID(&fakeHID{openErr: missing})

	err := manager.Apply(context.Background(), ProfileStandard)
	if !errors.Is(err, missing) {
		t.Fatalf("err = %v, want the reopen failure", err)
	}
}

func TestApplyRollsBackToLastKnownGood(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	previous := standardProfile()
	previous.Name = "mutable"
	previous.BuiltIn = false
	previous.Functions = append([]Function{NetworkFunction(FunctionRNDIS)}, previous.Functions...)
	previous.OSDesc = MSOSDesc()
	previous.Normalize()
	if err := manager.ApplyProfile(ctx, previous); err != nil {
		t.Fatalf("apply previous profile: %v", err)
	}
	if err := manager.store.SetActive(ProfileHIDOnly); err != nil {
		t.Fatal(err)
	}

	target := previous
	target.Device.Product = "Failed update"
	target.Functions = slices.DeleteFunc(target.Functions, func(function Function) bool {
		return function.Kind.isNet()
	})
	transient := NetworkFunction(FunctionNCM)
	transient.Instance = "transient"
	target.Functions = append([]Function{transient}, target.Functions...)
	target.OSDesc = MSOSDesc()
	target.Normalize()

	wantErr := errors.New("configfs write failed")
	ops.FailWriteOnce(functionsDir+"/hid.GS0/report_length", wantErr)
	err := manager.ApplyProfile(ctx, target)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want the configfs failure", err)
	}
	if !strings.Contains(err.Error(), "rolled back to "+previous.Name) {
		t.Fatalf("err = %v, want successful rollback detail", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}

	links := ops.Links()
	if target := links[configPrefix+"/rndis.usb0"]; target != functionsDir+"/rndis.usb0" {
		t.Fatalf("rollback network link = %q, want rndis", target)
	}
	if target, ok := links[configPrefix+"/ncm.transient"]; ok {
		t.Fatalf("rollback kept transient network link %q", target)
	}
	active, err := manager.store.Active()
	if err != nil || active != previous.Name {
		t.Fatalf("active = %q err = %v, want %q", active, err, previous.Name)
	}
	lastKnownGood, err := manager.store.LastKnownGood()
	if err != nil || lastKnownGood != previous.Name {
		t.Fatalf("last known good = %q err = %v, want %q", lastKnownGood, err, previous.Name)
	}
	stored, err := manager.store.LoadProfile(previous.Name)
	if err != nil || stored.Device.Product != previous.Device.Product {
		t.Fatalf("stored product = %q err = %v, want %q", stored.Device.Product, err, previous.Device.Product)
	}
}

func TestApplyFailureKeepsTheUDCBoundWhenRollbackFails(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}

	target := standardProfile()
	target.Name = "target"
	target.BuiltIn = false
	target.Normalize()
	path := functionsDir + "/hid.GS0/report_length"
	applyErr := errors.New("target write failed")
	rollbackErr := errors.New("rollback write failed")
	ops.FailWriteOnce(path, applyErr)
	ops.FailWriteOnce(path, rollbackErr)

	err := manager.ApplyProfile(ctx, target)
	if !errors.Is(err, applyErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("err = %v, want target and rollback failures", err)
	}
	if !strings.Contains(err.Error(), "rollback to "+ProfileStandard) {
		t.Fatalf("err = %v, want rollback diagnostic", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want emergency bind to %q", bound, dwc2Device)
	}
}

func TestRebindAndResetPHYVerifyTheBind(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	if err := manager.Rebind(ctx); err != nil {
		t.Fatalf("rebind: %v", err)
	}
	if err := manager.ResetPHY(ctx); err != nil {
		t.Fatalf("reset phy: %v", err)
	}
	if ops.PHYResets() != 1 {
		t.Fatalf("phy resets = %d, want 1", ops.PHYResets())
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want %q", bound, dwc2Device)
	}

	ops.SetUDC()
	if err := manager.Rebind(ctx); !errors.Is(err, ErrUDCCount) {
		t.Fatalf("err = %v, want the udc count error (H4)", err)
	}
}

func TestGadgetMutatorsSerialize(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.SetHID(&fakeHID{})
	ctx := context.Background()

	var group sync.WaitGroup
	for _, mutate := range []func() error{
		func() error { return manager.Apply(ctx, ProfileStandard) },
		func() error { return manager.Apply(ctx, ProfileHIDOnly) },
		func() error { return manager.Rebind(ctx) },
		func() error { return manager.ResetPHY(ctx) },
		func() error { return manager.SetLUN(ctx, LUN{File: "/data/boot.iso", CDROM: true}) },
	} {
		group.Add(1)
		go func() {
			defer group.Done()
			for i := 0; i < 8; i++ {
				if err := mutate(); err != nil {
					t.Errorf("mutate: %v", err)
					return
				}
				if _, err := manager.Snapshot(); err != nil {
					t.Errorf("snapshot: %v", err)
					return
				}
			}
		}()
	}
	group.Wait()
}

func TestSetLUNReleasesTheBackingFileBeforeTheFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		lun  LUN
		want []string
	}{
		{
			name: "unmount",
			want: []string{"file=\n", "ro=0", "cdrom=0", "inquiry_string=" + InquiryString, "file=" + DefaultDiskFile},
		},
		{
			name: "cdrom",
			lun:  LUN{File: "/data/boot.iso", CDROM: true},
			want: []string{"file=\n", "ro=1", "cdrom=1", "inquiry_string=" + InquiryStringCDROM, "file=/data/boot.iso"},
		},
		{
			name: "mass storage leaves ro and cdrom alone",
			lun:  LUN{File: "/data/disk.img"},
			want: []string{"inquiry_string=" + InquiryString, "file=/data/disk.img"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager, ops := newTestManager(t)
			if err := manager.SetLUN(context.Background(), tc.lun); err != nil {
				t.Fatalf("set lun: %v", err)
			}

			var writes []string
			for _, op := range ops.Trace() {
				if op.Kind == OpWrite {
					writes = append(writes, strings.TrimPrefix(op.Path, lunAttr(""))+"="+string(op.Data))
				}
			}
			if strings.Join(writes, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("lun writes = %v, want %v", writes, tc.want)
			}
			if bound := ops.Bound(); bound != dwc2Device {
				t.Fatalf("bound = %q, want %q", bound, dwc2Device)
			}
		})
	}
}

func TestLUNReadsBackTheRuntimeState(t *testing.T) {
	manager, ops := newTestManager(t)
	for rel, data := range map[string]string{lunAttr("file"): DefaultDiskFile + "\n", lunAttr("cdrom"): "0\n"} {
		if err := ops.Seed(rel, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}

	lun, err := manager.LUN()
	if err != nil || lun != (LUN{}) {
		t.Fatalf("lun = %+v err = %v, want the unmounted zero value", lun, err)
	}

	if err := manager.SetLUN(context.Background(), LUN{File: "/data/boot.iso", CDROM: true}); err != nil {
		t.Fatalf("set lun: %v", err)
	}
	lun, err = manager.LUN()
	if err != nil || lun != (LUN{File: "/data/boot.iso", CDROM: true}) {
		t.Fatalf("lun = %+v err = %v, want the mounted cdrom", lun, err)
	}
}

func TestModeResolvesInThreeTiers(t *testing.T) {
	for _, tc := range []struct {
		name      string
		active    string
		bcdDevice string
		want      string
		wantErr   bool
	}{
		{name: "active profile beats the marker", active: ProfileStandard, bcdDevice: BCDDeviceHIDOnly, want: ModeNormal},
		{name: "active hid-only beats the marker", active: ProfileHIDOnly, bcdDevice: BCDDeviceNormal, want: ModeHIDOnly},
		{name: "migrated profile is normal", active: ProfileCurrent, bcdDevice: BCDDeviceHIDOnly, want: ModeNormal},
		{name: "exact hid-only marker", bcdDevice: BCDDeviceHIDOnly, want: ModeHIDOnly},
		{name: "exact normal marker", bcdDevice: BCDDeviceNormal, want: ModeNormal},
		{name: "kernel 5.15 default", bcdDevice: "0x0515", want: ModeNormal},
		{name: "kernel 5.9 default", bcdDevice: "0x0509", want: ModeNormal},
		{name: "unknown marker", bcdDevice: "0x0601", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager, ops := newTestManager(t)
			if err := ops.Seed(attrBCDDevice, []byte(tc.bcdDevice+"\n")); err != nil {
				t.Fatal(err)
			}
			if tc.active != "" {
				if err := manager.store.SetActive(tc.active); err != nil {
					t.Fatal(err)
				}
			}

			mode, err := manager.Mode()
			if tc.wantErr {
				if !errors.Is(err, ErrUnknownMode) {
					t.Fatalf("mode = %q err = %v, want the unknown mode error", mode, err)
				}
				return
			}
			if err != nil || mode != tc.want {
				t.Fatalf("mode = %q err = %v, want %q", mode, err, tc.want)
			}
		})
	}
}

func TestApplyClearsOSDescWhenTheNetworkFunctionGoes(t *testing.T) {
	manager, ops := newTestManager(t, sentinelRNDIS)
	ctx := context.Background()

	profile := derivedProfile()
	if err := manager.ApplyProfile(ctx, profile); err != nil {
		t.Fatalf("apply with rndis: %v", err)
	}
	if target := ops.Links()[osDescDir+"/"+configName]; target != configPrefix {
		t.Fatalf("os_desc link = %q, want %q", target, configPrefix)
	}
	if use := osDescUse(t, ops); use != "1\n" {
		t.Fatalf("os_desc/use = %q, want it set", use)
	}

	profile.Functions = slices.DeleteFunc(profile.Functions, func(f Function) bool { return f.Kind.isNet() })
	if err := manager.ApplyProfile(ctx, profile); err != nil {
		t.Fatalf("apply without rndis: %v", err)
	}
	if target, ok := ops.Links()[osDescDir+"/"+configName]; ok {
		t.Fatalf("os_desc link survives at %q, the gadget keeps answering the 0xEE request (H11)", target)
	}
	if use := osDescUse(t, ops); use != "0\n" {
		t.Fatalf("os_desc/use = %q, want it cleared", use)
	}
}

func osDescUse(t *testing.T, ops *RecordOps) string {
	t.Helper()

	data, err := ops.ReadFile(osDescDir + "/use")
	if err != nil {
		t.Fatalf("read os_desc/use: %v", err)
	}
	return string(data)
}
