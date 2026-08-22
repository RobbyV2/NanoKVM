package tunnel

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"NanoKVM-Server/proto"
)

var (
	configMu  sync.Mutex
	configDir = "/etc/kvm"
)

type Config struct {
	Args string            `json:"args"`
	Env  map[string]string `json:"env"`
}

func configPath(name proto.TunnelName) string {
	return filepath.Join(configDir, string(name)+".json")
}

func loadConfig(name proto.TunnelName) (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	return loadConfigFromPath(configPath(name))
}

func saveConfig(name proto.TunnelName, cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	return saveConfigToPath(configPath(name), cfg)
}

func loadConfigFromPath(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode tunnel config: %w", err)
	}
	return cfg, nil
}

func saveConfigToPath(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tunnel config: %w", err)
	}
	data = append(data, '\n')

	file, err := newAtomicFile(path, 0o600)
	if err != nil {
		return err
	}
	defer file.Discard()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write tunnel config: %w", err)
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
