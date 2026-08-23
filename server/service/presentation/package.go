package presentation

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

const (
	PackageSchemaVersion = 1
	PackageMediaType     = "application/vnd.nanokvm.presentation+zip"

	packageArchiveLimit = 8 << 20
	packageFileLimit    = 2 << 20
	packageTotalLimit   = 6 << 20
	packageFileCount    = 64
	jsonDepthLimit      = 64
)

type AssetRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type PackageAssets struct {
	Device         *AssetRef           `json:"device,omitempty"`
	Configurations []AssetRef          `json:"configurations,omitempty"`
	BOS            *AssetRef           `json:"bos,omitempty"`
	Strings        map[string]AssetRef `json:"strings,omitempty"`
	HIDReports     map[string]AssetRef `json:"hid_reports,omitempty"`
}

type PackageManifest struct {
	SchemaVersion int           `json:"schema_version"`
	Profile       Profile       `json:"profile"`
	Assets        PackageAssets `json:"assets"`
}

func ExportPackage(w io.Writer, profile Profile) error {
	if err := profile.Validate(); err != nil {
		return err
	}

	descriptors := profile.Descriptors
	profile.Descriptors = nil
	manifest := PackageManifest{SchemaVersion: PackageSchemaVersion, Profile: profile}

	archive := zip.NewWriter(w)
	add := func(name string, data []byte) (AssetRef, error) {
		hash := sha256.Sum256(data)
		ref := AssetRef{Path: name, SHA256: hex.EncodeToString(hash[:])}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o600)
		file, err := archive.CreateHeader(header)
		if err != nil {
			return AssetRef{}, err
		}
		if _, err := file.Write(data); err != nil {
			return AssetRef{}, err
		}
		return ref, nil
	}

	if descriptors != nil {
		if len(descriptors.Device) != 0 {
			ref, err := add("descriptors/device.bin", descriptors.Device)
			if err != nil {
				return err
			}
			manifest.Assets.Device = &ref
		}
		for i, data := range descriptors.Configurations {
			ref, err := add(fmt.Sprintf("descriptors/configuration-%d.bin", i+1), data)
			if err != nil {
				return err
			}
			manifest.Assets.Configurations = append(manifest.Assets.Configurations, ref)
		}
		if len(descriptors.BOS) != 0 {
			ref, err := add("descriptors/bos.bin", descriptors.BOS)
			if err != nil {
				return err
			}
			manifest.Assets.BOS = &ref
		}

		stringKeys := sortedKeys(descriptors.Strings)
		if len(stringKeys) != 0 {
			manifest.Assets.Strings = make(map[string]AssetRef, len(stringKeys))
		}
		for _, key := range stringKeys {
			name := "strings/" + hex.EncodeToString([]byte(key)) + ".txt"
			ref, err := add(name, []byte(descriptors.Strings[key]))
			if err != nil {
				return err
			}
			manifest.Assets.Strings[key] = ref
		}

		reportKeys := sortedKeys(descriptors.HIDReports)
		if len(reportKeys) != 0 {
			manifest.Assets.HIDReports = make(map[string]AssetRef, len(reportKeys))
		}
		for _, key := range reportKeys {
			ref, err := add("hid/"+key+".bin", descriptors.HIDReports[key])
			if err != nil {
				return err
			}
			manifest.Assets.HIDReports[key] = ref
		}
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	data = append(data, '\n')
	header := &zip.FileHeader{Name: "manifest.json", Method: zip.Deflate}
	header.SetMode(0o600)
	file, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return archive.Close()
}

func ImportPackage(data []byte) (Profile, error) {
	if len(data) == 0 || len(data) > packageArchiveLimit {
		return Profile{}, fmt.Errorf("package size %d is outside 1..%d", len(data), packageArchiveLimit)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Profile{}, fmt.Errorf("open package: %w", err)
	}
	if len(reader.File) == 0 || len(reader.File) > packageFileCount {
		return Profile{}, fmt.Errorf("package contains %d files, limit is %d", len(reader.File), packageFileCount)
	}

	files := make(map[string][]byte, len(reader.File))
	total := uint64(0)
	for _, file := range reader.File {
		name := file.Name
		if strings.Contains(name, "\\") || path.IsAbs(name) || path.Clean(name) != name || name == "." || strings.HasPrefix(name, "../") {
			return Profile{}, fmt.Errorf("unsafe package path %q", name)
		}
		if file.FileInfo().IsDir() || file.Mode()&os.ModeSymlink != 0 {
			return Profile{}, fmt.Errorf("package entry %q is not a regular file", name)
		}
		if _, exists := files[name]; exists {
			return Profile{}, fmt.Errorf("duplicate package path %q", name)
		}
		if file.UncompressedSize64 > packageFileLimit || total+file.UncompressedSize64 > packageTotalLimit {
			return Profile{}, fmt.Errorf("package expands beyond its limit at %q", name)
		}
		opened, err := file.Open()
		if err != nil {
			return Profile{}, err
		}
		blob, readErr := io.ReadAll(io.LimitReader(opened, packageFileLimit+1))
		closeErr := opened.Close()
		if readErr != nil {
			return Profile{}, readErr
		}
		if closeErr != nil {
			return Profile{}, closeErr
		}
		if len(blob) > packageFileLimit {
			return Profile{}, fmt.Errorf("package entry %q is too large", name)
		}
		files[name] = blob
		total += uint64(len(blob))
	}

	manifestData, ok := files["manifest.json"]
	if !ok {
		return Profile{}, errors.New("package has no manifest.json")
	}
	if err := rejectDuplicateJSONKeys(manifestData); err != nil {
		return Profile{}, fmt.Errorf("manifest: %w", err)
	}
	var manifest PackageManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Profile{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Profile{}, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.SchemaVersion != PackageSchemaVersion {
		return Profile{}, fmt.Errorf("package schema version %d, want %d", manifest.SchemaVersion, PackageSchemaVersion)
	}
	if manifest.Profile.BuiltIn {
		return Profile{}, errors.New("imported profile cannot replace a built-in profile; clone it first")
	}
	if manifest.Profile.Descriptors != nil {
		return Profile{}, errors.New("manifest embeds descriptors instead of package assets")
	}

	used := map[string]bool{"manifest.json": true}
	load := func(ref *AssetRef) ([]byte, error) {
		if ref == nil {
			return nil, nil
		}
		if ref.Path == "manifest.json" || ref.Path == "" || !profileNamePatternForAsset(ref.Path) {
			return nil, fmt.Errorf("invalid asset path %q", ref.Path)
		}
		if used[ref.Path] {
			return nil, fmt.Errorf("asset %q is referenced more than once", ref.Path)
		}
		blob, ok := files[ref.Path]
		if !ok {
			return nil, fmt.Errorf("missing asset %q", ref.Path)
		}
		if len(blob) == 0 {
			return nil, fmt.Errorf("asset %q is empty", ref.Path)
		}
		want, err := hex.DecodeString(ref.SHA256)
		if err != nil || len(want) != sha256.Size {
			return nil, fmt.Errorf("invalid sha256 for %q", ref.Path)
		}
		got := sha256.Sum256(blob)
		if !bytes.Equal(want, got[:]) {
			return nil, fmt.Errorf("checksum mismatch for %q", ref.Path)
		}
		used[ref.Path] = true
		return blob, nil
	}

	descriptors := &DescriptorSet{}
	descriptors.Device, err = load(manifest.Assets.Device)
	if err != nil {
		return Profile{}, err
	}
	for i := range manifest.Assets.Configurations {
		blob, err := load(&manifest.Assets.Configurations[i])
		if err != nil {
			return Profile{}, err
		}
		descriptors.Configurations = append(descriptors.Configurations, blob)
	}
	descriptors.BOS, err = load(manifest.Assets.BOS)
	if err != nil {
		return Profile{}, err
	}
	if len(manifest.Assets.Strings) != 0 {
		descriptors.Strings = make(map[string]string, len(manifest.Assets.Strings))
	}
	for key, ref := range manifest.Assets.Strings {
		blob, err := load(&ref)
		if err != nil {
			return Profile{}, err
		}
		descriptors.Strings[key] = string(blob)
	}
	if len(manifest.Assets.HIDReports) != 0 {
		descriptors.HIDReports = make(map[string][]byte, len(manifest.Assets.HIDReports))
	}
	for key, ref := range manifest.Assets.HIDReports {
		blob, err := load(&ref)
		if err != nil {
			return Profile{}, err
		}
		descriptors.HIDReports[key] = blob
	}
	if len(descriptors.Device) != 0 || len(descriptors.Configurations) != 0 || len(descriptors.BOS) != 0 || len(descriptors.Strings) != 0 || len(descriptors.HIDReports) != 0 {
		manifest.Profile.Descriptors = descriptors
	}
	if len(used) != len(files) {
		var extras []string
		for name := range files {
			if !used[name] {
				extras = append(extras, name)
			}
		}
		sort.Strings(extras)
		return Profile{}, fmt.Errorf("package contains unreferenced files: %s", strings.Join(extras, ", "))
	}

	manifest.Profile.Normalize()
	if err := manifest.Profile.Validate(); err != nil {
		return Profile{}, fmt.Errorf("validate profile: %w", err)
	}
	return manifest.Profile, nil
}

func profileNamePatternForAsset(name string) bool {
	return !path.IsAbs(name) && path.Clean(name) == name && name != "." && !strings.HasPrefix(name, "../") && !strings.Contains(name, "\\") && strings.IndexFunc(name, func(r rune) bool { return r < 0x20 || r > 0x7E }) < 0
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if err := consumeJSONValue(decoder, token, 0); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
}

func consumeJSONValue(decoder *json.Decoder, token json.Token, depth int) error {
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if depth >= jsonDepthLimit {
		return fmt.Errorf("json nesting exceeds %d levels", jsonDepthLimit)
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate key %q", key)
			}
			seen[key] = true
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := consumeJSONValue(decoder, value, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := consumeJSONValue(decoder, value, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected delimiter %q", delim)
	}
	_, err := decoder.Token()
	return err
}

func ensureJSONEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple json values")
		}
		return err
	}
	return nil
}
