//go:build linux

package functionfs

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestFunctionFSEndpointFilesAreNamedByAddress(t *testing.T) {
	image, err := Import(vendorFixture(bulkEndpoint(0x02, 512), bulkEndpoint(0x81, 512)), fixtureFetcher{}, testCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	// bit 4 is FUNCTIONFS_VIRTUAL_ADDR; without it the kernel names endpoint files by index.
	if binary.LittleEndian.Uint32(image.Descriptors[8:12])&16 == 0 {
		t.Fatal("descriptor flags cleared FUNCTIONFS_VIRTUAL_ADDR")
	}
	want := []string{"ep01", "ep81"}
	for index, endpoint := range image.Function.Endpoints {
		if got := functionFSEndpointName(endpoint.Address); got != want[index] {
			t.Fatalf("endpoint %d (0x%02x) = %q, want %q", index, endpoint.Address, got, want[index])
		}
	}
}

// A reactor that takes a request and never answers stands in for an endpoint
// the controller has wedged: the transfer has to give up on it, and the URB it
// left behind stays pinned until the device file is closed.
func TestTransferAbandonsARequestThatOutlivesItsCancellation(t *testing.T) {
	previous := transferCancelGrace
	transferCancelGrace = 20 * time.Millisecond
	defer func() { transferCancelGrace = previous }()

	device := &linuxDevice{requests: make(chan *usbRequest), close: make(chan struct{}), done: make(chan struct{})}
	wedged := make(chan *usbRequest, 1)
	go func() { wedged <- <-device.requests }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := device.Transfer(ctx, Endpoint{SourceAddress: 0x81, Transfer: "bulk", MaxPacket: 512}, make([]byte, 8))
		result <- err
	}()

	select {
	case err := <-result:
		if !errors.Is(err, ErrTransfer) {
			t.Fatalf("wedged transfer returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a wedged transfer never returned")
	}
	(<-wedged).pinner.Unpin()
}
