package presentation

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	SchemaVersion = 1

	ProfileStandard = "standard"
	ProfileHIDOnly  = "hid-only"
	ProfileHybrid   = "hybrid"

	// D6: the normal-mode marker was the gadget core's get_default_bcdDevice(),
	// bin2bcd(VERSION)<<8|bin2bcd(PATCHLEVEL), so a vendor kernel bump moved it
	// and took /api/hid/mode with it. Both markers are now written deliberately.
	BCDDeviceNormal  = "0x0510"
	BCDDeviceHIDOnly = "0x0623"

	InquiryString      = "NanoKVM USB Mass Storage0520"
	InquiryStringCDROM = "NanoKVM USB CD/DVD-ROM  0520"
)

type FunctionKind string

const (
	FunctionHID         FunctionKind = "hid"
	FunctionNCM         FunctionKind = "ncm"
	FunctionRNDIS       FunctionKind = "rndis"
	FunctionMassStorage FunctionKind = "mass_storage"
	FunctionFFS         FunctionKind = "ffs"
	FunctionUVC         FunctionKind = "uvc"
	FunctionUAC2        FunctionKind = "uac2"
)

// The network protocols the gadget layer can actually build, in the precedence
// S03usbdev:53,61 gives them. ECM is not among them: there is no f_ecm branch
// in the script, no FunctionKind for it and no compile case, so offering it in
// a selector would offer a mode nothing downstream can produce.
var NetworkKinds = [...]FunctionKind{FunctionNCM, FunctionRNDIS}

var ErrUnknownNetworkKind = errors.New("unknown network protocol")

// The gate between a request-supplied string and a profile. It is the reason a
// selector cannot name a protocol the compiler has no case for.
func ParseNetworkKind(name string) (FunctionKind, error) {
	for _, kind := range NetworkKinds {
		if string(kind) == name {
			return kind, nil
		}
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownNetworkKind, name)
}

var hidInstances = [...]string{"GS0", "GS1", "GS2"}

var (
	profileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	instancePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)
	stringIndexPattern = regexp.MustCompile(`^(?:0x[0-9A-Fa-f]{4}:)?(?:[1-9][0-9]{0,2})$`)
	cameraPattern      = regexp.MustCompile(`^cam([0-9])$`)
	microphonePattern  = regexp.MustCompile(`^mic([0-9])$`)
)

var (
	descKeyboardStandard = []byte{
		0x05, 0x01, 0x09, 0x06, 0xA1, 0x01, 0x05, 0x07, 0x19, 0xE0, 0x29, 0xE7,
		0x15, 0x00, 0x25, 0x01, 0x75, 0x01, 0x95, 0x08, 0x81, 0x02, 0x95, 0x01,
		0x75, 0x08, 0x81, 0x03, 0x95, 0x05, 0x75, 0x01, 0x05, 0x08, 0x19, 0x01,
		0x29, 0x05, 0x91, 0x02, 0x95, 0x01, 0x75, 0x03, 0x91, 0x03, 0x95, 0x06,
		0x75, 0x08, 0x15, 0x00, 0x25, 0xE7, 0x05, 0x07, 0x19, 0x00, 0x29, 0xE7,
		0x81, 0x00, 0xC0,
	}

	descKeyboardHIDOnly = []byte{
		0x05, 0x01, 0x09, 0x06, 0xA1, 0x01, 0x05, 0x07, 0x19, 0xE0, 0x29, 0xE7,
		0x15, 0x00, 0x25, 0x01, 0x75, 0x01, 0x95, 0x08, 0x81, 0x02, 0x95, 0x01,
		0x75, 0x08, 0x81, 0x03, 0x95, 0x05, 0x75, 0x01, 0x05, 0x08, 0x19, 0x01,
		0x29, 0x05, 0x91, 0x02, 0x95, 0x01, 0x75, 0x03, 0x91, 0x03, 0x95, 0x06,
		0x75, 0x08, 0x15, 0x00, 0x25, 0x65, 0x05, 0x07, 0x19, 0x00, 0x29, 0x65,
		0x81, 0x00, 0xC0,
	}

	descMouseRelative = []byte{
		0x05, 0x01, 0x09, 0x02, 0xA1, 0x01, 0x09, 0x01, 0xA1, 0x00, 0x05, 0x09,
		0x19, 0x01, 0x29, 0x03, 0x15, 0x00, 0x25, 0x01, 0x95, 0x03, 0x75, 0x01,
		0x81, 0x02, 0x95, 0x01, 0x75, 0x05, 0x81, 0x03, 0x05, 0x01, 0x09, 0x30,
		0x09, 0x31, 0x09, 0x38, 0x15, 0x81, 0x25, 0x7F, 0x75, 0x08, 0x95, 0x03,
		0x81, 0x06, 0xC0, 0xC0,
	}

	descPointerStandard = []byte{
		0x05, 0x01, 0x09, 0x02, 0xA1, 0x01, 0x09, 0x01, 0xA1, 0x00, 0x05, 0x09,
		0x19, 0x01, 0x29, 0x05, 0x15, 0x00, 0x25, 0x01, 0x95, 0x05, 0x75, 0x01,
		0x81, 0x02, 0x95, 0x01, 0x75, 0x03, 0x81, 0x01, 0x05, 0x01, 0x09, 0x30,
		0x09, 0x31, 0x15, 0x00, 0x26, 0xFF, 0x7F, 0x35, 0x00, 0x46, 0xFF, 0x7F,
		0x75, 0x10, 0x95, 0x02, 0x81, 0x02, 0x05, 0x01, 0x09, 0x38, 0x15, 0x81,
		0x25, 0x7F, 0x35, 0x00, 0x45, 0x00, 0x75, 0x08, 0x95, 0x01, 0x81, 0x06,
		0xC0, 0xC0,
	}

	descPointerHIDOnly = []byte{
		0x05, 0x01, 0x09, 0x02, 0xA1, 0x01, 0x09, 0x01, 0xA1, 0x00, 0x05, 0x09,
		0x19, 0x01, 0x29, 0x03, 0x15, 0x00, 0x25, 0x01, 0x95, 0x03, 0x75, 0x01,
		0x81, 0x02, 0x95, 0x01, 0x75, 0x05, 0x81, 0x01, 0x05, 0x01, 0x09, 0x30,
		0x09, 0x31, 0x15, 0x00, 0x26, 0xFF, 0x7F, 0x35, 0x00, 0x46, 0xFF, 0x7F,
		0x75, 0x10, 0x95, 0x02, 0x81, 0x02, 0x05, 0x01, 0x09, 0x38, 0x15, 0x81,
		0x25, 0x7F, 0x35, 0x00, 0x45, 0x00, 0x75, 0x08, 0x95, 0x01, 0x81, 0x06,
		0xC0, 0xC0,
	}
)

type Profile struct {
	SchemaVersion int            `json:"schema_version"`
	Name          string         `json:"name"`
	BuiltIn       bool           `json:"built_in"`
	Provenance    Provenance     `json:"provenance"`
	Device        Device         `json:"device"`
	Config        ConfigDesc     `json:"config"`
	Functions     []Function     `json:"functions"`
	OSDesc        *OSDesc        `json:"os_desc,omitempty"`
	Descriptors   *DescriptorSet `json:"descriptors,omitempty"`
}

type DescriptorSet struct {
	Device         []byte            `json:"device,omitempty"`
	Configurations [][]byte          `json:"configurations,omitempty"`
	BOS            []byte            `json:"bos,omitempty"`
	Strings        map[string]string `json:"strings,omitempty"`
	HIDReports     map[string][]byte `json:"hid_reports,omitempty"`
}

type Device struct {
	VendorID     string  `json:"vendor_id"`
	ProductID    string  `json:"product_id"`
	BCDUSB       *string `json:"bcd_usb,omitempty"`    // nil = DO NOT WRITE
	BCDDevice    *string `json:"bcd_device,omitempty"` // nil = DO NOT WRITE
	Class        *uint8  `json:"class,omitempty"`
	SubClass     *uint8  `json:"subclass,omitempty"`
	Protocol     *uint8  `json:"protocol,omitempty"`
	Serial       *string `json:"serial,omitempty"`
	Manufacturer string  `json:"manufacturer"`
	Product      string  `json:"product"`
}

type ConfigDesc struct {
	BMAttributes  uint8  `json:"bm_attributes"`
	MaxPower      uint16 `json:"max_power"`
	Configuration string `json:"configuration"`
}

type Function struct {
	Kind     FunctionKind     `json:"kind"`
	Instance string           `json:"instance"`
	HID      *HIDFunction     `json:"hid,omitempty"`
	Net      *NetFunction     `json:"net,omitempty"`
	Storage  *StorageFunction `json:"storage,omitempty"`
	FFS      *FunctionFS      `json:"functionfs,omitempty"`
	Video    *VideoFunction   `json:"video,omitempty"`
	Audio    *AudioFunction   `json:"audio,omitempty"`
}

type EndpointTransfer string

const (
	EndpointBulk        EndpointTransfer = "bulk"
	EndpointInterrupt   EndpointTransfer = "interrupt"
	EndpointIsochronous EndpointTransfer = "isochronous"
)

type FunctionFS struct {
	Interfaces uint8                `json:"interfaces"`
	Endpoints  []FunctionFSEndpoint `json:"endpoints"`
}

type FunctionFSEndpoint struct {
	SourceAddress uint8            `json:"source_address"`
	Address       uint8            `json:"address"`
	Transfer      EndpointTransfer `json:"transfer"`
	MaxPacket     uint16           `json:"max_packet"`
	Interval      uint8            `json:"interval,omitempty"`
	Mult          uint8            `json:"mult,omitempty"`
}

type HIDFunction struct {
	Protocol      uint8     `json:"protocol"`
	SubClass      uint8     `json:"subclass"`
	ReportLength  uint16    `json:"report_length"`
	WakeupOnWrite bool      `json:"wakeup_on_write"`
	ReportDesc    []byte    `json:"report_desc"`
	Roles         []HIDRole `json:"roles,omitempty"`
	DevNodeIndex  int       `json:"-"`
}

type NetFunction struct {
	DevAddr         *string `json:"dev_addr,omitempty"`
	HostAddr        *string `json:"host_addr,omitempty"`
	Class           *uint8  `json:"class,omitempty"`
	SubClass        *uint8  `json:"subclass,omitempty"`
	Protocol        *uint8  `json:"protocol,omitempty"`
	CompatibleID    string  `json:"compatible_id"`
	SubCompatibleID string  `json:"sub_compatible_id,omitempty"`
}

type StorageFunction struct {
	Removable     bool   `json:"removable"`
	ReadOnly      bool   `json:"read_only"`
	InquiryString string `json:"inquiry_string"`
	File          string `json:"file"`
}

type VideoFunction struct {
	FunctionName string        `json:"function_name"`
	Formats      []VideoFormat `json:"formats"`
	// The iInterface string the host displays for this camera. nil = DO NOT
	// WRITE, which leaves f_uvc's own "UVC Camera" in place; a profile stored
	// before the attribute existed deserializes that way, and renaming every
	// host's camera on upgrade is not something an upgrade may do.
	HostName *string `json:"host_name,omitempty"`
	// nil = DO NOT WRITE, which leaves f_uvc's own default and the control
	// interrupt IN endpoint in place. Only a kernel whose uvc function group
	// carries enable_interrupt_ep can honour false.
	InterruptEndpoint  *bool  `json:"interrupt_endpoint,omitempty"`
	StreamingMaxPacket uint16 `json:"streaming_maxpacket"`
	StreamingMaxBurst  uint8  `json:"streaming_maxburst"`
	StreamingInterval  uint8  `json:"streaming_interval"`
}

type VideoFormat struct {
	Codec  string       `json:"codec"`
	Frames []VideoFrame `json:"frames"`
}

type VideoFrame struct {
	Width     uint16   `json:"width"`
	Height    uint16   `json:"height"`
	Intervals []uint32 `json:"intervals"`
}

type AudioFunction struct {
	FunctionName  string  `json:"function_name"`
	HostName      *string `json:"host_name,omitempty"`
	PChannelMask  uint32  `json:"p_chmask"`
	PSampleRate   uint32  `json:"p_srate"`
	PSampleSize   uint8   `json:"p_ssize"`
	CChannelMask  uint32  `json:"c_chmask"`
	CSampleRate   uint32  `json:"c_srate"`
	CSampleSize   uint8   `json:"c_ssize"`
	RequestNumber uint8   `json:"req_number"`
}

type OSDesc struct {
	VendorCode string `json:"b_vendor_code"`
	QwSign     string `json:"qw_sign"`
}

func (p *Profile) Normalize() {
	index := 0
	for i := range p.Functions {
		if p.Functions[i].Kind != FunctionHID || p.Functions[i].HID == nil {
			continue
		}
		p.Functions[i].HID.DevNodeIndex = index
		if len(p.Functions[i].HID.Roles) == 0 && index < len(HIDRoles) {
			p.Functions[i].HID.Roles = []HIDRole{HIDRoles[index]}
		}
		index++
	}
	p.Provenance.Descriptors = p.Descriptors != nil
	if p.Provenance.Origin == "" || (p.Provenance.Origin == OriginBuiltIn && !p.BuiltIn) {
		p.Provenance.Origin, p.Provenance.Source = OriginUser, ""
	}
	if p.Provenance.Origin == OriginPreset {
		if preset, ok := PresetByID(p.Provenance.Source); !ok || !preset.matches(p.Device) {
			p.Provenance.Origin, p.Provenance.Source = OriginUser, ""
		}
	}
}

func (p *Profile) Validate() error {
	if p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema version %d, want %d", p.SchemaVersion, SchemaVersion)
	}
	if p.Name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if !profileNamePattern.MatchString(p.Name) {
		return fmt.Errorf("profile name %q must match %s", p.Name, profileNamePattern)
	}
	if err := p.Provenance.validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	if err := p.Device.validate(); err != nil {
		return fmt.Errorf("device: %w", err)
	}
	if err := p.Config.validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if p.BuiltIn {
		base, ok := builtInProfile(p.Name)
		if !ok {
			return fmt.Errorf("unknown built-in profile %q", p.Name)
		}
		if !reflect.DeepEqual(p.Device, base.Device) || p.Config != base.Config {
			return fmt.Errorf("built-in profile %q alters device identity", p.Name)
		}
	}

	seen := make(map[string]bool, len(p.Functions))
	var hid []Function
	functionFS := false
	mediaStarted := false
	indices := map[FunctionKind]int{FunctionUVC: 0, FunctionUAC2: 0}
	for _, f := range p.Functions {
		if err := f.validate(); err != nil {
			return fmt.Errorf("function %s.%s: %w", f.Kind, f.Instance, err)
		}
		key := string(f.Kind) + "." + f.Instance
		if seen[key] {
			return fmt.Errorf("duplicate function %s", key)
		}
		seen[key] = true
		if f.Kind == FunctionHID {
			if mediaStarted {
				return fmt.Errorf("hid functions must precede media functions")
			}
			hid = append(hid, f)
		}
		functionFS = functionFS || f.Kind == FunctionFFS
		if f.Kind == FunctionUVC || f.Kind == FunctionUAC2 {
			mediaStarted = true
			pattern := cameraPattern
			if f.Kind == FunctionUAC2 {
				pattern = microphonePattern
			}
			match := pattern.FindStringSubmatch(f.Instance)
			index, _ := strconv.Atoi(match[1])
			if index != indices[f.Kind] {
				return fmt.Errorf("%s instances must be contiguous from zero, got %s", f.Kind, f.Instance)
			}
			indices[f.Kind]++
		}
	}
	if functionFS && mediaStarted {
		return fmt.Errorf("functionfs and media functions cannot coexist")
	}
	if err := validateHIDOrder(hid, functionFS); err != nil {
		return err
	}
	if p.OSDesc != nil {
		if err := p.OSDesc.validate(); err != nil {
			return err
		}
	}
	if p.Descriptors != nil {
		if err := p.Descriptors.validate(); err != nil {
			return fmt.Errorf("descriptors: %w", err)
		}
	}
	return nil
}

func validateHIDOrder(hid []Function, functionFS bool) error {
	if len(hid) == 0 {
		return nil
	}
	// A prefix of GS0,GS1,GS2 rather than all three, because the layout
	// distributes the three roles over one, two or three interfaces. The
	// prefix itself is not negotiable: f_hid hands out /dev/hidgN in mkdir
	// order and a gap would move every minor after it.
	want := len(hidInstances)
	if functionFS {
		want = 2
	}
	if len(hid) > want {
		return fmt.Errorf("%d hid functions, want at most %d", len(hid), want)
	}
	for i, f := range hid {
		if f.Instance != hidInstances[i] {
			return fmt.Errorf("hid function %d is %q, want %q", i, f.Instance, hidInstances[i])
		}
		if f.HID.DevNodeIndex != i {
			return fmt.Errorf("hid function %s has dev node index %d, want %d", f.Instance, f.HID.DevNodeIndex, i)
		}
	}
	return nil
}

func (d *Device) validate() error {
	if err := hexU16("vendor id", d.VendorID); err != nil {
		return err
	}
	if err := hexU16("product id", d.ProductID); err != nil {
		return err
	}
	if d.BCDUSB != nil {
		if err := hexU16("bcd usb", *d.BCDUSB); err != nil {
			return err
		}
	}
	if d.BCDDevice != nil {
		if err := hexU16("bcd device", *d.BCDDevice); err != nil {
			return err
		}
	}
	if d.Serial != nil && *d.Serial == "" {
		return fmt.Errorf("serial is present but empty")
	}
	if d.Manufacturer == "" || d.Product == "" {
		return fmt.Errorf("manufacturer and product are required")
	}
	for name, value := range map[string]string{"manufacturer": d.Manufacturer, "product": d.Product} {
		if err := usbString(name, value); err != nil {
			return err
		}
	}
	if d.Serial != nil {
		if err := usbString("serial", *d.Serial); err != nil {
			return err
		}
	}
	return nil
}

func (c *ConfigDesc) validate() error {
	if c.BMAttributes&0x80 == 0 {
		return fmt.Errorf("bmAttributes 0x%02X must set bit 7", c.BMAttributes)
	}
	if c.MaxPower == 0 || c.MaxPower > 500 {
		return fmt.Errorf("MaxPower %d mA out of range", c.MaxPower)
	}
	if c.Configuration == "" {
		return fmt.Errorf("configuration string is empty")
	}
	if c.BMAttributes&^0xE0 != 0 {
		return fmt.Errorf("bmAttributes 0x%02X contains reserved bits", c.BMAttributes)
	}
	return usbString("configuration", c.Configuration)
}

func (f *Function) validate() error {
	if !instancePattern.MatchString(f.Instance) {
		return fmt.Errorf("invalid instance %q", f.Instance)
	}
	switch f.Kind {
	case FunctionHID:
		if !slices.Contains(hidInstances[:], f.Instance) {
			return fmt.Errorf("unsupported hid instance %q", f.Instance)
		}
		if f.HID == nil || f.Net != nil || f.Storage != nil || f.FFS != nil || f.Video != nil || f.Audio != nil {
			return fmt.Errorf("expects exactly a hid payload")
		}
		return f.HID.validate()
	case FunctionNCM, FunctionRNDIS:
		if f.Net == nil || f.HID != nil || f.Storage != nil || f.FFS != nil || f.Video != nil || f.Audio != nil {
			return fmt.Errorf("expects exactly a net payload")
		}
		return f.Net.validate()
	case FunctionMassStorage:
		if f.Storage == nil || f.HID != nil || f.Net != nil || f.FFS != nil || f.Video != nil || f.Audio != nil {
			return fmt.Errorf("expects exactly a storage payload")
		}
		return f.Storage.validate()
	case FunctionFFS:
		if f.Instance != "hybrid" {
			return fmt.Errorf("unsupported functionfs instance %q", f.Instance)
		}
		if f.FFS == nil || f.HID != nil || f.Net != nil || f.Storage != nil || f.Video != nil || f.Audio != nil {
			return fmt.Errorf("expects exactly a functionfs payload")
		}
		return f.FFS.Validate()
	case FunctionUVC:
		if f.Video == nil || f.HID != nil || f.Net != nil || f.Storage != nil || f.FFS != nil || f.Audio != nil {
			return fmt.Errorf("expects exactly a video payload")
		}
		if !cameraPattern.MatchString(f.Instance) {
			return fmt.Errorf("unsupported uvc instance %q", f.Instance)
		}
		return f.Video.validate()
	case FunctionUAC2:
		if f.Audio == nil || f.HID != nil || f.Net != nil || f.Storage != nil || f.FFS != nil || f.Video != nil {
			return fmt.Errorf("expects exactly an audio payload")
		}
		if !microphonePattern.MatchString(f.Instance) {
			return fmt.Errorf("unsupported uac2 instance %q", f.Instance)
		}
		return f.Audio.validate()
	default:
		return fmt.Errorf("unknown kind")
	}
}

func (f *FunctionFS) Validate() error {
	if f.Interfaces == 0 || f.Interfaces > 16 {
		return fmt.Errorf("interfaces %d outside 1..16", f.Interfaces)
	}
	if len(f.Endpoints) == 0 || len(f.Endpoints) > 7 {
		return fmt.Errorf("endpoints %d outside 1..7", len(f.Endpoints))
	}
	seenSource := make(map[uint8]bool, len(f.Endpoints))
	seenMapped := make(map[uint8]bool, len(f.Endpoints))
	for i, endpoint := range f.Endpoints {
		if err := endpoint.validate(); err != nil {
			return fmt.Errorf("endpoint %d: %w", i, err)
		}
		if seenSource[endpoint.SourceAddress] || seenMapped[endpoint.Address] {
			return fmt.Errorf("endpoint %d has a duplicate address", i)
		}
		seenSource[endpoint.SourceAddress] = true
		seenMapped[endpoint.Address] = true
	}
	return nil
}

func (v *VideoFunction) validate() error {
	if err := usbString("function name", v.FunctionName); err != nil || v.FunctionName == "" {
		if err != nil {
			return err
		}
		return fmt.Errorf("function name is empty")
	}
	if err := hostName(v.HostName); err != nil {
		return err
	}
	if v.StreamingMaxPacket != 256 && v.StreamingMaxPacket != 512 && v.StreamingMaxPacket != 768 {
		return fmt.Errorf("streaming maxpacket %d, want 256, 512, or 768", v.StreamingMaxPacket)
	}
	if v.StreamingMaxBurst != 0 || v.StreamingInterval != 1 {
		return fmt.Errorf("high-speed streaming requires maxburst 0 and interval 1")
	}
	if len(v.Formats) != 1 || v.Formats[0].Codec != "mjpeg" || len(v.Formats[0].Frames) == 0 {
		return fmt.Errorf("exactly one non-empty mjpeg format is required")
	}
	seen := make(map[[2]uint16]bool, len(v.Formats[0].Frames))
	for _, frame := range v.Formats[0].Frames {
		if err := frame.validate(); err != nil {
			return err
		}
		key := [2]uint16{frame.Width, frame.Height}
		if seen[key] {
			return fmt.Errorf("duplicate frame %dx%d", frame.Width, frame.Height)
		}
		seen[key] = true
	}
	return nil
}

func (e FunctionFSEndpoint) validate() error {
	for name, address := range map[string]uint8{"source": e.SourceAddress, "mapped": e.Address} {
		if address&0x0f == 0 || address&0x70 != 0 {
			return fmt.Errorf("%s address 0x%02x is invalid", name, address)
		}
	}
	if e.SourceAddress&0x80 != e.Address&0x80 {
		return fmt.Errorf("source and mapped directions differ")
	}
	switch e.Transfer {
	case EndpointBulk:
		if e.MaxPacket == 0 || e.MaxPacket > 512 || e.Interval != 0 {
			return fmt.Errorf("bulk packet %d interval %d", e.MaxPacket, e.Interval)
		}
	case EndpointInterrupt:
		if e.MaxPacket == 0 || e.MaxPacket > 1024 || e.Interval == 0 || e.Interval > 16 {
			return fmt.Errorf("interrupt packet %d interval %d", e.MaxPacket, e.Interval)
		}
	case EndpointIsochronous:
		if e.MaxPacket == 0 || e.MaxPacket > 1024 || e.Interval == 0 || e.Interval > 16 || e.Mult > 2 {
			return fmt.Errorf("isochronous packet %d interval %d mult %d", e.MaxPacket, e.Interval, e.Mult)
		}
	default:
		return fmt.Errorf("transfer %q", e.Transfer)
	}
	return nil
}

func (f VideoFrame) validate() error {
	allowed := map[[2]uint16]bool{{1280, 720}: true, {640, 480}: true, {320, 240}: true, {160, 120}: true}
	if !allowed[[2]uint16{f.Width, f.Height}] {
		return fmt.Errorf("unsupported frame %dx%d", f.Width, f.Height)
	}
	if len(f.Intervals) == 0 || len(f.Intervals) > 2 {
		return fmt.Errorf("frame %dx%d needs one or two intervals", f.Width, f.Height)
	}
	seen := make(map[uint32]bool, len(f.Intervals))
	for _, interval := range f.Intervals {
		if interval != 333333 && interval != 666666 {
			return fmt.Errorf("frame %dx%d has unsupported interval %d", f.Width, f.Height, interval)
		}
		if seen[interval] {
			return fmt.Errorf("frame %dx%d repeats interval %d", f.Width, f.Height, interval)
		}
		seen[interval] = true
	}
	if len(f.Intervals) == 2 && f.Intervals[0] > f.Intervals[1] {
		return fmt.Errorf("frame %dx%d intervals are not ascending", f.Width, f.Height)
	}
	return nil
}

func (a *AudioFunction) validate() error {
	if err := usbString("function name", a.FunctionName); err != nil || a.FunctionName == "" {
		if err != nil {
			return err
		}
		return fmt.Errorf("function name is empty")
	}
	if err := hostName(a.HostName); err != nil {
		return err
	}
	if a.PChannelMask != 1 || a.PSampleRate != 48000 || a.PSampleSize != 2 {
		return fmt.Errorf("microphone must expose mono 48000 Hz signed 16-bit USB IN")
	}
	if a.CChannelMask != 0 {
		return fmt.Errorf("microphone cannot expose USB OUT channels")
	}
	if a.CSampleRate != 48000 || a.CSampleSize != 2 {
		return fmt.Errorf("disabled USB OUT must retain 48000 Hz signed 16-bit defaults")
	}
	if a.RequestNumber < 2 || a.RequestNumber > 8 {
		return fmt.Errorf("request number %d, want 2 through 8", a.RequestNumber)
	}
	return nil
}

func (h *HIDFunction) validate() error {
	if h.Protocol == 0 && len(h.Roles) < 2 {
		return fmt.Errorf("protocol is zero")
	}
	if h.SubClass > 1 {
		return fmt.Errorf("subclass %d, want 0 or 1", h.SubClass)
	}
	// Empty is a profile stored before roles existed; Normalize backfills it
	// from the function's position, which is what it has always meant.
	if len(h.Roles) > 0 {
		if err := ValidateHIDLayout([][]HIDRole{h.Roles}); err != nil {
			return err
		}
	}
	length, err := reportLength(h.ReportDesc)
	if err != nil {
		return fmt.Errorf("report descriptor: %w", err)
	}
	if length != h.ReportLength {
		return fmt.Errorf("report length %d, descriptor implies %d", h.ReportLength, length)
	}
	return nil
}

func (n *NetFunction) validate() error {
	for name, addr := range map[string]*string{"dev_addr": n.DevAddr, "host_addr": n.HostAddr} {
		if addr == nil {
			continue
		}
		if _, err := net.ParseMAC(*addr); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	for name, value := range map[string]string{"compatible id": n.CompatibleID, "sub-compatible id": n.SubCompatibleID} {
		if name == "compatible id" && value == "" {
			return fmt.Errorf("compatible id is empty")
		}
		if len(value) > 8 || strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r > 0x7E }) >= 0 {
			return fmt.Errorf("%s must be at most 8 printable ASCII bytes", name)
		}
	}
	return nil
}

func (s *StorageFunction) validate() error {
	if len(s.InquiryString) != 0 && len(s.InquiryString) != 28 {
		return fmt.Errorf("inquiry string is %d bytes, want 28", len(s.InquiryString))
	}
	if s.File == "" {
		return fmt.Errorf("backing file is empty")
	}
	clean := filepath.Clean(s.File)
	if clean != s.File || !filepath.IsAbs(clean) || strings.ContainsAny(s.File, "\x00\r\n") {
		return fmt.Errorf("backing file %q is not a clean absolute path", s.File)
	}
	return nil
}

func (o *OSDesc) validate() error {
	if err := hexUint("os desc vendor code", o.VendorCode, 8); err != nil {
		return err
	}
	if o.QwSign == "" || len(o.QwSign) > 14 || strings.IndexFunc(o.QwSign, func(r rune) bool { return r < 0x20 || r > 0x7E }) >= 0 {
		return fmt.Errorf("os desc qw_sign must be 1 to 14 printable ASCII bytes")
	}
	return nil
}

func (d *DescriptorSet) validate() error {
	if len(d.Device) == 0 && len(d.Configurations) == 0 && len(d.BOS) == 0 && len(d.Strings) == 0 && len(d.HIDReports) == 0 {
		return fmt.Errorf("descriptor set is empty")
	}
	if len(d.Device) != 0 {
		if err := validateDeviceDescriptor(d.Device, len(d.Configurations), d.Strings); err != nil {
			return err
		}
	}
	for i, config := range d.Configurations {
		if err := validateConfigurationDescriptor(config, d.Strings); err != nil {
			return fmt.Errorf("configuration %d: %w", i, err)
		}
	}
	if len(d.BOS) != 0 {
		if err := validateBOSDescriptor(d.BOS); err != nil {
			return err
		}
	}
	for index, value := range d.Strings {
		if !stringIndexPattern.MatchString(index) {
			return fmt.Errorf("invalid string index %q", index)
		}
		parts := strings.Split(index, ":")
		n, err := strconv.ParseUint(parts[len(parts)-1], 10, 8)
		if err != nil || n == 0 {
			return fmt.Errorf("invalid string index %q", index)
		}
		if err := usbString("descriptor string "+index, value); err != nil {
			return err
		}
	}
	for instance, report := range d.HIDReports {
		if !instancePattern.MatchString(instance) {
			return fmt.Errorf("invalid hid report name %q", instance)
		}
		if err := validateHIDReportDescriptor(report); err != nil {
			return fmt.Errorf("hid report %s: %w", instance, err)
		}
	}
	return nil
}

func (d *DescriptorSet) Validate() error {
	return d.validate()
}

func validateDeviceDescriptor(data []byte, configurations int, strings map[string]string) error {
	if len(data) != 18 || data[0] != 18 || data[1] != 1 {
		return fmt.Errorf("device descriptor must be one 18-byte device descriptor")
	}
	if packet := data[7]; packet != 8 && packet != 16 && packet != 32 && packet != 64 {
		return fmt.Errorf("device descriptor has invalid endpoint-zero packet size %d", packet)
	}
	if data[17] == 0 || int(data[17]) != configurations {
		return fmt.Errorf("device descriptor names %d configurations, package has %d", data[17], configurations)
	}
	for _, index := range data[14:17] {
		if index == 0 || descriptorStringExists(strings, index) {
			continue
		}
		return fmt.Errorf("device descriptor references missing string %d", index)
	}
	return nil
}

func descriptorStringExists(values map[string]string, index byte) bool {
	want := strconv.Itoa(int(index))
	for key := range values {
		parts := strings.Split(key, ":")
		if parts[len(parts)-1] == want {
			return true
		}
	}
	return false
}

func validateConfigurationDescriptor(data []byte, strings map[string]string) error {
	if len(data) < 9 || data[0] != 9 || data[1] != 2 {
		return fmt.Errorf("missing configuration header")
	}
	if total := int(binary.LittleEndian.Uint16(data[2:4])); total != len(data) {
		return fmt.Errorf("wTotalLength is %d, file is %d", total, len(data))
	}
	if data[5] == 0 {
		return fmt.Errorf("bConfigurationValue is zero")
	}
	if data[6] != 0 && !descriptorStringExists(strings, data[6]) {
		return fmt.Errorf("configuration references missing string %d", data[6])
	}

	type interfaceKey struct{ number, alternate byte }
	interfaces := make(map[byte]bool)
	endpoints := make(map[interfaceKey]map[byte]bool)
	endpointOwners := make(map[byte]byte)
	declared := make(map[interfaceKey]byte)
	var current *interfaceKey
	for offset := 0; offset < len(data); {
		length := int(data[offset])
		if length < 2 || offset+length > len(data) {
			return fmt.Errorf("descriptor at offset %d overruns the configuration", offset)
		}
		typeID := data[offset+1]
		switch typeID {
		case 4:
			if length < 9 {
				return fmt.Errorf("short interface descriptor at offset %d", offset)
			}
			key := interfaceKey{data[offset+2], data[offset+3]}
			if _, exists := declared[key]; exists {
				return fmt.Errorf("duplicate interface %d alternate %d", key.number, key.alternate)
			}
			interfaces[key.number] = true
			if data[offset+8] != 0 && !descriptorStringExists(strings, data[offset+8]) {
				return fmt.Errorf("interface %d references missing string %d", key.number, data[offset+8])
			}
			declared[key] = data[offset+4]
			endpoints[key] = make(map[byte]bool)
			current = &key
		case 5:
			if length < 7 || current == nil {
				return fmt.Errorf("endpoint descriptor at offset %d has no interface", offset)
			}
			address := data[offset+2]
			if address&0x0F == 0 || address&0x70 != 0 {
				return fmt.Errorf("invalid endpoint address 0x%02X", address)
			}
			if endpoints[*current][address] {
				return fmt.Errorf("duplicate endpoint 0x%02X", address)
			}
			if owner, exists := endpointOwners[address]; exists && owner != current.number {
				return fmt.Errorf("endpoint 0x%02X belongs to interfaces %d and %d", address, owner, current.number)
			}
			endpoints[*current][address] = true
			endpointOwners[address] = current.number
			packet := binary.LittleEndian.Uint16(data[offset+4:offset+6]) & 0x07FF
			if packet == 0 || packet > 1024 {
				return fmt.Errorf("invalid endpoint packet size %d", packet)
			}
		case 11:
			if length < 8 {
				return fmt.Errorf("short interface association descriptor at offset %d", offset)
			}
			if data[offset+7] != 0 && !descriptorStringExists(strings, data[offset+7]) {
				return fmt.Errorf("interface association references missing string %d", data[offset+7])
			}
		}
		offset += length
	}
	if len(interfaces) != int(data[4]) {
		return fmt.Errorf("bNumInterfaces is %d, descriptors define %d", data[4], len(interfaces))
	}
	for key := range declared {
		if key.alternate == 0 {
			continue
		}
		if _, ok := declared[interfaceKey{number: key.number}]; !ok {
			return fmt.Errorf("interface %d has no alternate setting zero", key.number)
		}
	}
	for key, count := range declared {
		if len(endpoints[key]) != int(count) {
			return fmt.Errorf("interface %d alternate %d declares %d endpoints, has %d", key.number, key.alternate, count, len(endpoints[key]))
		}
	}
	return nil
}

func validateBOSDescriptor(data []byte) error {
	if len(data) < 5 || data[0] != 5 || data[1] != 15 {
		return fmt.Errorf("bos descriptor is missing its header")
	}
	if total := int(binary.LittleEndian.Uint16(data[2:4])); total != len(data) {
		return fmt.Errorf("bos wTotalLength is %d, file is %d", total, len(data))
	}
	capabilities := 0
	for offset := 5; offset < len(data); {
		length := int(data[offset])
		if length < 3 || offset+length > len(data) || data[offset+1] != 16 {
			return fmt.Errorf("invalid bos capability at offset %d", offset)
		}
		capabilities++
		offset += length
	}
	if capabilities != int(data[4]) {
		return fmt.Errorf("bos names %d capabilities, has %d", data[4], capabilities)
	}
	return nil
}

func validateHIDReportDescriptor(data []byte) error {
	if len(data) == 0 || len(data) > 64<<10 {
		return fmt.Errorf("size %d is outside 1..65536", len(data))
	}
	mainItems := 0
	collections := 0
	for offset := 0; offset < len(data); {
		prefix := data[offset]
		if prefix == 0xFE {
			if offset+3 > len(data) || offset+3+int(data[offset+1]) > len(data) {
				return fmt.Errorf("truncated long item at offset %d", offset)
			}
			offset += 3 + int(data[offset+1])
			continue
		}
		size := int(prefix & 0x03)
		if size == 3 {
			size = 4
		}
		if offset+1+size > len(data) {
			return fmt.Errorf("truncated item at offset %d", offset)
		}
		if prefix&0x0C == 0 && prefix&0xF0 >= 0x80 && prefix&0xF0 <= 0xB0 {
			mainItems++
		}
		switch prefix & 0xFC {
		case 0xA0:
			collections++
		case 0xC0:
			collections--
			if collections < 0 {
				return fmt.Errorf("collection closes before it opens at offset %d", offset)
			}
		}
		offset += 1 + size
	}
	if collections != 0 {
		return fmt.Errorf("%d collections remain open", collections)
	}
	if mainItems == 0 {
		return fmt.Errorf("contains no input, output, feature, or collection item")
	}
	return nil
}

// A blank iInterface is worse on a host than the kernel default, and the 80
// bytes sources.validateLabel already enforces on the API that sets these is
// the stricter of the two limits, so it is the one that decides.
func hostName(value *string) error {
	if value == nil {
		return nil
	}
	if err := usbString("host name", *value); err != nil {
		return err
	}
	if *value == "" || len(*value) > 80 {
		return fmt.Errorf("host name must contain 1 to 80 bytes")
	}
	return nil
}

func usbString(name, value string) error {
	if !utf8.ValidString(value) || len(value) > 126 || strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7F }) >= 0 {
		return fmt.Errorf("%s must be valid UTF-8 without control characters and at most 126 bytes", name)
	}
	return nil
}

func reportLength(desc []byte) (uint16, error) {
	var size, count, bits, id int
	ids := make(map[int]int)
	for i := 0; i < len(desc); {
		prefix := desc[i]
		if prefix == 0xFE {
			return 0, fmt.Errorf("long item at offset %d", i)
		}
		n := int(prefix & 0x03)
		if n == 3 {
			n = 4
		}
		i++
		if i+n > len(desc) {
			return 0, fmt.Errorf("truncated item at offset %d", i-1)
		}
		value := 0
		for j := n - 1; j >= 0; j-- {
			value = value<<8 | int(desc[i+j])
		}
		i += n
		switch prefix &^ 0x03 {
		case 0x74:
			size = value
		case 0x94:
			count = value
		case 0x84:
			if value == 0 {
				return 0, fmt.Errorf("report id 0 at offset %d", i-n-1)
			}
			id = value
		case 0x80:
			ids[id] += size * count
			bits += size * count
		}
	}
	if id != 0 {
		bits = 0
		for _, total := range ids {
			bits = max(bits, total)
		}
		bits += 8
	}
	if bits == 0 || bits%8 != 0 {
		return 0, fmt.Errorf("input reports total %d bits", bits)
	}
	return uint16(bits / 8), nil
}

func hexU16(name, value string) error {
	return hexUint(name, value, 16)
}

func hexUint(name, value string, bits int) error {
	if !strings.HasPrefix(value, "0x") {
		return fmt.Errorf("%s %q is not 0x-prefixed", name, value)
	}
	if _, err := strconv.ParseUint(value[2:], 16, bits); err != nil {
		return fmt.Errorf("%s %q: %w", name, value, err)
	}
	return nil
}

func ptr[T any](v T) *T {
	return &v
}

func builtInProfile(name string) (Profile, bool) {
	switch name {
	case ProfileStandard:
		return standardProfile(), true
	case ProfileHIDOnly:
		return hidOnlyProfile(), true
	default:
		return Profile{}, false
	}
}

func standardProfile() Profile {
	return Profile{
		SchemaVersion: SchemaVersion,
		Name:          ProfileStandard,
		BuiltIn:       true,
		Provenance:    Provenance{Origin: OriginBuiltIn},
		Device: Device{
			VendorID:     "0x3346",
			ProductID:    "0x1009",
			BCDDevice:    ptr(BCDDeviceNormal),
			Class:        ptr[uint8](0xEF),
			SubClass:     ptr[uint8](0x02),
			Protocol:     ptr[uint8](0x01),
			Serial:       ptr(DeviceSerial()),
			Manufacturer: "sipeed",
			Product:      "NanoKVM",
		},
		Config:    ConfigDesc{BMAttributes: 0xE0, MaxPower: 120, Configuration: "NanoKVM"},
		Functions: hidFunctions(descKeyboardStandard, descMouseRelative, descPointerStandard, 0),
	}
}

func hidOnlyProfile() Profile {
	return Profile{
		SchemaVersion: SchemaVersion,
		Name:          ProfileHIDOnly,
		BuiltIn:       true,
		Provenance:    Provenance{Origin: OriginBuiltIn},
		Device: Device{
			VendorID:     "0x3346",
			ProductID:    "0x1009",
			BCDUSB:       ptr("0x0101"),
			BCDDevice:    ptr(BCDDeviceHIDOnly),
			Manufacturer: "sipeed",
			Product:      "NanoKVM",
		},
		Config:    ConfigDesc{BMAttributes: 0xA0, MaxPower: 200, Configuration: "NanoKVM"},
		Functions: hidFunctions(descKeyboardHIDOnly, descMouseRelative, descPointerHIDOnly, 1),
	}
}

func hidFunctions(keyboard, mouse, pointer []byte, subClass uint8) []Function {
	descs := [...][]byte{keyboard, mouse, pointer}
	protocols := [...]uint8{1, 2, 2}
	lengths := [...]uint16{8, 4, 6}

	functions := make([]Function, 0, len(hidInstances))
	for i, instance := range hidInstances {
		functions = append(functions, Function{
			Kind:     FunctionHID,
			Instance: instance,
			HID: &HIDFunction{
				Protocol:      protocols[i],
				SubClass:      subClass,
				ReportLength:  lengths[i],
				WakeupOnWrite: true,
				ReportDesc:    bytes.Clone(descs[i]),
				Roles:         []HIDRole{HIDRoles[i]},
				DevNodeIndex:  i,
			},
		})
	}
	return functions
}
