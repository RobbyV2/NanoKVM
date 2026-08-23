package presentation

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// newTestConfigFSOps points ConfigFSOps at a plain directory. The dirfd it
// carries is an ordinary open(2) on its root, so nothing about the type needs
// configfs underneath it. WriteFile opens relative names without O_CREAT,
// which is what the real attributes want, so every attribute a test writes has
// to be created up front the way configfs would have.
func newTestConfigFSOps(t *testing.T) (*ConfigFSOps, string) {
	t.Helper()

	root := filepath.Join(t.TempDir(), "g0")
	ops, err := NewConfigFSOps(root)
	if err != nil {
		t.Fatalf("new configfs ops: %v", err)
	}
	t.Cleanup(func() { _ = ops.Close() })
	return ops, root
}

// The UDC attribute takes the name plus a newline. Nothing in the golden-trace
// suite sees these bytes: it runs through RecordOps and renders the bind line
// from a constant, so the newline lives and dies here.
func TestBindUDCWritesTheNameWithATrailingNewline(t *testing.T) {
	ops, root := newTestConfigFSOps(t)

	udc := filepath.Join(root, udcAttr)
	// A longer previous value catches a bind that forgets O_TRUNC and leaves a
	// tail behind rather than replacing the name outright.
	if err := os.WriteFile(udc, []byte("a-much-longer-previous-name\n"), 0o644); err != nil {
		t.Fatalf("seed %s: %v", udcAttr, err)
	}

	if err := ops.BindUDC(dwc2Device); err != nil {
		t.Fatalf("bind udc: %v", err)
	}

	got, err := os.ReadFile(udc)
	if err != nil {
		t.Fatalf("read %s: %v", udcAttr, err)
	}
	if want := dwc2Device + "\n"; string(got) != want {
		t.Fatalf("%s = %q, want %q", udcAttr, got, want)
	}
}

func TestBindUDCRefusesAnEmptyName(t *testing.T) {
	ops, root := newTestConfigFSOps(t)

	udc := filepath.Join(root, udcAttr)
	if err := os.WriteFile(udc, []byte("4340000.usb\n"), 0o644); err != nil {
		t.Fatalf("seed %s: %v", udcAttr, err)
	}

	for _, name := range []string{"", " ", "\n", "\t "} {
		if err := ops.BindUDC(name); !errors.Is(err, ErrUDCName) {
			t.Fatalf("bind udc %q: err = %v, want %v", name, err, ErrUDCName)
		}
	}

	// A refused bind must not have touched the attribute on the way out.
	got, err := os.ReadFile(udc)
	if err != nil {
		t.Fatalf("read %s: %v", udcAttr, err)
	}
	if want := "4340000.usb\n"; string(got) != want {
		t.Fatalf("%s = %q, want %q", udcAttr, got, want)
	}
}

func TestUnbindUDCWritesABareNewline(t *testing.T) {
	ops, root := newTestConfigFSOps(t)

	udc := filepath.Join(root, udcAttr)
	if err := os.WriteFile(udc, []byte("4340000.usb\n"), 0o644); err != nil {
		t.Fatalf("seed %s: %v", udcAttr, err)
	}

	if err := ops.UnbindUDC(); err != nil {
		t.Fatalf("unbind udc: %v", err)
	}

	got, err := os.ReadFile(udc)
	if err != nil {
		t.Fatalf("read %s: %v", udcAttr, err)
	}
	if string(got) != "\n" {
		t.Fatalf("%s = %q, want %q", udcAttr, got, "\n")
	}
}

// SetOTGRole has the same shape as BindUDC: a bare value plus a newline, and
// no golden trace that sees the bytes it actually writes.
func TestSetOTGRoleWritesTheRoleWithATrailingNewline(t *testing.T) {
	ops, _ := newTestConfigFSOps(t)

	path := filepath.Join(t.TempDir(), "otg_role")
	old := otgRolePath
	otgRolePath = path
	t.Cleanup(func() { otgRolePath = old })

	if err := os.WriteFile(path, []byte("a-much-longer-previous-role\n"), 0o644); err != nil {
		t.Fatalf("seed otg role: %v", err)
	}

	if err := ops.SetOTGRole("device"); err != nil {
		t.Fatalf("set otg role: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read otg role: %v", err)
	}
	if want := "device\n"; string(got) != want {
		t.Fatalf("otg role = %q, want %q", got, want)
	}
}

// The dirfd ConfigFSOps holds is an ordinary directory fd: openat and mkdirat
// resolve ".." through it exactly as open(2) would, so the only thing keeping a
// gadget path inside g0 is validateRel. These prove that, by giving each op a
// path that reaches a real file one level above the root and checking the file
// is still there afterwards.
func TestConfigFSOpsRefuseEscapingPaths(t *testing.T) {
	ops, root := newTestConfigFSOps(t)
	outside := filepath.Join(filepath.Dir(root), "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatalf("create outside: %v", err)
	}
	victim := filepath.Join(outside, "victim")
	if err := os.WriteFile(victim, []byte("untouched"), 0o644); err != nil {
		t.Fatalf("seed victim: %v", err)
	}

	for _, rel := range []string{
		"../outside/victim",
		"functions/../../outside/victim",
		"/etc/passwd",
		"",
		"functions/./hid.usb0",
		"evil/hid.usb0",
	} {
		if err := ops.WriteFile(rel, []byte("owned")); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("WriteFile(%q): err = %v, want %v", rel, err, ErrInvalidPath)
		}
		if _, err := ops.ReadFile(rel); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("ReadFile(%q): err = %v, want %v", rel, err, ErrInvalidPath)
		}
		if err := ops.Mkdir(rel); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("Mkdir(%q): err = %v, want %v", rel, err, ErrInvalidPath)
		}
		if err := ops.Symlink(GadgetRoot, rel); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("Symlink(%q): err = %v, want %v", rel, err, ErrInvalidPath)
		}
	}
	// Symlink also validates a relative target, which it is about to join onto
	// the root and hand to symlinkat.
	if err := ops.Symlink("../outside/victim", "configs/c.1/hid.usb0"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Symlink target: err = %v, want %v", err, ErrInvalidPath)
	}

	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("read victim: %v", err)
	}
	if string(got) != "untouched" {
		t.Fatalf("victim = %q, want %q", got, "untouched")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 1 {
		t.Fatalf("outside = %v, err = %v", entries, err)
	}
}

// Remove exists to drop the function symlinks out of configs/c.1 and the
// os_desc link. Anything else it is pointed at is a gadget attribute or a
// function directory that unlinking would corrupt.
func TestRemoveIsLimitedToConfigAndOSDescSymlinks(t *testing.T) {
	ops, root := newTestConfigFSOps(t)
	for _, rel := range []string{"strings/0x409", "configs/c.1", "functions"} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("seed %s: %v", rel, err)
		}
	}
	stray := "strings/0x409/manufacturer"
	if err := os.Symlink(root, filepath.Join(root, stray)); err != nil {
		t.Fatalf("seed %s: %v", stray, err)
	}

	if err := ops.Remove(stray); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Remove(%q): err = %v, want %v", stray, err, ErrInvalidPath)
	}
	if _, err := os.Lstat(filepath.Join(root, stray)); err != nil {
		t.Fatalf("%s was unlinked: %v", stray, err)
	}

	allowed := "configs/c.1/hid.usb0"
	if err := os.Symlink(root, filepath.Join(root, allowed)); err != nil {
		t.Fatalf("seed %s: %v", allowed, err)
	}
	if err := ops.Remove(allowed); err != nil {
		t.Fatalf("Remove(%q): %v", allowed, err)
	}
}

// RemoveDir is only ever meant to tear down the descriptor groups f_uvc creates
// under a uvc.camN function. Aimed at a function directory or a config it would
// destroy the gadget.
func TestRemoveDirIsLimitedToUVCDescriptorGroups(t *testing.T) {
	ops, root := newTestConfigFSOps(t)
	for _, rel := range []string{
		"functions/hid.usb0",
		"configs/c.1",
		"functions/uvc.cam0/streaming/mjpeg/m/1920x1080",
		"functions/uvc.cam0/streaming/mjpeg/m/1280x720",
	} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("seed %s: %v", rel, err)
		}
	}

	for _, rel := range []string{
		"functions/hid.usb0",
		"configs/c.1",
		"functions/uvc.cam0/streaming/mjpeg/m/1920x1080",
	} {
		if err := ops.RemoveDir(rel); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("RemoveDir(%q): err = %v, want %v", rel, err, ErrInvalidPath)
		}
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("%s was removed: %v", rel, err)
		}
	}

	allowed := "functions/uvc.cam0/streaming/mjpeg/m/1280x720"
	if err := ops.RemoveDir(allowed); err != nil {
		t.Fatalf("RemoveDir(%q): %v", allowed, err)
	}
}
