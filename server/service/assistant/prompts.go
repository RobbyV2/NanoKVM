package assistant

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"NanoKVM-Server/utils"

	"github.com/pelletier/go-toml/v2"
)

// prompts.toml is the chaice extension's file, copied verbatim. content.js
// parses it once into a table of tables and reads it as
// universal.prompt + "\n\n" + <kind>.prompt; so do we. It is the firmware
// default; an admin can override it from settings (PromptsFile).
//
//go:embed prompts.toml
var promptsTOML []byte

const (
	PromptsFile    = "/etc/kvm/assistant/prompts.json"
	maxPromptsJSON = 256 << 10
)

var (
	promptsMu       sync.Mutex
	promptsFilePath = PromptsFile
	defaultPrompts  = embeddedPrompts

	embeddedOnce sync.Once
	embedded     map[string]any
	embeddedErr  error
)

type PromptField struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PromptEntry struct {
	Name   string        `json:"name"`
	Fields []PromptField `json:"fields"`
}

// PromptSet is the editable form: one entry per TOML table, one field per
// key, in the admin's order. IsDefault is only meaningful on responses.
type PromptSet struct {
	Entries   []PromptEntry `json:"entries"`
	IsDefault bool          `json:"isDefault"`
}

type promptsFile struct {
	Entries []PromptEntry `json:"entries"`
}

func embeddedPrompts() (map[string]any, error) {
	embeddedOnce.Do(func() {
		embedded, embeddedErr = parsePrompts(promptsTOML)
	})
	return embedded, embeddedErr
}

func parsePrompts(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := toml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse prompts.toml: %w", err)
	}
	return out, nil
}

// loadPrompts returns the current prompts as a table of tables: the saved
// override if there is one, else the embedded default. It reads the override
// on every call so asks see the latest save.
func loadPrompts() (map[string]any, error) {
	set, err := loadPromptSet()
	if err != nil {
		return nil, err
	}
	return promptSetToMap(set.Entries), nil
}

func loadPromptSet() (PromptSet, error) {
	promptsMu.Lock()
	defer promptsMu.Unlock()
	return loadPromptSetLocked()
}

func loadPromptSetLocked() (PromptSet, error) {
	data, err := os.ReadFile(promptsFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return defaultPromptSet()
	}
	if err != nil {
		return PromptSet{}, fmt.Errorf("read assistant prompts: %w", err)
	}
	var f promptsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return PromptSet{}, fmt.Errorf("decode assistant prompts: %w", err)
	}
	return PromptSet{Entries: normalizeEntries(f.Entries)}, nil
}

func defaultPromptSet() (PromptSet, error) {
	m, err := defaultPrompts()
	if err != nil {
		return PromptSet{}, err
	}
	set, err := promptSetFromMap(m)
	if err != nil {
		return PromptSet{}, err
	}
	set.IsDefault = true
	return set, nil
}

func savePrompts(entries []PromptEntry) (PromptSet, error) {
	entries = normalizeEntries(entries)
	for i := range entries {
		entries[i].Name = strings.TrimSpace(entries[i].Name)
		for j := range entries[i].Fields {
			entries[i].Fields[j].Key = strings.TrimSpace(entries[i].Fields[j].Key)
		}
	}
	if err := validatePromptEntries(entries); err != nil {
		return PromptSet{}, err
	}
	data, err := json.MarshalIndent(promptsFile{Entries: entries}, "", "  ")
	if err != nil {
		return PromptSet{}, fmt.Errorf("encode assistant prompts: %w", err)
	}
	if len(data) > maxPromptsJSON {
		return PromptSet{}, fmt.Errorf("%w: prompts exceed 256 KB", errInvalidConfig)
	}
	data = append(data, '\n')

	promptsMu.Lock()
	defer promptsMu.Unlock()
	if err := utils.WriteFileAtomic(promptsFilePath, data, 0o600); err != nil {
		return PromptSet{}, fmt.Errorf("save assistant prompts: %w", err)
	}
	return PromptSet{Entries: entries}, nil
}

func resetPrompts() (PromptSet, error) {
	promptsMu.Lock()
	defer promptsMu.Unlock()
	if err := os.Remove(promptsFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return PromptSet{}, fmt.Errorf("reset assistant prompts: %w", err)
	}
	return defaultPromptSet()
}

func validatePromptEntries(entries []PromptEntry) error {
	names := make(map[string]bool, len(entries))
	for i, e := range entries {
		if e.Name == "" {
			return fmt.Errorf("%w: entry %d has an empty name", errInvalidConfig, i+1)
		}
		if strings.ContainsAny(e.Name, ".[]") {
			return fmt.Errorf("%w: entry name %q must not contain '.', '[' or ']'", errInvalidConfig, e.Name)
		}
		if names[e.Name] {
			return fmt.Errorf("%w: duplicate entry name %q", errInvalidConfig, e.Name)
		}
		names[e.Name] = true
		keys := make(map[string]bool, len(e.Fields))
		for j, f := range e.Fields {
			if f.Key == "" {
				return fmt.Errorf("%w: field %d of entry %q has an empty key", errInvalidConfig, j+1, e.Name)
			}
			if keys[f.Key] {
				return fmt.Errorf("%w: duplicate field key %q in entry %q", errInvalidConfig, f.Key, e.Name)
			}
			keys[f.Key] = true
		}
	}
	return nil
}

// normalizeEntries makes nil lists empty so JSON always carries arrays.
func normalizeEntries(entries []PromptEntry) []PromptEntry {
	out := make([]PromptEntry, len(entries))
	for i, e := range entries {
		if e.Fields == nil {
			e.Fields = []PromptField{}
		}
		out[i] = PromptEntry{Name: e.Name, Fields: append([]PromptField{}, e.Fields...)}
	}
	return out
}

// promptSetFromMap orders a parsed TOML table of tables for editing:
// universal first, then the other tables alphabetically; within a table,
// prompt first, then the other keys alphabetically.
func promptSetFromMap(m map[string]any) (PromptSet, error) {
	names := orderedKeys(m, "universal")
	set := PromptSet{Entries: make([]PromptEntry, 0, len(names))}
	for _, name := range names {
		table, ok := m[name].(map[string]any)
		if !ok {
			return PromptSet{}, errors.New("prompts: every top-level value must be a table")
		}
		entry := PromptEntry{Name: name, Fields: make([]PromptField, 0, len(table))}
		for _, key := range orderedKeys(table, "prompt") {
			value, ok := table[key].(string)
			if !ok {
				return PromptSet{}, errors.New("prompts: every field value must be a string")
			}
			entry.Fields = append(entry.Fields, PromptField{Key: key, Value: value})
		}
		set.Entries = append(set.Entries, entry)
	}
	return set, nil
}

func orderedKeys(m map[string]any, first string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != first {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if _, ok := m[first]; ok {
		keys = append([]string{first}, keys...)
	}
	return keys
}

func promptSetToMap(entries []PromptEntry) map[string]any {
	out := make(map[string]any, len(entries))
	for _, e := range entries {
		table := make(map[string]any, len(e.Fields))
		for _, f := range e.Fields {
			table[f.Key] = f.Value
		}
		out[e.Name] = table
	}
	return out
}

func promptOf(p map[string]any, table string) string {
	t, _ := p[table].(map[string]any)
	s, _ := t["prompt"].(string)
	return s
}

func getPrompt(p map[string]any, kind string) string {
	if p == nil {
		return ""
	}
	return promptOf(p, "universal") + "\n\n" + promptOf(p, kind)
}
