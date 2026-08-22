package presentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	return NewManager(NewStore(), ops, staticV0), ops
}

type fakeHID struct {
	mu      sync.Mutex
	events  []string
	openErr error
}

func (f *fakeHID) Lock()        { f.append("lock") }
func (f *fakeHID) Unlock()      { f.append("unlock") }
func (f *fakeHID) CloseNoLock() { f.append("close") }

func (f *fakeHID) OpenNoLockWithRetry(timeout, delay time.Duration) error {
	f.append(fmt.Sprintf("open %s %s", timeout, delay))
	return f.openErr
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
		case op.Kind == OpUnlink:
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

	plan, err := Compile(standardProfile(), staticV0)
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

	want := []string{"lock", "close", "open 2s 100ms", "unlock"}
	if strings.Join(h.Events(), ",") != strings.Join(want, ",") {
		t.Fatalf("hid bracket = %v, want %v", h.Events(), want)
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
