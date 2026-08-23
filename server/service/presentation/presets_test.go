package presentation

import (
	"bytes"
	"testing"
)

func presetProfile(t *testing.T, preset Preset) Profile {
	t.Helper()

	profile := standardProfile()
	profile.Name, profile.BuiltIn = "preset-"+preset.ID, false
	profile.Device.VendorID, profile.Device.ProductID = preset.VendorID, preset.ProductID
	profile.Device.Manufacturer, profile.Device.Product = preset.Manufacturer, preset.Product
	profile.Provenance = Provenance{Origin: OriginPreset, Source: preset.ID}
	profile.Normalize()
	return profile
}

func TestShippedPresetsAreIdentityOnlyAndUsable(t *testing.T) {
	seen := make(map[string]bool)
	for _, preset := range Presets() {
		if preset.ID == "" || seen[preset.ID] {
			t.Fatalf("preset id %q is empty or repeated", preset.ID)
		}
		seen[preset.ID] = true
		if preset.Source == "" {
			t.Fatalf("preset %q names no source for its identity", preset.ID)
		}

		profile := presetProfile(t, preset)
		if profile.Provenance.Origin != OriginPreset || profile.Provenance.Source != preset.ID {
			t.Fatalf("preset %q: provenance = %+v", preset.ID, profile.Provenance)
		}
		if profile.Descriptors != nil || profile.Provenance.Descriptors {
			t.Fatalf("preset %q claims descriptor data", preset.ID)
		}
		if err := profile.Validate(); err != nil {
			t.Fatalf("preset %q: %v", preset.ID, err)
		}
	}
	if _, ok := PresetByID("no-such-preset"); ok {
		t.Fatal("PresetByID invented an entry")
	}
}

// Item 28 defers the capture tool, so the point of the schema is that the tree
// it produces drops into a shipped identity later without a version bump.
func TestAPresetIdentityTakesACapturedTreeAtTheSameSchemaVersion(t *testing.T) {
	preset, ok := PresetByID("logitech-unifying-receiver")
	if !ok {
		t.Fatal("preset missing")
	}
	profile := presetProfile(t, preset)
	profile.Descriptors = descriptorProfile().Descriptors
	profile.Normalize()

	if profile.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", profile.SchemaVersion, SchemaVersion)
	}
	if !profile.Provenance.Descriptors {
		t.Fatal("provenance did not notice the captured tree")
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	var archive bytes.Buffer
	if err := ExportPackage(&archive, profile); err != nil {
		t.Fatalf("export: %v", err)
	}
	got, err := ImportPackage(archive.Bytes())
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if got.Descriptors == nil || got.Provenance != profile.Provenance {
		t.Fatalf("captured tree did not survive: descriptors=%v provenance=%+v", got.Descriptors != nil, got.Provenance)
	}
}

func TestPresetProvenanceDoesNotOutliveTheIdentity(t *testing.T) {
	preset, ok := PresetByID("dell-kb216")
	if !ok {
		t.Fatal("preset missing")
	}
	profile := presetProfile(t, preset)

	profile.Device.ProductID = "0x9999"
	profile.Normalize()
	if profile.Provenance.Origin != OriginUser || profile.Provenance.Source != "" {
		t.Fatalf("edited profile still claims preset %q: %+v", preset.ID, profile.Provenance)
	}

	profile = presetProfile(t, preset)
	profile.Provenance.Source = "a-preset-that-does-not-ship"
	profile.Normalize()
	if profile.Provenance.Origin != OriginUser {
		t.Fatalf("profile kept provenance for an unshipped preset: %+v", profile.Provenance)
	}
}
