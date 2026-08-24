package bootslot

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		Root:          root,
		Cmdline:       cmd,
		Pending:       filepath.Join(root, "kernel_pending"),
		ConfirmPath:   filepath.Join(root, "confirmed"),
		InstalledPath: filepath.Join(root, "kernel.version"),
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

// A trial kernel that reaches userspace but leaves the device unreachable is
// exactly the failure A/B exists for. Committing on the listener alone would
// make it permanent.
func TestATrialWithNoRoutableAddressIsNotCommitted(t *testing.T) {
	p := rig(t, "nanokvm_slot=trial", uenvFixture)
	err := p.ConfirmWhenReady(Ready{
		Serving:  func() bool { return true },
		Routable: func() bool { return false },
		Timeout:  30 * time.Millisecond,
		Interval: time.Millisecond,
	})
	if !errors.Is(err, ErrNotReady) {
		t.Fatalf("ConfirmWhenReady = %v, want ErrNotReady", err)
	}
	good, _ := os.ReadFile(filepath.Join(p.Root, "boot.sd"))
	if string(good) != "boot.sd" {
		t.Errorf("an unreachable device committed its trial kernel over %q", good)
	}
	uenv, _ := os.ReadFile(filepath.Join(p.Root, "uEnv.txt"))
	if !strings.Contains(string(uenv), "ab_state=trial") {
		t.Errorf("an unreachable device left ab_state at %q", uenv)
	}
	if _, err := os.Stat(p.ConfirmPath); err == nil {
		t.Error("an unreachable device signalled the trial guard")
	}
}

func TestATrialThatServesAndIsRoutableIsCommitted(t *testing.T) {
	p := rig(t, "nanokvm_slot=trial", uenvFixture)
	err := p.ConfirmWhenReady(Ready{
		Serving:  func() bool { return true },
		Routable: func() bool { return true },
		Timeout:  time.Second,
		Interval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ConfirmWhenReady: %v", err)
	}
	good, _ := os.ReadFile(filepath.Join(p.Root, "boot.sd"))
	if string(good) != "boot.alt" {
		t.Errorf("boot.sd = %q, want the trial kernel", good)
	}
}

func TestRoutableIgnoresLoopbackAndLinkLocal(t *testing.T) {
	unreachable := []net.Addr{
		&net.IPNet{IP: net.ParseIP("127.0.0.1")},
		&net.IPNet{IP: net.ParseIP("::1")},
		&net.IPNet{IP: net.ParseIP("169.254.7.1")},
		&net.IPNet{IP: net.ParseIP("fe80::1")},
	}
	if routable(unreachable) {
		t.Error("a device with only loopback and link-local addresses looked routable")
	}
	if !routable(append(unreachable, &net.IPNet{IP: net.ParseIP("192.168.1.20")})) {
		t.Error("a device with a LAN address did not look routable")
	}
}

func TestConfirmRecordsTheInstalledKernelVersion(t *testing.T) {
	p := rig(t, "nanokvm_slot=trial", uenvFixture)
	if err := p.MarkPending("2.9.0"); err != nil {
		t.Fatal(err)
	}
	if err := p.Confirm(); err != nil {
		t.Fatal(err)
	}
	if got := p.InstalledVersion(); got != "2.9.0" {
		t.Errorf("InstalledVersion() = %q, want 2.9.0", got)
	}
	if _, ok := p.RolledBack(); ok {
		t.Error("a committed kernel still reported a rollback")
	}
}

func TestClearRollbackSilencesTheWarning(t *testing.T) {
	p := rig(t, "nanokvm_slot=good", uenvFixture)
	if err := p.MarkPending("2.9.0"); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.RolledBack(); !ok {
		t.Fatal("a rolled back kernel was not reported")
	}
	if err := p.ClearRollback(); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.RolledBack(); ok {
		t.Error("a dismissed rollback came back")
	}
	if err := p.ClearRollback(); err != nil {
		t.Errorf("dismissing twice: %v", err)
	}
}

func TestSetStateRejectsAnUnknownState(t *testing.T) {
	p := rig(t, "nanokvm_slot=good", uenvFixture)
	if err := p.SetState("maybe"); !errors.Is(err, ErrState) {
		t.Fatalf("SetState(\"maybe\") = %v, want ErrState", err)
	}
	uenv, _ := os.ReadFile(filepath.Join(p.Root, "uEnv.txt"))
	if string(uenv) != uenvFixture {
		t.Errorf("a rejected state still rewrote uEnv.txt: %q", uenv)
	}
}

func TestSetStateKeepsTheEnvTerminatorAndLastLine(t *testing.T) {
	p := rig(t, "nanokvm_slot=good", "ab_state=committed\nab_limit=1\nsdboot=run ab_pre\n\x00")
	if err := p.SetState(StateTrial); err != nil {
		t.Fatal(err)
	}
	uenv, _ := os.ReadFile(filepath.Join(p.Root, "uEnv.txt"))
	want := "ab_state=trial\nab_limit=1\nsdboot=run ab_pre\n\x00"
	if string(uenv) != want {
		t.Errorf("uEnv.txt = %q, want %q", uenv, want)
	}
}
