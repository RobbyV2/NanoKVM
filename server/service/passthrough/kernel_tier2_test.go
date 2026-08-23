//go:build linux && kernelint

package passthrough

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"NanoKVM-Server/service/functionfs"
	"NanoKVM-Server/service/kernelint"
	"NanoKVM-Server/service/presentation"
)

// The imported device is stubbed because usbip-host cannot export the gadget
// this test is about to build. Everything on the FunctionFS side of the relay
// is the shipped code against the real kernel: the configfs mkdir, the
// mount(2), the ep0 descriptor writes and the bind.
type stubSource struct{}

func (stubSource) Descriptor(uint8, uint8, uint16, int) ([]byte, error) {
	return nil, errors.New("no string or class descriptors in the fixture")
}
func (stubSource) Control(context.Context, functionfs.Setup, []byte) ([]byte, error) {
	return nil, functionfs.ErrStall
}
func (stubSource) Transfer(context.Context, functionfs.Endpoint, []byte) ([]byte, error) {
	return nil, functionfs.ErrStall
}
func (stubSource) ClearHalt(uint8) error { return nil }
func (stubSource) Reset() error          { return nil }
func (stubSource) Close() error          { return nil }

type kernelHybridFactory struct {
	caps presentation.CapabilityTable
}

func (f *kernelHybridFactory) Prepare(Local) (HybridRelay, presentation.FunctionFS, error) {
	prepared, err := functionfs.PrepareRemote(vendorDescriptors(), stubSource{}, stubSource{}, f.caps)
	if err != nil {
		return nil, presentation.FunctionFS{}, err
	}
	return &heldRelay{inner: prepared.Relay, done: make(chan struct{})}, prepared.Image.Function, nil
}

// f_ffs unbinds the gadget the moment ep0 is closed, and Relay.Run closes it on
// the way out, so a relay that exits on its own takes the mount and the binding
// with it before anything can be asserted. The data plane is not what this test
// is about; the mount and the ep0 writes inside PrepareRemote are.
type heldRelay struct {
	inner *functionfs.Relay
	once  sync.Once
	done  chan struct{}
}

func (r *heldRelay) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return nil
	}
}

func (r *heldRelay) Close() error {
	r.once.Do(func() { close(r.done) })
	return r.inner.Close()
}

func (f *kernelHybridFactory) Cleanup() error { return functionfs.Cleanup() }

func vendorDescriptors() []byte {
	device := []byte{18, 1, 0x00, 0x02, 0, 0, 0, 64, 0x34, 0x12, 0x78, 0x56, 0, 1, 0, 0, 0, 1}
	config := []byte{9, 2, 0, 0, 1, 1, 0, 0x80, 50, 9, 4, 0, 0, 2, 0xff, 0, 0, 0}
	for _, address := range []uint8{0x02, 0x81} {
		endpoint := []byte{7, 5, address, 2, 0, 0, 0}
		binary.LittleEndian.PutUint16(endpoint[4:6], 512)
		config = append(config, endpoint...)
	}
	binary.LittleEndian.PutUint16(config[2:4], uint16(len(config)))
	return append(device, config...)
}

// The device runs a vendor 5.10 kernel whose f_hid carries wakeup_on_write and
// whose cvitek driver exposes /proc/cviusb/otg_role. Stock 6.8 has neither, and
// they are the only two writes in the Hybrid plan this kernel cannot take.
// Everything else, configfs included, is the real ConfigFSOps.
type vmOps struct {
	*presentation.ConfigFSOps
}

func (o vmOps) WriteFile(rel string, data []byte) error {
	if strings.HasSuffix(rel, "/wakeup_on_write") {
		return nil
	}
	return o.ConfigFSOps.WriteFile(rel, data)
}

func (vmOps) SetOTGRole(string) error { return nil }

func kernelPresentation(t *testing.T) *presentation.Manager {
	t.Helper()

	kernelint.BootstrapGadget(t, presentation.GadgetRoot)
	configfs, err := presentation.NewConfigFSOps(presentation.GadgetRoot)
	if err != nil {
		t.Fatal(err)
	}
	ops := vmOps{ConfigFSOps: configfs}
	t.Cleanup(func() { _ = configfs.Close() })

	instance := filepath.Join(presentation.GadgetRoot, "functions", "ffs.hybrid")
	if err := os.Remove(instance); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("clear %s: %v", instance, err)
	}
	t.Cleanup(func() {
		_ = functionfs.Cleanup()
		_ = os.Remove(instance)
	})
	return presentation.NewManager(presentation.NewStore(), ops, presentation.LoadCapabilities())
}

// Reverting "register ffs instance before mounting" leaves startHybrid calling
// Prepare with no ffs.hybrid in configfs, and the mount(2) inside it returns
// ENODEV. That is the whole reason this suite exists, so it is asserted against
// the real filesystem type rather than against a recorded call order.
func TestKernelTier2HybridRegistersTheFFSInstanceBeforeMounting(t *testing.T) {
	kernelint.RequireTier2(t)

	manager, _, _, _ := newTestManager(t)
	manager.gadget = kernelPresentation(t)
	manager.hybrid = &kernelHybridFactory{caps: presentation.LoadCapabilities()}

	session, err := manager.StartMode(context.Background(), "127.0.0.1", "1-1", ModeHybrid, false)
	if err != nil {
		t.Fatalf("start hybrid against the real kernel: %v", err)
	}
	t.Cleanup(func() { _ = manager.Stop() })

	if session.Mode != ModeHybrid {
		t.Fatalf("mode = %q", session.Mode)
	}
	for _, path := range []string{
		filepath.Join(presentation.GadgetRoot, "functions", "ffs.hybrid"),
		filepath.Join(presentation.GadgetRoot, "configs", "c.1", "ffs.hybrid"),
		"/dev/ffs-hybrid/ep0",
		"/dev/ffs-hybrid/ep01",
		"/dev/ffs-hybrid/ep81",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}

	bound, err := os.ReadFile(filepath.Join(presentation.GadgetRoot, "UDC"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(bound); len(got) == 0 || got[0] == '\n' {
		t.Fatalf("UDC = %q after a Hybrid start", got)
	}
}
