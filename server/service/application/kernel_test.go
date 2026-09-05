package application

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/service/bootslot"
)

const uenvFixture = "ab_if=mmc\nab_dev=0:1\nab_limit=1\nab_state=committed\nab_good=boot.sd\nab_alt=boot.alt\nsdboot=run ab_pre\n\x00"

var (
	oldKernel = []byte("\xd0\x0d\xfe\xedthe kernel that already boots")
	newKernel = []byte("\xd0\x0d\xfe\xedthe kernel this update carries")
)

func kernelRig(t *testing.T) (bootslot.Paths, *kernelPayload) {
	t.Helper()
	root := t.TempDir()
	boot := filepath.Join(root, "boot")
	cache := filepath.Join(root, "cache", kernelSubtree)
	for _, dir := range []string{boot, cache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string, data []byte) {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(boot, "uEnv.txt"), []byte(uenvFixture))
	write(filepath.Join(boot, "boot.sd"), oldKernel)
	write(filepath.Join(boot, "boot.alt"), oldKernel)
	write(filepath.Join(boot, "bootcnt"), []byte{7, 0, 0, 0})
	write(filepath.Join(root, "cmdline"), []byte("console=ttyS0 nanokvm_slot=good"))
	itb := filepath.Join(cache, "boot.itb")
	write(itb, newKernel)

	return bootslot.Paths{
		Root:          boot,
		Cmdline:       filepath.Join(root, "cmdline"),
		Pending:       filepath.Join(root, "kernel_pending"),
		ConfirmPath:   filepath.Join(root, "confirmed"),
		InstalledPath: filepath.Join(root, "kernel.version"),
	}, &kernelPayload{itb: itb, version: "2.9.0"}
}

// simulateBoot is u-boot's sdboot from scripts/ab/uEnv.txt.in: the alt slot is
// only reachable while ab_state is trial and the bumped counter is still
// within ab_limit, and a counter it cannot read at all falls back to good.
func simulateBoot(t *testing.T, slot bootslot.Paths) string {
	t.Helper()
	uenv, err := os.ReadFile(filepath.Join(slot.Root, "uEnv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if value(t, uenv, "ab_state") != "trial" {
		return "boot.sd"
	}
	counter, err := os.ReadFile(filepath.Join(slot.Root, "bootcnt"))
	if err != nil || len(counter) != 4 {
		return "boot.sd"
	}
	if binary.LittleEndian.Uint32(counter)+1 > 1 {
		return "boot.sd"
	}
	return "boot.alt"
}

func value(t *testing.T, uenv []byte, key string) string {
	t.Helper()
	for _, line := range strings.Split(string(uenv), "\n") {
		if after, ok := strings.CutPrefix(line, key+"="); ok {
			return after
		}
	}
	t.Fatalf("uEnv.txt carries no %s", key)
	return ""
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Cutting the power between any two steps must leave a device that boots. The
// reverse ordering is the one that bricks: flipping ab_state before boot.alt
// holds a whole kernel makes a partial write selectable.
func TestKernelInstallSurvivesPowerLossAtEveryStep(t *testing.T) {
	for cut := 0; cut <= 5; cut++ {
		slot, payload := kernelRig(t)
		steps := kernelInstallSteps(payload, slot)
		for i := range cut {
			if err := steps[i].run(); err != nil {
				t.Fatalf("step %q: %v", steps[i].name, err)
			}
		}

		booted := simulateBoot(t, slot)
		if cut < 4 {
			if booted != "boot.sd" {
				t.Errorf("power loss after step %d boots %s, want the committed slot", cut, booted)
			}
			if got := read(t, filepath.Join(slot.Root, "boot.sd")); !bytes.Equal(got, oldKernel) {
				t.Errorf("power loss after step %d changed the committed kernel", cut)
			}
			continue
		}
		if booted != "boot.alt" {
			t.Errorf("power loss after step %d boots %s, want the trial slot", cut, booted)
		}
		if got := read(t, filepath.Join(slot.Root, "boot.alt")); !bytes.Equal(got, newKernel) {
			t.Errorf("power loss after step %d selects a trial slot that is not the package's kernel", cut)
		}
	}
}

// The write in step 1 is not atomic and cannot be: /boot has under 2 MiB free
// against a 7 MiB kernel, so there is no room for a temporary file beside it.
// What makes that safe is that ab_state has not moved yet.
func TestATornTrialWriteStillBootsTheCommittedSlot(t *testing.T) {
	slot, payload := kernelRig(t)
	if err := os.WriteFile(slot.Alt(), newKernel[:7], 0o644); err != nil {
		t.Fatal(err)
	}
	if booted := simulateBoot(t, slot); booted != "boot.sd" {
		t.Errorf("a torn trial write boots %s, want the committed slot", booted)
	}
	if err := installKernelPayload(payload, slot); err != nil {
		t.Fatalf("install after a torn write: %v", err)
	}
	if got := read(t, slot.Alt()); !bytes.Equal(got, newKernel) {
		t.Error("the retry did not replace the torn trial kernel")
	}
}

func TestKernelInstallZeroesTheCounterAndRecordsThePendingVersion(t *testing.T) {
	slot, payload := kernelRig(t)
	if err := installKernelPayload(payload, slot); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(slot.Root, "bootcnt")); !bytes.Equal(got, make([]byte, 4)) {
		t.Errorf("bootcnt = %v, want four zero bytes", got)
	}
	if got := strings.TrimSpace(string(read(t, slot.Pending))); got != "2.9.0" {
		t.Errorf("pending marker = %q, want 2.9.0", got)
	}
	uenv := read(t, filepath.Join(slot.Root, "uEnv.txt"))
	if !bytes.HasSuffix(uenv, []byte("sdboot=run ab_pre\n\x00")) {
		t.Errorf("the trial flip disturbed the sdboot line or the \\n\\0 terminator: %q", uenv)
	}
}

// Nothing may be written to /boot on a device whose bootloader has no slot
// policy: there is no second slot to fall back to.
func TestKernelInstallRefusesWithoutASlotPolicy(t *testing.T) {
	slot, payload := kernelRig(t)
	if err := os.WriteFile(slot.Cmdline, []byte("console=ttyS0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installKernelPayload(payload, slot); err == nil {
		t.Fatal("a pre-A/B bootloader accepted a kernel install")
	}
	if got := read(t, slot.Alt()); !bytes.Equal(got, oldKernel) {
		t.Error("the trial slot was written on a device with no slot policy")
	}
}

// A boot from the trial slot may take the next kernel only once its trial is
// confirmed: until then boot.alt is the running kernel and the one a rollback
// would abandon. After Confirm, boot.sd holds a copy and the slot is free.
func TestKernelInstallGateOnTheRunningSlot(t *testing.T) {
	armTrial := func(t *testing.T, slot bootslot.Paths) {
		t.Helper()
		if err := os.WriteFile(slot.Cmdline, []byte("console=ttyS0 nanokvm_slot=trial"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := slot.SetState(bootslot.StateTrial); err != nil {
			t.Fatal(err)
		}
		if err := slot.MarkPending("2.8.0"); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name    string
		prepare func(t *testing.T, slot bootslot.Paths)
		refused string
	}{
		{"good slot proceeds", func(*testing.T, bootslot.Paths) {}, ""},
		{"unconfirmed trial is refused", armTrial, "still unconfirmed"},
		{"confirmed trial proceeds", func(t *testing.T, slot bootslot.Paths) {
			armTrial(t, slot)
			if err := slot.Confirm(); err != nil {
				t.Fatal(err)
			}
		}, ""},
		{"trial whose commit failed and rolls back is refused", func(t *testing.T, slot bootslot.Paths) {
			armTrial(t, slot)
			if err := os.WriteFile(slot.ConfirmPath, nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}, "rolls back"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			slot, payload := kernelRig(t)
			tc.prepare(t, slot)
			err := installKernelPayload(payload, slot)
			if tc.refused != "" {
				if err == nil {
					t.Fatal("a kernel was installed over the slot the device is running from")
				}
				if !strings.Contains(err.Error(), tc.refused) {
					t.Errorf("refusal = %q, want it to mention %q", err, tc.refused)
				}
				if got := read(t, slot.Alt()); !bytes.Equal(got, oldKernel) {
					t.Error("a refused install still wrote the trial slot")
				}
				return
			}
			if err != nil {
				t.Fatalf("install: %v", err)
			}
			if booted := simulateBoot(t, slot); booted != "boot.alt" {
				t.Errorf("the next boot picks %s, want the trial slot", booted)
			}
			if got := read(t, slot.Alt()); !bytes.Equal(got, newKernel) {
				t.Error("the trial slot does not hold the package's kernel")
			}
			if got := read(t, filepath.Join(slot.Root, "boot.sd")); !bytes.Equal(got, oldKernel) {
				t.Error("the install changed the committed kernel")
			}
			if got := strings.TrimSpace(string(read(t, slot.Pending))); got != "2.9.0" {
				t.Errorf("pending marker = %q, want 2.9.0", got)
			}
		})
	}
}

func TestStagedKernelIsRejectedWhenTheReadBackDiffers(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "boot.alt")
	if err := os.WriteFile(dst, newKernel[:9], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyStaged(dst, bytes.Repeat([]byte{0}, 32)); err == nil {
		t.Fatal("a trial kernel that does not match the package was accepted")
	}
}

// A rolled back device, and one whose reboot never happened, both keep
// ab_state=trial. Writing the next kernel into an armed slot would make a torn
// write selectable, so the install disarms before it writes.
func TestAnInstallOverAnArmedTrialDisarmsFirst(t *testing.T) {
	slot, payload := kernelRig(t)
	if err := slot.SetState(bootslot.StateTrial); err != nil {
		t.Fatal(err)
	}
	if err := slot.ResetBootCount(); err != nil {
		t.Fatal(err)
	}
	if booted := simulateBoot(t, slot); booted != "boot.alt" {
		t.Fatalf("the armed precondition boots %s, want the trial slot", booted)
	}

	steps := kernelInstallSteps(payload, slot)
	if err := steps[0].run(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(slot.Alt(), newKernel[:7], 0o644); err != nil {
		t.Fatal(err)
	}
	if booted := simulateBoot(t, slot); booted != "boot.sd" {
		t.Errorf("a torn write over an armed trial boots %s, want the committed slot", booted)
	}
}
