package edid

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func useTestStore(t *testing.T) *Store {
	t.Helper()
	swapString(t, &storeDir, t.TempDir())
	return NewStore()
}

// variant returns a valid EDID that differs from the fixture, so an archive
// test can tell two generations of bytes apart.
func variant(t *testing.T, serial byte) []byte {
	t.Helper()

	blob := repaired(func() []byte {
		out := bytes.Clone(fixture(t))
		out[12] = serial
		return out
	}())

	if _, err := Decode(blob); err != nil {
		t.Fatalf("variant %d does not decode: %v", serial, err)
	}
	return blob
}

func decoded(t *testing.T, blob []byte) *EDID {
	t.Helper()
	e, err := Decode(blob)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return e
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s has mode %o, want %o", path, got, want)
	}
}

func TestLoadActiveOnAnEmptyStore(t *testing.T) {
	store := useTestStore(t)

	data, record, err := store.LoadActive()
	if err != nil {
		t.Fatalf("load active: %v", err)
	}
	if data != nil || record != nil {
		t.Fatalf("a device that has never been flashed reported %v / %+v", data, record)
	}
}

func TestArchiveWritesEveryFileAt0600(t *testing.T) {
	store := useTestStore(t)
	blob := fixture(t)

	record, err := store.Archive(blob, "profile:abc", decoded(t, blob))
	if err != nil {
		t.Fatalf("archive: %v", err)
	}

	if record.SHA256 != digest(blob) {
		t.Fatalf("record sha %s, want %s", record.SHA256, digest(blob))
	}
	if record.Manufacturer != "SPD" || record.PreferredMode != "1920x1080p60" {
		t.Fatalf("record %+v, want the decoded summary", record)
	}

	assertMode(t, store.activePath(), 0o600)
	assertMode(t, store.recordPath(), 0o600)

	info, err := os.Stat(store.historyDir())
	if err != nil {
		t.Fatalf("stat history: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("history directory has mode %o, want 0755", got)
	}
}

// A newer write verifying says nothing about whether the operator wants the
// older EDID back, so the previous bytes move into history rather than being
// overwritten.
func TestArchiveMovesThePreviousActiveIntoHistory(t *testing.T) {
	store := useTestStore(t)

	first := variant(t, 0x11)
	if _, err := store.Archive(first, "upload", decoded(t, first)); err != nil {
		t.Fatalf("archive first: %v", err)
	}

	// Distinct timestamps come from the record, so the second entry cannot
	// collide with the first inside the same second.
	backdateRecord(t, store, time.Now().Add(-time.Hour))

	second := variant(t, 0x22)
	if _, err := store.Archive(second, "factory", decoded(t, second)); err != nil {
		t.Fatalf("archive second: %v", err)
	}

	active, record, err := store.LoadActive()
	if err != nil {
		t.Fatalf("load active: %v", err)
	}
	if !bytes.Equal(active, second) {
		t.Fatal("the active file does not hold the newest bytes")
	}
	if record.Source != "factory" {
		t.Fatalf("record source %q, want factory", record.Source)
	}

	backups, err := store.History()
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("%d history entries, want 1", len(backups))
	}
	if backups[0].SHA256 != digest(first) {
		t.Fatal("history does not hold the bytes that were replaced")
	}
	assertMode(t, filepath.Join(store.historyDir(), backups[0].ID+".bin"), 0o600)

	restored, err := store.ReadBackup(backups[0].ID)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if !bytes.Equal(restored, first) {
		t.Fatal("the archived bytes came back changed")
	}
}

// Nothing is ever pruned: a third write leaves both earlier generations in
// place, because an archived file is the only rollback target that exists.
func TestArchiveNeverPrunes(t *testing.T) {
	store := useTestStore(t)

	for i, serial := range []byte{0x11, 0x22, 0x33} {
		blob := variant(t, serial)
		if _, err := store.Archive(blob, "upload", decoded(t, blob)); err != nil {
			t.Fatalf("archive %d: %v", i, err)
		}
		backdateRecord(t, store, time.Now().Add(-time.Duration(3-i)*time.Hour))
	}

	backups, err := store.History()
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("%d history entries, want 2", len(backups))
	}
	if backups[0].ID <= backups[1].ID {
		t.Fatal("history is not newest first")
	}
}

// backdateRecord rewrites last-applied.json's timestamp so the next archive
// files the outgoing entry under a distinct id.
func backdateRecord(t *testing.T, store *Store, at time.Time) {
	t.Helper()

	_, record, err := store.LoadActive()
	if err != nil || record == nil {
		t.Fatalf("load active: %v", err)
	}
	record.AppliedAt = at.UTC()

	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatalf("encode record: %v", err)
	}
	if err := os.WriteFile(store.recordPath(), encoded, 0o600); err != nil {
		t.Fatalf("write record: %v", err)
	}
}

func TestReadBackupRefusesTraversal(t *testing.T) {
	store := useTestStore(t)

	for _, id := range []string{"../last-applied", "", "not-an-id", "20260822T101500Z-ZZZZZZZZ"} {
		if _, err := store.ReadBackup(id); !errors.Is(err, ErrBackupNotFound) {
			t.Fatalf("ReadBackup(%q) error %v, want %v", id, err, ErrBackupNotFound)
		}
	}
}

func TestLockIsExclusiveAndBreaksAStaleHolder(t *testing.T) {
	store := useTestStore(t)

	unlock, err := store.Lock()
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	assertMode(t, filepath.Join(store.Dir(), lockName), 0o600)

	if _, err := store.Lock(); !errors.Is(err, ErrLocked) {
		t.Fatalf("second lock error %v, want %v", err, ErrLocked)
	}

	unlock()
	unlock() // releasing twice is harmless

	again, err := store.Lock()
	if err != nil {
		t.Fatalf("relock: %v", err)
	}
	again()

	// A lockfile left behind by a process that no longer exists is broken once.
	if err := os.WriteFile(filepath.Join(store.Dir(), lockName), []byte("424242\n"), 0o600); err != nil {
		t.Fatalf("plant stale lock: %v", err)
	}
	old := processAlive
	processAlive = func(int) bool { return false }
	t.Cleanup(func() { processAlive = old })

	unlock, err = store.Lock()
	if err != nil {
		t.Fatalf("stale lock was not broken: %v", err)
	}
	unlock()
}

func TestArchiveRejectsAnythingThatIsNot256Bytes(t *testing.T) {
	store := useTestStore(t)

	if _, err := store.Archive(make([]byte, BlockSize), "upload", nil); err == nil {
		t.Fatal("a 128 byte archive was accepted")
	}
	if _, _, err := store.LoadActive(); err != nil {
		t.Fatalf("load active: %v", err)
	}
	if _, err := os.Stat(store.activePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a rejected archive still wrote the active file")
	}
}
