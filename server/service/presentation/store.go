package presentation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"NanoKVM-Server/utils"
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

	return utils.WriteFileAtomic(path, data, 0o600)
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

	return utils.WriteFileAtomic(filepath.Join(s.dir, file), []byte(name+"\n"), 0o600)
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
