package sources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"NanoKVM-Server/utils"
)

const (
	DefaultConfigPath = "/etc/kvm/presentation/sources.json"
	maxConfigBytes    = 64 << 10
)

type Config struct {
	SchemaVersion int    `json:"schema_version"`
	Slots         []Slot `json:"slots"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	if path == "" {
		path = DefaultConfigPath
	}
	return &Store{path: path}
}

func (s *Store) Load() ([]Slot, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sources config: %w", err)
	}
	if len(data) > maxConfigBytes {
		return nil, errors.New("sources config exceeds 64 KiB")
	}
	var config Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode sources config: %w", err)
	}
	if err := requireJSONEnd(decoder); err != nil {
		return nil, fmt.Errorf("decode sources config: %w", err)
	}
	if config.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("sources config schema %d, want %d", config.SchemaVersion, SchemaVersion)
	}
	return validateSlots(config.Slots)
}

func (s *Store) Save(slots []Slot) error {
	validated, err := validateSlots(slots)
	if err != nil {
		return err
	}
	data, err := json.Marshal(Config{SchemaVersion: SchemaVersion, Slots: validated})
	if err != nil {
		return fmt.Errorf("encode sources config: %w", err)
	}
	return utils.WriteFileAtomic(s.path, append(data, '\n'), 0o600)
}

func requireJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
