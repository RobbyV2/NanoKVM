package passthrough

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestProxyPIDsMatchesOnlyTheConfiguredBinary(t *testing.T) {
	root := t.TempDir()
	previousRoot, previousBinary := procRoot, proxyBinary
	procRoot, proxyBinary = root, "/etc/kvm/bin/usb-proxy"
	t.Cleanup(func() { procRoot, proxyBinary = previousRoot, previousBinary })

	processes := map[string][]byte{
		"101": []byte("/etc/kvm/bin/usb-proxy\x00--device\x004340000.usb\x00"),
		"102": []byte("/usr/bin/usb-proxy\x00"),
		"103": []byte("/etc/kvm/bin/usb-proxy-helper\x00"),
	}
	for pid, cmdline := range processes {
		dir := filepath.Join(root, pid)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), cmdline, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "uptime"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	pids, err := proxyPIDs()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(pids, []int{101}) {
		t.Fatalf("pids = %v, want [101]", pids)
	}
}
