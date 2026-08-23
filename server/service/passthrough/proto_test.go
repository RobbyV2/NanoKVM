package passthrough

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func sampleDevice() Device {
	return Device{
		Path:      "/sys/devices/pci0000:00/0000:00:14.0/usb1/1-1",
		BusID:     "1-1",
		BusNum:    0x0102,
		DevNum:    0x0304,
		Speed:     SpeedHigh,
		IDVendor:  0x046d,
		IDProduct: 0xc31c,
		BCDDevice: 0x6410,

		DeviceClass:        0x00,
		DeviceSubClass:     0x01,
		DeviceProtocol:     0x02,
		ConfigurationValue: 0x03,
		NumConfigurations:  0x04,
		NumInterfaces:      0x05,
	}
}

func TestDeviceLayoutIsPacked312Bytes(t *testing.T) {
	if DeviceSize != 312 {
		t.Fatalf("DeviceSize = %d, want 312", DeviceSize)
	}
	if offTrailer+6 != DeviceSize {
		t.Fatalf("trailer ends at %d, want %d", offTrailer+6, DeviceSize)
	}

	device := sampleDevice()
	raw, err := device.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(raw) != 312 {
		t.Fatalf("encoded usbip_usb_device is %d bytes, want 312", len(raw))
	}

	for _, field := range []struct {
		name   string
		offset int
		want   []byte
	}{
		{"path", offPath, []byte(device.Path)},
		{"path padding", offPath + len(device.Path), bytes.Repeat([]byte{0}, pathSize-len(device.Path))},
		{"busid", offBusID, append([]byte("1-1"), bytes.Repeat([]byte{0}, busIDSize-3)...)},
		{"busnum", offBusNum, []byte{0x00, 0x00, 0x01, 0x02}},
		{"devnum", offDevNum, []byte{0x00, 0x00, 0x03, 0x04}},
		{"speed", offSpeed, []byte{0x00, 0x00, 0x00, 0x03}},
		{"idVendor", offIDVendor, []byte{0x04, 0x6d}},
		{"idProduct", offIDProduct, []byte{0xc3, 0x1c}},
		{"bcdDevice", offBCDDevice, []byte{0x64, 0x10}},
		{"trailer", offTrailer, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}},
	} {
		got := raw[field.offset : field.offset+len(field.want)]
		if !bytes.Equal(got, field.want) {
			t.Errorf("%s at offset %d = % x, want % x", field.name, field.offset, got, field.want)
		}
	}
}

func TestDeviceRoundTripsBigEndian(t *testing.T) {
	want := sampleDevice()

	raw, err := want.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := DecodeDevice(raw)
	if err != nil {
		t.Fatalf("DecodeDevice: %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
	if got.DevID() != 0x01020304 {
		t.Fatalf("DevID = %#08x, want 0x01020304", got.DevID())
	}
}

func TestDevIDKeepsFullBusnumAndDevnum(t *testing.T) {
	device := Device{BusNum: 0x1234, DevNum: 0x5678}
	if got := device.DevID(); got != 0x12345678 {
		t.Fatalf("DevID = %#08x, want 0x12345678: busnum and devnum must not be truncated", got)
	}
}

func TestOpCommonIsBigEndian(t *testing.T) {
	want := []byte{0x01, 0x11, 0x80, 0x03, 0x00, 0x00, 0x00, 0x00}
	got := OpCommon{Version: ProtocolVersion, Code: CodeReqImport, Status: StatusOK}.Encode()
	if !bytes.Equal(got, want) {
		t.Fatalf("op_common = % x, want % x", got, want)
	}

	header, err := DecodeOpCommon(got)
	if err != nil {
		t.Fatalf("DecodeOpCommon: %v", err)
	}
	if header != (OpCommon{Version: 0x0111, Code: 0x8003, Status: StatusOK}) {
		t.Fatalf("decoded = %+v", header)
	}

	reply, err := DecodeOpCommon([]byte{0x01, 0x11, 0x00, 0x03, 0x00, 0x00, 0x00, 0x02})
	if err != nil {
		t.Fatalf("DecodeOpCommon reply: %v", err)
	}
	if reply.Code != CodeRepImport || reply.Status != StatusDeviceBusy {
		t.Fatalf("reply = %+v, want OP_REP_IMPORT with a busy status", reply)
	}
}

func TestDecodeRejectsTruncatedInput(t *testing.T) {
	if _, err := DecodeOpCommon(make([]byte, HeaderSize-1)); !errors.Is(err, ErrTruncated) {
		t.Fatalf("DecodeOpCommon of 7 bytes: %v", err)
	}
	if _, err := DecodeDevice(make([]byte, DeviceSize-1)); !errors.Is(err, ErrTruncated) {
		t.Fatalf("DecodeDevice of 311 bytes: %v", err)
	}
}

func TestEncodeImportRequestIsFortyBytes(t *testing.T) {
	got, err := EncodeImportRequest("1-1.4")
	if err != nil {
		t.Fatalf("EncodeImportRequest: %v", err)
	}
	if len(got) != ImportRequestSize || ImportRequestSize != 40 {
		t.Fatalf("import request is %d bytes, want 40", len(got))
	}

	want := append([]byte{0x01, 0x11, 0x80, 0x03, 0x00, 0x00, 0x00, 0x00}, "1-1.4"...)
	want = append(want, bytes.Repeat([]byte{0}, busIDSize-5)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("import request = % x, want % x", got, want)
	}
}

func TestEncodeImportRequestRejectsUnusableBusIDs(t *testing.T) {
	for _, busID := range []string{
		"",
		"1-1\x00extra",
		"1-1 1-2",
		"../../etc/passwd",
		"11111111111111111111111111111111-1",
		"usb1",
	} {
		if _, err := EncodeImportRequest(busID); err == nil {
			t.Fatalf("EncodeImportRequest(%q) was accepted", busID)
		}
	}
}

func TestEncodeRejectsAnOverlongPath(t *testing.T) {
	device := sampleDevice()
	device.Path = string(bytes.Repeat([]byte("a"), pathSize))
	if _, err := device.Encode(); !errors.Is(err, ErrFieldTooLong) {
		t.Fatalf("Encode of a 256 byte path: %v", err)
	}
}

func TestInterfaceRoundTripsThePackedFourBytes(t *testing.T) {
	want := Interface{Class: 0x0e, SubClass: 0x02, Protocol: 0x01}

	raw := want.Encode()
	if len(raw) != InterfaceSize {
		t.Fatalf("Encode is %d bytes, want %d", len(raw), InterfaceSize)
	}
	if raw[3] != 0 {
		t.Fatalf("padding = %d, want 0", raw[3])
	}

	got, err := DecodeInterface(raw)
	if err != nil {
		t.Fatalf("DecodeInterface: %v", err)
	}
	if got != want {
		t.Fatalf("DecodeInterface = %+v, want %+v", got, want)
	}
	if _, err := DecodeInterface(raw[:3]); !errors.Is(err, ErrTruncated) {
		t.Fatalf("DecodeInterface of 3 bytes = %v, want ErrTruncated", err)
	}
}

func TestRefusalNamesTheDeviceAndTheReason(t *testing.T) {
	device := sampleDevice()
	device.BusID = "3-1.2"

	relayable := []Interface{
		{Class: 0x03},              // HID
		{Class: 0x08, SubClass: 6}, // mass storage
		{Class: 0x09},              // hub
		{Class: 0xe0, SubClass: 2}, // wireless, not the radio
	}
	if refusal := (RemoteDevice{Device: device, Interfaces: relayable}).Refusal(); refusal != "" {
		t.Fatalf("Refusal of a relayable device = %q, want none", refusal)
	}

	for name, iface := range map[string]Interface{
		"audio":     {Class: 0x01, SubClass: 0x02},
		"video":     {Class: 0x0e, SubClass: 0x02},
		"bluetooth": {Class: 0xe0, SubClass: 0x01, Protocol: 0x01},
	} {
		t.Run(name, func(t *testing.T) {
			refusal := (RemoteDevice{Device: device, Interfaces: append(relayable, iface)}).Refusal()
			if refusal == "" {
				t.Fatalf("Refusal of a %s device = %q, want a reason", name, refusal)
			}
			if !strings.Contains(refusal, "3-1.2") || !strings.Contains(refusal, "046d:c31c") {
				t.Fatalf("Refusal %q does not name the device", refusal)
			}
			if !strings.Contains(refusal, "isochronous") {
				t.Fatalf("Refusal %q does not give the reason", refusal)
			}
		})
	}
}
