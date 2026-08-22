package bridge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

func TestUplinkFileFallsBackToEth0(t *testing.T) {
	newHarness(t)

	// The stock device: no file at all. This is the state a disabled device is
	// returned to, so that it is byte-identical to one that never bridged.
	if got := ReadUplink(); got != StockUplink {
		t.Fatalf("ReadUplink with no file = %q, want eth0", got)
	}

	if err := WriteUplink(BridgeName); err != nil {
		t.Fatalf("WriteUplink: %v", err)
	}
	if got := ReadUplink(); got != BridgeName {
		t.Fatalf("ReadUplink = %q, want br0", got)
	}

	// The file other components read is not 0600: the init scripts and the C++
	// OLED daemon both have to read it.
	info, err := os.Stat(uplinkPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("l2-uplink mode = %v, want 0644", info.Mode().Perm())
	}

	if err := RemoveUplink(); err != nil {
		t.Fatalf("RemoveUplink: %v", err)
	}
	if got := ReadUplink(); got != StockUplink {
		t.Fatalf("ReadUplink after removal = %q, want eth0", got)
	}

	// Removing a file that is not there is the state the call guarantees, not
	// an error: a disable on a device that never enabled has to succeed.
	if err := RemoveUplink(); err != nil {
		t.Fatalf("RemoveUplink of an absent file: %v", err)
	}
}

func TestUplinkFileRejectsAnythingElse(t *testing.T) {
	newHarness(t)

	for _, name := range []string{RecoveryName, "eth1", "br1", "", "br0 wlan0", "../../etc/passwd"} {
		if err := WriteUplink(name); err == nil {
			t.Fatalf("WriteUplink(%q) was allowed", name)
		}
	}
}

// A file that has been corrupted must not turn into an argv. Everything that
// does not look like an interface name falls back to eth0, which is the state
// the rest of the system is built to handle.
func TestReadUplinkIgnoresJunk(t *testing.T) {
	newHarness(t)

	for _, content := range []string{
		"",
		"\n",
		"   \n",
		"br0 && reboot\n",
		"$(reboot)\n",
		"../../etc/passwd\n",
		"averyverylonginterfacename\n",
	} {
		writeFile(t, uplinkPath, content)
		if got := ReadUplink(); got != StockUplink {
			t.Fatalf("ReadUplink(%q) = %q, want the eth0 fallback", content, got)
		}
	}

	// A legible name still round-trips, so the guard is not simply rejecting
	// everything.
	writeFile(t, uplinkPath, "br0\n")
	if got := ReadUplink(); got != BridgeName {
		t.Fatalf("ReadUplink of a valid file = %q, want br0", got)
	}
}

func TestGatewayFileValidatesTheAddress(t *testing.T) {
	newHarness(t)

	if err := WriteGateway("192.168.1.1"); err != nil {
		t.Fatalf("WriteGateway: %v", err)
	}
	got, ok := ReadGateway()
	if !ok || got != "192.168.1.1\n" {
		t.Fatalf("ReadGateway = %q, %v", got, ok)
	}

	// system_state.cpp reads this file verbatim, so nothing that is not an IPv4
	// address is allowed to reach it.
	for _, bad := range []string{"", "not-an-ip", "fe80::1", "192.168.1.1; reboot", "1.2.3.4/24"} {
		if err := WriteGateway(bad); err == nil {
			t.Fatalf("WriteGateway(%q) was allowed", bad)
		}
	}
	if got, _ := ReadGateway(); got != "192.168.1.1\n" {
		t.Fatalf("a rejected write changed the file to %q", got)
	}
}

func TestSafeDeviceRejectsArgvInjection(t *testing.T) {
	for _, good := range []string{"eth0", "br0", "usb0", "wlan0", "lo", "enp0s25"} {
		if !safeDevice(good) {
			t.Errorf("safeDevice(%q) = false", good)
		}
	}
	for _, bad := range []string{
		"", " ", "-i", "--help", "eth0 up", "eth0\nlink", "eth0;reboot", "$(id)",
		"../eth0", "averyverylonginterfacename",
	} {
		if safeDevice(bad) {
			t.Errorf("safeDevice(%q) = true", bad)
		}
	}
}

func TestPendingRoundTrip(t *testing.T) {
	h := newHarness(t)

	if pending, err := h.store.Pending(); err != nil || pending != nil {
		t.Fatalf("Pending with none armed = %v, %v", pending, err)
	}

	armed := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	want := Pending{
		Operation:    operationEnable,
		SnapshotPath: h.store.SnapshotPath(),
		ArmedAt:      armed,
		Deadline:     armed.Add(DefaultWindow),
	}
	if err := h.store.Arm(want); err != nil {
		t.Fatalf("Arm: %v", err)
	}

	got, err := h.store.Pending()
	if err != nil || got == nil {
		t.Fatalf("Pending = %v, %v", got, err)
	}
	if *got != want {
		t.Fatalf("round trip = %+v, want %+v", *got, want)
	}

	// The path has to survive a reboot, so it is absolute.
	if !filepath.IsAbs(got.SnapshotPath) {
		t.Fatalf("snapshot path %q is not absolute", got.SnapshotPath)
	}

	if err := h.store.Disarm(); err != nil {
		t.Fatalf("Disarm: %v", err)
	}
	if pending, _ := h.store.Pending(); pending != nil {
		t.Fatal("the marker survived Disarm")
	}
	if err := h.store.Disarm(); err != nil {
		t.Fatalf("Disarm with none armed: %v", err)
	}
}

// A marker that cannot be parsed is an error and not a nil, because treating it
// as "nothing armed" is the one interpretation that skips a restore, and a
// device that skips its restore is a device somebody has to walk over to.
func TestCorruptPendingIsAnError(t *testing.T) {
	h := newHarness(t)
	writeFile(t, filepath.Join(h.store.Dir(), pendingName), "{ truncated")

	if _, err := h.store.Pending(); err == nil {
		t.Fatal("a corrupt marker decoded as no marker")
	}
}

func TestStoreFileModes(t *testing.T) {
	h := newHarness(t)

	if err := h.store.Arm(Pending{SnapshotPath: "/x", ArmedAt: h.clock}); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	if err := h.store.WriteLastKnownGood(LastKnownGood{Uplink: StockUplink}); err != nil {
		t.Fatalf("WriteLastKnownGood: %v", err)
	}
	if _, err := h.store.WriteSnapshot(&Snapshot{}); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	for _, name := range []string{pendingName, lastKnownGoodName, snapshotName} {
		info, err := os.Stat(filepath.Join(h.store.Dir(), name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s mode = %v, want 0600", name, info.Mode().Perm())
		}
	}

	info, err := os.Stat(h.store.Dir())
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("store dir mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestLastKnownGoodRoundTrip(t *testing.T) {
	h := newHarness(t)

	if lkg, err := h.store.LastKnownGood(); err != nil || lkg != nil {
		t.Fatalf("LastKnownGood on a fresh device = %v, %v", lkg, err)
	}

	want := LastKnownGood{
		Enabled:   true,
		Uplink:    BridgeName,
		State:     proto.BridgeEnabled,
		Checks:    proto.BridgeChecks{Address: true, Gateway: true, Inbound: true},
		Message:   "",
		AppliedAt: time.Date(2026, 8, 22, 12, 1, 0, 0, time.UTC),
	}
	if err := h.store.WriteLastKnownGood(want); err != nil {
		t.Fatalf("WriteLastKnownGood: %v", err)
	}

	got, err := h.store.LastKnownGood()
	if err != nil || got == nil {
		t.Fatalf("LastKnownGood = %v, %v", got, err)
	}
	if *got != want {
		t.Fatalf("round trip = %+v, want %+v", *got, want)
	}
}

// Commit is the step-12 sequence. The outcome has to be durable before the
// marker goes, because the caller's connection may not survive the apply and
// GET is where it reads the result back.
func TestCommitPublishesTheOutcomeAndClearsTheMarker(t *testing.T) {
	h := newHarness(t)

	if err := h.store.Arm(Pending{SnapshotPath: h.store.SnapshotPath(), ArmedAt: h.clock}); err != nil {
		t.Fatalf("Arm: %v", err)
	}

	if err := h.store.Commit(LastKnownGood{
		Enabled: true, Uplink: BridgeName, State: proto.BridgeEnabled,
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	lkg, err := h.store.LastKnownGood()
	if err != nil || lkg == nil || !lkg.Enabled {
		t.Fatalf("LastKnownGood = %v, %v", lkg, err)
	}
	if pending, _ := h.store.Pending(); pending != nil {
		t.Fatal("the marker survived Commit")
	}
}

// The window a crash inside Commit leaves behind: an outcome that says enabled
// with a marker still armed. The boot check has to act on the marker, because
// the outcome was written by a process that never proved anything.
func TestAnArmedMarkerOutranksTheRecordedOutcome(t *testing.T) {
	h := newHarness(t)

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	path, err := h.store.WriteSnapshot(snapshot)
	if err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	if err := h.store.WriteLastKnownGood(LastKnownGood{
		Enabled: true, Uplink: BridgeName, State: proto.BridgeEnabled,
	}); err != nil {
		t.Fatalf("WriteLastKnownGood: %v", err)
	}
	if err := h.store.Arm(Pending{
		Operation: operationEnable, SnapshotPath: path,
		ArmedAt: h.clock, Deadline: h.clock.Add(DefaultWindow),
	}); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	if err := WriteUplink(BridgeName); err != nil {
		t.Fatalf("WriteUplink: %v", err)
	}

	recovered, err := h.mgr.RecoverPending(context.Background())
	if err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}
	if !recovered {
		t.Fatal("the boot check trusted the recorded outcome over the armed marker")
	}

	lkg, _ := h.store.LastKnownGood()
	if lkg == nil || lkg.Enabled || lkg.State != proto.BridgeRolledBack {
		t.Fatalf("outcome after recovery = %+v, want a rolled-back disable", lkg)
	}
	if got := ReadUplink(); got != StockUplink {
		t.Fatalf("uplink = %q after recovery, want eth0", got)
	}
}

// Status reports the marker while one is armed, so a client whose connection
// the apply cut can tell "in flight" from "finished".
func TestStatusReportsAnArmedMarker(t *testing.T) {
	h := newHarness(t)

	if err := h.store.Arm(Pending{
		Operation: operationEnable, SnapshotPath: h.store.SnapshotPath(),
		ArmedAt: h.clock, Deadline: h.clock.Add(DefaultWindow),
	}); err != nil {
		t.Fatalf("Arm: %v", err)
	}

	status, err := h.mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.State != proto.BridgePending {
		t.Fatalf("state = %q, want pending", status.State)
	}
	if status.Pending == nil || status.Pending.Operation != operationEnable {
		t.Fatalf("pending = %+v", status.Pending)
	}
	if !status.Pending.Deadline.Equal(h.clock.Add(DefaultWindow)) {
		t.Fatalf("deadline = %s", status.Pending.Deadline)
	}
}

func TestRestoreFileRemovesWhereTheCaptureHadNone(t *testing.T) {
	h := newHarness(t)
	path := filepath.Join(h.root, "etc/kvm/gateway")
	writeFile(t, path, "10.0.0.1\n")

	if err := restoreFile(path, "", false, 0o644); err != nil {
		t.Fatalf("restoreFile: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("restoreFile left a file the capture shows was absent")
	}

	// Restoring an empty file that did exist is not the same thing.
	if err := restoreFile(path, "", true, 0o644); err != nil {
		t.Fatalf("restoreFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("restoreFile did not create an empty file that existed: %v", err)
	}
}
