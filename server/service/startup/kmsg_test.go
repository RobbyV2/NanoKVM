package startup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKmsgWritesOneInfoRecordPerLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kmsg")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	previous := kmsgPath
	kmsgPath = path
	t.Cleanup(func() { kmsgPath = previous })

	Kmsg("shutdown: on %s", "interrupt")
	Kmsg("shutdown: %s", Result{Name: "usb gadget", Elapsed: 3_000_000})
	Kmsg("two\nlines")
	Kmsg("%s", strings.Repeat("x", 2*kmsgLineMax))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	want := []string{
		"<6>NanoKVM-Server: shutdown: on interrupt",
		"<6>NanoKVM-Server: shutdown: usb gadget done in 3ms",
		"<6>NanoKVM-Server: two lines",
		"<6>NanoKVM-Server: " + strings.Repeat("x", kmsgLineMax),
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d records, want %d: %q", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("record %d: got %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestKmsgIsSilentWithoutTheDevice(t *testing.T) {
	previous := kmsgPath
	kmsgPath = filepath.Join(t.TempDir(), "absent")
	t.Cleanup(func() { kmsgPath = previous })
	Kmsg("shutdown: %s", "nothing to see")
}
