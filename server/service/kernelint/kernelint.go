// Package kernelint gates the //go:build kernelint tests, which mutate real
// kernel objects. Every check fails the test rather than skipping it: a silent
// skip is how a fake that never touched a kernel stayed green for weeks.
//
// Tier 1 needs only a private network namespace and vhci_hcd, so it runs on an
// ordinary Linux CI runner. Tier 2 needs a bound UDC and the gadget function
// modules, which no GitHub-hosted runner has; scripts/kernelint.sh boots a VM.
package kernelint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	ConfigFSRoot = "/sys/kernel/config"
	DummyUDC     = "dummy_udc.0"
	VHCIRoot     = "/sys/devices/platform/vhci_hcd.0"
)

func RequireTier1(t *testing.T) {
	t.Helper()
	requireRoot(t)
	requirePrivateNetNS(t)
}

func RequireTier1VHCI(t *testing.T) {
	t.Helper()
	requireRoot(t)
	requirePath(t, VHCIRoot, "modprobe vhci-hcd")
}

func RequireTier2(t *testing.T) {
	t.Helper()
	requireRoot(t)
	requirePath(t, ConfigFSRoot+"/usb_gadget", "mount -t configfs none "+ConfigFSRoot)
	requirePath(t, "/sys/class/udc/"+DummyUDC, "modprobe dummy_hcd, which Ubuntu does not ship; see scripts/kernelint.sh")
	for _, module := range []string{"libcomposite", "usb_f_fs", "usb_f_hid"} {
		requireModule(t, module)
	}
}

// The device boots with S03usbdev having already built and bound g0, and
// presentation.applyPlan relies on it: its first step is an unbind, and writing
// an empty UDC on a gadget that was never bound is ENODEV. This is that boot
// script, reduced to one function that costs no /dev/hidgN minor.
func BootstrapGadget(t *testing.T, root string) {
	t.Helper()
	RequireTier2(t)

	for _, dir := range []string{"", "strings/0x409", "configs/c.1", "configs/c.1/strings/0x409", "functions/ncm.boot"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for path, value := range map[string]string{
		"idVendor":                                "0x3346",
		"idProduct":                               "0x1009",
		"strings/0x409/manufacturer":              "sipeed",
		"strings/0x409/product":                   "NanoKVM",
		"strings/0x409/serialnumber":              "0123456789ABCDEF",
		"configs/c.1/strings/0x409/configuration": "NanoKVM",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(value+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	link := filepath.Join(root, "configs/c.1/ncm.boot")
	if err := os.Symlink(filepath.Join(root, "functions/ncm.boot"), link); err != nil && !os.IsExist(err) {
		t.Fatalf("link ncm.boot: %v", err)
	}
	udc := filepath.Join(root, "UDC")
	if bound, err := os.ReadFile(udc); err == nil && strings.TrimSpace(string(bound)) != "" {
		return
	}
	if err := os.WriteFile(udc, []byte(DummyUDC+"\n"), 0o644); err != nil {
		t.Fatalf("bind %s: %v", DummyUDC, err)
	}
}

func requireRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Fatal("kernelint needs root")
	}
}

func requirePath(t *testing.T, path, remedy string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s is absent: %v; %s", path, err, remedy)
	}
}

func requireModule(t *testing.T, name string) {
	t.Helper()
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		t.Fatalf("read /proc/modules: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if loaded, _, _ := strings.Cut(line, " "); loaded == name {
			return
		}
	}
	t.Fatalf("module %s is not loaded", name)
}

func requirePrivateNetNS(t *testing.T) {
	t.Helper()
	self, err := os.Readlink("/proc/self/ns/net")
	if err != nil {
		t.Fatalf("read own network namespace: %v", err)
	}
	initial, err := os.Readlink("/proc/1/ns/net")
	if err != nil {
		t.Fatalf("read init network namespace: %v", err)
	}
	if self == initial {
		t.Fatalf("running in the initial network namespace %s; these tests add and delete links and must run under ip netns exec", self)
	}
}
