package audit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
)

func useTempTrail(t *testing.T) string {
	t.Helper()
	original := path
	path = filepath.Join(t.TempDir(), "audit.jsonl")
	t.Cleanup(func() { path = original })
	return path
}

func read(t *testing.T, name string) []entry {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var entries []entry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var record entry
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("line %q: %s", line, err)
		}
		entries = append(entries, record)
	}
	return entries
}

func TestRecordNamesTheActorAndTheOutcome(t *testing.T) {
	trail := useTempTrail(t)
	alice := middleware.Principal{Username: "alice", Role: authn.RoleAdmin}

	Record(alice, "presentation.apply", "hid-only", nil)
	Record(alice, "edid.flash", strings.Repeat("x", maxFieldLen+50), errors.New("needs_recovery: mismatch"))

	entries := read(t, trail)
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Actor != "alice" || entries[0].Unauthenticated {
		t.Fatalf("actor = %q unauthenticated = %v", entries[0].Actor, entries[0].Unauthenticated)
	}
	if entries[0].Action != "presentation.apply" || entries[0].Target != "hid-only" || !entries[0].OK {
		t.Fatalf("apply record = %+v", entries[0])
	}
	if entries[0].Time == "" {
		t.Fatal("record carries no time")
	}
	if entries[1].OK || entries[1].Error != "needs_recovery: mismatch" {
		t.Fatalf("flash record = %+v", entries[1])
	}
	if len(entries[1].Target) != maxFieldLen {
		t.Fatalf("target length = %d, want %d", len(entries[1].Target), maxFieldLen)
	}
}

func TestDisabledAuthenticationIsNotAttributedToAdmin(t *testing.T) {
	trail := useTempTrail(t)

	Record(middleware.Principal{Username: "admin", Role: authn.RoleAdmin, Unauthenticated: true},
		"bridge.enable", "enabled", nil)
	Record(middleware.Principal{}, "bridge.revert", "", nil)

	for index, record := range read(t, trail) {
		if record.Actor != "" || !record.Unauthenticated {
			t.Fatalf("entry %d = %+v, want no actor and unauthenticated", index, record)
		}
	}
}

func TestTrailCannotOutgrowTwoFiles(t *testing.T) {
	trail := useTempTrail(t)
	admin := middleware.Principal{Username: "admin", Role: authn.RoleAdmin}
	target := strings.Repeat("p", maxFieldLen)

	for i := 0; i < 2000; i++ {
		Record(admin, "presentation.apply", target, nil)
	}
	Record(admin, "presentation.apply", "last-one", nil)

	names, err := filepath.Glob(trail + "*")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("trail files = %v, want the live file and one rotation", names)
	}
	var total int64
	for _, name := range names {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > maxFileBytes {
			t.Fatalf("%s is %d bytes, over the %d bound", name, info.Size(), maxFileBytes)
		}
		total += info.Size()
	}
	if total > 2*maxFileBytes {
		t.Fatalf("trail is %d bytes, over the %d bound", total, 2*maxFileBytes)
	}

	entries := read(t, trail)
	if entries[len(entries)-1].Target != "last-one" {
		t.Fatalf("newest record = %+v", entries[len(entries)-1])
	}
}
