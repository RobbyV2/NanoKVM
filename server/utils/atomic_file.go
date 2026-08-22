package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicFile replaces a file by writing a sibling temporary and renaming it
// over the destination, so a reader either sees the whole old file or the
// whole new one and a crash mid-write never leaves a truncated file behind.
//
// Every step below is load-bearing:
//   - the temporary is created in the destination's own directory, because
//     rename(2) is atomic only within a single filesystem;
//   - the mode is set on the temporary before any content is written, so the
//     content is never briefly readable under the umask's default mode;
//   - Sync is called and its error is propagated, because a rename that beats
//     the data to disk is exactly the torn write this type exists to prevent;
//   - the destination is chmod'd again after the rename, which costs nothing
//     and closes the window where an existing destination was replaced by a
//     temporary someone else had already tampered with;
//   - the directory is synced last, so the rename itself survives a crash.
//
// Use it as a writer when the content is streamed:
//
//	file, err := utils.NewAtomicFile(path, 0o600)
//	if err != nil {
//	    return err
//	}
//	defer file.Discard()
//	if _, err := io.Copy(file, source); err != nil {
//	    return err
//	}
//	return file.Commit()
//
// Discard after Commit is a no-op, so the deferred call is always correct.
// For a plain byte slice, WriteFileAtomic wraps the whole sequence.
type AtomicFile struct {
	file   *os.File
	tmp    string
	dest   string
	mode   os.FileMode
	closed bool
}

// NewAtomicFile creates the destination's directory if needed and opens a
// temporary file beside it, already carrying mode.
func NewAtomicFile(dest string, mode os.FileMode) (*AtomicFile, error) {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", dir, err)
	}

	file, err := os.CreateTemp(dir, "."+filepath.Base(dest)+".*")
	if err != nil {
		return nil, fmt.Errorf("create temporary file in %s: %w", dir, err)
	}
	if err := file.Chmod(mode); err != nil {
		tmp := file.Name()
		_ = file.Close()
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("set permissions on %s: %w", tmp, err)
	}

	return &AtomicFile{file: file, tmp: file.Name(), dest: dest, mode: mode}, nil
}

// Write appends to the temporary file. AtomicFile is an io.Writer.
func (a *AtomicFile) Write(p []byte) (int, error) {
	return a.file.Write(p)
}

// Path is the temporary file, so content can be inspected after Flush and
// before Commit publishes it.
func (a *AtomicFile) Path() string {
	return a.tmp
}

// Flush syncs and closes the temporary without publishing it. Commit calls it,
// so it is only needed when the temporary has to be examined first.
func (a *AtomicFile) Flush() error {
	if a.closed {
		return nil
	}
	a.closed = true

	if err := a.file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", a.tmp, err)
	}
	if err := a.file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", a.tmp, err)
	}
	return nil
}

// Commit flushes the temporary and renames it over the destination.
func (a *AtomicFile) Commit() error {
	if err := a.Flush(); err != nil {
		return err
	}
	if err := os.Rename(a.tmp, a.dest); err != nil {
		return fmt.Errorf("replace %s: %w", a.dest, err)
	}
	if err := os.Chmod(a.dest, a.mode); err != nil {
		return fmt.Errorf("set permissions on %s: %w", a.dest, err)
	}
	return SyncDir(filepath.Dir(a.dest))
}

// Discard closes and removes the temporary. It is safe to call after Commit,
// where the rename has already consumed the temporary and the remove fails
// harmlessly.
func (a *AtomicFile) Discard() {
	if !a.closed {
		a.closed = true
		_ = a.file.Close()
	}
	_ = os.Remove(a.tmp)
}

// WriteFileAtomic replaces path with data through the sequence AtomicFile
// documents.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	file, err := NewAtomicFile(path, mode)
	if err != nil {
		return err
	}
	defer file.Discard()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return file.Commit()
}

// SyncDir flushes a directory's own entries so a rename into it survives a
// crash. A directory that cannot be opened for reading is not an error: the
// rename already succeeded, and nothing further can be done about it here.
func SyncDir(dir string) error {
	directory, err := os.Open(dir)
	if err != nil {
		return nil
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("sync directory %s: %w", dir, err)
	}
	return directory.Close()
}
