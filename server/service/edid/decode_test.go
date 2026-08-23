package edid

import (
	"bytes"
	"errors"
	"math"
	"os"
	"testing"
)

const fixturePath = "testdata/E21_NanoKVM.bin"

func fixture(t testing.TB) []byte {
	t.Helper()

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if len(data) != Size {
		t.Fatalf("fixture is %d bytes, want %d", len(data), Size)
	}
	return data
}

func repaired(data []byte) []byte {
	out := bytes.Clone(data)
	out[BlockSize-1] = checksum(out[:BlockSize])
	out[Size-1] = checksum(out[BlockSize:])
	return out
}

func mutate(data []byte, edits map[int]byte) []byte {
	out := bytes.Clone(data)
	for offset, value := range edits {
		out[offset] = value
	}
	return out
}

func closeTo(got, want float64) bool {
	return math.Abs(got-want) < 1e-9
}

func TestFixtureRoundTrip(t *testing.T) {
	data := fixture(t)

	parsed, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	out, err := parsed.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Equal(out, data) {
		for i := range data {
			if out[i] != data[i] {
				t.Fatalf("byte %d = 0x%02X, want 0x%02X", i, out[i], data[i])
			}
		}
		t.Fatalf("encoded %d bytes, want %d", len(out), len(data))
	}
}

func TestFixtureIdentity(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if parsed.Manufacturer != "SPD" {
		t.Errorf("manufacturer = %q, want SPD", parsed.Manufacturer)
	}
	if parsed.ProductCode != 0x3301 {
		t.Errorf("product code = 0x%04X, want 0x3301", parsed.ProductCode)
	}
	if parsed.Serial != 21 {
		t.Errorf("serial = %d, want 21", parsed.Serial)
	}
	if parsed.Week != 30 || parsed.Year != 2025 {
		t.Errorf("manufactured week %d year %d, want 30 2025", parsed.Week, parsed.Year)
	}
	if parsed.Version != 1 || parsed.Revision != 3 {
		t.Errorf("version %d.%d, want 1.3", parsed.Version, parsed.Revision)
	}
	if parsed.Extensions != 1 {
		t.Errorf("extensions = %d, want 1", parsed.Extensions)
	}
	if parsed.Name() != "NanoKVM" {
		t.Errorf("name = %q, want NanoKVM", parsed.Name())
	}
}

func TestFixtureBasicDisplay(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	display := parsed.Display
	if !display.Digital {
		t.Error("input is analog, want digital")
	}
	if display.DFP1x {
		t.Error("dfp 1.x flag is set")
	}
	if display.WidthCM != 48 || display.HeightCM != 27 {
		t.Errorf("physical size = %dx%d cm, want 48x27", display.WidthCM, display.HeightCM)
	}
	if !closeTo(display.Gamma, 2.2) {
		t.Errorf("gamma = %v, want 2.2", display.Gamma)
	}

	want := Features{ActiveOff: true, ColorType: 1, PreferredNative: true}
	if display.Features != want {
		t.Errorf("features = %+v, want %+v", display.Features, want)
	}
}

func TestFixtureChromaticity(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	tests := []struct {
		name  string
		point Point
		x, y  int
	}{
		{"red", parsed.Chromaticity.Red, 653, 366},
		{"green", parsed.Chromaticity.Green, 322, 653},
		{"blue", parsed.Chromaticity.Blue, 156, 70},
		{"white", parsed.Chromaticity.White, 321, 337},
	}
	for _, tt := range tests {
		if tt.point.XUnits != tt.x || tt.point.YUnits != tt.y {
			t.Errorf("%s = %d %d units, want %d %d", tt.name, tt.point.XUnits, tt.point.YUnits, tt.x, tt.y)
		}
		if !closeTo(tt.point.X, float64(tt.x)/1024) || !closeTo(tt.point.Y, float64(tt.y)/1024) {
			t.Errorf("%s = %.4f %.4f, want %.4f %.4f", tt.name, tt.point.X, tt.point.Y, float64(tt.x)/1024, float64(tt.y)/1024)
		}
	}
}

func TestFixtureTimingTables(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	wantModes := []string{
		"720x400@70", "640x480@60", "640x480@75", "800x600@60",
		"800x600@75", "1024x768@60", "1024x768@75", "1280x1024@75",
	}
	if len(parsed.Established.Modes) != len(wantModes) {
		t.Fatalf("established modes = %v, want %v", parsed.Established.Modes, wantModes)
	}
	for i, mode := range wantModes {
		if parsed.Established.Modes[i] != mode {
			t.Errorf("established mode %d = %q, want %q", i, parsed.Established.Modes[i], mode)
		}
	}
	if parsed.Established.ManufacturerReserved != 0 {
		t.Errorf("manufacturer reserved = 0x%02X, want 0", parsed.Established.ManufacturerReserved)
	}

	standard := parsed.StandardTimings
	if len(standard) != standardCount {
		t.Fatalf("%d standard timings, want %d", len(standard), standardCount)
	}
	wantStandard := []StandardTiming{
		{Used: true, Horizontal: 1920, Vertical: 1080, AspectRatio: "16:9", RefreshHz: 60},
		{Used: true, Horizontal: 1280, Vertical: 1024, AspectRatio: "5:4", RefreshHz: 60},
	}
	for i, want := range wantStandard {
		got := standard[i]
		got.Raw = [2]byte{}
		if got != want {
			t.Errorf("standard timing %d = %+v, want %+v", i+1, got, want)
		}
	}
	for i := len(wantStandard); i < standardCount; i++ {
		if standard[i].Used {
			t.Errorf("standard timing %d = %+v, want the 01 01 unused marker", i+1, standard[i])
		}
		if standard[i].Raw != [2]byte{0x01, 0x01} {
			t.Errorf("standard timing %d raw = % X, want 01 01", i+1, standard[i].Raw)
		}
	}
}

func TestFixtureDescriptors(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	kinds := []DescriptorKind{DescriptorTiming, DescriptorSerial, DescriptorName, DescriptorRangeLimits}
	for i, kind := range kinds {
		if parsed.Descriptors[i].Kind != kind {
			t.Fatalf("descriptor %d is %q, want %q", i+1, parsed.Descriptors[i].Kind, kind)
		}
	}
	if text := parsed.Descriptors[1].Text; text != "NanoKVM" {
		t.Errorf("serial descriptor = %q, want NanoKVM", text)
	}
	if text := parsed.Descriptors[2].Text; text != "NanoKVM" {
		t.Errorf("name descriptor = %q, want NanoKVM", text)
	}

	want := RangeLimits{VerticalMin: 56, VerticalMax: 76, HorizontalMin: 30, HorizontalMax: 83, MaxPixelClock: 170}
	if got := *parsed.Descriptors[3].Range; got != want {
		t.Errorf("range limits = %+v, want %+v", got, want)
	}
}

func TestFixturePreferredTiming(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	timing := parsed.PreferredTiming()
	if timing == nil {
		t.Fatal("no preferred timing")
	}

	want := Timing{
		PixelClockKHz: 148500,
		HActive:       1920, HBlank: 280, HTotal: 2200,
		VActive: 1080, VBlank: 45, VTotal: 1125,
		HSyncOffset: 88, HSyncWidth: 44,
		VSyncOffset: 4, VSyncWidth: 5,
		WidthMM: 476, HeightMM: 268,
		Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true,
		RefreshHz: 60,
	}
	if got := *timing; got != want {
		t.Errorf("preferred timing = %+v, want %+v", got, want)
	}
	if !closeTo(timing.RefreshHz, 60) {
		t.Errorf("refresh = %v Hz, want 60.000", timing.RefreshHz)
	}
	if mode := timing.Mode(); mode != "1920x1080p60" {
		t.Errorf("mode = %q, want 1920x1080p60", mode)
	}
}

func TestTimingRoundTrip(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	for i, descriptor := range []Descriptor{parsed.Descriptors[0], parsed.CTA.DTDs[0], parsed.CTA.DTDs[1]} {
		timing := descriptor.Timing
		if timing == nil {
			t.Fatalf("descriptor %d is %q, want a detailed timing", i, descriptor.Kind)
		}
		hz := float64(timing.PixelClockKHz) * 1000 / float64(timing.HTotal*timing.VTotal)
		if !closeTo(timing.RefreshHz, hz) || !closeTo(hz, 60) {
			t.Errorf("timing %d: %d kHz over %d by %d = %v Hz, want 60.000", i, timing.PixelClockKHz, timing.HTotal, timing.VTotal, hz)
		}

		var out [descriptorSize]byte
		encodeTiming(out[:], timing)
		if out != descriptor.Raw {
			t.Errorf("timing %d re-encodes to % X, want % X", i, out, descriptor.Raw)
		}
	}

	secondary := parsed.CTA.DTDs[1].Timing
	want := Timing{
		PixelClockKHz: 74250,
		HActive:       1280, HBlank: 370, HTotal: 1650,
		VActive: 720, VBlank: 30, VTotal: 750,
		HSyncOffset: 110, HSyncWidth: 40,
		VSyncOffset: 5, VSyncWidth: 5,
		WidthMM: 476, HeightMM: 268,
		Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true,
		RefreshHz: 60,
	}
	if got := *secondary; got != want {
		t.Errorf("secondary timing = %+v, want %+v", got, want)
	}
	if mode := secondary.Mode(); mode != "1280x720p60" {
		t.Errorf("secondary mode = %q, want 1280x720p60", mode)
	}
}

func TestFixtureCTA(t *testing.T) {
	parsed, err := Decode(fixture(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	cta := parsed.CTA
	if cta == nil {
		t.Fatal("no cta block")
	}
	if cta.Revision != 3 || cta.DTDOffset != 17 {
		t.Errorf("cta revision %d offset %d, want 3 17", cta.Revision, cta.DTDOffset)
	}
	if cta.Underscan || cta.BasicAudio || !cta.YCbCr444 || !cta.YCbCr422 || cta.NativeDTDs != 0 {
		t.Errorf("cta flags = %+v, want underscan 0 audio 0 444 1 422 1 native 0", cta)
	}
	if len(cta.Blocks) != 2 {
		t.Fatalf("%d data blocks, want 2", len(cta.Blocks))
	}

	video := cta.Blocks[0]
	if video.Tag != CTAVideo {
		t.Fatalf("first data block tag = %d, want %d", video.Tag, CTAVideo)
	}
	wantVICs := []uint8{4, 31, 20, 19, 1, 16}
	if len(video.Video) != len(wantVICs) {
		t.Fatalf("%d short video descriptors, want %d", len(video.Video), len(wantVICs))
	}
	for i, vic := range wantVICs {
		if video.Video[i].VIC != vic || video.Video[i].Native {
			t.Errorf("svd %d = %+v, want vic %d not native", i, video.Video[i], vic)
		}
	}

	vendor := cta.Blocks[1]
	if vendor.Tag != CTAVendor || vendor.Vendor == nil {
		t.Fatalf("second data block = %+v, want a vendor block", vendor)
	}
	if vendor.Vendor.Kind != VendorHDMI14 || vendor.Vendor.OUI != "00-0C-03" {
		t.Errorf("vendor = %+v, want the hdmi 1.4 vsdb", vendor.Vendor)
	}
	if vendor.Vendor.SourcePhysicalAddress != "1.0.0.0" {
		t.Errorf("source physical address = %q, want 1.0.0.0", vendor.Vendor.SourcePhysicalAddress)
	}

	for _, block := range cta.Blocks {
		if block.Tag == CTAAudio || block.Tag == CTASpeakers {
			t.Errorf("cta advertises audio: %+v", block)
		}
	}
	if len(cta.DTDs) != 2 {
		t.Fatalf("%d extension detailed timings, want 2", len(cta.DTDs))
	}
	if cta.DTDs[0].Raw != parsed.Descriptors[0].Raw {
		t.Errorf("extension dtd A = % X, want the base preferred timing % X", cta.DTDs[0].Raw, parsed.Descriptors[0].Raw)
	}
	if !allZero(cta.Padding) {
		t.Errorf("cta padding is not zero: % X", cta.Padding)
	}
}

func TestPreservesUnknownBlocks(t *testing.T) {
	data := fixture(t)

	unknownDescriptor := [descriptorSize]byte{0x00, 0x00, 0x00, 0xF0, 0x00, 0xDE, 0xAD, 0xBE, 0xEF}
	copy(data[descriptorBase+3*descriptorSize:], unknownDescriptor[:])
	data[BlockSize+11] = 0xC5
	data = repaired(data)

	parsed, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	descriptor := parsed.Descriptors[3]
	if descriptor.Kind != DescriptorUnknown || descriptor.Tag != 0xF0 {
		t.Fatalf("descriptor 4 = %+v, want an unknown 0xF0 tag", descriptor)
	}
	if descriptor.Raw != unknownDescriptor {
		t.Errorf("descriptor 4 raw = % X, want % X", descriptor.Raw, unknownDescriptor)
	}

	block := parsed.CTA.Blocks[1]
	if block.Tag != 6 || block.Vendor != nil {
		t.Fatalf("second data block = %+v, want an undecoded tag 6", block)
	}

	out, err := parsed.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatal("unknown descriptor or data block was not preserved byte for byte")
	}
}

func TestPreservesUnknownExtension(t *testing.T) {
	data := fixture(t)
	for i := BlockSize; i < Size; i++ {
		data[i] = 0
	}
	data[BlockSize] = 0x70
	data[BlockSize+1] = 0x13
	data = repaired(data)

	parsed, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if parsed.CTA != nil {
		t.Fatal("a non cta extension decoded as cta")
	}

	out, err := parsed.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatal("unknown extension block was not preserved byte for byte")
	}
}

func TestHardRejects(t *testing.T) {
	data := fixture(t)

	blankExtension := bytes.Clone(data)
	for i := BlockSize; i < Size; i++ {
		blankExtension[i] = 0
	}

	noTiming := bytes.Clone(data)
	for i := descriptorBase; i < descriptorBase+descriptorSize; i++ {
		noTiming[i] = 0
	}
	for i := BlockSize + 17; i < BlockSize+53; i++ {
		noTiming[i] = 0
	}

	tests := []struct {
		name string
		data []byte
		want RejectKind
	}{
		{"oversized", append(bytes.Clone(data), 0x00), RejectOversize},
		{"short", data[:Size-1], RejectSize},
		{"header", repaired(mutate(data, map[int]byte{6: 0x00})), RejectHeader},
		{"base checksum", mutate(data, map[int]byte{BlockSize - 1: data[BlockSize-1] + 1}), RejectChecksum},
		{"extension checksum", mutate(data, map[int]byte{Size - 1: data[Size-1] + 1}), RejectChecksum},
		{"version 1.2", repaired(mutate(data, map[int]byte{19: 0x02})), RejectVersion},
		{"extension claimed but empty", repaired(mutate(blankExtension, map[int]byte{extensionCount: 0x01})), RejectExtensionCount},
		{"extension present but uncounted", repaired(mutate(data, map[int]byte{extensionCount: 0x00})), RejectExtensionCount},
		{"two extensions", repaired(mutate(data, map[int]byte{extensionCount: 0x02})), RejectExtensionCount},
		{"cta revision", repaired(mutate(data, map[int]byte{BlockSize + 1: 0x02})), RejectCTA},
		{"cta offset under", repaired(mutate(data, map[int]byte{BlockSize + 2: 0x03})), RejectCTA},
		{"cta offset over", repaired(mutate(data, map[int]byte{BlockSize + 2: 0x80})), RejectCTA},
		{"cta ragged data block", repaired(mutate(data, map[int]byte{BlockSize + 11: 0x6F})), RejectCTA},
		{"cta dirty dtd region", repaired(mutate(data, map[int]byte{BlockSize + 120: 0x01})), RejectCTA},
		{"no detailed timing", repaired(noTiming), RejectNoTiming},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := Decode(tt.data)
			if err == nil {
				t.Fatalf("decoded %+v, want reject %q", parsed, tt.want)
			}

			var rejected *RejectError
			if !errors.As(err, &rejected) {
				t.Fatalf("error %v is not a reject", err)
			}
			if rejected.Kind != tt.want {
				t.Fatalf("rejected as %q, want %q: %v", rejected.Kind, tt.want, err)
			}
		})
	}
}

func TestValidFixtureIsNotRejected(t *testing.T) {
	if _, err := Decode(fixture(t)); err != nil {
		t.Fatalf("the factory edid was rejected: %v", err)
	}
}
