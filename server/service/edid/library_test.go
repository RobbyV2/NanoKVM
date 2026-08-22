package edid

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math"
	"regexp"
	"testing"
)

const factorySource = "tools/nanokvm_update_edid/E21_NanoKVM.bin"

var profileCorpus = regexp.MustCompile(`^Digital/.+@[0-9a-f]{40}$`)

func TestShippedProfiles(t *testing.T) {
	shipped := Profiles()
	if len(shipped) < 15 || len(shipped) > 30 {
		t.Fatalf("shipped %d profiles, want 15 to 30", len(shipped))
	}

	seen := make(map[string]string, len(shipped))
	for _, profile := range shipped {
		if previous, ok := seen[profile.ID()]; ok {
			t.Errorf("profile %s duplicates %s", profile.Source, previous)
		}
		seen[profile.ID()] = profile.Source

		t.Run(profile.Model, func(t *testing.T) {
			data := profile.Data
			if len(data) != Size {
				t.Fatalf("blob is %d bytes, want %d", len(data), Size)
			}
			if got, want := data[BlockSize-1], checksum(data[:BlockSize]); got != want {
				t.Errorf("base checksum 0x%02X, recomputed 0x%02X", got, want)
			}
			if got, want := data[Size-1], checksum(data[BlockSize:]); got != want {
				t.Errorf("extension checksum 0x%02X, recomputed 0x%02X", got, want)
			}
			if sha256.Sum256(data) != profile.SHA256 {
				t.Errorf("sha256 %s does not cover the blob", profile.ID())
			}
			if data[extensionCount] == 0 && !allZero(data[BlockSize:]) {
				t.Error("no extension declared but block 1 carries data")
			}
			if profile.Source != factorySource && !profileCorpus.MatchString(profile.Source) {
				t.Errorf("provenance %q carries no upstream path and commit", profile.Source)
			}

			parsed, err := Decode(data)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}

			encoded, err := parsed.Encode()
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if !bytes.Equal(encoded, data) {
				t.Error("decode then encode does not reproduce the blob")
			}

			if parsed.Manufacturer != profile.Manufacturer {
				t.Errorf("manufacturer %q, blob carries %q", profile.Manufacturer, parsed.Manufacturer)
			}
			if parsed.Name() != profile.Model {
				t.Errorf("model %q, blob carries %q", profile.Model, parsed.Name())
			}

			timing := parsed.PreferredTiming()
			if timing == nil {
				t.Fatal("no preferred timing")
			}
			if timing.Mode() != profile.PreferredMode {
				t.Errorf("preferred mode %q, blob carries %q", profile.PreferredMode, timing.Mode())
			}
			if timing.HActive > 1920 || timing.VActive > 1080 || math.Round(timing.RefreshHz) > 60 {
				t.Errorf("preferred mode %s is beyond the capture path", timing.Mode())
			}
		})
	}
}

func TestProfileLookup(t *testing.T) {
	for _, want := range Profiles() {
		got, err := ProfileByID(want.ID())
		if err != nil {
			t.Fatalf("lookup %s: %v", want.Source, err)
		}
		if got.Source != want.Source {
			t.Errorf("lookup %s returned %s", want.Source, got.Source)
		}
	}

	if _, err := ProfileByID("0000"); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("unknown id: %v", err)
	}

	shipped := Profiles()
	shipped[0] = Profile{}
	if Profiles()[0].Source == "" {
		t.Error("Profiles hands out the table itself")
	}
}

func TestNormalizeBlob(t *testing.T) {
	factory := fixture(t)

	var padded Profile
	for _, profile := range Profiles() {
		if profile.Data[extensionCount] == 0 {
			padded = profile
			break
		}
	}
	if padded.Data == nil {
		t.Fatal("no shipped profile without an extension block")
	}

	blob, err := Normalize(padded.Data[:BlockSize])
	if err != nil {
		t.Fatalf("normalize %s: %v", padded.Source, err)
	}
	if !bytes.Equal(blob, padded.Data) {
		t.Errorf("padding %s did not reproduce the shipped blob", padded.Source)
	}
	if blob[extensionCount] != 0 {
		t.Errorf("padding changed the extension count to %d", blob[extensionCount])
	}
	if got, want := blob[Size-1], checksum(blob[BlockSize:]); got != want {
		t.Errorf("padded block checksum 0x%02X, recomputed 0x%02X", got, want)
	}

	blob, err = Normalize(factory)
	if err != nil {
		t.Fatalf("normalize factory blob: %v", err)
	}
	if !bytes.Equal(blob, factory) {
		t.Error("normalizing 256 bytes changed them")
	}
	blob[0] = 0xFF
	if factory[0] != 0x00 {
		t.Error("Normalize aliases its input")
	}

	for _, size := range []int{0, 127, 129, 255, 257, 512} {
		if _, err := Normalize(make([]byte, size)); !errors.Is(err, ErrBlobSize) {
			t.Errorf("normalize %d bytes: %v", size, err)
		}
	}
	if _, err := Normalize(factory[:BlockSize]); !errors.Is(err, ErrBlobSize) {
		t.Error("128 bytes declaring an extension block was accepted")
	}
}

func TestFactoryProfile(t *testing.T) {
	want := fixture(t)

	for _, profile := range Profiles() {
		if profile.Source != factorySource {
			continue
		}
		if !bytes.Equal(profile.Data, want) {
			t.Error("factory profile does not carry E21_NanoKVM.bin")
		}
		if profile.Manufacturer != "SPD" || profile.Model != "NanoKVM" || profile.PreferredMode != "1920x1080p60" {
			t.Errorf("factory profile decoded as %s %s %s", profile.Manufacturer, profile.Model, profile.PreferredMode)
		}
		return
	}
	t.Error("no factory profile")
}
