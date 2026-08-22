package presentation

import (
	"os"
	"path/filepath"
	"reflect"
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
