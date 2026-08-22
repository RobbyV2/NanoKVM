package presentation

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

const (
	hexKeyboardStandard = "05010906a101050719e029e71500250175019508810295017508810395057501050819012905910295017503910395067508150025e70507190029e78100c0"
	hexKeyboardHIDOnly  = "05010906a101050719e029e71500250175019508810295017508810395057501050819012905910295017503910395067508150025650507190029658100c0"
	hexMouseRelative    = "05010902a1010901a1000509190129031500250195037501810295017505810305010930093109381581257f750895038106c0c0"
	hexPointerStandard  = "05010902a1010901a10005091901290515002501950575018102950175038101050109300931150026ff7f350046ff7f751095028102050109381581257f35004500750895018106c0c0"
	hexPointerHIDOnly   = "05010902a1010901a10005091901290315002501950375018102950175058101050109300931150026ff7f350046ff7f751095028102050109381581257f35004500750895018106c0c0"
)

func TestBuiltInReportDescriptors(t *testing.T) {
	tests := []struct {
		profile  Profile
		instance string
		want     string
		length   int
		reported uint16
	}{
		{standardProfile(), "GS0", hexKeyboardStandard, 63, 8},
		{standardProfile(), "GS1", hexMouseRelative, 52, 4},
		{standardProfile(), "GS2", hexPointerStandard, 74, 6},
		{hidOnlyProfile(), "GS0", hexKeyboardHIDOnly, 63, 8},
		{hidOnlyProfile(), "GS1", hexMouseRelative, 52, 4},
		{hidOnlyProfile(), "GS2", hexPointerHIDOnly, 74, 6},
	}

	for _, tt := range tests {
		t.Run(tt.profile.Name+"/"+tt.instance, func(t *testing.T) {
			want, err := hex.DecodeString(tt.want)
			if err != nil {
				t.Fatal(err)
			}
			if len(want) != tt.length {
				t.Fatalf("fixture is %d bytes, want %d", len(want), tt.length)
			}

			fn := hidFunction(t, tt.profile, tt.instance)
			if !bytes.Equal(fn.ReportDesc, want) {
				t.Fatalf("report_desc = %x, want %x", fn.ReportDesc, want)
			}
			if len(fn.ReportDesc) != tt.length {
				t.Fatalf("report_desc is %d bytes, want %d", len(fn.ReportDesc), tt.length)
			}
			if fn.ReportLength != tt.reported {
				t.Fatalf("report_length = %d, want %d", fn.ReportLength, tt.reported)
			}

			implied, err := reportLength(fn.ReportDesc)
			if err != nil {
				t.Fatalf("report length: %v", err)
			}
			if implied != tt.reported {
				t.Fatalf("descriptor implies %d, report_length = %d", implied, tt.reported)
			}
		})
	}
}

func TestBuiltInProfilesValidate(t *testing.T) {
	for _, p := range []Profile{standardProfile(), hidOnlyProfile()} {
		t.Run(p.Name, func(t *testing.T) {
			if err := p.Validate(); err != nil {
				t.Fatalf("validate: %v", err)
			}
		})
	}
}

func TestBuiltInDeviceDescriptorFields(t *testing.T) {
	standard := standardProfile().Device
	if standard.BCDUSB != nil {
		t.Fatalf("standard writes bcdUSB: %q", *standard.BCDUSB)
	}
	if standard.BCDDevice == nil || *standard.BCDDevice != BCDDeviceNormal {
		t.Fatalf("standard bcdDevice = %v, want %q", standard.BCDDevice, BCDDeviceNormal)
	}
	if standard.Serial == nil || *standard.Serial != "0123456789ABCDEF" {
		t.Fatalf("standard serial = %v", standard.Serial)
	}

	hidOnly := hidOnlyProfile().Device
	if hidOnly.Serial != nil {
		t.Fatalf("hid-only serial = %q, want unwritten", *hidOnly.Serial)
	}
	if hidOnly.Class != nil || hidOnly.SubClass != nil || hidOnly.Protocol != nil {
		t.Fatal("hid-only writes device class triple")
	}
	if hidOnly.BCDUSB == nil || *hidOnly.BCDUSB != "0x0101" || hidOnly.BCDDevice == nil || *hidOnly.BCDDevice != BCDDeviceHIDOnly {
		t.Fatalf("hid-only bcd = %v %v", hidOnly.BCDUSB, hidOnly.BCDDevice)
	}
	if subClass := hidFunction(t, hidOnlyProfile(), "GS1").SubClass; subClass != 1 {
		t.Fatalf("hid-only subclass = %d, want 1", subClass)
	}
}

func TestValidateRejectsReorderedHID(t *testing.T) {
	p := standardProfile()
	p.Functions[1], p.Functions[2] = p.Functions[2], p.Functions[1]
	p.Normalize()

	if err := p.Validate(); err == nil {
		t.Fatal("expected reordered hid functions to be rejected")
	}
}

func TestValidateRejectsMismatchedReportLength(t *testing.T) {
	p := standardProfile()
	hidFunction(t, p, "GS0").ReportLength = 6

	if err := p.Validate(); err == nil {
		t.Fatal("expected mismatched report length to be rejected")
	}
}

func TestValidateRejectsHIDOnlySerial(t *testing.T) {
	p := hidOnlyProfile()
	p.Device.Serial = ptr("0123456789ABCDEF")

	if err := p.Validate(); err == nil {
		t.Fatal("expected hid-only serial to be rejected")
	}
}

func hidFunction(t *testing.T, p Profile, instance string) *HIDFunction {
	t.Helper()
	for _, f := range p.Functions {
		if f.Kind == FunctionHID && f.Instance == instance {
			return f.HID
		}
	}
	t.Fatalf("profile %s has no hid.%s", p.Name, instance)
	return nil
}

// The selector's whole contract. ECM is a real USB network class and an obvious
// third entry, and there is no f_ecm anywhere in this tree: no FunctionKind, no
// compile case, no branch in S03usbdev. Offering it would offer a gadget the
// layer below cannot build, so the parser is what keeps the set honest.
func TestParseNetworkKindOffersOnlyWhatTheGadgetLayerBuilds(t *testing.T) {
	if got := len(NetworkKinds); got != 2 {
		t.Fatalf("NetworkKinds holds %d entries %v, want exactly ncm and rndis", got, NetworkKinds)
	}

	for _, name := range []string{"ncm", "rndis"} {
		kind, err := ParseNetworkKind(name)
		if err != nil {
			t.Fatalf("ParseNetworkKind(%q) = %v", name, err)
		}
		if string(kind) != name {
			t.Fatalf("ParseNetworkKind(%q) = %q", name, kind)
		}
	}

	for _, name := range []string{"ecm", "eem", "hid", "mass_storage", "", "NCM"} {
		if _, err := ParseNetworkKind(name); !errors.Is(err, ErrUnknownNetworkKind) {
			t.Fatalf("ParseNetworkKind(%q) = %v, want ErrUnknownNetworkKind", name, err)
		}
	}
}
