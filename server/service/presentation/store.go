package presentation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	PresentationDir = "/etc/kvm/presentation"

	activeFile        = "active"
	lastKnownGoodFile = ".last-known-good"
)

var (
	configMu        sync.Mutex
	presentationDir = PresentationDir
)

type Store struct {
	dir string
}

func NewStore() *Store {
	return &Store{dir: presentationDir}
}

func (s *Store) LoadProfile(name string) (Profile, error) {
	if profile, ok := builtinByName(name); ok {
		return profile, nil
	}

	path, err := s.profilePath(name)
	if err != nil {
		return Profile{}, err
	}

	configMu.Lock()
	defer configMu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Profile{}, nil
		}
		return Profile{}, fmt.Errorf("read profile %s: %w", name, err)
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("decode profile %s: %w", name, err)
	}
	profile.Normalize()
	return profile, nil
}

func (s *Store) SaveProfile(profile Profile) error {
	path, err := s.profilePath(profile.Name)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile %s: %w", profile.Name, err)
	}
	data = append(data, '\n')

	configMu.Lock()
	defer configMu.Unlock()

	return writeAtomic(path, data)
}

func (s *Store) WriteBuiltins() error {
	for _, profile := range builtinProfiles() {
		if err := s.SaveProfile(profile); err != nil {
			return fmt.Errorf("write built-in %s: %w", profile.Name, err)
		}
	}
	return nil
}

func (s *Store) Active() (string, error) {
	return s.readMarker(activeFile)
}

func (s *Store) SetActive(name string) error {
	return s.writeMarker(activeFile, name)
}

func (s *Store) LastKnownGood() (string, error) {
	return s.readMarker(lastKnownGoodFile)
}

func (s *Store) SetLastKnownGood(name string) error {
	return s.writeMarker(lastKnownGoodFile, name)
}

func (s *Store) profilePath(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("invalid profile name %q", name)
	}
	return filepath.Join(s.dir, name+".json"), nil
}

func (s *Store) readMarker(file string) (string, error) {
	configMu.Lock()
	defer configMu.Unlock()

	data, err := os.ReadFile(filepath.Join(s.dir, file))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (s *Store) writeMarker(file string, name string) error {
	if _, err := s.profilePath(name); err != nil {
		return err
	}

	configMu.Lock()
	defer configMu.Unlock()

	return writeAtomic(filepath.Join(s.dir, file), []byte(name+"\n"))
}

func builtinProfiles() []Profile {
	return []Profile{standardProfile(), hidOnlyProfile()}
}

func builtinByName(name string) (Profile, bool) {
	for _, profile := range builtinProfiles() {
		if profile.Name == name {
			return profile, true
		}
	}
	return Profile{}, false
}

func writeAtomic(path string, data []byte) error {
	file, err := newAtomicFile(path, 0o600)
	if err != nil {
		return err
	}
	defer file.Discard()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return file.Commit()
}

type atomicFile struct {
	file   *os.File
	tmp    string
	dest   string
	mode   os.FileMode
	closed bool
}

func newAtomicFile(dest string, mode os.FileMode) (*atomicFile, error) {
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

	return &atomicFile{file: file, tmp: file.Name(), dest: dest, mode: mode}, nil
}

func (a *atomicFile) Write(p []byte) (int, error) {
	return a.file.Write(p)
}

func (a *atomicFile) Path() string {
	return a.tmp
}

func (a *atomicFile) Flush() error {
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

func (a *atomicFile) Commit() error {
	if err := a.Flush(); err != nil {
		return err
	}
	if err := os.Rename(a.tmp, a.dest); err != nil {
		return fmt.Errorf("replace %s: %w", a.dest, err)
	}
	if err := os.Chmod(a.dest, a.mode); err != nil {
		return fmt.Errorf("set permissions on %s: %w", a.dest, err)
	}
	return syncDir(filepath.Dir(a.dest))
}

func (a *atomicFile) Discard() {
	if !a.closed {
		a.closed = true
		_ = a.file.Close()
	}
	_ = os.Remove(a.tmp)
}

func syncDir(dir string) error {
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
