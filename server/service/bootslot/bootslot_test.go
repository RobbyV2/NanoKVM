package bootslot

import (
	"os"
	"path/filepath"
	"testing"
)

func rig(t *testing.T, cmdline, uenv string) Paths {
	t.Helper()
	root := t.TempDir()
	if uenv != "" {
		if err := os.WriteFile(filepath.Join(root, "uEnv.txt"), []byte(uenv), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"boot.sd", "boot.alt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "bootcnt"), []byte{1, 0, 0, 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := filepath.Join(root, "cmdline")
	if err := os.WriteFile(cmd, []byte(cmdline), 0o644); err != nil {
		t.Fatal(err)
	}
	return Paths{
		Root:        root,
		Cmdline:     cmd,
		Pending:     filepath.Join(root, "kernel_pending"),
		ConfirmPath: filepath.Join(root, "confirmed"),
	}
}

const uenvFixture = "ab_if=mmc\nab_state=trial\nab_good=boot.sd\nsdboot=run ab_pre\n\x00"

func TestConfirmRewritesOnlyTheStateLine(t *testing.T) {
	p := rig(t, "console=ttyS0 nanokvm_slot=trial", uenvFixture)
	if err := p.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(p.Root, "uEnv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "ab_if=mmc\nab_state=committed\nab_good=boot.sd\nsdboot=run ab_pre\n\x00"
	if string(got) != want {
		t.Errorf("uEnv.txt = %q, want %q", got, want)
	}
}

func TestConfirmCommitsTheTrialKernelAndResetsTheCounter(t *testing.T) {
	p := rig(t, "nanokvm_slot=trial", uenvFixture)
	if err := p.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	good, _ := os.ReadFile(filepath.Join(p.Root, "boot.sd"))
	if string(good) != "boot.alt" {
		t.Errorf("boot.sd = %q, want the trial kernel", good)
	}
	cnt, _ := os.ReadFile(filepath.Join(p.Root, "bootcnt"))
	if len(cnt) != 4 || cnt[0] != 0 {
		t.Errorf("bootcnt = %v, want four zero bytes", cnt)
	}
	if _, err := os.Stat(p.ConfirmPath); err != nil {
		t.Errorf("trial guard was never signalled: %v", err)
	}
}

func TestConfirmDoesNothingOnACommittedBoot(t *testing.T) {
	p := rig(t, "nanokvm_slot=good", uenvFixture)
	if err := p.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	good, _ := os.ReadFile(filepath.Join(p.Root, "boot.sd"))
	if string(good) != "boot.sd" {
		t.Errorf("a committed boot overwrote boot.sd with %q", good)
	}
	if _, err := os.Stat(p.ConfirmPath); err == nil {
		t.Error("a committed boot signalled the trial guard")
	}
}

func TestRolledBackReportsTheVersionThatFailed(t *testing.T) {
	p := rig(t, "nanokvm_slot=good", uenvFixture)
	if err := os.WriteFile(p.Pending, []byte("2.8.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	version, ok := p.RolledBack()
	if !ok || version != "2.8.0" {
		t.Errorf("RolledBack() = %q, %v, want 2.8.0, true", version, ok)
	}
}

func TestRolledBackIsSilentDuringATrial(t *testing.T) {
	p := rig(t, "nanokvm_slot=trial", uenvFixture)
	if err := os.WriteFile(p.Pending, []byte("2.8.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.RolledBack(); ok {
		t.Error("a trial boot reported itself as a rollback")
	}
}

func TestSlotIsEmptyWithoutTheBootloaderPolicy(t *testing.T) {
	p := rig(t, "console=ttyS0 root=/dev/mmcblk0p2", uenvFixture)
	if got := p.Slot(); got != "" {
		t.Errorf("Slot() = %q, want empty on a pre-A/B bootloader", got)
	}
}
