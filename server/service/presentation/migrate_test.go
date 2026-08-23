package presentation

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const serialOp = "write\tstrings/0x409/serialnumber\t"

func useTestBootDir(t *testing.T, flags ...string) string {
	t.Helper()

	dir := t.TempDir()
	oldBoot, oldUID := bootDir, baseUIDPath
	bootDir, baseUIDPath = dir, filepath.Join(dir, "base_uid")
	t.Cleanup(func() { bootDir, baseUIDPath = oldBoot, oldUID })

	if err := os.WriteFile(baseUIDPath, []byte(goldenBaseUID), 0o600); err != nil {
		t.Fatalf("write base uid: %v", err)
	}
	for _, flag := range flags {
		if err := os.WriteFile(filepath.Join(dir, flag), nil, 0o600); err != nil {
			t.Fatalf("write sentinel %s: %v", flag, err)
		}
	}
	return dir
}

func migrationFixtures() map[string][]string {
	return map[string][]string{
		"normal.none.nodisk":           nil,
		"normal.ncm.nodisk":            {sentinelNCM},
		"normal.rndis.nodisk":          {sentinelRNDIS},
		"normal.ncmrndis.nodisk":       {sentinelNCM, sentinelRNDIS},
		"normal.none.disk":             {sentinelDisk},
		"normal.rndis.disk":            {sentinelRNDIS, sentinelDisk},
		"normal.rndis.disk.bios":       {sentinelRNDIS, sentinelDisk, sentinelBIOS},
		"normal.rndis.disk.notwakeup":  {sentinelRNDIS, sentinelDisk, sentinelNoWakeup},
		"normal.rndis.disk.disablehid": {sentinelRNDIS, sentinelDisk, sentinelDisableHID},
		"normal.rndis.disk.diskro":     {sentinelRNDIS, sentinelDisk, sentinelDiskRO},
	}
}

func TestMigratedProfileCompilesToTheScriptTrace(t *testing.T) {
	for name, fixture := range migrationFixtures() {
		t.Run(name, func(t *testing.T) {
			useTestBootDir(t, fixture...)

			plan, err := Compile(derivedProfile(), staticV1)
			if err != nil {
				t.Fatalf("compile migrated profile: %v", err)
			}

			want, got := readTrace(t, name), renderTrace(plan)
			for i := 0; i < len(want) || i < len(got); i++ {
				switch {
				case i >= len(got):
					t.Fatalf("op %d: missing, script has %q", i, want[i])
				case i >= len(want):
					t.Fatalf("op %d: migrated %q, script stops here", i, got[i])
				// The one deliberate divergence from the script: S03usbdev:32
				// writes the same sixteen characters on every board, and this
				// serial is per-device. Every other op still has to match.
				case strings.HasPrefix(want[i], serialOp):
					derived := serialOp + hex.EncodeToString([]byte(DeviceSerial()+"\n"))
					if got[i] == want[i] || got[i] != derived {
						t.Fatalf("op %d: migrated %q, want the per-device serial %q", i, got[i], derived)
					}
				case want[i] != got[i]:
					t.Fatalf("op %d:\n script   %q\n migrated %q", i, want[i], got[i])
				}
			}
		})
	}
}

func TestMigrateIsOneShot(t *testing.T) {
	useTestPresentationDir(t)
	useTestBootDir(t, sentinelRNDIS)
	store := NewStore()

	if err := Migrate(store, nil); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	active, err := store.Active()
	if err != nil || active != ProfileCurrent {
		t.Fatalf("active = %q err = %v, want %q", active, err, ProfileCurrent)
	}

	if err := os.WriteFile(filepath.Join(bootDir, sentinelDisk), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(store, nil); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	profile, err := store.LoadProfile(ProfileCurrent)
	if err != nil {
		t.Fatalf("load migrated profile: %v", err)
	}
	for _, function := range profile.Functions {
		if function.Kind == FunctionMassStorage {
			t.Fatal("second migrate rewrote the profile from the sentinels")
		}
	}
	if !sentinelExists(sentinelRNDIS) {
		t.Fatal("migrate removed a /boot sentinel")
	}
}

func TestMigrateKeepsTheHIDOnlyGadget(t *testing.T) {
	useTestPresentationDir(t)
	useTestBootDir(t, sentinelRNDIS)
	store := NewStore()

	ops := NewRecordOps()
	if err := ops.Seed("bcdDevice", []byte("0x0623\n")); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(store, ops); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	active, err := store.Active()
	if err != nil || active != ProfileHIDOnly {
		t.Fatalf("active = %q err = %v, want %q", active, err, ProfileHIDOnly)
	}
	profile, err := store.LoadProfile(ProfileCurrent)
	if err != nil || profile.Name != ProfileCurrent {
		t.Fatalf("migrated profile = %q err = %v", profile.Name, err)
	}
}

func TestMigrateWithoutBaseUIDDropsBothMACs(t *testing.T) {
	useTestBootDir(t, sentinelRNDIS)
	if err := os.Remove(baseUIDPath); err != nil {
		t.Fatal(err)
	}

	for _, function := range derivedProfile().Functions {
		if function.Net == nil {
			continue
		}
		if function.Net.DevAddr != nil || function.Net.HostAddr != nil {
			t.Fatalf("net addresses = %v %v, want both unset", function.Net.DevAddr, function.Net.HostAddr)
		}
	}
}

func TestMirrorSentinelsTracksTheProfile(t *testing.T) {
	useTestBootDir(t, sentinelNCM, sentinelDisk)

	if err := mirrorSentinels(derivedProfileWith(t, sentinelRNDIS)); err != nil {
		t.Fatalf("mirror sentinels: %v", err)
	}
	if sentinelExists(sentinelNCM) || sentinelExists(sentinelDisk) {
		t.Fatal("mirror kept a sentinel the profile dropped")
	}
	if !sentinelExists(sentinelRNDIS) {
		t.Fatal("mirror did not write the rndis sentinel")
	}
}

func TestMirrorSentinelsSparesHIDOnly(t *testing.T) {
	useTestBootDir(t, sentinelRNDIS)

	if err := mirrorSentinels(hidOnlyProfile()); err != nil {
		t.Fatalf("mirror sentinels: %v", err)
	}
	if !sentinelExists(sentinelRNDIS) {
		t.Fatal("hid-only mode destroyed the user's network sentinel")
	}
}

func derivedProfileWith(t *testing.T, flags ...string) Profile {
	t.Helper()

	dir := t.TempDir()
	old := bootDir
	bootDir = dir
	defer func() { bootDir = old }()

	for _, flag := range flags {
		if err := os.WriteFile(filepath.Join(dir, flag), nil, 0o600); err != nil {
			t.Fatalf("write sentinel %s: %v", flag, err)
		}
	}
	return derivedProfile()
}
