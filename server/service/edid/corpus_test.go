package edid

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Real blobs from linuxhw/EDID at the commit gen_edid_profiles.go pins; see
// testdata/corpus/SOURCES. They are here because the shipped library is 25
// entries curated for one mode, which leaves audio, speakers, the HDMI Forum
// VSDB, analog input, borders, stereo and zero length data blocks decoded by
// nothing. Every literal below is cross checked against edid-decode by
// oracle_test.go.

const corpusDir = "testdata/corpus"

type corpusEntry struct {
	file         string
	manufacturer string
	product      uint16
	serial       uint32
	week         uint8
	year         int
	version      string
	digital      bool
	widthCM      uint8
	heightCM     uint8
	gamma        float64
	model        string
	serialText   string
	descriptors  string
	established  string
	standard     string
	preferred    Timing
	ctaBlocks    string
	vics         string
	audio        string
	speakers     string
	ctaDTDs      int
}

var corpusEntries = []corpusEntry{
	{
		file:         "Analog-Samsung-SAM0017-0AC443B4B859.bin",
		manufacturer: "SAM", product: 0x0017, serial: 1195847989, week: 24, year: 2002,
		version: "1.3", digital: false, widthCM: 30, heightCM: 23, gamma: 2.78,
		model: "SyncMaster", serialText: "H9NT604549",
		descriptors: "timing,range_limits,name,serial",
		established: "720x400@70 640x480@60 640x480@72 640x480@75 800x600@56 800x600@60 800x600@72 800x600@75 1024x768@60 1024x768@70 1024x768@75",
		standard:    "640x480@75 4:3,800x600@75 4:3,1024x768@75 4:3,640x480@60 4:3",
		preferred: Timing{PixelClockKHz: 65000, HActive: 1024, HBlank: 320, HTotal: 1344, VActive: 768, VBlank: 38, VTotal: 806,
			HSyncOffset: 24, HSyncWidth: 136, VSyncOffset: 3, VSyncWidth: 6, WidthMM: 304, HeightMM: 228,
			Sync: SyncDigitalSeparate, RefreshHz: 60.00384024577573},
	},
	{
		file:         "Analog-Sony-SNY2601-F15DAD249021.bin",
		manufacturer: "SNY", product: 0x2601, serial: 16843009, week: 1, year: 2007,
		version: "1.3", digital: false, widthCM: 71, heightCM: 40, gamma: 2.2,
		model: "SONY TV", serialText: "",
		descriptors: "timing,timing,range_limits,name",
		established: "640x480@60 640x480@72 640x480@75 800x600@56 800x600@60 800x600@72 800x600@75 1024x768@60 1024x768@70",
		standard:    "640x480@60 4:3,800x600@60 4:3,1024x768@60 4:3,1280x720@60 16:9,640x480@75 4:3,800x600@75 4:3,640x480@85 4:3,800x600@85 4:3",
		preferred: Timing{PixelClockKHz: 72000, HActive: 1360, HBlank: 160, HTotal: 1520, VActive: 768, VBlank: 22, VTotal: 790,
			HSyncOffset: 48, HSyncWidth: 32, VSyncOffset: 3, VSyncWidth: 5, WidthMM: 71, HeightMM: 40,
			Sync: SyncDigitalSeparate, HSyncPositive: true, RefreshHz: 59.96002664890073},
	},
	{
		file:         "Digital-Panasonic-MEIA0A6-D5C15A943730.bin",
		manufacturer: "MEI", product: 0xA0A6, serial: 16843009, week: 0, year: 2010,
		version: "1.3", digital: true, widthCM: 0, heightCM: 0, gamma: 2.2,
		model: "Panasonic-TV", serialText: "",
		descriptors: "timing,timing,name,range_limits",
		established: "",
		standard:    "",
		preferred: Timing{PixelClockKHz: 148500, HActive: 1920, HBlank: 720, HTotal: 2640, VActive: 1080, VBlank: 45, VTotal: 1125,
			HSyncOffset: 528, HSyncWidth: 44, VSyncOffset: 4, VSyncWidth: 5, WidthMM: 698, HeightMM: 392,
			Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true, RefreshHz: 50},
		ctaBlocks: "rev3 flags 0x72,tag 2 len 16,tag 1 len 3,vendor hdmi_1.4 00-0C-03 2.0.0.0,extended 5",
		vics:      "31* 16* 20 5 32 19 4 18 3 17 2 22 7 21 6 1",
		audio:     "lpcm 2ch [32000 44100 48000] [16] 0",
		ctaDTDs:   4,
	},
	{
		file:         "Digital-Panasonic-MEIA296-F205F59911ED.bin",
		manufacturer: "MEI", product: 0xA296, serial: 16843009, week: 0, year: 2019,
		version: "1.3", digital: true, widthCM: 128, heightCM: 72, gamma: 2.2,
		model: "Panasonic-TV", serialText: "",
		descriptors: "timing,timing,name,range_limits",
		established: "640x480@60 1024x768@60",
		standard:    "640x480@60 4:3,1024x768@60 4:3",
		preferred: Timing{PixelClockKHz: 594000, HActive: 3840, HBlank: 560, HTotal: 4400, VActive: 2160, VBlank: 90, VTotal: 2250,
			HSyncOffset: 176, HSyncWidth: 88, VSyncOffset: 8, VSyncWidth: 10, WidthMM: 698, HeightMM: 392,
			Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true, RefreshHz: 60},
		ctaBlocks: "rev3 flags 0xF0,tag 2 len 23,tag 1 len 12,tag 4 len 3,vendor hdmi_1.4 00-0C-03 2.0.0.0,vendor hdmi_forum C4-5D-D8 ,extended 0,extended 5,extended 15,extended 6,extended 1,extended 1,extended 17",
		vics:      "97 96 16 31 102 101 5 20 32 33 34 4 19 3 18 7 22 93 94 95 98 99 100",
		audio:     "lpcm 6ch [32000 44100 48000] [16] 0;ac-3 6ch [32000 44100 48000] [] 640;e-ac-3 8ch [32000 44100 48000] [] 8;mlp 8ch [48000] [] 24",
		speakers:  "FL/FR LFE FC RL/RR",
		ctaDTDs:   1,
	},
	{
		file:         "Digital-Samsung-SAM0027-3CFB11465D51.bin",
		manufacturer: "SAM", product: 0x0027, serial: 1095840055, week: 7, year: 2002,
		version: "1.3", digital: true, widthCM: 32, heightCM: 24, gamma: 2.07,
		model: "SyncMaster", serialText: "HGDT203221",
		descriptors: "timing,range_limits,name,serial",
		established: "720x400@70 720x400@88 640x480@60 640x480@67 640x480@72 640x480@75 800x600@56 800x600@60 800x600@72 800x600@75 832x624@75 1024x768@87i 1024x768@60 1024x768@70 1024x768@75 1152x870@75",
		standard:    "640x480@85 4:3,800x600@85 4:3,1024x768@85 4:3,1280x1024@60 5:4",
		preferred: Timing{PixelClockKHz: 94500, HActive: 1024, HBlank: 352, HTotal: 1376, VActive: 768, VBlank: 40, VTotal: 808,
			HSyncOffset: 48, HSyncWidth: 96, VSyncOffset: 1, VSyncWidth: 3, WidthMM: 312, HeightMM: 234,
			Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true, RefreshHz: 84.99669007598435},
		ctaBlocks: "rev3 flags 0x61,tag 1 len 3,tag 4 len 3,vendor hdmi_1.4 00-0C-03 2.0.0.0,tag 2 len 3,extended 0",
		vics:      "16* 4* 2",
		audio:     "lpcm 2ch [32000 44100 48000] [16 20 24] 0",
		speakers:  "FL/FR",
		ctaDTDs:   1,
	},
	{
		file:         "Digital-Samsung-SAM0D22-F20C544C86F0.bin",
		manufacturer: "SAM", product: 0x0D22, serial: 1129791798, week: 46, year: 2019,
		version: "1.3", digital: true, widthCM: 60, heightCM: 34, gamma: 2.2,
		model: "S27F350", serialText: "H4ZMB00793",
		descriptors: "timing,range_limits,name,serial",
		established: "720x400@70 640x480@60 640x480@67 640x480@72 640x480@75 800x600@56 800x600@60 800x600@72 800x600@75 832x624@75 1024x768@60 1024x768@70 1024x768@75 1280x1024@75 1152x870@75",
		standard:    "1152x864@75 4:3,1280x720@60 16:9,1280x800@60 16:10,1280x1024@60 5:4,1440x900@60 16:10,1600x900@60 16:9,1680x1050@60 16:10",
		preferred: Timing{PixelClockKHz: 148500, HActive: 1920, HBlank: 280, HTotal: 2200, VActive: 1080, VBlank: 45, VTotal: 1125,
			HSyncOffset: 88, HSyncWidth: 44, VSyncOffset: 4, VSyncWidth: 5, WidthMM: 598, HeightMM: 336,
			Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true, RefreshHz: 60},
		ctaBlocks: "foreign 0x00",
	},
	{
		file:         "Digital-Samsung-SAM0FEF-FE7B0F681537.bin",
		manufacturer: "SAM", product: 0x0FEF, serial: 16780800, week: 1, year: 2019,
		version: "1.3", digital: true, widthCM: 111, heightCM: 62, gamma: 2.2,
		model: "SAMSUNG", serialText: "",
		descriptors: "timing,timing,range_limits,name",
		established: "720x400@70 640x480@60 640x480@67 640x480@72 640x480@75 800x600@60 800x600@72 800x600@75 832x624@75 1024x768@60 1024x768@70 1024x768@75 1280x1024@75 1152x870@75",
		standard:    "1152x864@75 4:3,1280x720@60 16:9,1280x800@60 16:10,1280x1024@60 5:4,1440x900@60 16:10,1600x900@60 16:9,1680x1050@60 16:10,1920x1080@60 16:9",
		preferred: Timing{PixelClockKHz: 297000, HActive: 3840, HBlank: 560, HTotal: 4400, VActive: 2160, VBlank: 90, VTotal: 2250,
			HSyncOffset: 176, HSyncWidth: 88, VSyncOffset: 8, VSyncWidth: 10, WidthMM: 708, HeightMM: 398,
			Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true, RefreshHz: 30},
		ctaBlocks: "rev3 flags 0xF0,tag 2 len 22,tag 1 len 9,tag 4 len 3,extended 0,extended 5,vendor hdmi_1.4 00-0C-03 1.0.0.0,tag 0 len 0,tag 0 len 0,tag 0 len 0,tag 0 len 0,tag 0 len 0,tag 0 len 0,tag 0 len 0,tag 0 len 0,tag 0 len 0,extended 6,extended 14,extended 1",
		vics:      "74 16 31 4 19 5 20 32 33 34 93 94 95 74 74 74 98 100 7 22 3 18",
		audio:     "lpcm 2ch [32000 44100 48000] [16 20 24] 0;ac-3 6ch [32000 44100 48000] [] 640;e-ac-3 8ch [32000 44100 48000] [] 0",
		speakers:  "FL/FR",
		ctaDTDs:   1,
	},
	{
		file:         "Digital-Samsung-SAM7017-F42E76586CC3.bin",
		manufacturer: "SAM", product: 0x7017, serial: 16780800, week: 1, year: 2020,
		version: "1.3", digital: true, widthCM: 129, heightCM: 72, gamma: 2.2,
		model: "SONY AVSYSTEM", serialText: "",
		descriptors: "timing,timing,name,range_limits",
		established: "640x480@60",
		standard:    "1152x864@75 4:3,1280x720@60 16:9,1280x800@60 16:10,1280x1024@60 5:4,1440x900@60 16:10,1600x900@60 16:9,1680x1050@60 16:10,1920x1080@60 16:9",
		preferred: Timing{PixelClockKHz: 74250, HActive: 1280, HBlank: 370, HTotal: 1650, VActive: 720, VBlank: 30, VTotal: 750,
			HSyncOffset: 110, HSyncWidth: 40, VSyncOffset: 5, VSyncWidth: 5, WidthMM: 708, HeightMM: 398,
			Sync: SyncDigitalSeparate, HSyncPositive: true, VSyncPositive: true, RefreshHz: 60},
		ctaBlocks: "rev3 flags 0x71,tag 2 len 4,tag 1 len 24,tag 4 len 3,vendor hdmi_1.4 00-0C-03 1.0.0.0",
		vics:      "4* 19 3 18",
		audio:     "lpcm 2ch [32000 44100 48000] [16 20 24] 0;ac-3 6ch [32000 44100 48000] [] 680;e-ac-3 8ch [32000 44100 48000] [] 0;mlp 8ch [48000 96000 192000] [] 0;lpcm 8ch [32000 44100 48000 88200 96000 176400 192000] [16 20 24] 0;dts 6ch [32000 44100 48000 88200 96000] [] 1536;dts-hd 8ch [44100 48000 88200 96000 176400 192000] [] 8;dsd 6ch [44100] [] 0",
		speakers:  "FL/FR LFE FC RL/RR RC RLC/RRC",
		ctaDTDs:   4,
	},
	{
		file:         "Digital-Samsung-SDC414D-CB33C5F31E5A.bin",
		manufacturer: "SDC", product: 0x414D, serial: 0, week: 17, year: 2020,
		version: "1.4", digital: true, widthCM: 34, heightCM: 21, gamma: 2.2,
		model: "", serialText: "",
		descriptors: "timing,timing,unknown 0xFE,unknown 0x00",
		established: "",
		standard:    "",
		preferred: Timing{PixelClockKHz: 528190, HActive: 3456, HBlank: 560, HTotal: 4016, VActive: 2160, VBlank: 32, VTotal: 2192,
			HSyncOffset: 48, HSyncWidth: 32, VSyncOffset: 8, VSyncWidth: 8, WidthMM: 336, HeightMM: 210,
			Stereo: 1, Sync: SyncDigitalSeparate, HSyncPositive: true, RefreshHz: 60.000645229301774},
		ctaBlocks: "rev3 flags 0x00,extended 5,extended 6",
	},
	{
		file:         "Digital-Sony-MS_0003-DA6FC6C2CDD8.bin",
		manufacturer: "MS_", product: 0x0003, serial: 3, week: 0, year: 2002,
		version: "1.4", digital: true, widthCM: 19, heightCM: 12, gamma: 0,
		model: "JDI_8.9_LCD", serialText: "",
		descriptors: "timing,name,range_limits,dummy",
		established: "",
		standard:    "",
		preferred: Timing{PixelClockKHz: 271380, HActive: 2560, HBlank: 232, HTotal: 2792, VActive: 1600, VBlank: 20, VTotal: 1620,
			HSyncOffset: 120, HSyncWidth: 32, VSyncOffset: 13, VSyncWidth: 3, WidthMM: 192, HeightMM: 120,
			Sync: SyncDigitalSeparate, HSyncPositive: true, RefreshHz: 59.99946938342354},
	},
}

// Real monitors the strict validator refuses, so the reject paths are exercised
// by blobs nobody edited into shape.
var corpusRejects = []struct {
	file string
	kind RejectKind
}{
	{"Analog-Sony-SNY0000-119C70A7CE0B.bin", RejectVersion},
	{"Digital-Samsung-SAM04EA-D0B977BDC18A.bin", RejectCTA},
	{"Digital-Samsung-SAM0B32-1CB13EE08621.bin", RejectCTA},
	{"Digital-Samsung-SAM7435-B5E58036B270.bin", RejectChecksum},
	{"Digital-Samsung-SDC4196-210D5A00AAA3.bin", RejectNoTiming},
}

func corpus(t testing.TB) map[string][]byte {
	t.Helper()

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}

	blobs := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".bin") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(corpusDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		blobs[entry.Name()] = data
	}
	if len(blobs) != len(corpusEntries)+len(corpusRejects) {
		t.Fatalf("%d corpus files, %d described", len(blobs), len(corpusEntries)+len(corpusRejects))
	}
	return blobs
}

func TestCorpusDecodes(t *testing.T) {
	blobs := corpus(t)

	for _, want := range corpusEntries {
		t.Run(want.file, func(t *testing.T) {
			raw, ok := blobs[want.file]
			if !ok {
				t.Fatalf("%s is not in %s", want.file, corpusDir)
			}
			blob, err := Normalize(raw)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			parsed, err := Decode(blob)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}

			got := describe(parsed)
			got.file = want.file
			if got != want {
				diff(t, got, want)
			}

			out, err := parsed.Encode()
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if !bytes.Equal(out, blob) {
				t.Error("decode then encode does not reproduce the blob")
			}
		})
	}
}

func TestCorpusRejects(t *testing.T) {
	blobs := corpus(t)

	for _, want := range corpusRejects {
		t.Run(want.file, func(t *testing.T) {
			blob, err := Normalize(blobs[want.file])
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			parsed, err := Decode(blob)
			if err == nil {
				t.Fatalf("decoded %+v, want reject %q", parsed, want.kind)
			}

			var rejected *RejectError
			if !errors.As(err, &rejected) {
				t.Fatalf("error %v is not a reject", err)
			}
			if rejected.Kind != want.kind {
				t.Fatalf("rejected as %q, want %q: %v", rejected.Kind, want.kind, err)
			}
		})
	}
}

// The corpus earns its place only while it still reaches the branches the
// shipped library does not.
func TestCorpusCoversWhatTheLibraryDoesNot(t *testing.T) {
	reached := map[string]bool{}
	for _, entry := range corpusEntries {
		if !entry.digital {
			reached["analog"] = true
		}
		if entry.gamma == 0 {
			reached["gamma in extension"] = true
		}
		if strings.Contains(entry.descriptors, "dummy") {
			reached["dummy descriptor"] = true
		}
		if strings.Contains(entry.descriptors, "unknown") {
			reached["unknown descriptor"] = true
		}
		if strings.Contains(entry.descriptors, "serial") {
			reached["serial descriptor"] = true
		}
		if entry.version == "1.4" {
			reached["edid 1.4"] = true
		}
		if entry.preferred.Stereo != 0 {
			reached["stereo"] = true
		}
		if strings.Contains(entry.established, "1152x870@75") {
			reached["established block iii bit"] = true
		}
		if strings.Contains(entry.established, "1024x768@87i") {
			reached["interlaced established mode"] = true
		}
		for _, aspect := range aspectRatios {
			if strings.Contains(entry.standard, " "+aspect) {
				reached["standard aspect "+aspect] = true
			}
		}
		if strings.Contains(entry.ctaBlocks, "foreign") {
			reached["non cta extension"] = true
		}
		if strings.Contains(entry.ctaBlocks, "hdmi_forum") {
			reached["hdmi forum vsdb"] = true
		}
		if strings.Contains(entry.ctaBlocks, "hdmi_1.4") {
			reached["hdmi 1.4 vsdb"] = true
		}
		if strings.Contains(entry.ctaBlocks, "extended") {
			reached["cta extended block"] = true
		}
		if strings.Contains(entry.ctaBlocks, "tag 0 len 0") {
			reached["zero length data block"] = true
		}
		if strings.Contains(entry.vics, "*") {
			reached["native short video descriptor"] = true
		}
		if entry.speakers != "" {
			reached["speaker allocation"] = true
		}
		if strings.Contains(entry.audio, "16 20 24") {
			reached["lpcm bit depths"] = true
		}
		for _, format := range []string{"ac-3", "e-ac-3", "dts", "dts-hd", "mlp", "dsd"} {
			if strings.Contains(entry.audio, format+" ") {
				reached["audio "+format] = true
			}
		}
	}

	want := []string{
		"analog", "gamma in extension", "dummy descriptor", "unknown descriptor",
		"serial descriptor", "edid 1.4", "stereo", "established block iii bit",
		"interlaced established mode", "standard aspect 16:10", "standard aspect 4:3",
		"standard aspect 5:4", "standard aspect 16:9", "non cta extension",
		"hdmi forum vsdb", "hdmi 1.4 vsdb", "cta extended block", "zero length data block",
		"native short video descriptor", "speaker allocation", "lpcm bit depths",
		"audio ac-3", "audio e-ac-3", "audio dts", "audio dts-hd", "audio mlp", "audio dsd",
	}
	for _, name := range want {
		if !reached[name] {
			t.Errorf("no corpus entry reaches %s", name)
		}
	}
}

func describe(parsed *EDID) corpusEntry {
	entry := corpusEntry{
		manufacturer: parsed.Manufacturer,
		product:      parsed.ProductCode,
		serial:       parsed.Serial,
		week:         parsed.Week,
		year:         parsed.Year,
		version:      fmt.Sprintf("%d.%d", parsed.Version, parsed.Revision),
		digital:      parsed.Display.Digital,
		widthCM:      parsed.Display.WidthCM,
		heightCM:     parsed.Display.HeightCM,
		gamma:        math.Round(parsed.Display.Gamma*100) / 100,
		model:        parsed.Name(),
		established:  strings.Join(parsed.Established.Modes, " "),
	}

	var descriptors, standard []string
	for _, descriptor := range parsed.Descriptors {
		if descriptor.Kind == DescriptorUnknown {
			descriptors = append(descriptors, fmt.Sprintf("unknown 0x%02X", descriptor.Tag))
			continue
		}
		descriptors = append(descriptors, string(descriptor.Kind))
		if descriptor.Kind == DescriptorSerial {
			entry.serialText = descriptor.Text
		}
	}
	entry.descriptors = strings.Join(descriptors, ",")

	for _, timing := range parsed.StandardTimings {
		if timing.Used {
			standard = append(standard, fmt.Sprintf("%dx%d@%d %s", timing.Horizontal, timing.Vertical, timing.RefreshHz, timing.AspectRatio))
		}
	}
	entry.standard = strings.Join(standard, ",")

	if timing := parsed.PreferredTiming(); timing != nil {
		entry.preferred = *timing
	}
	if parsed.CTA == nil {
		if len(parsed.Extension) > 0 {
			entry.ctaBlocks = fmt.Sprintf("foreign 0x%02X", parsed.Extension[0])
		}
		return entry
	}

	blocks := []string{fmt.Sprintf("rev%d flags 0x%02X", parsed.CTA.Revision, parsed.CTA.FlagByte)}
	var vics, audio []string
	for _, block := range parsed.CTA.Blocks {
		switch block.Tag {
		case CTAVendor:
			blocks = append(blocks, fmt.Sprintf("vendor %s %s %s", block.Vendor.Kind, block.Vendor.OUI, block.Vendor.SourcePhysicalAddress))
		case CTAExtended:
			blocks = append(blocks, fmt.Sprintf("extended %d", block.ExtendedTag))
		default:
			blocks = append(blocks, fmt.Sprintf("tag %d len %d", block.Tag, len(block.Raw)-1))
		}
		for _, video := range block.Video {
			if video.Native {
				vics = append(vics, fmt.Sprintf("%d*", video.VIC))
				continue
			}
			vics = append(vics, fmt.Sprintf("%d", video.VIC))
		}
		for _, descriptor := range block.Audio {
			audio = append(audio, fmt.Sprintf("%s %dch %v %v %d", descriptor.FormatName, descriptor.Channels,
				descriptor.SampleRatesHz, descriptor.BitDepths, descriptor.MaxBitrateKbps))
		}
		if block.Speakers != nil {
			entry.speakers = strings.Join(block.Speakers.Present, " ")
		}
	}
	entry.ctaBlocks = strings.Join(blocks, ",")
	entry.vics = strings.Join(vics, " ")
	entry.audio = strings.Join(audio, ";")
	entry.ctaDTDs = len(parsed.CTA.DTDs)
	return entry
}

func diff(t *testing.T, got, want corpusEntry) {
	t.Helper()

	if got.preferred != want.preferred {
		t.Errorf("preferred timing = %+v, want %+v", got.preferred, want.preferred)
	}
	fields := []struct {
		name      string
		got, want any
	}{
		{"manufacturer", got.manufacturer, want.manufacturer},
		{"product", got.product, want.product},
		{"serial", got.serial, want.serial},
		{"week", got.week, want.week},
		{"year", got.year, want.year},
		{"version", got.version, want.version},
		{"digital", got.digital, want.digital},
		{"width cm", got.widthCM, want.widthCM},
		{"height cm", got.heightCM, want.heightCM},
		{"gamma", got.gamma, want.gamma},
		{"model", got.model, want.model},
		{"serial text", got.serialText, want.serialText},
		{"descriptors", got.descriptors, want.descriptors},
		{"established", got.established, want.established},
		{"standard timings", got.standard, want.standard},
		{"cta blocks", got.ctaBlocks, want.ctaBlocks},
		{"short video descriptors", got.vics, want.vics},
		{"audio", got.audio, want.audio},
		{"speakers", got.speakers, want.speakers},
		{"cta detailed timings", got.ctaDTDs, want.ctaDTDs},
	}
	for _, field := range fields {
		if field.got != field.want {
			t.Errorf("%s = %v, want %v", field.name, field.got, field.want)
		}
	}
}
