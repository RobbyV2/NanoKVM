package assistant

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// prompts.toml is the chaice extension's file, copied verbatim. content.js
// parses it once into a table of tables and reads it as
// universal.prompt + "\n\n" + <kind>.prompt; so do we.
//
//go:embed prompts.toml
var promptsTOML []byte

var (
	promptsOnce sync.Once
	prompts     map[string]any
	promptsErr  error
)

func loadPrompts() (map[string]any, error) {
	promptsOnce.Do(func() {
		prompts, promptsErr = parsePrompts(promptsTOML)
	})
	return prompts, promptsErr
}

func parsePrompts(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := toml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse prompts.toml: %w", err)
	}
	return out, nil
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
