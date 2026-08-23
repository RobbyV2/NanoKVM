package presentation

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestDeviceSerialIsPerBoardAndLeaksNothing(t *testing.T) {
	useTestBootDir(t)

	first := DeviceSerial()
	if first == fallbackSerial {
		t.Fatalf("serial = %q, want a value derived from base_uid", first)
	}
	if again := DeviceSerial(); again != first {
		t.Fatalf("serial = %q then %q for one board", first, again)
	}
	if len(first) != len(fallbackSerial) || strings.Trim(first, "0123456789ABCDEF") != "" {
		t.Fatalf("serial %q is not %d hex characters", first, len(fallbackSerial))
	}
	if err := usbString("serial", first); err != nil {
		t.Fatalf("serial %q: %v", first, err)
	}

	// The host reads this string. The gadget MAC comes off the same UID, so a
	// serial that started with it would hand the controlled host an address.
	dev, _ := gadgetMACs()
	if dev == nil {
		t.Fatal("no gadget mac to compare against")
	}
	mac := strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(*dev, macDevPrefix+":"), ":", ""))
	if strings.Contains(first, mac) {
		t.Fatalf("serial %q carries the gadget mac %q", first, *dev)
	}

	if err := os.WriteFile(baseUIDPath, []byte("a-second-board"), 0o600); err != nil {
		t.Fatal(err)
	}
	if second := DeviceSerial(); second == first {
		t.Fatalf("two boards share the serial %q", second)
	}

	if err := os.Remove(baseUIDPath); err != nil {
		t.Fatal(err)
	}
	if got := DeviceSerial(); got != fallbackSerial {
		t.Fatalf("serial without base_uid = %q, want the stable fallback %q", got, fallbackSerial)
	}
}

func TestStandardProfileCarriesTheDerivedSerial(t *testing.T) {
	useTestBootDir(t)

	serial := standardProfile().Device.Serial
	if serial == nil {
		t.Fatal("standard profile writes no serial")
	}
	if *serial == fallbackSerial {
		t.Fatalf("standard serial = %q, the fleet-wide constant", *serial)
	}
	if *serial != DeviceSerial() {
		t.Fatalf("standard serial = %q, want the derived %q", *serial, DeviceSerial())
	}
}

func TestForeignVendorNamesSomebodyElsesVendor(t *testing.T) {
	for vendor, want := range map[string]bool{
		VendorSipeed: false,
		"0X3346":     false,
		"0x046d":     true,
		"":           false,
	} {
		if got := ForeignVendor(vendor); got != want {
			t.Fatalf("ForeignVendor(%q) = %v, want %v", vendor, got, want)
		}
	}

	profile := standardProfile()
	if plan, err := Compile(profile, staticV1); err != nil || plan.Device.ForeignVendor {
		t.Fatalf("own vendor flagged as foreign: %v %v", plan.Device, err)
	}

	preset, ok := PresetByID("logitech-unifying-receiver")
	if !ok {
		t.Fatal("preset missing")
	}
	profile.BuiltIn = false
	profile.Device.VendorID, profile.Device.ProductID = preset.VendorID, preset.ProductID
	plan, err := Compile(profile, staticV1)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !plan.Device.ForeignVendor {
		t.Fatalf("device %+v is not flagged as foreign", plan.Device)
	}
}

func TestPackageRoundTripsProvenance(t *testing.T) {
	want := descriptorProfile()
	want.Provenance = Provenance{Origin: OriginPreset, Source: "nanokvm"}
	want.Normalize()
	if !want.Provenance.Descriptors {
		t.Fatal("normalize did not record the captured tree")
	}

	var archive bytes.Buffer
	if err := ExportPackage(&archive, want); err != nil {
		t.Fatalf("export: %v", err)
	}
	got, err := ImportPackage(archive.Bytes())
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if got.Provenance != want.Provenance {
		t.Fatalf("provenance = %+v, want %+v", got.Provenance, want.Provenance)
	}
}

func TestImportRejectsAClaimOfDescriptorsThePackageDoesNotShip(t *testing.T) {
	profile := standardProfile()
	profile.Name, profile.BuiltIn = "claimer", false
	profile.Normalize()
	profile.Provenance.Descriptors = true

	manifest, err := json.Marshal(PackageManifest{SchemaVersion: PackageSchemaVersion, Profile: profile})
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	archive := zip.NewWriter(&data)
	file, err := archive.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(manifest); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := ImportPackage(data.Bytes()); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("import error = %v, want a provenance mismatch", err)
	}
}

func TestValidateRejectsAnUnknownOrigin(t *testing.T) {
	profile := standardProfile()
	profile.BuiltIn = false
	profile.Provenance.Origin = "vendor-said-so"

	if err := profile.Validate(); err == nil || !strings.Contains(err.Error(), "unknown origin") {
		t.Fatalf("validate = %v, want an unknown origin rejection", err)
	}
}
