package presentation

import (
	"bytes"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
)

const (
	SchemaVersion = 1

	ProfileStandard = "standard"
	ProfileHIDOnly  = "hid-only"

	InquiryString      = "NanoKVM USB Mass Storage0520"
	InquiryStringCDROM = "NanoKVM USB CD/DVD-ROM  0520"
)

type FunctionKind string

const (
	FunctionHID         FunctionKind = "hid"
	FunctionNCM         FunctionKind = "ncm"
	FunctionRNDIS       FunctionKind = "rndis"
	FunctionMassStorage FunctionKind = "mass_storage"
)

var hidInstances = [...]string{"GS0", "GS1", "GS2"}

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
	SchemaVersion int        `json:"schema_version"`
	Name          string     `json:"name"`
	BuiltIn       bool       `json:"built_in"`
	Device        Device     `json:"device"`
	Config        ConfigDesc `json:"config"`
	Functions     []Function `json:"functions"`
	OSDesc        *OSDesc    `json:"os_desc,omitempty"`
}

type Device struct {
	VendorID     string  `json:"vendor_id"`
	ProductID    string  `json:"product_id"`
	BCDUSB       *string `json:"bcd_usb,omitempty"`    // nil = DO NOT WRITE
	BCDDevice    *string `json:"bcd_device,omitempty"` // nil = DO NOT WRITE (H14)
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
}

type HIDFunction struct {
	Protocol      uint8  `json:"protocol"`
	SubClass      uint8  `json:"subclass"`
	ReportLength  uint16 `json:"report_length"`
	WakeupOnWrite bool   `json:"wakeup_on_write"`
	ReportDesc    []byte `json:"report_desc"`
	DevNodeIndex  int    `json:"-"`
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
		index++
	}
}

func (p *Profile) Validate() error {
	if p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema version %d, want %d", p.SchemaVersion, SchemaVersion)
	}
	if p.Name == "" {
		return fmt.Errorf("profile name is empty")
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
			hid = append(hid, f)
		}
	}
	if err := validateHIDOrder(hid); err != nil {
		return err
	}
	if p.OSDesc != nil {
		return p.OSDesc.validate()
	}
	return nil
}

func validateHIDOrder(hid []Function) error {
	if len(hid) == 0 {
		return nil
	}
	if len(hid) != len(hidInstances) {
		return fmt.Errorf("%d hid functions, want %d", len(hid), len(hidInstances))
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
	return nil
}

func (f *Function) validate() error {
	if f.Instance == "" {
		return fmt.Errorf("instance is empty")
	}
	switch f.Kind {
	case FunctionHID:
		if f.HID == nil || f.Net != nil || f.Storage != nil {
			return fmt.Errorf("expects exactly a hid payload")
		}
		return f.HID.validate()
	case FunctionNCM, FunctionRNDIS:
		if f.Net == nil || f.HID != nil || f.Storage != nil {
			return fmt.Errorf("expects exactly a net payload")
		}
		return f.Net.validate()
	case FunctionMassStorage:
		if f.Storage == nil || f.HID != nil || f.Net != nil {
			return fmt.Errorf("expects exactly a storage payload")
		}
		return f.Storage.validate()
	default:
		return fmt.Errorf("unknown kind")
	}
}

func (h *HIDFunction) validate() error {
	if h.Protocol == 0 {
		return fmt.Errorf("protocol is zero")
	}
	if h.SubClass > 1 {
		return fmt.Errorf("subclass %d, want 0 or 1", h.SubClass)
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
	if n.CompatibleID == "" {
		return fmt.Errorf("compatible id is empty")
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
	return nil
}

func (o *OSDesc) validate() error {
	if err := hexU16("os desc vendor code", o.VendorCode); err != nil {
		return err
	}
	if o.QwSign == "" {
		return fmt.Errorf("os desc qw_sign is empty")
	}
	return nil
}

func reportLength(desc []byte) (uint16, error) {
	var size, count, bits int
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
			return 0, fmt.Errorf("report id at offset %d", i-n-1)
		case 0x80:
			bits += size * count
		}
	}
	if bits == 0 || bits%8 != 0 {
		return 0, fmt.Errorf("input reports total %d bits", bits)
	}
	return uint16(bits / 8), nil
}

func hexU16(name, value string) error {
	if !strings.HasPrefix(value, "0x") {
		return fmt.Errorf("%s %q is not 0x-prefixed", name, value)
	}
	if _, err := strconv.ParseUint(value[2:], 16, 16); err != nil {
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
		Device: Device{
			VendorID:     "0x3346",
			ProductID:    "0x1009",
			Class:        ptr[uint8](0xEF),
			SubClass:     ptr[uint8](0x02),
			Protocol:     ptr[uint8](0x01),
			Serial:       ptr("0123456789ABCDEF"),
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
		Device: Device{
			VendorID:     "0x3346",
			ProductID:    "0x1009",
			BCDUSB:       ptr("0x0101"),
			BCDDevice:    ptr("0x0623"),
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
				DevNodeIndex:  i,
			},
		})
	}
	return functions
}
