package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicReplacesContentAndMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte("a much longer previous value\n"), 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	if err := WriteFileAtomic(path, []byte("new\n"), 0o600); err != nil {
		t.Fatalf("write atomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "new\n" {
		t.Fatalf("content = %q, want %q", got, "new\n")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}

	// The temporary is gone, and it never lived anywhere but this directory:
	// a rename across filesystems is not atomic.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.json" {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("directory holds %v, want only config.json", names)
	}
}

// The mode has to land on the temporary before any content does, or the
// content is briefly readable under the umask's default mode.
func TestNewAtomicFileAppliesTheModeBeforeAnyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")

	file, err := NewAtomicFile(path, 0o600)
	if err != nil {
		t.Fatalf("new atomic file: %v", err)
	}
	defer file.Discard()

	info, err := os.Stat(file.Path())
	if err != nil {
		t.Fatalf("stat temporary: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("temporary permissions = %o, want 600", info.Mode().Perm())
	}
	if info.Size() != 0 {
		t.Fatalf("temporary holds %d bytes before any write", info.Size())
	}
	if got := filepath.Dir(file.Path()); got != dir {
		t.Fatalf("temporary lives in %s, want %s", got, dir)
	}
	if base := filepath.Base(file.Path()); !strings.HasPrefix(base, ".secret.") {
		t.Fatalf("temporary named %q, want a hidden sibling of secret", base)
	}
}

// An abandoned write must leave both the destination and the directory as they
// were, which is what the deferred Discard buys every caller.
func TestDiscardLeavesTheDestinationUntouched(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatalf("seed destination: %v", err)
	}

	file, err := NewAtomicFile(path, 0o600)
	if err != nil {
		t.Fatalf("new atomic file: %v", err)
	}
	if _, err := file.Write([]byte("half a wri")); err != nil {
		t.Fatalf("write: %v", err)
	}
	file.Discard()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "old\n" {
		t.Fatalf("content = %q, want the untouched %q", got, "old\n")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("discard left %d entries behind", len(entries))
	}
}

// Flush publishes nothing: it exists so a caller can inspect the temporary,
// as the tunnel binary upload does when it verifies the ELF header, and only
// then commit.
func TestFlushThenCommitPublishesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "binary")

	file, err := NewAtomicFile(path, 0o755)
	if err != nil {
		t.Fatalf("new atomic file: %v", err)
	}
	defer file.Discard()

	if _, err := file.Write([]byte("payload")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := file.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("flush published the destination: err = %v", err)
	}

	staged, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("read flushed temporary: %v", err)
	}
	if string(staged) != "payload" {
		t.Fatalf("temporary = %q, want %q", staged, "payload")
	}

	if err := file.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("permissions = %o, want 755", info.Mode().Perm())
	}
}
