package edid

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	BlockSize = 128
	Size      = 2 * BlockSize

	baseYear        = 1990
	standardBase    = 38
	standardCount   = 8
	descriptorBase  = 54
	descriptorSize  = 18
	descriptorCount = 4
	extensionCount  = 126
	dtdRegionEnd    = 126

	ctaTag         = 0x02
	ctaMinRevision = 3
	ctaMinOffset   = 4
	ctaMaxOffset   = 127
)

var edidHeader = [8]byte{0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00}

type RejectKind string

const (
	RejectSize           RejectKind = "size"
	RejectOversize       RejectKind = "oversize"
	RejectHeader         RejectKind = "header"
	RejectChecksum       RejectKind = "checksum"
	RejectVersion        RejectKind = "version"
	RejectExtensionCount RejectKind = "extension_count"
	RejectCTA            RejectKind = "cta"
	RejectNoTiming       RejectKind = "no_timing"
)

type RejectError struct {
	Kind   RejectKind
	Detail string
}

func (e *RejectError) Error() string {
	return fmt.Sprintf("edid %s: %s", e.Kind, e.Detail)
}

func reject(kind RejectKind, format string, args ...any) error {
	return &RejectError{Kind: kind, Detail: fmt.Sprintf(format, args...)}
}

type DescriptorKind string

const (
	DescriptorTiming         DescriptorKind = "timing"
	DescriptorSerial         DescriptorKind = "serial"
	DescriptorName           DescriptorKind = "name"
	DescriptorRangeLimits    DescriptorKind = "range_limits"
	DescriptorEstablishedIII DescriptorKind = "established_iii"
	DescriptorDummy          DescriptorKind = "dummy"
	DescriptorUnknown        DescriptorKind = "unknown"
)

const (
	tagSerial         = 0xFF
	tagName           = 0xFC
	tagRangeLimits    = 0xFD
	tagEstablishedIII = 0xF7
	tagDummy          = 0x10
)

type SyncKind string

const (
	SyncAnalogComposite        SyncKind = "analog_composite"
	SyncBipolarAnalogComposite SyncKind = "bipolar_analog_composite"
	SyncDigitalComposite       SyncKind = "digital_composite"
	SyncDigitalSeparate        SyncKind = "digital_separate"
)

var syncKinds = [4]SyncKind{SyncAnalogComposite, SyncBipolarAnalogComposite, SyncDigitalComposite, SyncDigitalSeparate}

type CTATag uint8

const (
	CTAAudio    CTATag = 1
	CTAVideo    CTATag = 2
	CTAVendor   CTATag = 3
	CTASpeakers CTATag = 4
	CTAExtended CTATag = 7
)

type VendorKind string

const (
	VendorHDMI14    VendorKind = "hdmi_1.4"
	VendorHDMIForum VendorKind = "hdmi_forum"
	VendorOther     VendorKind = "other"
)

type EDID struct {
	ManufacturerID  uint16           `json:"-"`
	Manufacturer    string           `json:"manufacturer"`
	ProductCode     uint16           `json:"product_code"`
	Serial          uint32           `json:"serial"`
	Week            uint8            `json:"week"`
	Year            int              `json:"year"`
	Version         uint8            `json:"version"`
	Revision        uint8            `json:"revision"`
	Display         Display          `json:"display"`
	Chromaticity    Chromaticity     `json:"chromaticity"`
	Established     Established      `json:"established"`
	StandardTimings []StandardTiming `json:"standard_timings"`
	Descriptors     []Descriptor     `json:"descriptors"`
	Extensions      uint8            `json:"extensions"`
	CTA             *CTA             `json:"cta,omitempty"`
	Extension       []byte           `json:"-"`
}

type Display struct {
	InputByte     uint8    `json:"-"`
	Digital       bool     `json:"digital"`
	BitDepth      uint8    `json:"bit_depth,omitempty"`
	Interface     uint8    `json:"interface,omitempty"`
	DFP1x         bool     `json:"dfp_1x,omitempty"`
	VideoLevel    uint8    `json:"video_level,omitempty"`
	BlankToBlack  bool     `json:"blank_to_black,omitempty"`
	SeparateSync  bool     `json:"separate_sync,omitempty"`
	CompositeSync bool     `json:"composite_sync,omitempty"`
	SyncOnGreen   bool     `json:"sync_on_green,omitempty"`
	SerratedVSync bool     `json:"serrated_vsync,omitempty"`
	WidthCM       uint8    `json:"width_cm"`
	HeightCM      uint8    `json:"height_cm"`
	GammaByte     uint8    `json:"-"`
	Gamma         float64  `json:"gamma"`
	FeatureByte   uint8    `json:"-"`
	Features      Features `json:"features"`
}

type Features struct {
	Standby         bool  `json:"standby"`
	Suspend         bool  `json:"suspend"`
	ActiveOff       bool  `json:"active_off"`
	ColorType       uint8 `json:"color_type"`
	SRGBDefault     bool  `json:"srgb_default"`
	PreferredNative bool  `json:"preferred_native"`
	ContinuousFreq  bool  `json:"continuous_frequency"`
}

type Point struct {
	XUnits int     `json:"x_units"`
	YUnits int     `json:"y_units"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

type Chromaticity struct {
	Raw   [10]byte `json:"-"`
	Red   Point    `json:"red"`
	Green Point    `json:"green"`
	Blue  Point    `json:"blue"`
	White Point    `json:"white"`
}

type Established struct {
	Raw                  [3]byte  `json:"-"`
	Modes                []string `json:"modes"`
	ManufacturerReserved uint8    `json:"manufacturer_reserved"`
}

type StandardTiming struct {
	Raw         [2]byte `json:"-"`
	Used        bool    `json:"used"`
	Horizontal  int     `json:"horizontal,omitempty"`
	Vertical    int     `json:"vertical,omitempty"`
	AspectRatio string  `json:"aspect_ratio,omitempty"`
	RefreshHz   int     `json:"refresh_hz,omitempty"`
}

type Descriptor struct {
	Kind           DescriptorKind `json:"kind"`
	Tag            uint8          `json:"tag,omitempty"`
	Text           string         `json:"text,omitempty"`
	Timing         *Timing        `json:"timing,omitempty"`
	Range          *RangeLimits   `json:"range,omitempty"`
	EstablishedIII []byte         `json:"established_iii,omitempty"`
	Raw            [18]byte       `json:"-"`
}

type Timing struct {
	PixelClockKHz int      `json:"pixel_clock_khz"`
	HActive       int      `json:"h_active"`
	HBlank        int      `json:"h_blank"`
	HTotal        int      `json:"h_total"`
	VActive       int      `json:"v_active"`
	VBlank        int      `json:"v_blank"`
	VTotal        int      `json:"v_total"`
	HSyncOffset   int      `json:"h_sync_offset"`
	HSyncWidth    int      `json:"h_sync_width"`
	VSyncOffset   int      `json:"v_sync_offset"`
	VSyncWidth    int      `json:"v_sync_width"`
	WidthMM       int      `json:"width_mm"`
	HeightMM      int      `json:"height_mm"`
	HBorder       int      `json:"h_border"`
	VBorder       int      `json:"v_border"`
	Interlaced    bool     `json:"interlaced"`
	Stereo        uint8    `json:"stereo"`
	Sync          SyncKind `json:"sync"`
	HSyncPositive bool     `json:"h_sync_positive"`
	VSyncPositive bool     `json:"v_sync_positive"`
	RefreshHz     float64  `json:"refresh_hz"`
}

type RangeLimits struct {
	OffsetFlags      uint8 `json:"offset_flags"`
	VerticalMin      int   `json:"vertical_min_hz"`
	VerticalMax      int   `json:"vertical_max_hz"`
	HorizontalMin    int   `json:"horizontal_min_khz"`
	HorizontalMax    int   `json:"horizontal_max_khz"`
	MaxPixelClock    int   `json:"max_pixel_clock_mhz"`
	SecondaryFormula uint8 `json:"secondary_formula"`
}

type CTA struct {
	Revision   uint8        `json:"revision"`
	DTDOffset  uint8        `json:"dtd_offset"`
	FlagByte   uint8        `json:"-"`
	Underscan  bool         `json:"underscan"`
	BasicAudio bool         `json:"basic_audio"`
	YCbCr444   bool         `json:"ycbcr_444"`
	YCbCr422   bool         `json:"ycbcr_422"`
	NativeDTDs uint8        `json:"native_dtds"`
	Blocks     []CTABlock   `json:"blocks"`
	DTDs       []Descriptor `json:"dtds"`
	Padding    []byte       `json:"-"`
}

type CTABlock struct {
	Tag         CTATag            `json:"tag"`
	ExtendedTag uint8             `json:"extended_tag,omitempty"`
	Video       []SVD             `json:"video,omitempty"`
	Audio       []AudioDescriptor `json:"audio,omitempty"`
	Vendor      *Vendor           `json:"vendor,omitempty"`
	Speakers    *Speakers         `json:"speakers,omitempty"`
	Raw         []byte            `json:"-"`
}

type SVD struct {
	VIC    uint8 `json:"vic"`
	Native bool  `json:"native"`
}

type AudioDescriptor struct {
	Format         uint8  `json:"format"`
	FormatName     string `json:"format_name"`
	Channels       int    `json:"channels"`
	SampleRatesHz  []int  `json:"sample_rates_hz"`
	BitDepths      []int  `json:"bit_depths,omitempty"`
	MaxBitrateKbps int    `json:"max_bitrate_kbps,omitempty"`
}

type Vendor struct {
	Kind                  VendorKind `json:"kind"`
	OUI                   string     `json:"oui"`
	SourcePhysicalAddress string     `json:"source_physical_address,omitempty"`
}

type Speakers struct {
	Mask    uint8    `json:"mask"`
	Present []string `json:"present"`
}

var (
	establishedI  = [8]string{"720x400@70", "720x400@88", "640x480@60", "640x480@67", "640x480@72", "640x480@75", "800x600@56", "800x600@60"}
	establishedII = [8]string{"800x600@72", "800x600@75", "832x624@75", "1024x768@87i", "1024x768@60", "1024x768@70", "1024x768@75", "1280x1024@75"}

	aspectRatios = [4]string{"16:10", "4:3", "5:4", "16:9"}
	bitDepths    = [8]uint8{0, 6, 8, 10, 12, 14, 16, 0}

	audioFormats = [16]string{
		"reserved", "lpcm", "ac-3", "mpeg-1", "mp3", "mpeg-2", "aac-lc", "dts",
		"atrac", "dsd", "e-ac-3", "dts-hd", "mlp", "dst", "wma-pro", "extended",
	}
	sampleRates  = [7]int{32000, 44100, 48000, 88200, 96000, 176400, 192000}
	speakerNames = [7]string{"FL/FR", "LFE", "FC", "RL/RR", "RC", "FLC/FRC", "RLC/RRC"}
)

func Decode(data []byte) (*EDID, error) {
	if len(data) > Size {
		return nil, reject(RejectOversize, "%d bytes, the tool would truncate to the first %d", len(data), Size)
	}
	if len(data) != Size {
		return nil, reject(RejectSize, "%d bytes, want %d", len(data), Size)
	}
	if !bytes.Equal(data[:len(edidHeader)], edidHeader[:]) {
		return nil, reject(RejectHeader, "% X, want % X", data[:len(edidHeader)], edidHeader)
	}
	for i := 0; i < 2; i++ {
		block := data[i*BlockSize : (i+1)*BlockSize]
		if sum := sum(block); sum != 0 {
			return nil, reject(RejectChecksum, "block %d sums to %d mod 256", i, sum)
		}
	}
	if data[18] < 1 || (data[18] == 1 && data[19] < 3) {
		return nil, reject(RejectVersion, "version %d.%d, want at least 1.3", data[18], data[19])
	}

	base, extension := data[:BlockSize], data[BlockSize:]
	e := &EDID{
		ManufacturerID: binary.BigEndian.Uint16(base[8:10]),
		ProductCode:    binary.LittleEndian.Uint16(base[10:12]),
		Serial:         binary.LittleEndian.Uint32(base[12:16]),
		Week:           base[16],
		Year:           baseYear + int(base[17]),
		Version:        base[18],
		Revision:       base[19],
		Display:        parseDisplay(base),
		Chromaticity:   parseChromaticity(base[25:35]),
		Established:    parseEstablished(base[35:38]),
		Extensions:     base[extensionCount],
	}
	e.Manufacturer = decodeManufacturer(e.ManufacturerID)

	for i := 0; i < standardCount; i++ {
		e.StandardTimings = append(e.StandardTimings, parseStandardTiming(base[standardBase+2*i:]))
	}
	for i := 0; i < descriptorCount; i++ {
		offset := descriptorBase + descriptorSize*i
		e.Descriptors = append(e.Descriptors, parseDescriptor(base[offset:offset+descriptorSize]))
	}

	blank := allZero(extension)
	switch {
	case e.Extensions == 0 && !blank:
		return nil, reject(RejectExtensionCount, "block 1 carries data but byte 126 is 0")
	case e.Extensions == 1 && blank:
		return nil, reject(RejectExtensionCount, "byte 126 claims an extension but block 1 is empty")
	case e.Extensions > 1:
		return nil, reject(RejectExtensionCount, "byte 126 is %d, want 0 or 1 in a %d byte edid", e.Extensions, Size)
	}

	if e.Extensions == 1 {
		if extension[0] == ctaTag {
			cta, err := parseCTA(extension)
			if err != nil {
				return nil, err
			}
			e.CTA = cta
		} else {
			e.Extension = bytes.Clone(extension)
		}
	}

	if !e.hasTiming() {
		return nil, reject(RejectNoTiming, "no detailed timing descriptor carries a pixel clock")
	}
	return e, nil
}

func (e *EDID) Encode() ([]byte, error) {
	out := make([]byte, Size)
	copy(out, edidHeader[:])
	binary.BigEndian.PutUint16(out[8:10], e.ManufacturerID)
	binary.LittleEndian.PutUint16(out[10:12], e.ProductCode)
	binary.LittleEndian.PutUint32(out[12:16], e.Serial)
	out[16] = e.Week
	out[17] = byte(e.Year - baseYear)
	out[18], out[19] = e.Version, e.Revision
	out[20] = e.Display.InputByte
	out[21], out[22] = e.Display.WidthCM, e.Display.HeightCM
	out[23] = e.Display.GammaByte
	out[24] = e.Display.FeatureByte
	copy(out[25:35], e.Chromaticity.Raw[:])
	copy(out[35:38], e.Established.Raw[:])

	if len(e.StandardTimings) != standardCount {
		return nil, fmt.Errorf("encode edid: %d standard timings, want %d", len(e.StandardTimings), standardCount)
	}
	for i, timing := range e.StandardTimings {
		copy(out[standardBase+2*i:], timing.Raw[:])
	}

	if len(e.Descriptors) != descriptorCount {
		return nil, fmt.Errorf("encode edid: %d descriptors, want %d", len(e.Descriptors), descriptorCount)
	}
	for i, descriptor := range e.Descriptors {
		encodeDescriptor(out[descriptorBase+descriptorSize*i:], descriptor)
	}

	out[extensionCount] = e.Extensions
	if err := e.encodeExtension(out[BlockSize:]); err != nil {
		return nil, err
	}
	out[BlockSize-1] = checksum(out[:BlockSize])
	out[Size-1] = checksum(out[BlockSize:])
	return out, nil
}

func (e *EDID) encodeExtension(dst []byte) error {
	if e.CTA == nil {
		copy(dst, e.Extension)
		return nil
	}

	cta := e.CTA
	dst[0], dst[1], dst[2], dst[3] = ctaTag, cta.Revision, cta.DTDOffset, cta.FlagByte
	offset := ctaMinOffset
	for _, block := range cta.Blocks {
		if offset+len(block.Raw) > int(cta.DTDOffset) {
			return fmt.Errorf("encode cta: data block collection overruns offset %d", cta.DTDOffset)
		}
		copy(dst[offset:], block.Raw)
		offset += len(block.Raw)
	}
	if offset != int(cta.DTDOffset) {
		return fmt.Errorf("encode cta: data block collection ends at %d, want %d", offset, cta.DTDOffset)
	}
	for _, descriptor := range cta.DTDs {
		if offset+descriptorSize > dtdRegionEnd {
			return fmt.Errorf("encode cta: detailed timing at %d overruns the dtd region", offset)
		}
		encodeDescriptor(dst[offset:], descriptor)
		offset += descriptorSize
	}
	if offset+len(cta.Padding) > BlockSize-1 {
		return fmt.Errorf("encode cta: padding overruns the block")
	}
	copy(dst[offset:], cta.Padding)
	return nil
}

func (e *EDID) PreferredTiming() *Timing {
	for _, descriptor := range e.Descriptors {
		if descriptor.Kind == DescriptorTiming && descriptor.Timing.PixelClockKHz > 0 {
			return descriptor.Timing
		}
	}
	return nil
}

func (e *EDID) Name() string {
	for _, descriptor := range e.Descriptors {
		if descriptor.Kind == DescriptorName {
			return descriptor.Text
		}
	}
	return ""
}

func (e *EDID) hasTiming() bool {
	descriptors := e.Descriptors
	if e.CTA != nil {
		descriptors = append(descriptors, e.CTA.DTDs...)
	}
	for _, descriptor := range descriptors {
		if descriptor.Kind == DescriptorTiming && descriptor.Timing.PixelClockKHz > 0 {
			return true
		}
	}
	return false
}

func (t *Timing) Mode() string {
	scan := "p"
	if t.Interlaced {
		scan = "i"
	}
	return fmt.Sprintf("%dx%d%s%d", t.HActive, t.VActive, scan, int(math.Round(t.RefreshHz)))
}

func parseDisplay(base []byte) Display {
	input, feature := base[20], base[24]
	display := Display{
		InputByte:   input,
		Digital:     input&0x80 != 0,
		WidthCM:     base[21],
		HeightCM:    base[22],
		GammaByte:   base[23],
		FeatureByte: feature,
		Features: Features{
			Standby:         feature&0x80 != 0,
			Suspend:         feature&0x40 != 0,
			ActiveOff:       feature&0x20 != 0,
			ColorType:       (feature >> 3) & 0x03,
			SRGBDefault:     feature&0x04 != 0,
			PreferredNative: feature&0x02 != 0,
			ContinuousFreq:  feature&0x01 != 0,
		},
	}
	if display.GammaByte != 0xFF {
		display.Gamma = float64(display.GammaByte)/100 + 1
	}
	if display.Digital {
		display.BitDepth = bitDepths[(input>>4)&0x07]
		display.Interface = input & 0x0F
		display.DFP1x = input&0x01 != 0
		return display
	}
	display.VideoLevel = (input >> 5) & 0x03
	display.BlankToBlack = input&0x10 != 0
	display.SeparateSync = input&0x08 != 0
	display.CompositeSync = input&0x04 != 0
	display.SyncOnGreen = input&0x02 != 0
	display.SerratedVSync = input&0x01 != 0
	return display
}

func parseChromaticity(b []byte) Chromaticity {
	var chroma Chromaticity
	copy(chroma.Raw[:], b)

	units := [8]int{}
	for i := range units {
		low := (b[i/4] >> (6 - 2*(i%4))) & 0x03
		units[i] = int(b[2+i])<<2 | int(low)
	}
	points := [4]*Point{&chroma.Red, &chroma.Green, &chroma.Blue, &chroma.White}
	for i, point := range points {
		point.XUnits, point.YUnits = units[2*i], units[2*i+1]
		point.X, point.Y = float64(point.XUnits)/1024, float64(point.YUnits)/1024
	}
	return chroma
}

func parseEstablished(b []byte) Established {
	established := Established{ManufacturerReserved: b[2] & 0x7F}
	copy(established.Raw[:], b)
	for i, name := range establishedI {
		if b[0]&(0x80>>i) != 0 {
			established.Modes = append(established.Modes, name)
		}
	}
	for i, name := range establishedII {
		if b[1]&(0x80>>i) != 0 {
			established.Modes = append(established.Modes, name)
		}
	}
	if b[2]&0x80 != 0 {
		established.Modes = append(established.Modes, "1152x870@75")
	}
	return established
}

func parseStandardTiming(b []byte) StandardTiming {
	timing := StandardTiming{Raw: [2]byte{b[0], b[1]}}
	if b[0] == 0x01 && b[1] == 0x01 {
		return timing
	}
	timing.Used = true
	timing.Horizontal = (int(b[0]) + 31) * 8
	timing.AspectRatio = aspectRatios[(b[1]>>6)&0x03]
	timing.RefreshHz = int(b[1]&0x3F) + 60
	switch (b[1] >> 6) & 0x03 {
	case 0:
		timing.Vertical = timing.Horizontal * 10 / 16
	case 1:
		timing.Vertical = timing.Horizontal * 3 / 4
	case 2:
		timing.Vertical = timing.Horizontal * 4 / 5
	default:
		timing.Vertical = timing.Horizontal * 9 / 16
	}
	return timing
}

func parseDescriptor(b []byte) Descriptor {
	descriptor := Descriptor{}
	copy(descriptor.Raw[:], b)

	if b[0] != 0 || b[1] != 0 || b[2] != 0 {
		descriptor.Kind = DescriptorTiming
		descriptor.Timing = parseTiming(b)
		return descriptor
	}

	descriptor.Tag = b[3]
	switch b[3] {
	case tagSerial:
		descriptor.Kind, descriptor.Text = DescriptorSerial, descriptorText(b)
	case tagName:
		descriptor.Kind, descriptor.Text = DescriptorName, descriptorText(b)
	case tagRangeLimits:
		descriptor.Kind, descriptor.Range = DescriptorRangeLimits, parseRangeLimits(b)
	case tagEstablishedIII:
		descriptor.Kind, descriptor.EstablishedIII = DescriptorEstablishedIII, bytes.Clone(b[6:18])
	case tagDummy:
		descriptor.Kind = DescriptorDummy
	default:
		descriptor.Kind = DescriptorUnknown
	}
	return descriptor
}

func encodeDescriptor(dst []byte, descriptor Descriptor) {
	if descriptor.Kind == DescriptorTiming && descriptor.Timing != nil {
		encodeTiming(dst, descriptor.Timing)
		return
	}
	copy(dst, descriptor.Raw[:])
}

func descriptorText(b []byte) string {
	text := string(b[5:18])
	if i := strings.IndexByte(text, 0x0A); i >= 0 {
		text = text[:i]
	}
	return strings.TrimRight(text, " ")
}

func parseRangeLimits(b []byte) *RangeLimits {
	flags := b[4]
	limits := &RangeLimits{
		OffsetFlags:      flags,
		VerticalMin:      int(b[5]),
		VerticalMax:      int(b[6]),
		HorizontalMin:    int(b[7]),
		HorizontalMax:    int(b[8]),
		MaxPixelClock:    int(b[9]) * 10,
		SecondaryFormula: b[10],
	}
	if flags&0x02 != 0 {
		limits.VerticalMax += 255
	}
	if flags&0x03 == 0x03 {
		limits.VerticalMin += 255
	}
	if flags&0x08 != 0 {
		limits.HorizontalMax += 255
	}
	if flags&0x0C == 0x0C {
		limits.HorizontalMin += 255
	}
	return limits
}

func parseTiming(b []byte) *Timing {
	clock := int(binary.LittleEndian.Uint16(b[0:2])) * 10
	timing := &Timing{
		PixelClockKHz: clock,
		HActive:       int(b[2]) | int(b[4]>>4)<<8,
		HBlank:        int(b[3]) | int(b[4]&0x0F)<<8,
		VActive:       int(b[5]) | int(b[7]>>4)<<8,
		VBlank:        int(b[6]) | int(b[7]&0x0F)<<8,
		HSyncOffset:   int(b[8]) | int((b[11]>>6)&0x03)<<8,
		HSyncWidth:    int(b[9]) | int((b[11]>>4)&0x03)<<8,
		VSyncOffset:   int(b[10]>>4) | int((b[11]>>2)&0x03)<<4,
		VSyncWidth:    int(b[10]&0x0F) | int(b[11]&0x03)<<4,
		WidthMM:       int(b[12]) | int(b[14]>>4)<<8,
		HeightMM:      int(b[13]) | int(b[14]&0x0F)<<8,
		HBorder:       int(b[15]),
		VBorder:       int(b[16]),
		Interlaced:    b[17]&0x80 != 0,
		Stereo:        ((b[17]>>5)&0x03)<<1 | b[17]&0x01,
		Sync:          syncKinds[(b[17]>>3)&0x03],
		VSyncPositive: b[17]&0x04 != 0,
		HSyncPositive: b[17]&0x02 != 0,
	}
	timing.HTotal = timing.HActive + timing.HBlank
	timing.VTotal = timing.VActive + timing.VBlank
	if timing.HTotal > 0 && timing.VTotal > 0 {
		timing.RefreshHz = float64(clock) * 1000 / float64(timing.HTotal*timing.VTotal)
	}
	return timing
}

func encodeTiming(dst []byte, t *Timing) {
	binary.LittleEndian.PutUint16(dst[0:2], uint16(t.PixelClockKHz/10))
	dst[2] = byte(t.HActive)
	dst[3] = byte(t.HBlank)
	dst[4] = byte(t.HActive>>8)<<4 | byte((t.HBlank>>8)&0x0F)
	dst[5] = byte(t.VActive)
	dst[6] = byte(t.VBlank)
	dst[7] = byte(t.VActive>>8)<<4 | byte((t.VBlank>>8)&0x0F)
	dst[8] = byte(t.HSyncOffset)
	dst[9] = byte(t.HSyncWidth)
	dst[10] = byte(t.VSyncOffset&0x0F)<<4 | byte(t.VSyncWidth&0x0F)
	dst[11] = byte((t.HSyncOffset>>8)&0x03)<<6 | byte((t.HSyncWidth>>8)&0x03)<<4 |
		byte((t.VSyncOffset>>4)&0x03)<<2 | byte((t.VSyncWidth>>4)&0x03)
	dst[12] = byte(t.WidthMM)
	dst[13] = byte(t.HeightMM)
	dst[14] = byte(t.WidthMM>>8)<<4 | byte((t.HeightMM>>8)&0x0F)
	dst[15] = byte(t.HBorder)
	dst[16] = byte(t.VBorder)

	var flags byte
	if t.Interlaced {
		flags |= 0x80
	}
	flags |= ((t.Stereo >> 1) & 0x03) << 5
	flags |= t.Stereo & 0x01
	for i, kind := range syncKinds {
		if kind == t.Sync {
			flags |= byte(i) << 3
		}
	}
	if t.VSyncPositive {
		flags |= 0x04
	}
	if t.HSyncPositive {
		flags |= 0x02
	}
	dst[17] = flags
}

func parseCTA(block []byte) (*CTA, error) {
	if block[1] < ctaMinRevision {
		return nil, reject(RejectCTA, "revision %d, want at least %d", block[1], ctaMinRevision)
	}
	offset := int(block[2])
	if offset < ctaMinOffset || offset > ctaMaxOffset {
		return nil, reject(RejectCTA, "dtd offset %d, want [%d, %d]", offset, ctaMinOffset, ctaMaxOffset)
	}

	flags := block[3]
	cta := &CTA{
		Revision:   block[1],
		DTDOffset:  block[2],
		FlagByte:   flags,
		Underscan:  flags&0x80 != 0,
		BasicAudio: flags&0x40 != 0,
		YCbCr444:   flags&0x20 != 0,
		YCbCr422:   flags&0x10 != 0,
		NativeDTDs: flags & 0x0F,
	}

	for cursor := ctaMinOffset; cursor < offset; {
		length := int(block[cursor] & 0x1F)
		end := cursor + 1 + length
		if end > offset {
			return nil, reject(RejectCTA, "data block at %d runs past the dtd offset %d", cursor, offset)
		}
		parsed, err := parseCTABlock(block[cursor:end])
		if err != nil {
			return nil, err
		}
		cta.Blocks = append(cta.Blocks, parsed)
		cursor = end
	}

	cursor := offset
	for cursor+descriptorSize <= dtdRegionEnd {
		chunk := block[cursor : cursor+descriptorSize]
		if allZero(chunk) {
			break
		}
		cta.DTDs = append(cta.DTDs, parseDescriptor(chunk))
		cursor += descriptorSize
	}
	cta.Padding = bytes.Clone(block[cursor : BlockSize-1])
	if !allZero(cta.Padding) {
		return nil, reject(RejectCTA, "the dtd region carries %d non zero trailing bytes", len(cta.Padding))
	}
	return cta, nil
}

func parseCTABlock(raw []byte) (CTABlock, error) {
	block := CTABlock{Tag: CTATag(raw[0] >> 5), Raw: bytes.Clone(raw)}
	payload := raw[1:]

	switch block.Tag {
	case CTAVideo:
		for _, b := range payload {
			block.Video = append(block.Video, SVD{VIC: b & 0x7F, Native: b&0x80 != 0})
		}
	case CTAAudio:
		for i := 0; i+3 <= len(payload); i += 3 {
			block.Audio = append(block.Audio, parseAudio(payload[i:i+3]))
		}
	case CTAVendor:
		if len(payload) < 3 {
			return CTABlock{}, reject(RejectCTA, "vendor data block carries %d bytes, want at least 3", len(payload))
		}
		block.Vendor = parseVendor(payload)
	case CTASpeakers:
		if len(payload) < 1 {
			return CTABlock{}, reject(RejectCTA, "speaker allocation block is empty")
		}
		block.Speakers = parseSpeakers(payload[0])
	case CTAExtended:
		if len(payload) < 1 {
			return CTABlock{}, reject(RejectCTA, "extended data block carries no tag")
		}
		block.ExtendedTag = payload[0]
	}
	return block, nil
}

func parseAudio(b []byte) AudioDescriptor {
	descriptor := AudioDescriptor{
		Format:     (b[0] >> 3) & 0x0F,
		Channels:   int(b[0]&0x07) + 1,
		FormatName: audioFormats[(b[0]>>3)&0x0F],
	}
	for i, rate := range sampleRates {
		if b[1]&(1<<i) != 0 {
			descriptor.SampleRatesHz = append(descriptor.SampleRatesHz, rate)
		}
	}
	if descriptor.Format == 1 {
		for i, depth := range [3]int{16, 20, 24} {
			if b[2]&(1<<i) != 0 {
				descriptor.BitDepths = append(descriptor.BitDepths, depth)
			}
		}
		return descriptor
	}
	descriptor.MaxBitrateKbps = int(b[2]) * 8
	return descriptor
}

func parseVendor(payload []byte) *Vendor {
	oui := fmt.Sprintf("%02X-%02X-%02X", payload[2], payload[1], payload[0])
	vendor := &Vendor{Kind: VendorOther, OUI: oui}
	switch oui {
	case "00-0C-03":
		vendor.Kind = VendorHDMI14
	case "C4-5D-D8":
		vendor.Kind = VendorHDMIForum
	}
	if vendor.Kind == VendorHDMI14 && len(payload) >= 5 {
		vendor.SourcePhysicalAddress = fmt.Sprintf("%d.%d.%d.%d", payload[3]>>4, payload[3]&0x0F, payload[4]>>4, payload[4]&0x0F)
	}
	return vendor
}

func parseSpeakers(mask uint8) *Speakers {
	speakers := &Speakers{Mask: mask}
	for i, name := range speakerNames {
		if mask&(1<<i) != 0 {
			speakers.Present = append(speakers.Present, name)
		}
	}
	return speakers
}

func decodeManufacturer(id uint16) string {
	letters := make([]byte, 3)
	for i := range letters {
		letters[i] = byte('A'-1) + byte((id>>(10-5*i))&0x1F)
	}
	return string(letters)
}

func sum(block []byte) byte {
	var total byte
	for _, b := range block {
		total += b
	}
	return total
}

func checksum(block []byte) byte {
	return -sum(block[:BlockSize-1])
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
