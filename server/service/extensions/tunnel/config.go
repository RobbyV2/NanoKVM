package tunnel

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
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

	return utils.WriteFileAtomic(path, data, 0o600)
}
