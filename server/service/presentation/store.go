package presentation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"NanoKVM-Server/utils"
)

const (
	PresentationDir = "/etc/kvm/presentation"

	activeFile        = "active"
	lastKnownGoodFile = ".last-known-good"
	previousFile      = ".previous"
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

func (s *Store) Profiles() ([]Profile, error) {
	configMu.Lock()
	defer configMu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read profiles: %w", err)
	}

	profiles := builtinProfiles()
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".json")
		if entry.Name() == "capability.json" || name == entry.Name() {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if _, ok := builtinByName(name); ok {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read profile %s: %w", name, err)
		}
		var profile Profile
		if err := json.Unmarshal(data, &profile); err != nil {
			return nil, fmt.Errorf("decode profile %s: %w", name, err)
		}
		profile.Normalize()
		if profile.Name != name {
			return nil, fmt.Errorf("profile file %s contains name %q", name, profile.Name)
		}
		if err := profile.Validate(); err != nil {
			return nil, fmt.Errorf("validate profile %s: %w", name, err)
		}
		profiles = append(profiles, profile)
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].BuiltIn != profiles[j].BuiltIn {
			return profiles[i].BuiltIn
		}
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}

func (s *Store) DeleteProfile(name string) error {
	if _, ok := builtinByName(name); ok {
		return fmt.Errorf("built-in profile %q cannot be deleted", name)
	}
	path, err := s.profilePath(name)
	if err != nil {
		return err
	}

	configMu.Lock()
	defer configMu.Unlock()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete profile %s: %w", name, err)
	}
	return nil
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

// A profile that binds and verifies becomes the last-known-good one, so the
// marker never names anything but the gadget that is running. What a manual
// rollback needs is the name that marker displaced.
func (s *Store) SetLastKnownGood(name string) error {
	current, err := s.readMarker(lastKnownGoodFile)
	if err != nil {
		return err
	}
	previous, err := s.readMarker(previousFile)
	if err != nil {
		return err
	}
	// A rollback landing on the profile it was aimed at does not make the
	// gadget the operator just walked away from the next target.
	if current != "" && current != name && previous != name {
		if err := s.writeMarker(previousFile, current); err != nil {
			return err
		}
	}
	return s.writeMarker(lastKnownGoodFile, name)
}

func (s *Store) Previous() (string, error) {
	return s.readMarker(previousFile)
}

func (s *Store) profilePath(name string) (string, error) {
	if !profileNamePattern.MatchString(name) {
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
