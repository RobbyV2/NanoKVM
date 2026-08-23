package functionfs

import (
	"encoding/binary"
	"errors"
	"testing"

	"NanoKVM-Server/service/presentation"
)

type fixtureFetcher map[[3]uint16][]byte

func (f fixtureFetcher) Descriptor(kind uint8, index uint8, recipient uint16, limit int) ([]byte, error) {
	value, ok := f[[3]uint16{uint16(kind), uint16(index), recipient}]
	if !ok {
		return nil, errors.New("missing fixture descriptor")
	}
	if len(value) > limit {
		value = value[:limit]
	}
	return append([]byte(nil), value...), nil
}

func testCapabilities() presentation.CapabilityTable {
	return presentation.CapabilityTable{
		Source: "test", MaxInEndpoints: 6, MaxOutEndpoints: 5,
		InFIFOWords: []int{768, 512, 512, 384, 128, 128},
		Functions: map[presentation.FunctionKind]presentation.FunctionCaps{
			presentation.FunctionHID: {Available: true, InEPs: 1, OutEPs: 1},
			presentation.FunctionFFS: {Available: true},
		},
	}
}

func vendorFixture(endpoints ...[]byte) []byte {
	device := []byte{18, 1, 0x00, 0x02, 0, 0, 0, 64, 0x34, 0x12, 0x78, 0x56, 0, 1, 0, 0, 0, 1}
	config := []byte{9, 2, 0, 0, 1, 1, 0, 0x80, 50, 9, 4, 0, 0, uint8(len(endpoints)), 0xff, 0, 0, 0}
	for _, endpoint := range endpoints {
		config = append(config, endpoint...)
	}
	binary.LittleEndian.PutUint16(config[2:4], uint16(len(config)))
	return append(device, config...)
}

func bulkEndpoint(address uint8, packet uint16) []byte {
	value := []byte{7, 5, address, 2, 0, 0, 0}
	binary.LittleEndian.PutUint16(value[4:6], packet)
	return value
}

func TestImportVendorBulk(t *testing.T) {
	raw := vendorFixture(bulkEndpoint(0x02, 512), bulkEndpoint(0x81, 512))
	image, err := Import(raw, fixtureFetcher{}, testCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	if image.Interfaces[0] != 0 || image.Endpoints[0x02] != 0x01 || image.Endpoints[0x81] != 0x81 {
		t.Fatalf("unexpected maps: interfaces %#v endpoints %#v", image.Interfaces, image.Endpoints)
	}
	if got := image.Function.Endpoints; len(got) != 2 || got[0].SourceAddress != 0x02 || got[1].SourceAddress != 0x81 {
		t.Fatalf("unexpected function endpoints: %#v", got)
	}
	if binary.LittleEndian.Uint32(image.Descriptors[8:12]) != 19 {
		t.Fatalf("FunctionFS flags = %d", binary.LittleEndian.Uint32(image.Descriptors[8:12]))
	}
}

func TestImportRefusesUnsafeLayouts(t *testing.T) {
	base := vendorFixture(bulkEndpoint(0x81, 512))
	tests := map[string]struct {
		mutate func([]byte)
		want   error
	}{
		"hub":                {func(raw []byte) { raw[32] = 0x09 }, ErrProtected},
		"isochronous":        {func(raw []byte) { raw[39] = 0x01 }, ErrIsochronous},
		"alternate":          {func(raw []byte) { raw[30] = 1 }, ErrAmbiguous},
		"oversize":           {func(raw []byte) { binary.LittleEndian.PutUint16(raw[40:42], 513) }, ErrEndpointSize},
		"unknown descriptor": {func(raw []byte) { raw[37] = 0x30 }, ErrUnsupported},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			raw := append([]byte(nil), base...)
			test.mutate(raw)
			_, err := Import(raw, fixtureFetcher{}, testCapabilities())
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestImportFIFOBudgetIsLoadBearing(t *testing.T) {
	raw := vendorFixture(bulkEndpoint(0x81, 512), bulkEndpoint(0x82, 512))
	limited := testCapabilities()
	limited.InFIFOWords = []int{768, 127, 127, 127, 127, 127}
	if _, err := Import(raw, fixtureFetcher{}, limited); !errors.Is(err, presentation.ErrFIFOBudget) {
		t.Fatalf("got %v, want FIFO budget failure", err)
	}
	limited.InFIFOWords[1] = 128
	if _, err := Import(raw, fixtureFetcher{}, limited); err != nil {
		t.Fatalf("mutated FIFO should admit the layout: %v", err)
	}
}

func TestImportHIDReport(t *testing.T) {
	device := []byte{18, 1, 0, 2, 0, 0, 0, 64, 1, 0, 2, 0, 0, 1, 0, 0, 0, 1}
	report := []byte{0x05, 0x01, 0x09, 0x06, 0xa1, 0x01, 0xc0}
	hid := []byte{9, 0x21, 0x11, 0x01, 0, 1, 0x22, byte(len(report)), 0}
	config := []byte{9, 2, 0, 0, 1, 1, 0, 0x80, 50, 9, 4, 0, 0, 1, 3, 1, 1, 0}
	config = append(config, hid...)
	config = append(config, []byte{7, 5, 0x81, 3, 8, 0, 10}...)
	binary.LittleEndian.PutUint16(config[2:4], uint16(len(config)))
	fetcher := fixtureFetcher{[3]uint16{0x22, 0, 0}: report}
	image, err := Import(append(device, config...), fetcher, testCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	if string(image.HIDReports[0]) != string(report) {
		t.Fatalf("report not retained: %x", image.HIDReports[0])
	}
}

func TestImportNotificationlessCDCACM(t *testing.T) {
	device := []byte{18, 1, 0, 2, 0, 0, 0, 64, 1, 0, 2, 0, 0, 1, 0, 0, 0, 1}
	config := []byte{9, 2, 0, 0, 2, 1, 0, 0x80, 50}
	config = append(config, []byte{8, 11, 0, 2, 2, 2, 1, 0}...)
	config = append(config, []byte{9, 4, 0, 0, 0, 2, 2, 1, 0}...)
	config = append(config, []byte{5, 0x24, 0, 0x10, 0x01}...)
	config = append(config, []byte{5, 0x24, 1, 0, 1}...)
	config = append(config, []byte{4, 0x24, 2, 2}...)
	config = append(config, []byte{5, 0x24, 6, 0, 1}...)
	config = append(config, []byte{9, 4, 1, 0, 2, 0x0a, 0, 0, 0}...)
	config = append(config, bulkEndpoint(0x02, 512)...)
	config = append(config, bulkEndpoint(0x81, 512)...)
	binary.LittleEndian.PutUint16(config[2:4], uint16(len(config)))
	if _, err := Import(append(device, config...), fixtureFetcher{}, testCapabilities()); err != nil {
		t.Fatal(err)
	}
	mutated := append([]byte(nil), config...)
	mutated[35] = 3
	if _, err := Import(append(device, mutated...), fixtureFetcher{}, testCapabilities()); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("unknown CDC reference returned %v", err)
	}
	mutated = append([]byte(nil), config...)
	mutated[28] = 0x02
	if _, err := Import(append(device, mutated...), fixtureFetcher{}, testCapabilities()); err == nil {
		t.Fatalf("missing CDC header returned %v", err)
	}
}

func TestDecodeUSBStringRejectsUnpairedSurrogate(t *testing.T) {
	if _, err := decodeUSBString([]byte{4, 3, 0x00, 0xd8}); err == nil {
		t.Fatal("unpaired UTF-16 surrogate was accepted")
	}
	if _, err := decodeUSBString([]byte{4, 3, 0, 0}); err == nil {
		t.Fatal("embedded NUL was accepted")
	}
}

func FuzzImportDescriptors(f *testing.F) {
	f.Add(vendorFixture(bulkEndpoint(0x81, 512)))
	f.Add(vendorFixture(bulkEndpoint(0x02, 64), bulkEndpoint(0x81, 64)))
	f.Add([]byte{18, 1})
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > MaxDescriptorBytes+1 {
			return
		}
		_, _ = Import(raw, fixtureFetcher{}, testCapabilities())
	})
}
