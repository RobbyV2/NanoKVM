//go:build linux

package functionfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

func TestUSBFSABI(t *testing.T) {
	if unsafe.Sizeof(usbControl{}) != 24 || unsafe.Sizeof(usbURB{}) != 56 || unsafe.Sizeof(disconnectClaim{}) != 264 {
		t.Fatalf("USBFS sizes = %d/%d/%d", unsafe.Sizeof(usbControl{}), unsafe.Sizeof(usbURB{}), unsafe.Sizeof(disconnectClaim{}))
	}
	if usbdevfsControl != 0xc0185500 || usbdevfsSubmitURB != 0x8038550a || usbdevfsReapURBNoDelay != 0x4008550d {
		t.Fatalf("USBFS ioctls = %#x/%#x/%#x", usbdevfsControl, usbdevfsSubmitURB, usbdevfsReapURBNoDelay)
	}
}

func TestHIDDescriptorUsesInterfaceRecipient(t *testing.T) {
	setup := descriptorSetup(0x22, 0, 4, 64)
	if setup.RequestType != 0x81 || setup.Index != 4 || setup.Length != 64 {
		t.Fatalf("HID setup = %+v", setup)
	}
	if setup := descriptorSetup(3, 1, 0x0409, 255); setup.RequestType != 0x80 {
		t.Fatalf("string setup = %+v", setup)
	}
}

func TestDescriptorFileIsBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "descriptors")
	if err := os.WriteFile(path, make([]byte, MaxDescriptorBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readDescriptors(path); !errors.Is(err, ErrMalformed) {
		t.Fatalf("oversize descriptor file returned %v", err)
	}
}
