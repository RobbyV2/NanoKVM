//go:build linux && kernelint

package functionfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"NanoKVM-Server/service/kernelint"
	"NanoKVM-Server/service/presentation"

	"golang.org/x/sys/unix"
)

const ffsInstance = "functions/ffs.hybrid"

// The mkdir under test is presentation.Manager.CreateFunctionFS, and the mount
// is openFunctionFS, so this asserts the contract between the two shipped
// halves rather than between two lines written here.
func kernelGadget(t *testing.T) *presentation.Manager {
	t.Helper()

	ops, err := presentation.NewConfigFSOps(presentation.GadgetRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ops.Close() })

	instance := filepath.Join(presentation.GadgetRoot, ffsInstance)
	if err := os.Remove(instance); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("clear %s: %v", instance, err)
	}
	if _, err := os.Stat(instance); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s survived removal: %v", instance, err)
	}
	t.Cleanup(func() {
		_ = Cleanup()
		_ = os.Remove(instance)
	})
	return presentation.NewManager(presentation.NewStore(), ops, presentation.LoadCapabilities())
}

func vendorImage(t *testing.T) Image {
	t.Helper()
	image, err := Import(vendorFixture(bulkEndpoint(0x02, 512), bulkEndpoint(0x81, 512)), fixtureFetcher{}, testCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	return image
}

func filesystemRegistered(t *testing.T, name string) bool {
	t.Helper()
	data, err := os.ReadFile("/proc/filesystems")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[len(fields)-1] == name {
			return true
		}
	}
	return false
}

// The relay's isochronous path stands on io_submit, and the arm64 container the
// normal suite runs in is not the kernel the device runs, so the same batch round
// trip is repeated against the VM kernel that carries the gadget stack.
func TestKernelTier2AIOBatchRoundTrip(t *testing.T) {
	kernelint.RequireTier2(t)
	aioBatchRoundTrip(t)
}

func TestKernelTier2FunctionFSMountRequiresTheConfigFSInstance(t *testing.T) {
	kernelint.RequireTier2(t)
	gadget := kernelGadget(t)
	image := vendorImage(t)

	if filesystemRegistered(t, "functionfs") {
		t.Fatal("functionfs is registered in /proc/filesystems with no ffs instance; a preflight that greps it would be reading stale state")
	}

	_, _, err := openFunctionFS(image)
	if !errors.Is(err, syscall.ENODEV) {
		t.Fatalf("mount before the configfs mkdir = %v, want ENODEV", err)
	}

	if err := gadget.CreateFunctionFS(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !filesystemRegistered(t, "functionfs") {
		t.Fatal("functionfs is still absent from /proc/filesystems after the configfs mkdir")
	}

	control, endpoints, err := openFunctionFS(image)
	if err != nil {
		t.Fatalf("mount after the configfs mkdir: %v", err)
	}

	// FUNCTIONFS_VIRTUAL_ADDR is set in descriptorBlock, so the kernel names the
	// endpoint files after the addresses the compiler assigned rather than 1..N.
	// Closing ep0 destroys them, so they are checked while it is still open.
	for _, name := range []string{"ep0", "ep01", "ep81"} {
		if _, err := os.Stat(filepath.Join(functionFSMount, name)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, endpoint := range endpoints {
		_ = endpoint.Close()
	}
	_ = control.Close()
	if err := Cleanup(); err != nil {
		t.Fatal(err)
	}
}

// f_fs parses the whole descriptor block on the ep0 write and creates one
// endpoint file per endpoint descriptor, named by address because
// FUNCTIONFS_VIRTUAL_ADDR is set. That is the parser that decides whether a
// two-alternate isochronous interface is presentable at all, and it runs long
// before any endpoint is autoconfigured, so the answer does not depend on the
// UDC having an isochronous endpoint to give.
func TestKernelTier2FunctionFSAcceptsTwoAlternatesOverOneEndpointFile(t *testing.T) {
	kernelint.RequireTier2(t)
	gadget := kernelGadget(t)
	if err := gadget.CreateFunctionFS(context.Background()); err != nil {
		t.Fatal(err)
	}

	image, err := Import(streamingFixture(isoEndpoint(0x81, 192, 0), isoEndpoint(0x81, 768, 0)), fixtureFetcher{}, testCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	control, endpoints, err := openFunctionFS(image)
	if err != nil {
		t.Fatalf("write a two-alternate isochronous descriptor block: %v", err)
	}
	entries, err := os.ReadDir(functionFSMount)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"ep0", "ep81"}) {
		t.Fatalf("FunctionFS created %v, want exactly ep0 and ep81: a second endpoint descriptor at 0x81 would collide on the name", names)
	}
	for _, endpoint := range endpoints {
		_ = endpoint.Close()
	}
	_ = control.Close()
	if err := Cleanup(); err != nil {
		t.Fatal(err)
	}
}

// ENODEV says the instance was never registered; ENOENT says it was registered
// under another name. Collapsing the two loses the only signal that separates a
// missing mkdir from a misspelling.
func TestKernelTier2FunctionFSWrongInstanceNameIsENOENT(t *testing.T) {
	kernelint.RequireTier2(t)
	gadget := kernelGadget(t)

	if err := gadget.CreateFunctionFS(context.Background()); err != nil {
		t.Fatal(err)
	}

	target := t.TempDir()
	err := unix.Mount("nosuchinstance", target, "functionfs", unix.MS_NOSUID|unix.MS_NODEV, "")
	if !errors.Is(err, syscall.ENOENT) {
		if err == nil {
			_ = unix.Unmount(target, 0)
		}
		t.Fatalf("mount of an unregistered instance name = %v, want ENOENT", err)
	}
}
