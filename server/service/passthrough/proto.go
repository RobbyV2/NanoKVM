package passthrough

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
)

const (
	ProtocolVersion uint16 = 0x0111
	CodeReqImport   uint16 = 0x8003
	CodeRepImport   uint16 = 0x0003
	CodeReqDevlist  uint16 = 0x8005
	CodeRepDevlist  uint16 = 0x0005

	HeaderSize        = 8
	DeviceSize        = 312
	InterfaceSize     = 4
	CountSize         = 4
	ImportRequestSize = HeaderSize + busIDSize

	maxDevices = 128

	pathSize  = 256
	busIDSize = 32
)

// Offsets into the packed usbip_usb_device. Only the numeric fields are byte
// swapped; path and busid are NUL padded bytes on the wire.
const (
	offPath      = 0
	offBusID     = offPath + pathSize
	offBusNum    = offBusID + busIDSize
	offDevNum    = offBusNum + 4
	offSpeed     = offDevNum + 4
	offIDVendor  = offSpeed + 4
	offIDProduct = offIDVendor + 2
	offBCDDevice = offIDProduct + 2
	offTrailer   = offBCDDevice + 2
)

var (
	ErrTruncated        = errors.New("passthrough: truncated usbip message")
	ErrFieldTooLong     = errors.New("passthrough: field does not fit the wire layout")
	ErrBusID            = errors.New("passthrough: not a usb busid")
	ErrVersion          = errors.New("passthrough: unexpected usbip version")
	ErrUnexpectedCode   = errors.New("passthrough: unexpected usbip reply code")
	ErrUnexpectedDevice = errors.New("passthrough: unexpected usbip device")
	ErrRejected         = errors.New("passthrough: exporter rejected the import")
)

var busIDPattern = regexp.MustCompile(`^[0-9]+-[0-9]+(\.[0-9]+)*$`)

type OpStatus uint32

const (
	StatusOK OpStatus = iota
	StatusUnavailable
	StatusDeviceBusy
	StatusDeviceError
	StatusNoDevice
	StatusUnexpected
)

var opStatusNames = map[OpStatus]string{
	StatusOK:          "ok",
	StatusUnavailable: "device unavailable",
	StatusDeviceBusy:  "device already exported",
	StatusDeviceError: "device in error state",
	StatusNoDevice:    "no such device",
	StatusUnexpected:  "unexpected request",
}

func (s OpStatus) String() string {
	if name, ok := opStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("status %d", uint32(s))
}

type Speed uint32

const (
	SpeedUnknown Speed = iota
	SpeedLow
	SpeedFull
	SpeedHigh
	SpeedWireless
	SpeedSuper
	SpeedSuperPlus
)

var speedNames = map[Speed]string{
	SpeedUnknown:   "unknown",
	SpeedLow:       "low",
	SpeedFull:      "full",
	SpeedHigh:      "high",
	SpeedWireless:  "wireless",
	SpeedSuper:     "super",
	SpeedSuperPlus: "super-plus",
}

func (s Speed) String() string {
	if name, ok := speedNames[s]; ok {
		return name
	}
	return fmt.Sprintf("speed %d", uint32(s))
}

type OpCommon struct {
	Version uint16
	Code    uint16
	Status  OpStatus
}

func (o OpCommon) Encode() []byte {
	out := make([]byte, HeaderSize)
	binary.BigEndian.PutUint16(out[0:2], o.Version)
	binary.BigEndian.PutUint16(out[2:4], o.Code)
	binary.BigEndian.PutUint32(out[4:8], uint32(o.Status))
	return out
}

func DecodeOpCommon(raw []byte) (OpCommon, error) {
	if len(raw) < HeaderSize {
		return OpCommon{}, fmt.Errorf("%w: op_common is %d of %d bytes", ErrTruncated, len(raw), HeaderSize)
	}
	return OpCommon{
		Version: binary.BigEndian.Uint16(raw[0:2]),
		Code:    binary.BigEndian.Uint16(raw[2:4]),
		Status:  OpStatus(binary.BigEndian.Uint32(raw[4:8])),
	}, nil
}

type Device struct {
	Path      string
	BusID     string
	BusNum    uint32
	DevNum    uint32
	Speed     Speed
	IDVendor  uint16
	IDProduct uint16
	BCDDevice uint16

	DeviceClass        uint8
	DeviceSubClass     uint8
	DeviceProtocol     uint8
	ConfigurationValue uint8
	NumConfigurations  uint8
	NumInterfaces      uint8
}

// What attach_store parses out of the third decimal.
func (d Device) DevID() uint32 {
	return d.BusNum<<16 | d.DevNum
}

func (d Device) Encode() ([]byte, error) {
	out := make([]byte, DeviceSize)
	if err := putPadded(out[offPath:offBusID], d.Path); err != nil {
		return nil, fmt.Errorf("encode path: %w", err)
	}
	if err := putPadded(out[offBusID:offBusNum], d.BusID); err != nil {
		return nil, fmt.Errorf("encode busid: %w", err)
	}

	binary.BigEndian.PutUint32(out[offBusNum:offDevNum], d.BusNum)
	binary.BigEndian.PutUint32(out[offDevNum:offSpeed], d.DevNum)
	binary.BigEndian.PutUint32(out[offSpeed:offIDVendor], uint32(d.Speed))
	binary.BigEndian.PutUint16(out[offIDVendor:offIDProduct], d.IDVendor)
	binary.BigEndian.PutUint16(out[offIDProduct:offBCDDevice], d.IDProduct)
	binary.BigEndian.PutUint16(out[offBCDDevice:offTrailer], d.BCDDevice)
	copy(out[offTrailer:DeviceSize], []byte{
		d.DeviceClass, d.DeviceSubClass, d.DeviceProtocol,
		d.ConfigurationValue, d.NumConfigurations, d.NumInterfaces,
	})
	return out, nil
}

func DecodeDevice(raw []byte) (Device, error) {
	if len(raw) < DeviceSize {
		return Device{}, fmt.Errorf("%w: usbip_usb_device is %d of %d bytes", ErrTruncated, len(raw), DeviceSize)
	}
	trailer := raw[offTrailer:DeviceSize]
	return Device{
		Path:      trimPadding(raw[offPath:offBusID]),
		BusID:     trimPadding(raw[offBusID:offBusNum]),
		BusNum:    binary.BigEndian.Uint32(raw[offBusNum:offDevNum]),
		DevNum:    binary.BigEndian.Uint32(raw[offDevNum:offSpeed]),
		Speed:     Speed(binary.BigEndian.Uint32(raw[offSpeed:offIDVendor])),
		IDVendor:  binary.BigEndian.Uint16(raw[offIDVendor:offIDProduct]),
		IDProduct: binary.BigEndian.Uint16(raw[offIDProduct:offBCDDevice]),
		BCDDevice: binary.BigEndian.Uint16(raw[offBCDDevice:offTrailer]),

		DeviceClass:        trailer[0],
		DeviceSubClass:     trailer[1],
		DeviceProtocol:     trailer[2],
		ConfigurationValue: trailer[3],
		NumConfigurations:  trailer[4],
		NumInterfaces:      trailer[5],
	}, nil
}

// usbip_usb_interface, the only description of a remote device's function the
// exporter offers before it is imported.
type Interface struct {
	Class    uint8
	SubClass uint8
	Protocol uint8
}

func (i Interface) Encode() []byte {
	return []byte{i.Class, i.SubClass, i.Protocol, 0}
}

func DecodeInterface(raw []byte) (Interface, error) {
	if len(raw) < InterfaceSize {
		return Interface{}, fmt.Errorf("%w: usbip_usb_interface is %d of %d bytes", ErrTruncated, len(raw), InterfaceSize)
	}
	return Interface{Class: raw[0], SubClass: raw[1], Protocol: raw[2]}, nil
}

const (
	classAudio    uint8 = 0x01
	classVideo    uint8 = 0x0e
	classWireless uint8 = 0xe0
	subclassRadio uint8 = 0x01
)

// Exact can relay an isochronous endpoint when the start allows it, and Hybrid
// has no isochronous data path at all, so a streaming device is refused unless
// it was asked for. An interface class is all the exporter tells us, and these
// are the classes whose interfaces carry the isochronous endpoints.
func (i Interface) Unsupported() string {
	switch {
	case i.Class == classAudio:
		return fmt.Sprintf("audio interface (class %02x) streams over isochronous endpoints", i.Class)
	case i.Class == classVideo:
		return fmt.Sprintf("video interface (class %02x) streams over isochronous endpoints", i.Class)
	case i.Class == classWireless && i.SubClass == subclassRadio:
		return fmt.Sprintf("the Bluetooth SCO channel (class %02x) is isochronous", i.Class)
	}
	return ""
}

// One entry of an exporter's device list.
type RemoteDevice struct {
	Device
	Interfaces []Interface
}

// Names the device as well as the reason: a refusal has to read as an account
// of the device the operator picked, not of the feature.
func (d RemoteDevice) Refusal() string {
	for _, iface := range d.Interfaces {
		if reason := iface.Unsupported(); reason != "" {
			return fmt.Sprintf("%s %04x:%04x: %s", d.BusID, d.IDVendor, d.IDProduct, reason)
		}
	}
	return ""
}

func EncodeDevlistRequest() []byte {
	return OpCommon{Version: ProtocolVersion, Code: CodeReqDevlist, Status: StatusOK}.Encode()
}

func EncodeImportRequest(busID string) ([]byte, error) {
	if !busIDPattern.MatchString(busID) {
		return nil, fmt.Errorf("%w: %q", ErrBusID, busID)
	}

	out := make([]byte, ImportRequestSize)
	copy(out, OpCommon{Version: ProtocolVersion, Code: CodeReqImport, Status: StatusOK}.Encode())
	if err := putPadded(out[HeaderSize:], busID); err != nil {
		return nil, fmt.Errorf("encode busid: %w", err)
	}
	return out, nil
}

func putPadded(dst []byte, value string) error {
	if len(value) >= len(dst) {
		return fmt.Errorf("%w: %q needs %d of %d bytes", ErrFieldTooLong, value, len(value)+1, len(dst))
	}
	copy(dst, value)
	return nil
}

func trimPadding(raw []byte) string {
	if end := bytes.IndexByte(raw, 0); end >= 0 {
		return string(raw[:end])
	}
	return string(raw)
}
