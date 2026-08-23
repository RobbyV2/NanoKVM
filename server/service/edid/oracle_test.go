//go:build edidoracle

package edid

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Differential test against edid-decode, the linuxtv reference decoder. Run by
// scripts/edid-repro.sh, which installs it. Everything asserted here is a field
// two independent implementations agree on, so a decode_test.go literal that
// drifted from the spec fails here rather than pinning the drift.

const oracleTool = "edid-decode"

type reference struct {
	version, revision    uint8
	manufacturer         string
	product              uint16
	serial               uint32
	week                 int
	year                 int
	digital              bool
	widthCM, heightCM    uint8
	gamma                float64
	gammaInExtension     bool
	extensions           int
	name, serialText     string
	rangeVMin, rangeVMax int
	rangeHMin, rangeHMax int
	rangeClock           int
	standard             []refStandard
	dtds                 []refTiming
	cta                  *refCTA
}

type refStandard struct {
	horizontal, vertical int
	aspect               string
	refresh              int
}

type refTiming struct {
	hActive, vActive              int
	interlaced                    bool
	clockKHz                      int
	widthMM, heightMM             int
	hFront, hSync, hBack, hBorder int
	vFront, vSync, vBack, vBorder int
	hPositive, vPositive          bool
}

type refCTA struct {
	revision   uint8
	underscan  bool
	basicAudio bool
	ycbcr444   bool
	ycbcr422   bool
	nativeDTDs uint8
	vics       []SVD
	audio      []refAudio
	speakers   []string
	vendors    []refVendor
}

type refAudio struct {
	format   string
	channels int
	rates    []int
	depths   []int
	byte2    int
}

type refVendor struct {
	oui string
	spa string
}

var (
	reVersion    = regexp.MustCompile(`^ {2}EDID Structure Version & Revision: (\d+)\.(\d+)$`)
	reMfr        = regexp.MustCompile(`^ {4}Manufacturer: (\S+)$`)
	reModel      = regexp.MustCompile(`^ {4}Model: (\d+)$`)
	reSerial     = regexp.MustCompile(`^ {4}Serial Number: (\d+)$`)
	reMade       = regexp.MustCompile(`^ {4}(?:Made in|Model year): (?:week (\d+) of )?(\d+)$`)
	reSize       = regexp.MustCompile(`^ {4}Maximum image size: (\d+) cm x (\d+) cm$`)
	reGamma      = regexp.MustCompile(`^ {4}Gamma: ([\d.]+)$`)
	reExtensions = regexp.MustCompile(`^ {2}Extension blocks: (\d+)$`)
	reName       = regexp.MustCompile(`^ {4}Display Product Name: '(.*)'$`)
	reSerialText = regexp.MustCompile(`^ {4}Display Product Serial Number: '(.*)'$`)
	reRanges     = regexp.MustCompile(`^ {6}Monitor ranges(?: \(.*\))?: (\d+)-(\d+) Hz V, (\d+)-(\d+) kHz H(?:, max dotclock (\d+) MHz)?$`)
	reDTD        = regexp.MustCompile(`^ +DTD \d+: +(\d+)x(\d+)(i?) +[\d.]+ Hz +\S+ +[\d.]+ kHz +([\d.]+) MHz \((?:[^()]*, )?(\d+) mm x (\d+) mm\)$`)
	reH          = regexp.MustCompile(`^ +Hfront +(\d+) +Hsync +(\d+) +Hback +(-?\d+) +Hpol ([PN])(?: +Hborder +(\d+))?`)
	reV          = regexp.MustCompile(`^ +Vfront +(\d+) +Vsync +(\d+) +Vback +(-?\d+) +Vpol ([PN])(?: +Vborder +(\d+))?`)
	reMode       = regexp.MustCompile(`^ {4}\S.*?: +(\d+)x(\d+)i? +([\d.]+) Hz +(\S+)`)
	reVIC        = regexp.MustCompile(`^ {4}VIC +(\d+):`)
	reCTARev     = regexp.MustCompile(`^ {2}Revision: (\d+)$`)
	reNativeDTD  = regexp.MustCompile(`^ {2}Native detailed modes: (\d+)$`)
	reVSDB       = regexp.MustCompile(`^ {2}Vendor-Specific Data Block \(.*\), OUI (\S+):$`)
	reSPA        = regexp.MustCompile(`^ {4}Source physical address: (\S+)$`)
	reChannels   = regexp.MustCompile(`^ {6}Max channels: (\d+)$`)
	reRates      = regexp.MustCompile(`^ {6}Supported sample rates \(kHz\): (.*)$`)
	reDepths     = regexp.MustCompile(`^ {6}Supported sample sizes \(bits\): (.*)$`)
	reBitrate    = regexp.MustCompile(`^ {6}Maximum bit rate: (\d+) kb/s$`)
	reDependent  = regexp.MustCompile(`^ {6}Audio Format Code dependent value: 0x([0-9a-f]+)$`)
	reSpeaker    = regexp.MustCompile(`^ {4}(\S+) - .*$`)
	reHeading    = regexp.MustCompile(`^( {2,4})(\S.*):$`)
)

var audioFormatNames = map[string]string{
	"Linear PCM":            "lpcm",
	"AC-3":                  "ac-3",
	"MPEG 1 (Layers 1 & 2)": "mpeg-1",
	"MP3 (MPEG1 Layer 3)":   "mp3",
	"MPEG2 (multichannel)":  "mpeg-2",
	"AAC LC":                "aac-lc",
	"DTS":                   "dts",
	"ATRAC":                 "atrac",
	"One Bit Audio":         "dsd",
	"Enhanced AC-3 (DD+)":   "e-ac-3",
	"DTS-HD":                "dts-hd",
	"MAT (MLP)":             "mlp",
	"DST":                   "dst",
	"WMA Pro":               "wma-pro",
}

var speakerNamesRef = map[string]string{
	"FL/FR":   "FL/FR",
	"LFE1":    "LFE",
	"FC":      "FC",
	"BL/BR":   "RL/RR",
	"BC":      "RC",
	"FLc/FRc": "FLC/FRC",
	"RLC/RRC": "RLC/RRC",
}

func TestOracleAgreesOnEveryShippedBlob(t *testing.T) {
	if _, err := exec.LookPath(oracleTool); err != nil {
		t.Fatalf("%s is not on PATH; run scripts/edid-repro.sh", oracleTool)
	}

	blobs := map[string][]byte{"testdata/E21_NanoKVM.bin": fixture(t)}
	for name, data := range corpus(t) {
		blobs[name] = data
	}
	for _, profile := range Profiles() {
		blobs[profile.Source] = profile.Data
	}

	for name, data := range blobs {
		t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			blob, err := Normalize(data)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			parsed, err := Decode(blob)
			if err != nil {
				t.Skipf("rejected: %v", err)
			}
			compare(t, parsed, runOracle(t, data))
		})
	}
}

func runOracle(t *testing.T, data []byte) reference {
	t.Helper()

	path := filepath.Join(t.TempDir(), "edid.bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("stage blob: %v", err)
	}
	out, err := exec.Command(oracleTool, "--skip-hex-dump", path).CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", oracleTool, err, out)
	}
	return parseOracle(t, string(out))
}

func parseOracle(t *testing.T, out string) reference {
	t.Helper()

	var ref reference
	ref.week = -1
	section, block := "", 0
	var audio *refAudio
	var vendor *refVendor

	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "Block 0"):
			block = 0
			continue
		case strings.HasPrefix(line, "Block 1, CTA-861"):
			block, ref.cta = 1, &refCTA{}
			continue
		case strings.HasPrefix(line, "Block "):
			block = 2
			continue
		}
		if match := reHeading.FindStringSubmatch(line); match != nil && len(match[1]) == 2 {
			section = match[2]
			if audio != nil {
				ref.cta.audio = append(ref.cta.audio, *audio)
				audio = nil
			}
			vendor = nil
		}

		if block == 0 {
			parseBaseLine(&ref, section, line)
			continue
		}
		if block != 1 || ref.cta == nil {
			continue
		}

		switch section {
		case "Video Data Block":
			if match := reVIC.FindStringSubmatch(line); match != nil {
				vic, _ := strconv.Atoi(match[1])
				ref.cta.vics = append(ref.cta.vics, SVD{VIC: uint8(vic), Native: strings.HasSuffix(line, "(native)")})
			}
		case "Audio Data Block":
			if match := reHeading.FindStringSubmatch(line); match != nil && len(match[1]) == 4 {
				if audio != nil {
					ref.cta.audio = append(ref.cta.audio, *audio)
				}
				format, ok := audioFormatNames[match[2]]
				if !ok {
					t.Fatalf("no mapping for audio format %q", match[2])
				}
				audio = &refAudio{format: format, byte2: -1}
				continue
			}
			if audio == nil {
				continue
			}
			if match := reChannels.FindStringSubmatch(line); match != nil {
				audio.channels, _ = strconv.Atoi(match[1])
			}
			if match := reRates.FindStringSubmatch(line); match != nil {
				for _, field := range strings.Fields(match[1]) {
					khz, err := strconv.ParseFloat(field, 64)
					if err != nil {
						t.Fatalf("sample rate %q: %v", field, err)
					}
					audio.rates = append(audio.rates, int(math.Round(khz*1000)))
				}
				sort.Ints(audio.rates)
			}
			if match := reDepths.FindStringSubmatch(line); match != nil {
				for _, field := range strings.Fields(match[1]) {
					bits, _ := strconv.Atoi(field)
					audio.depths = append(audio.depths, bits)
				}
				sort.Ints(audio.depths)
			}
			if match := reBitrate.FindStringSubmatch(line); match != nil {
				kbps, _ := strconv.Atoi(match[1])
				audio.byte2 = kbps / 8
			}
			if match := reDependent.FindStringSubmatch(line); match != nil {
				value, _ := strconv.ParseUint(match[1], 16, 8)
				audio.byte2 = int(value)
			}
		case "Speaker Allocation Data Block":
			if match := reSpeaker.FindStringSubmatch(line); match != nil {
				if name, ok := speakerNamesRef[match[1]]; ok {
					ref.cta.speakers = append(ref.cta.speakers, name)
				}
			}
		case "Detailed Timing Descriptors":
			parseTimingLine(&ref, line)
		default:
			if match := reVSDB.FindStringSubmatch(line); match != nil {
				ref.cta.vendors = append(ref.cta.vendors, refVendor{oui: match[1]})
				vendor = &ref.cta.vendors[len(ref.cta.vendors)-1]
			}
			if match := reSPA.FindStringSubmatch(line); match != nil && vendor != nil {
				vendor.spa = match[1]
			}
		}

		if match := reCTARev.FindStringSubmatch(line); match != nil {
			revision, _ := strconv.Atoi(match[1])
			ref.cta.revision = uint8(revision)
		}
		if match := reNativeDTD.FindStringSubmatch(line); match != nil {
			native, _ := strconv.Atoi(match[1])
			ref.cta.nativeDTDs = uint8(native)
		}
		switch strings.TrimSpace(line) {
		case "Underscans IT Video Formats by default":
			ref.cta.underscan = true
		case "Basic audio support":
			ref.cta.basicAudio = true
		case "Supports YCbCr 4:4:4":
			ref.cta.ycbcr444 = true
		case "Supports YCbCr 4:2:2":
			ref.cta.ycbcr422 = true
		}
	}
	if audio != nil {
		ref.cta.audio = append(ref.cta.audio, *audio)
	}
	return ref
}

func parseBaseLine(ref *reference, section, line string) {
	if match := reVersion.FindStringSubmatch(line); match != nil {
		version, _ := strconv.Atoi(match[1])
		revision, _ := strconv.Atoi(match[2])
		ref.version, ref.revision = uint8(version), uint8(revision)
	}
	if match := reMfr.FindStringSubmatch(line); match != nil {
		ref.manufacturer = match[1]
	}
	if match := reModel.FindStringSubmatch(line); match != nil {
		product, _ := strconv.Atoi(match[1])
		ref.product = uint16(product)
	}
	if match := reSerial.FindStringSubmatch(line); match != nil {
		serial, _ := strconv.ParseUint(match[1], 10, 32)
		ref.serial = uint32(serial)
	}
	if match := reMade.FindStringSubmatch(line); match != nil {
		if match[1] != "" {
			ref.week, _ = strconv.Atoi(match[1])
		}
		ref.year, _ = strconv.Atoi(match[2])
	}
	if match := reSize.FindStringSubmatch(line); match != nil {
		width, _ := strconv.Atoi(match[1])
		height, _ := strconv.Atoi(match[2])
		ref.widthCM, ref.heightCM = uint8(width), uint8(height)
	}
	if match := reGamma.FindStringSubmatch(line); match != nil {
		ref.gamma, _ = strconv.ParseFloat(match[1], 64)
	}
	if strings.TrimSpace(line) == "Gamma is defined in an extension block" {
		ref.gammaInExtension = true
	}
	if strings.TrimSpace(line) == "Digital display" {
		ref.digital = true
	}
	if match := reExtensions.FindStringSubmatch(line); match != nil {
		ref.extensions, _ = strconv.Atoi(match[1])
	}
	if match := reName.FindStringSubmatch(line); match != nil {
		ref.name = match[1]
	}
	if match := reSerialText.FindStringSubmatch(line); match != nil {
		ref.serialText = match[1]
	}
	if match := reRanges.FindStringSubmatch(line); match != nil {
		ref.rangeVMin, _ = strconv.Atoi(match[1])
		ref.rangeVMax, _ = strconv.Atoi(match[2])
		ref.rangeHMin, _ = strconv.Atoi(match[3])
		ref.rangeHMax, _ = strconv.Atoi(match[4])
		ref.rangeClock, _ = strconv.Atoi(match[5])
	}
	if section == "Standard Timings" {
		if match := reMode.FindStringSubmatch(line); match != nil {
			horizontal, _ := strconv.Atoi(match[1])
			vertical, _ := strconv.Atoi(match[2])
			refresh, _ := strconv.ParseFloat(match[3], 64)
			ref.standard = append(ref.standard, refStandard{
				horizontal: horizontal, vertical: vertical,
				aspect: match[4], refresh: int(math.Round(refresh)),
			})
		}
	}
	if section == "Detailed Timing Descriptors" {
		parseTimingLine(ref, line)
	}
}

func parseTimingLine(ref *reference, line string) {
	if match := reDTD.FindStringSubmatch(line); match != nil {
		hActive, _ := strconv.Atoi(match[1])
		vActive, _ := strconv.Atoi(match[2])
		clock, _ := strconv.ParseFloat(match[4], 64)
		widthMM, _ := strconv.Atoi(match[5])
		heightMM, _ := strconv.Atoi(match[6])
		ref.dtds = append(ref.dtds, refTiming{
			hActive: hActive, vActive: vActive, interlaced: match[3] == "i",
			clockKHz: int(math.Round(clock * 1000)), widthMM: widthMM, heightMM: heightMM,
		})
		return
	}
	if len(ref.dtds) == 0 {
		return
	}
	dtd := &ref.dtds[len(ref.dtds)-1]
	if match := reH.FindStringSubmatch(line); match != nil {
		dtd.hFront, _ = strconv.Atoi(match[1])
		dtd.hSync, _ = strconv.Atoi(match[2])
		dtd.hBack, _ = strconv.Atoi(match[3])
		dtd.hPositive = match[4] == "P"
		dtd.hBorder, _ = strconv.Atoi(match[5])
	}
	if match := reV.FindStringSubmatch(line); match != nil {
		dtd.vFront, _ = strconv.Atoi(match[1])
		dtd.vSync, _ = strconv.Atoi(match[2])
		dtd.vBack, _ = strconv.Atoi(match[3])
		dtd.vPositive = match[4] == "P"
		dtd.vBorder, _ = strconv.Atoi(match[5])
	}
}

func compare(t *testing.T, parsed *EDID, ref reference) {
	t.Helper()

	if parsed.Version != ref.version || parsed.Revision != ref.revision {
		t.Errorf("version %d.%d, reference %d.%d", parsed.Version, parsed.Revision, ref.version, ref.revision)
	}
	if parsed.Manufacturer != ref.manufacturer {
		t.Errorf("manufacturer %q, reference %q", parsed.Manufacturer, ref.manufacturer)
	}
	if parsed.ProductCode != ref.product {
		t.Errorf("product code %d, reference %d", parsed.ProductCode, ref.product)
	}
	if parsed.Serial != ref.serial {
		t.Errorf("serial %d, reference %d", parsed.Serial, ref.serial)
	}
	if ref.week >= 0 && int(parsed.Week) != ref.week {
		t.Errorf("week %d, reference %d", parsed.Week, ref.week)
	}
	if parsed.Year != ref.year {
		t.Errorf("year %d, reference %d", parsed.Year, ref.year)
	}
	if parsed.Display.Digital != ref.digital {
		t.Errorf("digital %v, reference %v", parsed.Display.Digital, ref.digital)
	}
	if parsed.Display.WidthCM != ref.widthCM || parsed.Display.HeightCM != ref.heightCM {
		t.Errorf("image size %dx%d cm, reference %dx%d", parsed.Display.WidthCM, parsed.Display.HeightCM, ref.widthCM, ref.heightCM)
	}
	if ref.gammaInExtension {
		if parsed.Display.Gamma != 0 {
			t.Errorf("gamma %v, reference defers it to an extension block", parsed.Display.Gamma)
		}
	} else if math.Abs(parsed.Display.Gamma-ref.gamma) > 0.005 {
		t.Errorf("gamma %v, reference %v", parsed.Display.Gamma, ref.gamma)
	}
	if int(parsed.Extensions) != ref.extensions {
		t.Errorf("extension count %d, reference %d", parsed.Extensions, ref.extensions)
	}
	if parsed.Name() != ref.name {
		t.Errorf("name %q, reference %q", parsed.Name(), ref.name)
	}

	compareText(t, parsed, ref)
	compareRanges(t, parsed, ref)
	compareStandard(t, parsed, ref)
	compareTimings(t, parsed, ref)
	compareCTA(t, parsed, ref)
}

func compareText(t *testing.T, parsed *EDID, ref reference) {
	t.Helper()

	var serialText string
	for _, descriptor := range parsed.Descriptors {
		if descriptor.Kind == DescriptorSerial {
			serialText = descriptor.Text
		}
	}
	if serialText != strings.TrimRight(ref.serialText, " ") {
		t.Errorf("serial descriptor %q, reference %q", serialText, strings.TrimRight(ref.serialText, " "))
	}
}

func compareRanges(t *testing.T, parsed *EDID, ref reference) {
	t.Helper()

	var limits *RangeLimits
	for _, descriptor := range parsed.Descriptors {
		if descriptor.Range != nil {
			limits = descriptor.Range
		}
	}
	if limits == nil {
		if ref.rangeClock != 0 {
			t.Errorf("no range limits, reference reports max dotclock %d MHz", ref.rangeClock)
		}
		return
	}
	want := RangeLimits{
		OffsetFlags: limits.OffsetFlags, SecondaryFormula: limits.SecondaryFormula,
		VerticalMin: ref.rangeVMin, VerticalMax: ref.rangeVMax,
		HorizontalMin: ref.rangeHMin, HorizontalMax: ref.rangeHMax,
		MaxPixelClock: ref.rangeClock,
	}
	if *limits != want {
		t.Errorf("range limits %+v, reference %+v", *limits, want)
	}
}

func compareStandard(t *testing.T, parsed *EDID, ref reference) {
	t.Helper()

	var used []StandardTiming
	for _, timing := range parsed.StandardTimings {
		if timing.Used {
			used = append(used, timing)
		}
	}
	if len(used) != len(ref.standard) {
		t.Fatalf("%d standard timings, reference %d", len(used), len(ref.standard))
	}
	for i, want := range ref.standard {
		got := used[i]
		if got.Horizontal != want.horizontal || got.Vertical != want.vertical ||
			got.AspectRatio != want.aspect || got.RefreshHz != want.refresh {
			t.Errorf("standard timing %d = %dx%d %s %dHz, reference %dx%d %s %dHz", i,
				got.Horizontal, got.Vertical, got.AspectRatio, got.RefreshHz,
				want.horizontal, want.vertical, want.aspect, want.refresh)
		}
	}
}

func compareTimings(t *testing.T, parsed *EDID, ref reference) {
	t.Helper()

	var got []*Timing
	descriptors := parsed.Descriptors
	if parsed.CTA != nil {
		descriptors = append(descriptors, parsed.CTA.DTDs...)
	}
	for _, descriptor := range descriptors {
		if descriptor.Kind == DescriptorTiming {
			got = append(got, descriptor.Timing)
		}
	}
	if len(got) != len(ref.dtds) {
		t.Fatalf("%d detailed timings, reference %d", len(got), len(ref.dtds))
	}

	for i, want := range ref.dtds {
		timing := got[i]
		// EDID states vertical active in lines per field; edid-decode reports
		// frame lines, so an interlaced DTD is half our figure.
		vActive := timing.VActive
		if timing.Interlaced {
			vActive *= 2
		}
		have := refTiming{
			hActive: timing.HActive, vActive: vActive, interlaced: timing.Interlaced,
			clockKHz: timing.PixelClockKHz, widthMM: timing.WidthMM, heightMM: timing.HeightMM,
			hFront: timing.HSyncOffset, hSync: timing.HSyncWidth,
			hBack:   timing.HBlank - timing.HSyncOffset - timing.HSyncWidth - 2*timing.HBorder,
			hBorder: timing.HBorder,
			vFront:  timing.VSyncOffset, vSync: timing.VSyncWidth,
			vBack:     timing.VBlank - timing.VSyncOffset - timing.VSyncWidth - 2*timing.VBorder,
			vBorder:   timing.VBorder,
			hPositive: timing.HSyncPositive, vPositive: timing.VSyncPositive,
		}
		if have != want {
			t.Errorf("detailed timing %d = %+v, reference %+v", i, have, want)
		}
	}
}

func compareCTA(t *testing.T, parsed *EDID, ref reference) {
	t.Helper()

	if ref.cta == nil {
		if parsed.CTA != nil {
			t.Error("decoded a cta block the reference did not report")
		}
		return
	}
	if parsed.CTA == nil {
		t.Fatal("reference reports a cta block, decoded none")
	}
	cta := parsed.CTA

	if cta.Revision != ref.cta.revision || cta.NativeDTDs != ref.cta.nativeDTDs {
		t.Errorf("cta revision %d native %d, reference %d %d", cta.Revision, cta.NativeDTDs, ref.cta.revision, ref.cta.nativeDTDs)
	}
	if cta.Underscan != ref.cta.underscan || cta.BasicAudio != ref.cta.basicAudio ||
		cta.YCbCr444 != ref.cta.ycbcr444 || cta.YCbCr422 != ref.cta.ycbcr422 {
		t.Errorf("cta flags underscan %v audio %v 444 %v 422 %v, reference %v %v %v %v",
			cta.Underscan, cta.BasicAudio, cta.YCbCr444, cta.YCbCr422,
			ref.cta.underscan, ref.cta.basicAudio, ref.cta.ycbcr444, ref.cta.ycbcr422)
	}

	var vics []SVD
	var audio []refAudio
	var speakers []string
	var vendors []refVendor
	for _, block := range cta.Blocks {
		switch block.Tag {
		case CTAVideo:
			// edid-decode drops the reserved VIC 0 a few real blobs carry.
			for _, video := range block.Video {
				if video.VIC != 0 {
					vics = append(vics, video)
				}
			}
		case CTAAudio:
			for _, descriptor := range block.Audio {
				audio = append(audio, refAudio{
					format: descriptor.FormatName, channels: descriptor.Channels,
					rates: descriptor.SampleRatesHz, depths: descriptor.BitDepths,
					byte2: descriptor.MaxBitrateKbps / 8,
				})
			}
		case CTASpeakers:
			speakers = append(speakers, block.Speakers.Present...)
		case CTAVendor:
			vendors = append(vendors, refVendor{oui: block.Vendor.OUI, spa: block.Vendor.SourcePhysicalAddress})
		}
	}

	if fmt.Sprint(vics) != fmt.Sprint(ref.cta.vics) {
		t.Errorf("short video descriptors %v, reference %v", vics, ref.cta.vics)
	}
	if fmt.Sprint(speakers) != fmt.Sprint(ref.cta.speakers) {
		t.Errorf("speakers %v, reference %v", speakers, ref.cta.speakers)
	}
	if fmt.Sprint(vendors) != fmt.Sprint(ref.cta.vendors) {
		t.Errorf("vendor blocks %v, reference %v", vendors, ref.cta.vendors)
	}
	if len(audio) != len(ref.cta.audio) {
		t.Fatalf("%d audio descriptors, reference %d", len(audio), len(ref.cta.audio))
	}
	for i, want := range ref.cta.audio {
		got := audio[i]
		if want.byte2 < 0 {
			got.byte2 = -1
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("audio descriptor %d = %+v, reference %+v", i, got, want)
		}
	}
}
