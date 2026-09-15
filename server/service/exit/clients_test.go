package exit

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The native clients in clients/ are templated by commands.go and must carry
// every placeholder; they also mirror the destination policy (D21) by hand, so
// the prefixes in policy.go must appear verbatim in each of them.

var clientMarkers = map[string]string{
	"client.sh":  "nexit/1 sh bootstrap",
	"client.py":  "nexit/1 python client",
	"client.pl":  "nexit/1 perl client",
	"client.ps1": "nexit/1 powershell client",
}

func readClient(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("clients", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func TestClientsCarryPlaceholdersAndMarkers(t *testing.T) {
	for name, marker := range clientMarkers {
		text := readClient(t, name)
		for _, ph := range []string{"__SCHEME__", "__HOST__", "__SLOT__", "__TOKEN__", "__FINGERPRINT__"} {
			if !strings.Contains(text, ph) {
				t.Errorf("%s: missing placeholder %s", name, ph)
			}
		}
		if name != "client.sh" && !strings.Contains(text, "__ALLOW_PRIVATE__") {
			t.Errorf("%s: missing placeholder __ALLOW_PRIVATE__", name)
		}
		head := strings.SplitN(text, "\n", 4)
		if len(head) < 3 || !strings.Contains(strings.Join(head[:3], "\n"), marker) {
			t.Errorf("%s: marker %q not in the first three lines (client.sh checks it after the fetch)", name, marker)
		}
		if strings.Contains(text, "\r") {
			t.Errorf("%s: contains CR; the scripts are served as-is and run by sh/perl/python", name)
		}
		if !strings.HasSuffix(text, "\n") {
			t.Errorf("%s: no trailing newline", name)
		}
	}
}

func TestClientsMirrorPolicyPrefixes(t *testing.T) {
	var all []netip.Prefix
	all = append(all, alwaysDenied4...)
	all = append(all, privateDenied4...)
	all = append(all, alwaysDenied6...)
	all = append(all, privateDenied6...)
	for _, name := range []string{"client.py", "client.pl", "client.ps1"} {
		text := readClient(t, name)
		for _, pfx := range all {
			if !strings.Contains(text, pfx.String()) {
				t.Errorf("%s: policy prefix %s from policy.go is not mirrored", name, pfx)
			}
		}
	}
}
