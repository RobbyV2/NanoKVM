package presentation

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	SourceStaticV0 = "static-v0"
	SourceProbeV1  = "probe-v1"
)

var capabilityMu sync.Mutex

type CapabilityTable struct {
	Source          string                        `json:"source"`
	GeneratedAt     time.Time                     `json:"generated_at"`
	MaxInEndpoints  int                           `json:"max_in_endpoints"`
	MaxOutEndpoints int                           `json:"max_out_endpoints"`
	InFIFOWords     []int                         `json:"in_fifo_words,omitempty"`
	Functions       map[FunctionKind]FunctionCaps `json:"functions"`
}

type FunctionCaps struct {
	Available  bool            `json:"available"`
	InEPs      int             `json:"in_eps"`
	OutEPs     int             `json:"out_eps"`
	Attributes map[string]bool `json:"attributes,omitempty"`
}

var staticV0 = CapabilityTable{
	Source:          SourceStaticV0,
	GeneratedAt:     time.Date(2026, time.August, 22, 0, 0, 0, 0, time.UTC),
	MaxInEndpoints:  6,
	MaxOutEndpoints: 5,
	InFIFOWords:     []int{768, 512, 512, 384, 128, 128},
	Functions: map[FunctionKind]FunctionCaps{
		FunctionHID:         {Available: true, InEPs: 1, OutEPs: 1},
		FunctionNCM:         {Available: true, InEPs: 2, OutEPs: 1, Attributes: map[string]bool{"os_desc/interface.ncm": true}},
		FunctionRNDIS:       {Available: true, InEPs: 2, OutEPs: 1, Attributes: map[string]bool{"os_desc/interface.rndis": true}},
		FunctionMassStorage: {Available: true, InEPs: 1, OutEPs: 1},
		FunctionFFS:         {Available: true},
	},
}

// f_hid allocates a /dev/hidgN minor at mkdir time, so hid is never probed.
var probeKinds = []FunctionKind{FunctionNCM, FunctionRNDIS, FunctionMassStorage, FunctionFFS}

func LoadCapabilities() CapabilityTable {
	capabilityMu.Lock()
	defer capabilityMu.Unlock()

	path := capabilityPath()
	table, err := loadCapabilityTable(path)
	switch {
	case err == nil:
		return table
	case !errors.Is(err, os.ErrNotExist):
		log.Warnf("ignoring capability table %s: %s", path, err)
	}

	available, err := probeAvailability()
	if err != nil {
		log.Debugf("capability probe unavailable, using %s: %s", staticV0.Source, err)
		return staticV0.clone()
	}
	return staticV0.withAvailability(available)
}

func capabilityPath() string {
	return filepath.Join(presentationDir, "capability.json")
}

func loadCapabilityTable(path string) (CapabilityTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CapabilityTable{}, err
	}

	var table CapabilityTable
	if err := json.Unmarshal(data, &table); err != nil {
		return CapabilityTable{}, fmt.Errorf("decode capability table %s: %w", path, err)
	}
	if err := table.Validate(); err != nil {
		return CapabilityTable{}, fmt.Errorf("validate capability table %s: %w", path, err)
	}
	return table, nil
}

func (t CapabilityTable) Validate() error {
	switch {
	case t.Source == "":
		return errors.New("source is empty")
	case t.MaxInEndpoints <= 0 || t.MaxOutEndpoints <= 0:
		return fmt.Errorf("budget %d IN %d OUT is not positive", t.MaxInEndpoints, t.MaxOutEndpoints)
	case len(t.Functions) == 0:
		return errors.New("no functions")
	case len(t.InFIFOWords) != 0 && len(t.InFIFOWords) != t.MaxInEndpoints:
		return fmt.Errorf("%d IN FIFOs for %d endpoints", len(t.InFIFOWords), t.MaxInEndpoints)
	}
	for index, words := range t.InFIFOWords {
		if words <= 0 {
			return fmt.Errorf("IN FIFO %d has nonpositive depth %d", index+1, words)
		}
	}
	return nil
}

func (t CapabilityTable) withAvailability(available map[FunctionKind]bool) CapabilityTable {
	merged := t.clone()
	merged.Source = SourceProbeV1
	merged.GeneratedAt = time.Now()

	for kind, caps := range merged.Functions {
		got, ok := available[kind]
		if !ok {
			continue
		}
		caps.Available = got
		merged.Functions[kind] = caps
	}
	return merged
}

func (t CapabilityTable) clone() CapabilityTable {
	cloned := t
	cloned.InFIFOWords = append([]int(nil), t.InFIFOWords...)
	cloned.Functions = make(map[FunctionKind]FunctionCaps, len(t.Functions))

	for kind, caps := range t.Functions {
		if caps.Attributes != nil {
			attributes := make(map[string]bool, len(caps.Attributes))
			maps.Copy(attributes, caps.Attributes)
			caps.Attributes = attributes
		}
		cloned.Functions[kind] = caps
	}
	return cloned
}
