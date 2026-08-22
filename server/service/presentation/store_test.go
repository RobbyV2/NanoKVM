package presentation

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func useTestPresentationDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "presentation")
	old := presentationDir
	presentationDir = dir
	t.Cleanup(func() { presentationDir = old })
	return dir
}

func TestProfileRoundTrip(t *testing.T) {
	useTestPresentationDir(t)
	store := NewStore()

	want := Profile{
		SchemaVersion: 1,
		Name:          "user",
		Device: Device{
			VendorID:     "0x3346",
			ProductID:    "0x1009",
			Manufacturer: "sipeed",
			Product:      "NanoKVM",
		},
	}
	if err := store.SaveProfile(want); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if err := store.SetActive(want.Name); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if err := store.SetLastKnownGood(want.Name); err != nil {
		t.Fatalf("set last known good: %v", err)
	}

	got, err := store.LoadProfile(want.Name)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profile = %+v, want %+v", got, want)
	}

	active, err := store.Active()
	if err != nil || active != want.Name {
		t.Fatalf("active = %q err = %v", active, err)
	}
	lastKnownGood, err := store.LastKnownGood()
	if err != nil || lastKnownGood != want.Name {
		t.Fatalf("last known good = %q err = %v", lastKnownGood, err)
	}
}

func TestSaveProfilePermissions(t *testing.T) {
	dir := useTestPresentationDir(t)
	store := NewStore()

	if err := store.WriteBuiltins(); err != nil {
		t.Fatalf("write built-ins: %v", err)
	}
	if err := store.SetActive(standardProfile().Name); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if err := store.SetLastKnownGood(standardProfile().Name); err != nil {
		t.Fatalf("set last known good: %v", err)
	}

	files := []string{activeFile, lastKnownGoodFile}
	for _, profile := range builtinProfiles() {
		files = append(files, profile.Name+".json")
	}
	for _, name := range files {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s permissions = %o, want 600", name, info.Mode().Perm())
		}
	}
}

func TestLoadProfileMissingFile(t *testing.T) {
	useTestPresentationDir(t)
	store := NewStore()

	profile, err := store.LoadProfile("user")
	if err != nil {
		t.Fatalf("load missing profile: %v", err)
	}
	if !reflect.DeepEqual(profile, Profile{}) {
		t.Fatalf("profile = %+v, want zero value", profile)
	}

	active, err := store.Active()
	if err != nil || active != "" {
		t.Fatalf("active = %q err = %v", active, err)
	}
}

func TestLoadProfileRejectsCorruptJSON(t *testing.T) {
	dir := useTestPresentationDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore().LoadProfile("user"); err == nil {
		t.Fatal("expected corrupt profile error")
	}
}

func TestLoadProfileIgnoresCorruptBuiltin(t *testing.T) {
	dir := useTestPresentationDir(t)
	store := NewStore()

	if err := store.WriteBuiltins(); err != nil {
		t.Fatalf("write built-ins: %v", err)
	}
	for _, want := range builtinProfiles() {
		if err := os.WriteFile(filepath.Join(dir, want.Name+".json"), []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := store.LoadProfile(want.Name)
		if err != nil {
			t.Fatalf("load built-in %s: %v", want.Name, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("built-in %s = %+v, want the code version", want.Name, got)
		}
	}
}

func TestProfileNameRejectsTraversal(t *testing.T) {
	useTestPresentationDir(t)
	store := NewStore()

	for _, name := range []string{"", "..", "a/b", "../escape", ".last-known-good"} {
		if _, err := store.LoadProfile(name); err == nil {
			t.Fatalf("load accepted name %q", name)
		}
		if err := store.SaveProfile(Profile{Name: name}); err == nil {
			t.Fatalf("save accepted name %q", name)
		}
		if err := store.SetActive(name); err == nil {
			t.Fatalf("set active accepted name %q", name)
		}
	}
}

// HIDFunction.DevNodeIndex is json:"-", so it comes back zero for every
// function no matter what was saved. LoadProfile repairs that by calling
// Normalize, and hid/hid.go depends on the repair: the index picks the
// /dev/hidgN minor. Drop the Normalize call and the second and third functions
// both claim node 0, which Validate also rejects.
func TestLoadProfileNormalizesDevNodeIndex(t *testing.T) {
	dir := useTestPresentationDir(t)
	store := NewStore()

	want := standardProfile()
	want.Name = "user-hid"
	want.BuiltIn = false
	if err := store.SaveProfile(want); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	// Guard the premise: the index is not in the file, so nothing but
	// Normalize can put it back.
	raw, err := os.ReadFile(filepath.Join(dir, want.Name+".json"))
	if err != nil {
		t.Fatalf("read saved profile: %v", err)
	}
	if strings.Contains(string(raw), "dev_node") || strings.Contains(string(raw), "DevNodeIndex") {
		t.Fatalf("saved profile carries a dev node index:\n%s", raw)
	}

	got, err := store.LoadProfile(want.Name)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	var indexes []int
	for _, function := range got.Functions {
		if function.Kind != FunctionHID || function.HID == nil {
			t.Fatalf("function %s.%s is not a hid function", function.Kind, function.Instance)
		}
		indexes = append(indexes, function.HID.DevNodeIndex)
	}
	if !reflect.DeepEqual(indexes, []int{0, 1, 2}) {
		t.Fatalf("dev node indexes = %v, want [0 1 2]", indexes)
	}

	if err := got.Validate(); err != nil {
		t.Fatalf("loaded profile does not validate: %v", err)
	}
}
