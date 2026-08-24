package functionfs

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestProjectKeepsOnlyEligibleInterfaces(t *testing.T) {
	device := []byte{18, 1, 0, 2, 0, 0, 0, 64, 1, 0, 2, 0, 0, 1, 0, 0, 0, 1}
	config := []byte{9, 2, 0, 0, 2, 1, 0, 0x80, 50}
	config = append(config, []byte{9, 4, 0, 0, 1, 0xff, 0, 0, 0}...)
	config = append(config, bulkEndpoint(0x81, 64)...)
	config = append(config, []byte{9, 4, 1, 0, 1, 0x03, 1, 1, 0}...)
	config = append(config, []byte{7, 5, 0x82, 3, 8, 0, 10}...)
	binary.LittleEndian.PutUint16(config[2:4], uint16(len(config)))

	projected, err := Project(device, config, []uint8{0})
	if err != nil {
		t.Fatal(err)
	}
	if projected[22] != 1 || len(projected) != 18+9+9+7 {
		t.Fatalf("projected interfaces=%d bytes=%d", projected[22], len(projected))
	}
	if _, err := Project(device, config, []uint8{1}); !errors.Is(err, ErrProtected) {
		t.Fatalf("protected interface returned %v", err)
	}
}

func TestProjectRefusesPartialAssociation(t *testing.T) {
	device := []byte{18, 1, 0, 2, 0, 0, 0, 64, 1, 0, 2, 0, 0, 1, 0, 0, 0, 1}
	config := []byte{9, 2, 0, 0, 2, 1, 0, 0x80, 50}
	config = append(config, []byte{8, 11, 0, 2, 0xff, 0, 0, 0}...)
	config = append(config, []byte{9, 4, 0, 0, 1, 0xff, 0, 0, 0}...)
	config = append(config, []byte{7, 5, 0x81, 2, 64, 0, 0}...)
	config = append(config, []byte{9, 4, 1, 0, 0, 0xff, 0, 0, 0}...)
	binary.LittleEndian.PutUint16(config[2:4], uint16(len(config)))
	if _, err := Project(device, config, []uint8{0}); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("partial association returned %v", err)
	}
	iso := append([]byte(nil), config...)
	iso[29] = 1
	if _, err := Project(device, iso, []uint8{0, 1}); err != nil {
		t.Fatalf("isochronous endpoint returned %v", err)
	}
}

func TestProjectRejectsWrappedAssociation(t *testing.T) {
	device := []byte{18, 1, 0, 2, 0, 0, 0, 64, 1, 0, 2, 0, 0, 1, 0, 0, 0, 1}
	config := []byte{9, 2, 0, 0, 1, 1, 0, 0x80, 50}
	config = append(config, []byte{8, 11, 250, 10, 0xff, 0, 0, 0}...)
	config = append(config, []byte{9, 4, 0, 0, 1, 0xff, 0, 0, 0}...)
	config = append(config, bulkEndpoint(0x81, 64)...)
	binary.LittleEndian.PutUint16(config[2:4], uint16(len(config)))
	if _, err := Project(device, config, []uint8{0}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Project() error = %v, want ErrMalformed", err)
	}
}

func FuzzProject(f *testing.F) {
	raw := vendorFixture(bulkEndpoint(0x81, 64))
	f.Add(raw[:18], raw[18:], []byte{0})
	f.Fuzz(func(t *testing.T, device, config, selected []byte) {
		if len(device) > 64 || len(config) > MaxDescriptorBytes || len(selected) > MaxInterfaces {
			return
		}
		_, _ = Project(device, config, selected)
	})
}
