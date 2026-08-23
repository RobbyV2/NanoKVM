package presentation

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func descriptorProfile() Profile {
	profile := standardProfile()
	profile.Name = "desk-profile"
	profile.BuiltIn = false
	profile.Descriptors = &DescriptorSet{
		Device: []byte{18, 1, 0, 2, 0, 0, 0, 64, 0x46, 0x33, 0x09, 0x10, 0, 1, 1, 2, 3, 1},
		Configurations: [][]byte{{
			9, 2, 18, 0, 1, 1, 0, 0x80, 50,
			9, 4, 0, 0, 0, 3, 1, 1, 0,
		}},
		BOS:        []byte{5, 15, 5, 0, 0},
		Strings:    map[string]string{"1": "Example", "2": "Desk Device", "3": "A100"},
		HIDReports: map[string][]byte{"GS0": append([]byte(nil), descKeyboardStandard...)},
	}
	profile.Normalize()
	return profile
}

func TestPackageRoundTrip(t *testing.T) {
	want := descriptorProfile()
	var archive bytes.Buffer
	if err := ExportPackage(&archive, want); err != nil {
		t.Fatalf("export: %v", err)
	}
	got, err := ImportPackage(archive.Bytes())
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profile differs after round trip\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestImportRejectsUnsafeArchiveEntries(t *testing.T) {
	for _, test := range []struct {
		name string
		make func(*zip.FileHeader)
	}{
		{name: "traversal", make: func(header *zip.FileHeader) { header.Name = "../profile.json" }},
		{name: "symlink", make: func(header *zip.FileHeader) { header.SetMode(0o777 | 0o120000) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var data bytes.Buffer
			archive := zip.NewWriter(&data)
			header := &zip.FileHeader{Name: "asset.bin"}
			test.make(header)
			file, err := archive.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := ImportPackage(data.Bytes()); err == nil {
				t.Fatal("unsafe package accepted")
			}
		})
	}
}

func TestImportRejectsDuplicateManifestKeys(t *testing.T) {
	data := packageWith(t, map[string][]byte{
		"manifest.json": []byte(`{"schema_version":1,"schema_version":1,"profile":{},"assets":{}}`),
	})
	_, err := ImportPackage(data)
	if err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("err = %v", err)
	}
}

func TestImportRejectsDeepManifest(t *testing.T) {
	manifest := strings.Repeat("[", jsonDepthLimit+1) + "0" + strings.Repeat("]", jsonDepthLimit+1)
	data := packageWith(t, map[string][]byte{"manifest.json": []byte(manifest)})
	_, err := ImportPackage(data)
	if err == nil || !strings.Contains(err.Error(), "nesting") {
		t.Fatalf("err = %v", err)
	}
}

func TestImportRejectsBadAssetChecksum(t *testing.T) {
	profile := descriptorProfile()
	profile.Descriptors = nil
	manifest := PackageManifest{
		SchemaVersion: 1,
		Profile:       profile,
		Assets: PackageAssets{
			Device: &AssetRef{Path: "descriptors/device.bin", SHA256: strings.Repeat("0", 64)},
		},
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	data := packageWith(t, map[string][]byte{
		"manifest.json":          encoded,
		"descriptors/device.bin": descriptorProfile().Descriptors.Device,
	})
	_, err = ImportPackage(data)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v", err)
	}
}

func TestProfileRejectsUnsafePathsAndMalformedDescriptors(t *testing.T) {
	profile := descriptorProfile()
	profile.Functions[0].Instance = "../GS0"
	if err := profile.Validate(); err == nil || !strings.Contains(err.Error(), "invalid instance") {
		t.Fatalf("path err = %v", err)
	}

	profile = descriptorProfile()
	profile.Descriptors.Configurations[0][2]++
	if err := profile.Validate(); err == nil || !strings.Contains(err.Error(), "wTotalLength") {
		t.Fatalf("descriptor err = %v", err)
	}

	profile = descriptorProfile()
	profile.Device.Product = "bad\nvalue"
	if err := profile.Validate(); err == nil || !strings.Contains(err.Error(), "control characters") {
		t.Fatalf("identity err = %v", err)
	}

	profile = descriptorProfile()
	profile.Descriptors.Configurations[0] = []byte{
		9, 2, 41, 0, 2, 1, 0, 0x80, 50,
		9, 4, 0, 0, 1, 3, 1, 1, 0,
		7, 5, 0x81, 3, 8, 0, 10,
		9, 4, 1, 0, 1, 3, 1, 1, 0,
		7, 5, 0x81, 3, 8, 0, 10,
	}
	if err := profile.Validate(); err == nil || !strings.Contains(err.Error(), "belongs to interfaces") {
		t.Fatalf("endpoint err = %v", err)
	}
}

func packageWith(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var data bytes.Buffer
	archive := zip.NewWriter(&data)
	for name, content := range files {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
