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
