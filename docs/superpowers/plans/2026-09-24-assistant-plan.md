# Assistant (chaice, native in NanoKVM) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Re-implement the chaice browser extension as a NanoKVM feature. Ctrl-Ctrl + key on the desktop page captures the target screen on the device, asks an LLM from the Go server, and shows, copies or places the answer exactly as the extension does.

**Architecture:** A new Go package `server/service/assistant` ports `llm.js` (request bodies), `background.js` (`performApiRequest` transport: relay, retry, fallback) and the ask flows of `content.js`. The flows capture through the existing MCP capture backend. The web side adds a settings tab and a desktop runtime. The runtime ports the on-page parts of `content.js` (hotkeys, status bar, toast, FRQ box, selection, freeze, crop-lock, quad-click) and uses the same inline styles.

**Tech Stack:** Go 1.25, gin, logrus, `github.com/pelletier/go-toml/v2` (already in go.sum), React 18 + jotai + antd, Node's built-in test runner.

**Spec:** `docs/superpowers/specs/2026-09-24-assistant-design.md`. Behavioural reference: `../../chaice/chaice/chaice-extension/{content.js,llm.js,background.js,options.js}` (paths relative to the NanoKVM repo root).

## Deviations from the spec (user instructions and findings from reading the source)

1. **Prompts (user instruction).** Do not open or read `prompts.toml` or any other chaice config file. Copy it byte-for-byte into the Go package with `cp` and use it the way `content.js` does. `//go:embed` it, parse it once as TOML into a table of tables, and build the prompt as `universal.prompt + "\n\n" + <kind>.prompt`. The prompts are **not** a setting. The spec's `prompts` settings row and the settings "prompts" section are dropped.
2. **Custom (M) prompt.** `content.js` uses `getPrompt("custom")`, which is universal + `[custom]`. The spec said "universal prompt"; the source wins.
3. **Crop coordinates.** The browser sends the crop as fractions (0–1) of the video frame, not source pixels. The stream resolution can differ from the capture resolution, so the server multiplies the fractions by the captured JPEG's own dimensions.
4. **Freeze overlay placement.** The extension's screenshot was of the viewport, so `cover` lined it up. Ours is the target frame, so the overlay goes over the `#screen` rendered-media rect with `100% 100%` sizing. Everything else about its style is unchanged.
5. **Page CSS `#screen` rule.** This rule is scoped to `#screen-viewport[data-cropped="false"] #screen`. Otherwise `object-fit: contain !important` breaks the manual input-region (cropped) view.
6. **Kept faithful on purpose (quirks the user may want changed):**
   - The Ctrl-Ctrl prefix stays armed until an action key or the next Ctrl press. There is no time limit on the action key.
   - Pressing ↑ when OpenRouter effort is already `high` shows "Reasoning: off (already)".
   - An F answer goes into the FRQ box without making it visible.

## Global Constraints

- Settings file `/etc/kvm/assistant.json`, mode 0600, atomic write (`utils.WriteFileAtomic`). Attachments dir `/etc/kvm/assistant/attachments/`, total cap 20 MB.
- All `/api/assistant/*` routes are behind `middleware.CheckToken()` + `middleware.RequireRole(authn.RoleAdmin)`.
- Error envelope is `proto.Response`. The codes are `-1` generic, `-2` disabled and `-3` no answer.
- The API keys and `proxyPass` never leave the device. They are never logged and never appear in error messages. `GET` returns the `has*` booleans instead.
- Default provider `openrouter`. Default `geminiBaseUrl` is `https://generativelanguage.googleapis.com/v1beta` and default `orBaseUrl` is `https://openrouter.ai/api/v1`. `thinkingBudget` defaults to 512 and `orReasoningEffort` to `low`. `ui` defaults to true and `enabled` to false.
- OpenRouter headers: `HTTP-Referer: https://github.com/chaice`, `X-Title: ChAIce`.
- Transport: 300 s per attempt (until response headers), 2 proxy attempts 5 s apart, then a direct fallback. Unwrap `X-Relay-Wrap: 1`.
- Status colours, verbatim: idle `rgb(40,62,159)`, answer-select/loading `#b3cfff`, FRQ `#e0cfff`, context-select `#0a1333`, success `rgb(89, 105, 192)`, error `red`, no-answer `yellow`, cleared `#444`, crop `#285e9f`, crop-select `orange`. Reset to idle after 3 s.
- The package `server/service/assistant` must not import cgo packages (`NanoKVM-Server/common`, `service/mcp/...`). The router adapts the capture backend.
- i18n keys go in `web/src/i18n/locales/en.ts` only.
- Go tests run in the builder container (no Go on the host). From the NanoKVM root:
  `GOTEST` ≡ `docker run --rm -e UID=1000 -e GID=1000 -v "$PWD":/home/build/NanoKVM nanokvm-builder-local-1000-1000 /bin/bash -c 'cd /home/build/NanoKVM/server && go test ./service/assistant/ <ARGS>'`
- Web checks: `cd web && pnpm exec tsc --noEmit && pnpm test`.
- Work on branch `assistant` (`git switch -c assistant` before Task 1). Commit per task.

## Review Focus

1. **A secret leaking through an error or log line.** A failed direct Gemini call returns a `*url.Error` whose text contains `?key=<key>`. The user expects the error the UI logs to name the failure, not the key. Pinned in Task 4: `TestDirectFailureDoesNotLeakKey`.
2. **An action key reaching the target while it is held.** Auto-repeat keydowns and the final keyup of a swallowed key must also be swallowed. Pinned in Task 10: `held key repeats and keyup are swallowed`.
3. **A selection that is a click (zero area) or lies entirely on the black letterbox.** Expect a red status and no LLM call, never a garbage image. Pinned in Task 5 (`TestCropBoundsEmpty`) and Task 10 (`selection outside the picture is null`).
4. **Long thinking calls.** The worst case is 2×300 s + 5 s + 300 s. The browser request must not time out first (axios `timeout: 0`, Task 9). Closing the tab must cancel the upstream call. Pinned in Task 4: `TestSendHonoursContextCancel`.
5. **Crop-lock while the screen viewport has a video scale ≠ 1.** The translate must be in the element's local units or the locked region is off-centre. Pinned in Task 10: `crop-lock translate is divided by the layout scale`.

---

### Task 1: Prompts (copied TOML, parsed like content.js)

**Files:**
- Create: `server/service/assistant/prompts.toml` (by `cp`, never opened)
- Create: `server/service/assistant/prompts.go`
- Test: `server/service/assistant/prompts_test.go`
- Modify: `server/go.mod` (move `github.com/pelletier/go-toml/v2 v2.2.2` from the indirect block into the direct `require` block, alphabetically after `github.com/modelcontextprotocol/go-sdk`)

**Interfaces:**
- Produces: `loadPrompts() (map[string]any, error)`, `parsePrompts([]byte) (map[string]any, error)`, `getPrompt(p map[string]any, kind string) string`

- [ ] **Step 1: Copy the file without reading it**

```bash
git switch -c assistant
mkdir -p server/service/assistant
cp ../../chaice/chaice/chaice-extension/prompts.toml server/service/assistant/prompts.toml
```

- [ ] **Step 2: Write the failing test**

```go
package assistant

import "testing"

func TestGetPromptJoinsUniversalAndKind(t *testing.T) {
	p, err := parsePrompts([]byte("[universal]\nprompt = \"U\"\n[mcq]\nprompt = \"M\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := getPrompt(p, "mcq"); got != "U\n\nM" {
		t.Fatalf("mcq prompt = %q", got)
	}
	// content.js: a missing table still yields universal + "\n\n".
	if got := getPrompt(p, "custom"); got != "U\n\n" {
		t.Fatalf("custom prompt = %q", got)
	}
	if got := getPrompt(nil, "mcq"); got != "" {
		t.Fatalf("nil prompts = %q", got)
	}
}

func TestEmbeddedPromptsParse(t *testing.T) {
	if _, err := loadPrompts(); err != nil {
		t.Fatalf("embedded prompts.toml: %v", err)
	}
}
```

- [ ] **Step 3: Run it and confirm it fails.** Run `GOTEST` with `-run Prompt -v`. Expected: build failure, `undefined: parsePrompts`.

- [ ] **Step 4: Implement**

```go
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
```

- [ ] **Step 5: Run the tests and confirm they pass.** Run `GOTEST` with `-run Prompt -v`. Expected: PASS.
- [ ] **Step 6: Commit**

```bash
git add server/go.mod server/service/assistant/prompts.toml server/service/assistant/prompts.go server/service/assistant/prompts_test.go
git commit -m "assistant: embed the chaice prompts and read them as content.js does"
```

---

### Task 2: Settings (load, update, mask)

**Files:**
- Create: `server/service/assistant/config.go`
- Test: `server/service/assistant/config_test.go`

**Interfaces:**
- Produces: the `Config` struct (JSON tags equal the extension's setting names), `PublicConfig`, `ConfigUpdate`, `defaultConfig()`, `loadConfig()`, `updateConfig(func(Config) (Config, error))`, `applyUpdate(Config, ConfigUpdate) (Config, error)`, `publicConfig(Config) PublicConfig`, `errInvalidConfig`, and the var `configFilePath`.

- [ ] **Step 1: Write the failing test**

```go
package assistant

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func useTestConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "assistant.json")
	old := configFilePath
	configFilePath = path
	t.Cleanup(func() { configFilePath = old })
	return path
}

func ptr[T any](v T) *T { return &v }

func TestMissingConfigIsDefaults(t *testing.T) {
	useTestConfig(t)
	cfg, err := loadConfig()
	if err != nil || cfg != defaultConfig() {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	if cfg.Provider != "openrouter" || !cfg.UI || cfg.Enabled || cfg.ThinkingBudget != 512 || cfg.ORReasoningEffort != "low" {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
}

func TestPartialFileKeepsDefaults(t *testing.T) {
	path := useTestConfig(t)
	os.WriteFile(path, []byte(`{"enabled":true,"orModel":"m"}`), 0o600)
	cfg, err := loadConfig()
	if err != nil || !cfg.Enabled || cfg.ORModel != "m" || cfg.ORBaseURL != defaultORBaseURL || !cfg.UI {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestUpdateSecretsAndMask(t *testing.T) {
	path := useTestConfig(t)
	cfg, err := updateConfig(func(c Config) (Config, error) {
		return applyUpdate(c, ConfigUpdate{ORAPIKey: ptr("sk-1"), ProxyPass: ptr("pw")})
	})
	if err != nil || cfg.ORAPIKey != "sk-1" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	// Empty secret leaves it; clear removes it.
	cfg, _ = applyUpdate(cfg, ConfigUpdate{ORAPIKey: ptr(""), ClearProxyPass: true})
	if cfg.ORAPIKey != "sk-1" || cfg.ProxyPass != "" {
		t.Fatalf("secret update wrong: %+v", cfg)
	}
	pub := publicConfig(cfg)
	if !pub.HasORAPIKey || pub.HasProxyPass || pub.HasGeminiAPIKey {
		t.Fatalf("mask wrong: %+v", pub)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode())
	}
}

func TestApplyUpdateValidates(t *testing.T) {
	base := defaultConfig()
	for _, u := range []ConfigUpdate{
		{Provider: ptr("claude")},
		{ORReasoningEffort: ptr("max")},
		{ThinkingBudget: ptr(-1)},
	} {
		if _, err := applyUpdate(base, u); !errors.Is(err, errInvalidConfig) {
			t.Fatalf("update %+v: err=%v", u, err)
		}
	}
	cfg, _ := applyUpdate(base, ConfigUpdate{ORBaseURL: ptr("  "), ProxyURL: ptr(" http://r:1 ")})
	if cfg.ORBaseURL != defaultORBaseURL || cfg.ProxyURL != "http://r:1" {
		t.Fatalf("normalise wrong: %+v", cfg)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails.** Run `GOTEST` with `-run Config -v`. Expected: build failure.
- [ ] **Step 3: Implement**

```go
package assistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"NanoKVM-Server/utils"
)

const (
	ConfigFile           = "/etc/kvm/assistant.json"
	defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	defaultORBaseURL     = "https://openrouter.ai/api/v1"
)

var (
	configMu         sync.Mutex
	configFilePath   = ConfigFile
	errInvalidConfig = errors.New("invalid arguments")
)

// Config keeps the chaice extension's setting names so its llm.js port reads
// the same fields.
type Config struct {
	Enabled           bool   `json:"enabled"`
	UI                bool   `json:"ui"`
	Provider          string `json:"provider"`
	GeminiBaseURL     string `json:"geminiBaseUrl"`
	GeminiModel       string `json:"geminiModel"`
	GeminiAPIKey      string `json:"geminiApiKey"`
	GeminiThinking    bool   `json:"geminiThinking"`
	ThinkingBudget    int    `json:"thinkingBudget"`
	ORBaseURL         string `json:"orBaseUrl"`
	ORModel           string `json:"orModel"`
	ORAPIKey          string `json:"orApiKey"`
	ORReasoning       bool   `json:"orReasoning"`
	ORReasoningEffort string `json:"orReasoningEffort"`
	ProxyURL          string `json:"proxyUrl"`
	ProxyPass         string `json:"proxyPass"`
	CopyClipboard     bool   `json:"copyClipboard"`
	AnswerSel         bool   `json:"answerSel"`
	ContextSel        bool   `json:"contextSel"`
	FRQSel            bool   `json:"frqSel"`
	QuadClickMCQ      bool   `json:"quadClickMCQ"`
}

type PublicConfig struct {
	Enabled           bool   `json:"enabled"`
	UI                bool   `json:"ui"`
	Provider          string `json:"provider"`
	GeminiBaseURL     string `json:"geminiBaseUrl"`
	GeminiModel       string `json:"geminiModel"`
	GeminiThinking    bool   `json:"geminiThinking"`
	ThinkingBudget    int    `json:"thinkingBudget"`
	ORBaseURL         string `json:"orBaseUrl"`
	ORModel           string `json:"orModel"`
	ORReasoning       bool   `json:"orReasoning"`
	ORReasoningEffort string `json:"orReasoningEffort"`
	ProxyURL          string `json:"proxyUrl"`
	CopyClipboard     bool   `json:"copyClipboard"`
	AnswerSel         bool   `json:"answerSel"`
	ContextSel        bool   `json:"contextSel"`
	FRQSel            bool   `json:"frqSel"`
	QuadClickMCQ      bool   `json:"quadClickMCQ"`
	HasGeminiAPIKey   bool   `json:"hasGeminiApiKey"`
	HasORAPIKey       bool   `json:"hasOrApiKey"`
	HasProxyPass      bool   `json:"hasProxyPass"`
}

// ConfigUpdate: nil fields are left alone; an empty secret is left alone and
// only a clear flag removes one (the MCP/PicoClaw masking convention).
type ConfigUpdate struct {
	Enabled           *bool   `json:"enabled"`
	UI                *bool   `json:"ui"`
	Provider          *string `json:"provider"`
	GeminiBaseURL     *string `json:"geminiBaseUrl"`
	GeminiModel       *string `json:"geminiModel"`
	GeminiAPIKey      *string `json:"geminiApiKey"`
	GeminiThinking    *bool   `json:"geminiThinking"`
	ThinkingBudget    *int    `json:"thinkingBudget"`
	ORBaseURL         *string `json:"orBaseUrl"`
	ORModel           *string `json:"orModel"`
	ORAPIKey          *string `json:"orApiKey"`
	ORReasoning       *bool   `json:"orReasoning"`
	ORReasoningEffort *string `json:"orReasoningEffort"`
	ProxyURL          *string `json:"proxyUrl"`
	ProxyPass         *string `json:"proxyPass"`
	CopyClipboard     *bool   `json:"copyClipboard"`
	AnswerSel         *bool   `json:"answerSel"`
	ContextSel        *bool   `json:"contextSel"`
	FRQSel            *bool   `json:"frqSel"`
	QuadClickMCQ      *bool   `json:"quadClickMCQ"`
	ClearGeminiAPIKey bool    `json:"clearGeminiApiKey"`
	ClearORAPIKey     bool    `json:"clearOrApiKey"`
	ClearProxyPass    bool    `json:"clearProxyPass"`
}

func defaultConfig() Config {
	return Config{
		UI:                true,
		Provider:          "openrouter",
		GeminiBaseURL:     defaultGeminiBaseURL,
		ThinkingBudget:    thinkingStep,
		ORBaseURL:         defaultORBaseURL,
		ORReasoningEffort: "low",
	}
}

func loadConfig() (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()
	return loadConfigFromPath(configFilePath)
}

func updateConfig(update func(Config) (Config, error)) (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	cfg, err := loadConfigFromPath(configFilePath)
	if err != nil {
		return Config{}, err
	}
	updated, err := update(cfg)
	if err != nil {
		return Config{}, err
	}
	if err := saveConfigToPath(configFilePath, updated); err != nil {
		return Config{}, err
	}
	return updated, nil
}

func loadConfigFromPath(path string) (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode assistant config: %w", err)
	}
	return cfg, nil
}

func saveConfigToPath(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode assistant config: %w", err)
	}
	data = append(data, '\n')
	if err := utils.WriteFileAtomic(path, data, 0o600); err != nil {
		return fmt.Errorf("save assistant config: %w", err)
	}
	return nil
}

func publicConfig(c Config) PublicConfig {
	return PublicConfig{
		Enabled: c.Enabled, UI: c.UI, Provider: c.Provider,
		GeminiBaseURL: c.GeminiBaseURL, GeminiModel: c.GeminiModel,
		GeminiThinking: c.GeminiThinking, ThinkingBudget: c.ThinkingBudget,
		ORBaseURL: c.ORBaseURL, ORModel: c.ORModel, ORReasoning: c.ORReasoning,
		ORReasoningEffort: c.ORReasoningEffort, ProxyURL: c.ProxyURL,
		CopyClipboard: c.CopyClipboard, AnswerSel: c.AnswerSel, ContextSel: c.ContextSel,
		FRQSel: c.FRQSel, QuadClickMCQ: c.QuadClickMCQ,
		HasGeminiAPIKey: c.GeminiAPIKey != "", HasORAPIKey: c.ORAPIKey != "", HasProxyPass: c.ProxyPass != "",
	}
}

func applyUpdate(c Config, u ConfigUpdate) (Config, error) {
	setBool := func(dst *bool, v *bool) {
		if v != nil {
			*dst = *v
		}
	}
	setTrimmed := func(dst *string, v *string, fallback string) {
		if v == nil {
			return
		}
		*dst = strings.TrimSpace(*v)
		if *dst == "" {
			*dst = fallback
		}
	}
	setSecret := func(dst *string, v *string, clear bool) {
		if v != nil && *v != "" {
			*dst = *v
		}
		if clear {
			*dst = ""
		}
	}

	if u.Provider != nil {
		if *u.Provider != "gemini" && *u.Provider != "openrouter" {
			return Config{}, errInvalidConfig
		}
		c.Provider = *u.Provider
	}
	if u.ORReasoningEffort != nil {
		switch *u.ORReasoningEffort {
		case "low", "medium", "high":
			c.ORReasoningEffort = *u.ORReasoningEffort
		default:
			return Config{}, errInvalidConfig
		}
	}
	if u.ThinkingBudget != nil {
		if *u.ThinkingBudget < 0 {
			return Config{}, errInvalidConfig
		}
		c.ThinkingBudget = *u.ThinkingBudget
	}
	setBool(&c.Enabled, u.Enabled)
	setBool(&c.UI, u.UI)
	setBool(&c.GeminiThinking, u.GeminiThinking)
	setBool(&c.ORReasoning, u.ORReasoning)
	setBool(&c.CopyClipboard, u.CopyClipboard)
	setBool(&c.AnswerSel, u.AnswerSel)
	setBool(&c.ContextSel, u.ContextSel)
	setBool(&c.FRQSel, u.FRQSel)
	setBool(&c.QuadClickMCQ, u.QuadClickMCQ)
	setTrimmed(&c.GeminiBaseURL, u.GeminiBaseURL, defaultGeminiBaseURL)
	setTrimmed(&c.ORBaseURL, u.ORBaseURL, defaultORBaseURL)
	setTrimmed(&c.GeminiModel, u.GeminiModel, "")
	setTrimmed(&c.ORModel, u.ORModel, "")
	setTrimmed(&c.ProxyURL, u.ProxyURL, "")
	setSecret(&c.GeminiAPIKey, u.GeminiAPIKey, u.ClearGeminiAPIKey)
	setSecret(&c.ORAPIKey, u.ORAPIKey, u.ClearORAPIKey)
	setSecret(&c.ProxyPass, u.ProxyPass, u.ClearProxyPass)
	return c, nil
}
```

(`thinkingStep` is defined in Task 3. Until Task 3 lands, add `const thinkingStep = 512` temporarily to `config.go` and move it in Task 3.)

- [ ] **Step 4: Run the tests and confirm they pass.** Run `GOTEST` with `-run Config -v`. Expected: PASS.
- [ ] **Step 5: Commit.** `git commit -m "assistant: settings file with masked secrets"` (add `config.go` and `config_test.go`).

---

### Task 3: llm.js port, checked against fixtures produced by llm.js

**Files:**
- Create: `server/service/assistant/llm.go`
- Create: `server/service/assistant/testdata/cases.json`, `testdata/gen-fixtures.mjs`, `testdata/fixtures.json` (generated)
- Test: `server/service/assistant/llm_test.go`

**Interfaces:**
- Consumes: `Config` (Task 2).
- Produces:
  - `type Image struct{ Mime, B64 string }`
  - `type Attachment struct{ Name, Kind, Mime, AudioFormat, B64, Text string }`
  - `type Turn struct{ Role, Text string; Images []Image; Attachments []Attachment }`
  - `type Request struct{ URL string; Headers map[string]string; Body []byte }`
  - `BuildConversation(provider string, cfg Config, turns []Turn, thinking *bool) Request`
  - `ParseResponse(provider string, data json.RawMessage) (string, bool)`
  - `classifyFile(name string) fileType{Kind, Mime, AudioFormat}`
  - `formatTextAttachments([]Attachment) string`
  - `const thinkingStep = 512`

- [ ] **Step 1: Write the cases.** Every image is passed as `image/png` so the Go body can be compared with llm.js, which hard-codes png.

`testdata/cases.json`:

```json
[
  {"name": "gemini-single", "provider": "gemini",
   "settings": {"geminiBaseUrl": "https://g.example/v1beta", "geminiModel": "gm", "geminiApiKey": "k"},
   "turns": [{"role": "user", "text": "Q", "images": ["aW1nMQ==", "aW1nMg=="]}]},
  {"name": "gemini-thinking", "provider": "gemini",
   "settings": {"geminiBaseUrl": "https://g.example/v1beta/", "geminiModel": "gm", "geminiApiKey": "k", "geminiThinking": true, "thinkingBudget": 1024},
   "turns": [{"role": "user", "text": "Q", "images": ["aW1n"]}]},
  {"name": "gemini-thinking-zero-budget", "provider": "gemini",
   "settings": {"geminiBaseUrl": "https://g.example/v1beta", "geminiModel": "gm", "geminiApiKey": "k", "geminiThinking": true, "thinkingBudget": 0},
   "turns": [{"role": "user", "text": "", "images": ["aW1n"]}]},
  {"name": "gemini-two-turn", "provider": "gemini", "thinking": false,
   "settings": {"geminiBaseUrl": "https://g.example/v1beta//", "geminiModel": "gm", "geminiApiKey": "k y/+=&!*'()~", "geminiThinking": true},
   "turns": [
     {"role": "user", "text": "anchor", "images": ["aW1n"]},
     {"role": "assistant", "text": "OK"},
     {"role": "user", "text": "P", "attachments": [
       {"name": "notes.TXT", "b64": "aGVsbG8=", "text": "hello"},
       {"name": "a.pdf", "b64": "cGRm"}, {"name": "p.png", "b64": "cG5n"},
       {"name": "s.mp3", "b64": "bXAz"}, {"name": "noext", "b64": "eA==", "text": "x"}]}]},
  {"name": "openrouter-single-default-base", "provider": "openrouter",
   "settings": {"orModel": "om", "orApiKey": "sk"},
   "turns": [{"role": "user", "text": "Q", "images": ["aW1nMQ==", "aW1nMg=="]}]},
  {"name": "openrouter-reasoning", "provider": "openrouter",
   "settings": {"orBaseUrl": "https://or.example/api/v1/", "orModel": "om", "orApiKey": "sk", "orReasoning": true, "orReasoningEffort": "high"},
   "turns": [{"role": "user", "text": "Q", "images": ["aW1n"]}]},
  {"name": "openrouter-two-turn-all-kinds", "provider": "openrouter", "thinking": false,
   "settings": {"orModel": "om", "orApiKey": "sk", "orReasoning": true},
   "turns": [
     {"role": "user", "text": "anchor", "images": ["aW1n"]},
     {"role": "assistant", "text": "OK"},
     {"role": "user", "text": "P", "attachments": [
       {"name": "a.md", "b64": "IyBo", "text": "# h"}, {"name": "b.csv", "b64": "MSwy", "text": "1,2"},
       {"name": "c.pdf", "b64": "cGRm"}, {"name": "d.jpg", "b64": "anBn"},
       {"name": "e.wav", "b64": "d2F2"}, {"name": "f.m4a", "b64": "bTRh"},
       {"name": "g.mov", "b64": "bW92"}]}]},
  {"name": "openrouter-empty-text", "provider": "openrouter",
   "settings": {"orModel": "om"},
   "turns": [{"role": "user", "text": ""}]}
]
```

- [ ] **Step 2: Write the generator and run it**

`testdata/gen-fixtures.mjs`:

```js
// Regenerate fixtures.json by running the chaice extension's own llm.js.
// Usage (from the NanoKVM root):
//   node server/service/assistant/testdata/gen-fixtures.mjs ../../chaice/chaice/chaice-extension/llm.js
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const here = dirname(fileURLToPath(import.meta.url));
const ctx = { window: {} };
vm.createContext(ctx);
vm.runInContext(readFileSync(process.argv[2], "utf8"), ctx);
const L = ctx.window.ChaiceLLM;

const cases = JSON.parse(readFileSync(join(here, "cases.json"), "utf8"));
const out = {};
for (const c of cases) {
  const turns = c.turns.map((t) => ({
    role: t.role,
    text: t.text,
    images: t.images || [],
    attachments: (t.attachments || []).map((a) => {
      const cls = L.classifyFile(a.name);
      const att = { name: a.name, kind: cls.kind, mime: cls.mime, audioFormat: cls.audioFormat, b64: a.b64 };
      if (cls.kind === "text") att.text = a.text;
      return att;
    }),
  }));
  const req = L.buildConversation({ provider: c.provider, settings: c.settings, turns, thinking: c.thinking });
  out[c.name] = { url: req.url, headers: req.headers, body: JSON.parse(req.body) };
}
writeFileSync(join(here, "fixtures.json"), JSON.stringify(out, null, 2) + "\n");
console.log(`wrote ${Object.keys(out).length} fixtures`);
```

Run: `node server/service/assistant/testdata/gen-fixtures.mjs ../../chaice/chaice/chaice-extension/llm.js`. Expected output: `wrote 8 fixtures`.

- [ ] **Step 3: Write the failing test**

```go
package assistant

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

type fixtureCase struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Settings Config `json:"settings"`
	Thinking *bool  `json:"thinking"`
	Turns    []struct {
		Role        string   `json:"role"`
		Text        string   `json:"text"`
		Images      []string `json:"images"`
		Attachments []struct {
			Name string `json:"name"`
			B64  string `json:"b64"`
			Text string `json:"text"`
		} `json:"attachments"`
	} `json:"turns"`
}

type fixture struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

func TestBuildConversationMatchesLLMJS(t *testing.T) {
	var cases []fixtureCase
	var fixtures map[string]fixture
	mustReadJSON(t, "testdata/cases.json", &cases)
	mustReadJSON(t, "testdata/fixtures.json", &fixtures)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var turns []Turn
			for _, ct := range c.Turns {
				turn := Turn{Role: ct.Role, Text: ct.Text}
				for _, b64 := range ct.Images {
					turn.Images = append(turn.Images, Image{Mime: "image/png", B64: b64})
				}
				for _, a := range ct.Attachments {
					ft := classifyFile(a.Name)
					att := Attachment{Name: a.Name, Kind: ft.Kind, Mime: ft.Mime, AudioFormat: ft.AudioFormat, B64: a.B64}
					if ft.Kind == "text" {
						att.Text = a.Text
					}
					turn.Attachments = append(turn.Attachments, att)
				}
				turns = append(turns, turn)
			}
			got := BuildConversation(c.Provider, c.Settings, turns, c.Thinking)
			want := fixtures[c.Name]
			if got.URL != want.URL {
				t.Fatalf("url\n got %s\nwant %s", got.URL, want.URL)
			}
			if !reflect.DeepEqual(got.Headers, want.Headers) {
				t.Fatalf("headers\n got %v\nwant %v", got.Headers, want.Headers)
			}
			var body any
			if err := json.Unmarshal(got.Body, &body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, want.Body) {
				t.Fatalf("body\n got %s\nwant %v", got.Body, want.Body)
			}
		})
	}
}

func TestScreenshotPartsCarryTheirMime(t *testing.T) {
	turns := []Turn{{Role: "user", Text: "Q", Images: []Image{{Mime: "image/jpeg", B64: "eA=="}}}}
	or := BuildConversation("openrouter", defaultConfig(), turns, nil)
	if !strings.Contains(string(or.Body), `"data:image/jpeg;base64,eA=="`) {
		t.Fatalf("openrouter body %s", or.Body)
	}
	g := BuildConversation("gemini", defaultConfig(), turns, nil)
	if !strings.Contains(string(g.Body), `"mime_type":"image/jpeg"`) {
		t.Fatalf("gemini body %s", g.Body)
	}
}

func TestParseResponse(t *testing.T) {
	cases := []struct {
		provider, data, want string
		ok                   bool
	}{
		{"openrouter", `{"choices":[{"message":{"content":"B"}}]}`, "B", true},
		{"openrouter", `{"choices":[]}`, "", false},
		{"openrouter", `{"choices":[{"message":{"content":null}}]}`, "", false},
		{"gemini", `{"candidates":[{"content":{"parts":[{"text":"C"}]}}]}`, "C", true},
		{"gemini", `{"candidates":[{"content":{"parts":[]}}]}`, "", false},
		{"gemini", `"not an object"`, "", false},
	}
	for _, c := range cases {
		got, ok := ParseResponse(c.provider, json.RawMessage(c.data))
		if got != c.want || ok != c.ok {
			t.Fatalf("%s %s: got %q %v", c.provider, c.data, got, ok)
		}
	}
}

func mustReadJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 4: Run it and confirm it fails.** Run `GOTEST` with `-run 'Conversation|Mime|Parse' -v`. Expected: build failure.
- [ ] **Step 5: Implement `llm.go`.** Remove the temporary `thinkingStep` from `config.go`.

```go
package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Go port of the chaice extension's llm.js: turns -> provider-native request,
// response JSON -> answer. No network I/O.

const thinkingStep = 512

type Image struct {
	Mime string
	B64  string
}

type Attachment struct {
	Name        string
	Kind        string
	Mime        string
	AudioFormat string
	B64         string
	Text        string
}

type Turn struct {
	Role        string
	Text        string
	Images      []Image
	Attachments []Attachment
}

type Request struct {
	URL     string
	Headers map[string]string
	Body    []byte
}

type fileType struct {
	Kind        string
	Mime        string
	AudioFormat string
}

var fileTypes = map[string]fileType{
	"png": {"image", "image/png", ""}, "jpg": {"image", "image/jpeg", ""},
	"jpeg": {"image", "image/jpeg", ""}, "webp": {"image", "image/webp", ""},
	"gif": {"image", "image/gif", ""}, "heic": {"image", "image/heic", ""},
	"heif": {"image", "image/heif", ""},
	"pdf": {"pdf", "application/pdf", ""},
	"mp3": {"audio", "audio/mpeg", "mp3"}, "wav": {"audio", "audio/wav", "wav"},
	"ogg": {"audio", "audio/ogg", "ogg"}, "flac": {"audio", "audio/flac", "flac"},
	"aac": {"audio", "audio/aac", "aac"}, "m4a": {"audio", "audio/mp4", "m4a"},
	"aiff": {"audio", "audio/aiff", "aiff"},
	"mp4": {"video", "video/mp4", ""}, "mov": {"video", "video/quicktime", ""},
	"webm": {"video", "video/webm", ""},
	"txt": {"text", "text/plain", ""}, "md": {"text", "text/markdown", ""},
	"markdown": {"text", "text/markdown", ""}, "csv": {"text", "text/csv", ""},
	"tsv": {"text", "text/tab-separated-values", ""}, "json": {"text", "application/json", ""},
	"xml": {"text", "application/xml", ""}, "html": {"text", "text/html", ""},
	"htm": {"text", "text/html", ""}, "yaml": {"text", "text/yaml", ""},
	"yml": {"text", "text/yaml", ""}, "log": {"text", "text/plain", ""},
	"ini": {"text", "text/plain", ""}, "conf": {"text", "text/plain", ""},
}

var extPattern = regexp.MustCompile(`\.([^./\\]+)$`)

func classifyFile(name string) fileType {
	if m := extPattern.FindStringSubmatch(name); m != nil {
		if ft, ok := fileTypes[strings.ToLower(m[1])]; ok {
			return ft
		}
	}
	return fileType{Kind: "text", Mime: "text/plain"}
}

func formatTextAttachments(files []Attachment) string {
	var b strings.Builder
	b.WriteString("\n\nThe following text files were attached:\n")
	for _, f := range files {
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", f.Name, f.Text)
	}
	return b.String()
}

func rawBase64(s string) string {
	return strings.TrimPrefix(s, "data:image/png;base64,")
}

// encodeURIComponent matches JavaScript's: everything but A-Z a-z 0-9 -_.!~*'() is escaped.
func encodeURIComponent(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.IndexByte("-_.!~*'()", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func marshalBody(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// --- Gemini ---

type geminiInline struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiPart struct {
	Text       *string       `json:"text,omitempty"`
	InlineData *geminiInline `json:"inline_data,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type geminiGenerationConfig struct {
	ThinkingConfig geminiThinkingConfig `json:"thinkingConfig"`
}

type geminiBody struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

func geminiParts(t Turn) []geminiPart {
	parts := []geminiPart{}
	if t.Text != "" {
		text := t.Text
		parts = append(parts, geminiPart{Text: &text})
	}
	for _, img := range t.Images {
		parts = append(parts, geminiPart{InlineData: &geminiInline{MimeType: img.Mime, Data: rawBase64(img.B64)}})
	}
	for _, a := range t.Attachments {
		mime := a.Mime
		if a.Kind == "text" {
			mime = "text/plain"
		}
		parts = append(parts, geminiPart{InlineData: &geminiInline{MimeType: mime, Data: a.B64}})
	}
	return parts
}

func buildGemini(cfg Config, turns []Turn, thinking *bool) Request {
	base := strings.TrimRight(cfg.GeminiBaseURL, "/")
	url := base + "/models/" + cfg.GeminiModel + ":generateContent?key=" + encodeURIComponent(cfg.GeminiAPIKey)

	contents := make([]geminiContent, 0, len(turns))
	for _, t := range turns {
		role := "user"
		if t.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{Role: role, Parts: geminiParts(t)})
	}
	body := geminiBody{Contents: contents}
	want := cfg.GeminiThinking
	if thinking != nil {
		want = *thinking
	}
	if want {
		budget := cfg.ThinkingBudget
		if budget == 0 {
			budget = thinkingStep
		}
		body.GenerationConfig = &geminiGenerationConfig{ThinkingConfig: geminiThinkingConfig{ThinkingBudget: budget}}
	}
	return Request{URL: url, Headers: map[string]string{"Content-Type": "application/json"}, Body: marshalBody(body)}
}

// --- OpenRouter ---

type orURL struct {
	URL string `json:"url"`
}

type orFile struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

type orAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type orPart struct {
	Type       string   `json:"type"`
	Text       *string  `json:"text,omitempty"`
	ImageURL   *orURL   `json:"image_url,omitempty"`
	File       *orFile  `json:"file,omitempty"`
	InputAudio *orAudio `json:"input_audio,omitempty"`
	VideoURL   *orURL   `json:"video_url,omitempty"`
}

type orMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type orReasoning struct {
	Effort string `json:"effort"`
}

type orBody struct {
	Model     string       `json:"model"`
	Messages  []orMessage  `json:"messages"`
	Reasoning *orReasoning `json:"reasoning,omitempty"`
}

func openRouterContent(t Turn) []orPart {
	var textFiles, media []Attachment
	for _, a := range t.Attachments {
		if a.Kind == "text" {
			textFiles = append(textFiles, a)
		} else {
			media = append(media, a)
		}
	}
	text := t.Text
	if len(textFiles) > 0 {
		text += formatTextAttachments(textFiles)
	}
	content := []orPart{{Type: "text", Text: &text}}
	for _, img := range t.Images {
		content = append(content, orPart{Type: "image_url", ImageURL: &orURL{URL: "data:" + img.Mime + ";base64," + rawBase64(img.B64)}})
	}
	for _, a := range media {
		switch a.Kind {
		case "image":
			content = append(content, orPart{Type: "image_url", ImageURL: &orURL{URL: "data:" + a.Mime + ";base64," + a.B64}})
		case "pdf":
			content = append(content, orPart{Type: "file", File: &orFile{Filename: a.Name, FileData: "data:application/pdf;base64," + a.B64}})
		case "audio":
			format := a.AudioFormat
			if format == "" {
				format = "mp3"
			}
			content = append(content, orPart{Type: "input_audio", InputAudio: &orAudio{Data: a.B64, Format: format}})
		case "video":
			content = append(content, orPart{Type: "video_url", VideoURL: &orURL{URL: "data:" + a.Mime + ";base64," + a.B64}})
		}
	}
	return content
}

func buildOpenRouter(cfg Config, turns []Turn, thinking *bool) Request {
	base := cfg.ORBaseURL
	if base == "" {
		base = defaultORBaseURL
	}
	url := strings.TrimRight(base, "/") + "/chat/completions"

	messages := make([]orMessage, 0, len(turns))
	for _, t := range turns {
		if t.Role == "assistant" {
			messages = append(messages, orMessage{Role: "assistant", Content: t.Text})
		} else {
			messages = append(messages, orMessage{Role: "user", Content: openRouterContent(t)})
		}
	}
	body := orBody{Model: cfg.ORModel, Messages: messages}
	want := cfg.ORReasoning
	if thinking != nil {
		want = *thinking
	}
	if want {
		effort := cfg.ORReasoningEffort
		if effort == "" {
			effort = "low"
		}
		body.Reasoning = &orReasoning{Effort: effort}
	}
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + cfg.ORAPIKey,
		"HTTP-Referer":  "https://github.com/chaice",
		"X-Title":       "ChAIce",
	}
	return Request{URL: url, Headers: headers, Body: marshalBody(body)}
}

// BuildConversation: provider "openrouter", anything else is Gemini (llm.js default).
func BuildConversation(provider string, cfg Config, turns []Turn, thinking *bool) Request {
	if provider == "openrouter" {
		return buildOpenRouter(cfg, turns, thinking)
	}
	return buildGemini(cfg, turns, thinking)
}

func ParseResponse(provider string, data json.RawMessage) (string, bool) {
	if provider == "openrouter" {
		var r struct {
			Choices []struct {
				Message struct {
					Content any `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if json.Unmarshal(data, &r) != nil || len(r.Choices) == 0 {
			return "", false
		}
		s, ok := r.Choices[0].Message.Content.(string)
		return s, ok
	}
	var r struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text any `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if json.Unmarshal(data, &r) != nil || len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", false
	}
	s, ok := r.Candidates[0].Content.Parts[0].Text.(string)
	return s, ok
}
```

- [ ] **Step 6: Run the tests and confirm they pass.** Run `GOTEST` with `-run 'Conversation|Mime|Parse|Config|Prompt' -v`. Expected: all PASS. If a fixture differs, fix the Go port. Never edit `fixtures.json` by hand.
- [ ] **Step 7: Commit.** `git commit -m "assistant: port llm.js, checked against llm.js output"` (add `llm.go`, `llm_test.go`, `testdata/`, `config.go`).

---

### Task 4: Transport (performApiRequest port)

**Files:**
- Create: `server/service/assistant/transport.go`
- Test: `server/service/assistant/transport_test.go`

**Interfaces:**
- Consumes: `Request` (Task 3).
- Produces: `type Result struct{ OK bool; Status int; Data json.RawMessage; BodyText, Route string }`, `type Transport struct{ Client *http.Client; AttemptTimeout, RetryGap time.Duration; MaxAttempts int }`, `NewTransport() *Transport`, `(*Transport).Send(ctx, req Request, proxyURL, proxyPass string) (Result, error)`

- [ ] **Step 1: Write the failing test**

```go
package assistant

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testTransport() *Transport {
	return &Transport{Client: &http.Client{}, AttemptTimeout: 2 * time.Second, RetryGap: time.Millisecond, MaxAttempts: 2}
}

func provider(t *testing.T, status int, body string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSendDirect(t *testing.T) {
	p := provider(t, 200, `{"a":1}`)
	r, err := testTransport().Send(context.Background(), Request{URL: p.URL, Body: []byte("{}")}, "", "")
	if err != nil || !r.OK || string(r.Data) != `{"a":1}` || r.Route != "direct (no proxy configured)" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendNonJSONBody(t *testing.T) {
	p := provider(t, 502, "bad gateway")
	r, err := testTransport().Send(context.Background(), Request{URL: p.URL}, "", "")
	if err != nil || r.OK || r.Status != 502 || r.BodyText != "bad gateway" || r.Data != nil {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendViaRelayUnwraps(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Target-URL") != "https://up/x" || r.Header.Get("X-Proxy-Pass") != "pw" || r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("relay headers %v", r.Header)
		}
		w.Header().Set("X-Relay-Wrap", "1")
		io.WriteString(w, `{"_relayWrap":1,"pad":"   ","status":200,"headers":{},"body":{"ok":true}}`)
	}))
	defer relay.Close()
	req := Request{URL: "https://up/x", Headers: map[string]string{"Authorization": "Bearer k"}}
	r, err := testTransport().Send(context.Background(), req, " "+relay.URL+" ", "pw")
	if err != nil || !r.OK || string(r.Data) != `{"ok":true}` || r.Route != "proxy "+relay.URL+" (attempt 1/2)" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendRelayStringBody(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Relay-Wrap", "1")
		io.WriteString(w, `{"_relayWrap":1,"pad":"","status":500,"body":"upstream down"}`)
	}))
	defer relay.Close()
	r, err := testTransport().Send(context.Background(), Request{URL: "https://up"}, relay.URL, "")
	if err != nil || r.OK || r.Status != 500 || r.BodyText != "upstream down" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendRelayAuthFailureIsReturnedNotRetried(t *testing.T) {
	calls := 0
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(401)
		io.WriteString(w, `{"error":{"message":"invalid proxy password"}}`)
	}))
	defer relay.Close()
	r, err := testTransport().Send(context.Background(), Request{URL: "https://up"}, relay.URL, "x")
	if err != nil || r.Status != 401 || calls != 1 {
		t.Fatalf("r=%+v err=%v calls=%d", r, err, calls)
	}
}

func TestSendFallsBackToDirect(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()
	p := provider(t, 200, `{"b":2}`)
	r, err := testTransport().Send(context.Background(), Request{URL: p.URL}, deadURL, "")
	want := "direct (FALLBACK after 2 failed proxy attempts to " + deadURL + ")"
	if err != nil || string(r.Data) != `{"b":2}` || r.Route != want {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendAttemptTimeoutCountsAsFailure(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer slow.Close()
	p := provider(t, 200, `{}`)
	tr := testTransport()
	tr.AttemptTimeout = 20 * time.Millisecond
	r, err := tr.Send(context.Background(), Request{URL: p.URL}, slow.URL, "")
	if err != nil || !strings.HasPrefix(r.Route, "direct (FALLBACK") {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendHonoursContextCancel(t *testing.T) {
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer hang.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := testTransport().Send(ctx, Request{URL: hang.URL}, "", "")
	if err == nil || time.Since(start) > time.Second {
		t.Fatalf("err=%v after %s", err, time.Since(start))
	}
}

func TestDirectFailureDoesNotLeakKey(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	u := dead.URL + "/models/m:generateContent?key=SECRETKEY"
	dead.Close()
	_, err := testTransport().Send(context.Background(), Request{URL: u}, "", "")
	if err == nil || strings.Contains(err.Error(), "SECRETKEY") {
		t.Fatalf("err=%v", err)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails.** Run `GOTEST` with `-run Send -v` (also covers `DirectFailure`). Expected: build failure.
- [ ] **Step 3: Implement**

```go
package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Port of background.js performApiRequest: direct when no proxy; otherwise the
// chaice relay, 2 attempts 5 s apart, then direct. Thinking calls can take
// minutes, so each attempt gets 300 s to produce response headers.

type Result struct {
	OK       bool
	Status   int
	Data     json.RawMessage
	BodyText string
	Route    string
}

type Transport struct {
	Client         *http.Client
	AttemptTimeout time.Duration
	RetryGap       time.Duration
	MaxAttempts    int
}

func NewTransport() *Transport {
	return &Transport{Client: &http.Client{}, AttemptTimeout: 300 * time.Second, RetryGap: 5 * time.Second, MaxAttempts: 2}
}

func (t *Transport) Send(ctx context.Context, req Request, proxyURL, proxyPass string) (Result, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		r, err := t.direct(ctx, req)
		r.Route = "direct (no proxy configured)"
		return r, err
	}

	for attempt := 1; attempt <= t.MaxAttempts; attempt++ {
		r, err := t.viaProxy(ctx, req, proxyURL, proxyPass)
		if err == nil {
			r.Route = fmt.Sprintf("proxy %s (attempt %d/%d)", proxyURL, attempt, t.MaxAttempts)
			return r, nil
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		log.Warnf("assistant: proxy attempt %d/%d failed: %v", attempt, t.MaxAttempts, err)
		if attempt < t.MaxAttempts {
			select {
			case <-time.After(t.RetryGap):
			case <-ctx.Done():
				return Result{}, ctx.Err()
			}
		}
	}
	log.Warnf("assistant: proxy unreachable after %d attempts; falling back to direct", t.MaxAttempts)
	r, err := t.direct(ctx, req)
	r.Route = fmt.Sprintf("direct (FALLBACK after %d failed proxy attempts to %s)", t.MaxAttempts, proxyURL)
	return r, err
}

// post returns once headers arrive; the attempt timeout stops there, as
// fetchWithTimeout's does. cancel must be called after the body is read.
func (t *Transport) post(ctx context.Context, target string, headers map[string]string, body []byte) (*http.Response, context.CancelFunc, error) {
	attemptCtx, cancel := context.WithCancel(ctx)
	timer := time.AfterFunc(t.AttemptTimeout, cancel)
	hr, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		timer.Stop()
		cancel()
		return nil, nil, redact(err)
	}
	for k, v := range headers {
		hr.Header.Set(k, v)
	}
	resp, err := t.Client.Do(hr)
	timer.Stop()
	if err != nil {
		cancel()
		return nil, nil, redact(err)
	}
	return resp, cancel, nil
}

// A *url.Error prints the URL, and a Gemini URL carries the API key.
func redact(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("%s request failed: %w", ue.Op, ue.Err)
	}
	return err
}

func (t *Transport) direct(ctx context.Context, req Request) (Result, error) {
	resp, cancel, err := t.post(ctx, req.URL, req.Headers, req.Body)
	if err != nil {
		return Result{}, err
	}
	defer cancel()
	defer resp.Body.Close()
	return readResponse(resp)
}

func (t *Transport) viaProxy(ctx context.Context, req Request, proxyURL, proxyPass string) (Result, error) {
	headers := make(map[string]string, len(req.Headers)+2)
	for k, v := range req.Headers {
		headers[k] = v
	}
	headers["X-Target-URL"] = req.URL
	if proxyPass != "" {
		headers["X-Proxy-Pass"] = proxyPass
	}
	resp, cancel, err := t.post(ctx, proxyURL, headers, req.Body)
	if err != nil {
		return Result{}, err
	}
	defer cancel()
	defer resp.Body.Close()

	if resp.Header.Get("X-Relay-Wrap") != "1" {
		return readResponse(resp)
	}
	var wrap struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrap); err != nil {
		return Result{}, fmt.Errorf("relay wrap: %w", err)
	}
	r := Result{OK: wrap.Status >= 200 && wrap.Status < 300, Status: wrap.Status}
	body := bytes.TrimSpace(wrap.Body)
	switch {
	case len(body) > 0 && (body[0] == '{' || body[0] == '['):
		r.Data = json.RawMessage(body)
	case len(body) > 0 && body[0] == '"':
		_ = json.Unmarshal(body, &r.BodyText)
	}
	return r, nil
}

func readResponse(resp *http.Response) (Result, error) {
	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, redact(err)
	}
	r := Result{OK: resp.StatusCode >= 200 && resp.StatusCode < 300, Status: resp.StatusCode}
	trimmed := bytes.TrimSpace(text)
	if len(trimmed) > 0 && json.Valid(trimmed) && !bytes.Equal(trimmed, []byte("null")) {
		r.Data = json.RawMessage(trimmed)
	} else if len(text) > 0 {
		r.BodyText = string(text)
	}
	return r, nil
}
```

- [ ] **Step 4: Run the tests and confirm they pass.** Run `GOTEST` with `-run 'Send|DirectFailure' -v`. Expected: PASS.
- [ ] **Step 5: Commit.** `git commit -m "assistant: relay transport with retry and direct fallback"`

---

### Task 5: Capture and crop

**Files:**
- Create: `server/service/assistant/capture.go`
- Test: `server/service/assistant/capture_test.go`

**Interfaces:**
- Produces: `type Frame struct{ JPEG []byte; Width, Height int }`, `type Capturer interface{ Capture(context.Context) (Frame, error) }`, `type Crop struct{ X, Y, W, H float64 }` (JSON tags `x`, `y`, `w`, `h`, fractions of the frame), `cropBounds(Crop, w, h int) (image.Rectangle, error)`, `cropJPEG([]byte, Crop) ([]byte, error)`, `ErrEmptyCrop`

- [ ] **Step 1: Write the failing test**

```go
package assistant

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"testing"
)

func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCropBounds(t *testing.T) {
	r, err := cropBounds(Crop{X: 0.25, Y: 0.5, W: 0.5, H: 0.25}, 1920, 1080)
	if err != nil || r != image.Rect(480, 540, 1440, 810) {
		t.Fatalf("r=%v err=%v", r, err)
	}
	// Outside the frame is clamped.
	r, err = cropBounds(Crop{X: -0.1, Y: 0.9, W: 0.3, H: 0.5}, 100, 100)
	if err != nil || r != image.Rect(0, 90, 20, 100) {
		t.Fatalf("clamp r=%v err=%v", r, err)
	}
}

func TestCropBoundsEmpty(t *testing.T) {
	for _, c := range []Crop{{X: 0.5, Y: 0.5}, {X: 1.2, Y: 0, W: 0.5, H: 1}, {X: math.NaN(), W: 1, H: 1}} {
		if _, err := cropBounds(c, 100, 100); !errors.Is(err, ErrEmptyCrop) {
			t.Fatalf("crop %+v: err=%v", c, err)
		}
	}
}

func TestCropJPEG(t *testing.T) {
	out, err := cropJPEG(testJPEG(t, 200, 100), Crop{X: 0.5, Y: 0, W: 0.5, H: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 100 || cfg.Height != 50 {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails.** Run `GOTEST` with `-run Crop -v`.
- [ ] **Step 3: Implement**

```go
package assistant

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"math"
)

var ErrEmptyCrop = errors.New("selection is empty")

type Frame struct {
	JPEG   []byte
	Width  int
	Height int
}

// Capturer is the device's screen capture; the router adapts the MCP backend
// so this package stays free of cgo.
type Capturer interface {
	Capture(ctx context.Context) (Frame, error)
}

// Crop is a selection as fractions of the video frame.
type Crop struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

func cropBounds(c Crop, w, h int) (image.Rectangle, error) {
	for _, v := range []float64{c.X, c.Y, c.W, c.H} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return image.Rectangle{}, ErrEmptyCrop
		}
	}
	x0 := math.Floor(c.X * float64(w))
	y0 := math.Floor(c.Y * float64(h))
	x1 := math.Ceil((c.X + c.W) * float64(w))
	y1 := math.Ceil((c.Y + c.H) * float64(h))
	clamp := func(v float64, max int) int { return int(math.Max(0, math.Min(v, float64(max)))) }
	r := image.Rect(clamp(x0, w), clamp(y0, h), clamp(x1, w), clamp(y1, h))
	if r.Empty() {
		return image.Rectangle{}, ErrEmptyCrop
	}
	return r, nil
}

func cropJPEG(data []byte, c Crop) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode capture: %w", err)
	}
	b := img.Bounds()
	r, err := cropBounds(c, b.Dx(), b.Dy())
	if err != nil {
		return nil, err
	}
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil, errors.New("capture cannot be cropped")
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, sub.SubImage(r.Add(b.Min)), &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("encode crop: %w", err)
	}
	return out.Bytes(), nil
}
```

- [ ] **Step 4: Run the tests and confirm they pass.** Run `GOTEST` with `-run Crop -v`.
- [ ] **Step 5: Commit.** `git commit -m "assistant: crop captures by frame fractions"`

---

### Task 6: Context images and attachments

**Files:**
- Create: `server/service/assistant/contexts.go`, `server/service/assistant/attachments.go`
- Test: `server/service/assistant/store_test.go`

**Interfaces:**
- Consumes: `Image`, `Attachment`, `classifyFile` (Task 3).
- Produces:
  - `type Contexts struct` with `Add(Image) (int, error)`, `Count() int`, `Clear()`, `Snapshot() []Image`; `ErrContextsFull`
  - `type AttachmentInfo struct{ Name string \`json:"name"\`; Size int64 \`json:"size"\` }`
  - `NewAttachmentStore(dir string) *AttachmentStore` with `List() ([]AttachmentInfo, error)`, `Put(name string, r io.Reader) error`, `Delete(name string) error`, `Load() []Attachment`
  - `AttachmentsDir`, `ErrAttachmentName`, `ErrAttachmentsFull`

- [ ] **Step 1: Write the failing test**

```go
package assistant

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContexts(t *testing.T) {
	var c Contexts
	if n, _ := c.Add(Image{Mime: "image/jpeg", B64: "YQ=="}); n != 1 {
		t.Fatalf("n=%d", n)
	}
	snap := c.Snapshot()
	c.Add(Image{Mime: "image/jpeg", B64: "Yg=="})
	if len(snap) != 1 || c.Count() != 2 {
		t.Fatalf("snapshot aliased or count wrong")
	}
	c.Clear()
	if c.Count() != 0 {
		t.Fatal("not cleared")
	}
	if _, err := c.Add(Image{B64: strings.Repeat("a", maxContextBytes+1)}); !errors.Is(err, ErrContextsFull) {
		t.Fatalf("err=%v", err)
	}
}

func TestAttachmentStore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "att")
	s := NewAttachmentStore(dir)
	if list, err := s.List(); err != nil || len(list) != 0 {
		t.Fatalf("missing dir: %v %v", list, err)
	}
	for _, bad := range []string{"", ".", "..", "../x", "a/b", ".hidden", "index.json"} {
		if err := s.Put(bad, strings.NewReader("x")); !errors.Is(err, ErrAttachmentName) {
			t.Fatalf("name %q: err=%v", bad, err)
		}
	}
	s.Put("b.txt", strings.NewReader("﻿hello"))
	s.Put("a.pdf", strings.NewReader("%PDF"))
	s.Put("bin.txt", bytes.NewReader([]byte{'a', 0, 'b'}))
	list, _ := s.List()
	if len(list) != 3 || list[0].Name != "a.pdf" || list[1].Name != "b.txt" {
		t.Fatalf("list %+v", list)
	}
	loaded := s.Load()
	// The NUL-containing text file is skipped, the BOM is stripped (TextDecoder).
	if len(loaded) != 2 || loaded[0].Kind != "pdf" || loaded[1].Text != "hello" || loaded[1].B64 == "" {
		t.Fatalf("loaded %+v", loaded)
	}
	info, _ := os.Stat(filepath.Join(dir, "b.txt"))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode())
	}
	if err := s.Put("big.bin", bytes.NewReader(make([]byte, maxAttachmentBytes))); !errors.Is(err, ErrAttachmentsFull) {
		t.Fatalf("cap err=%v", err)
	}
	if err := s.Delete("b.txt"); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.List(); len(list) != 2 {
		t.Fatalf("after delete %+v", list)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails.** Run `GOTEST` with `-run 'Contexts|AttachmentStore' -v`.
- [ ] **Step 3: Implement `contexts.go`**

```go
package assistant

import (
	"errors"
	"sync"
)

// Context images live in memory, per device (the extension kept them per
// browser profile). T and a server restart clear them.
const maxContextBytes = 16 << 20

var ErrContextsFull = errors.New("context images exceed 16 MB; clear them with T")

type Contexts struct {
	mu     sync.Mutex
	images []Image
	size   int
}

func (c *Contexts) Add(img Image) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.size+len(img.B64) > maxContextBytes {
		return len(c.images), ErrContextsFull
	}
	c.images = append(c.images, img)
	c.size += len(img.B64)
	return len(c.images), nil
}

func (c *Contexts) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.images)
}

func (c *Contexts) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.images = nil
	c.size = 0
}

func (c *Contexts) Snapshot() []Image {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Image(nil), c.images...)
}
```

- [ ] **Step 4: Implement `attachments.go`**

```go
package assistant

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

// Files uploaded in settings, sent with every ask exactly as the extension
// sends its bundled attachments/ folder.
const (
	AttachmentsDir     = "/etc/kvm/assistant/attachments"
	maxAttachmentBytes = 20 << 20
)

var (
	ErrAttachmentName  = errors.New("invalid attachment name")
	ErrAttachmentsFull = errors.New("attachments exceed 20 MB")
	skippedNames       = map[string]bool{".gitignore": true, "index.json": true, ".DS_Store": true}
)

type AttachmentInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type AttachmentStore struct {
	dir string
	mu  sync.Mutex
}

func NewAttachmentStore(dir string) *AttachmentStore {
	return &AttachmentStore{dir: dir}
}

func validAttachmentName(name string) bool {
	return name != "" && name == filepath.Base(name) && !strings.HasPrefix(name, ".") &&
		!skippedNames[name] && !strings.ContainsAny(name, "/\\\x00")
}

func (s *AttachmentStore) List() ([]AttachmentInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list()
}

func (s *AttachmentStore) list() ([]AttachmentInfo, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return []AttachmentInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []AttachmentInfo{}
	for _, e := range entries {
		if !validAttachmentName(e.Name()) || !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, AttachmentInfo{Name: e.Name(), Size: info.Size()})
	}
	return out, nil
}

func (s *AttachmentStore) Put(name string, r io.Reader) error {
	if !validAttachmentName(name) {
		return ErrAttachmentName
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.list()
	if err != nil {
		return err
	}
	var used int64
	for _, a := range existing {
		if a.Name != name {
			used += a.Size
		}
	}
	data, err := io.ReadAll(io.LimitReader(r, maxAttachmentBytes-used+1))
	if err != nil {
		return err
	}
	if used+int64(len(data)) > maxAttachmentBytes {
		return ErrAttachmentsFull
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	return utils.WriteFileAtomic(filepath.Join(s.dir, name), data, 0o600)
}

func (s *AttachmentStore) Delete(name string) error {
	if !validAttachmentName(name) {
		return ErrAttachmentName
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(filepath.Join(s.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Load mirrors content.js loadAttachments: classify by extension, base64
// everything, decode text files and skip ones that look binary.
func (s *AttachmentStore) Load() []Attachment {
	s.mu.Lock()
	defer s.mu.Unlock()
	infos, err := s.list()
	if err != nil {
		log.Warnf("assistant: list attachments: %v", err)
		return nil
	}
	var out []Attachment
	for _, info := range infos {
		data, err := os.ReadFile(filepath.Join(s.dir, info.Name))
		if err != nil {
			log.Warnf("assistant: attachment load failed: %s: %v", info.Name, err)
			continue
		}
		ft := classifyFile(info.Name)
		a := Attachment{Name: info.Name, Kind: ft.Kind, Mime: ft.Mime, AudioFormat: ft.AudioFormat, B64: base64.StdEncoding.EncodeToString(data)}
		if ft.Kind == "text" {
			if bytes.IndexByte(data, 0) >= 0 {
				log.Warnf("assistant: skipping %s: looks binary, not text", info.Name)
				continue
			}
			a.Text = strings.TrimPrefix(strings.ToValidUTF8(string(data), "�"), "﻿")
		}
		out = append(out, a)
	}
	return out
}
```

- [ ] **Step 5: Run the tests and confirm they pass.** Run `GOTEST` with `-run 'Contexts|AttachmentStore' -v`.
- [ ] **Step 6: Commit.** `git commit -m "assistant: context images and uploaded attachments"`

---

### Task 7: Ask flows and reasoning adjust

**Files:**
- Create: `server/service/assistant/service.go` (Service, flows), `server/service/assistant/reasoning.go`
- Test: `server/service/assistant/service_test.go`, `server/service/assistant/reasoning_test.go`

**Interfaces:**
- Consumes: everything above.
- Produces:
  - `type Service struct{ capture Capturer; transport *Transport; contexts *Contexts; attachments *AttachmentStore; prompts func() (map[string]any, error); loadConfig func() (Config, error) }`
  - `NewService(capture Capturer) *Service`
  - `type AskRequest struct{ Kind string \`json:"kind"\`; Crop *Crop \`json:"crop"\`; Text string \`json:"text"\` }`
  - `type AskResult struct{ Answer string \`json:"answer"\`; Route string \`json:"route"\` }`
  - `(*Service).Ask(ctx, AskRequest) (AskResult, error)`, `AddContext(ctx, *Crop) (int, error)`, `ClearContexts() error`, `ContextCount() (int, error)`, `Screenshot(ctx) ([]byte, error)`, `AdjustReasoning(up bool) (ReasoningResult, error)`
  - `type ReasoningResult struct{ Label string \`json:"label"\`; Changed bool \`json:"changed"\`; Config PublicConfig \`json:"config"\` }`
  - `adjustReasoning(Config, up bool) (Config, string, bool)`
  - `ErrDisabled`, `ErrNoAnswer`, `ErrInvalidKind`, `ackPrompt`

- [ ] **Step 1: Write the failing reasoning test** (a port of `nextGemini` / `nextOpenRouter` and their labels)

```go
package assistant

import "testing"

func TestAdjustReasoning(t *testing.T) {
	g := func(on bool, b int) Config { c := defaultConfig(); c.Provider = "gemini"; c.GeminiThinking = on; c.ThinkingBudget = b; return c }
	o := func(on bool, e string) Config { c := defaultConfig(); c.ORReasoning = on; c.ORReasoningEffort = e; return c }
	cases := []struct {
		name    string
		in      Config
		up      bool
		label   string
		changed bool
		check   func(Config) bool
	}{
		{"gemini off up", g(false, 2048), true, "Thinking: 512", true, func(c Config) bool { return c.GeminiThinking && c.ThinkingBudget == 512 }},
		{"gemini on up", g(true, 512), true, "Thinking: 1024", true, func(c Config) bool { return c.ThinkingBudget == 1024 }},
		{"gemini 512 down", g(true, 512), false, "Thinking: off", true, func(c Config) bool { return !c.GeminiThinking && c.ThinkingBudget == 512 }},
		{"gemini 1536 down", g(true, 1536), false, "Thinking: 1024", true, func(c Config) bool { return c.GeminiThinking && c.ThinkingBudget == 1024 }},
		{"gemini off down", g(false, 512), false, "Thinking: off (already)", false, func(c Config) bool { return !c.GeminiThinking }},
		{"gemini zero budget up", g(true, 0), true, "Thinking: 1024", true, func(c Config) bool { return c.ThinkingBudget == 1024 }},
		{"or off up", o(false, "high"), true, "Reasoning: low", true, func(c Config) bool { return c.ORReasoning && c.ORReasoningEffort == "low" }},
		{"or low up", o(true, "low"), true, "Reasoning: medium", true, func(c Config) bool { return c.ORReasoningEffort == "medium" }},
		{"or low down", o(true, "low"), false, "Reasoning: off", true, func(c Config) bool { return !c.ORReasoning && c.ORReasoningEffort == "low" }},
		{"or off down", o(false, "medium"), false, "Reasoning: off", true, func(c Config) bool { return !c.ORReasoning && c.ORReasoningEffort == "medium" }},
		// content.js labels every no-op "off (already)", even at high.
		{"or high up", o(true, "high"), true, "Reasoning: off (already)", false, func(c Config) bool { return c.ORReasoning && c.ORReasoningEffort == "high" }},
	}
	for _, c := range cases {
		next, label, changed := adjustReasoning(c.in, c.up)
		if label != c.label || changed != c.changed || !c.check(next) {
			t.Fatalf("%s: label=%q changed=%v cfg=%+v", c.name, label, changed, next)
		}
	}
}
```

- [ ] **Step 2: Write the failing flow tests**

```go
package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeCapturer struct {
	jpeg  []byte
	calls int
}

func (f *fakeCapturer) Capture(context.Context) (Frame, error) {
	f.calls++
	return Frame{JPEG: f.jpeg, Width: 200, Height: 100}, nil
}

type orServer struct {
	bodies  []map[string]any
	replies []string // one per call; "" means {"choices":[]}
	status  int
}

func (o *orServer) start(t *testing.T) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		o.bodies = append(o.bodies, body)
		if o.status != 0 {
			w.WriteHeader(o.status)
			io.WriteString(w, `{"error":{"message":"bad key"}}`)
			return
		}
		reply := o.replies[len(o.bodies)-1]
		if reply == "" {
			io.WriteString(w, `{"choices":[]}`)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": reply}}}})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func testService(t *testing.T, providerURL string, cap *fakeCapturer) *Service {
	cfg := defaultConfig()
	cfg.Enabled = true
	cfg.ORBaseURL = providerURL
	cfg.ORModel = "om"
	cfg.ORAPIKey = "sk"
	p, _ := parsePrompts([]byte("[universal]\nprompt = \"U\"\n[mcq]\nprompt = \"M\"\n[frq]\nprompt = \"F\"\n[custom]\nprompt = \"C\"\n"))
	return &Service{
		capture:     cap,
		transport:   &Transport{Client: &http.Client{}, AttemptTimeout: 5 * time.Second, RetryGap: time.Millisecond, MaxAttempts: 2},
		contexts:    &Contexts{},
		attachments: NewAttachmentStore(filepath.Join(t.TempDir(), "att")),
		prompts:     func() (map[string]any, error) { return p, nil },
		loadConfig:  func() (Config, error) { return cfg, nil },
	}
}

func userContent(t *testing.T, body map[string]any, i int) []any {
	t.Helper()
	msgs := body["messages"].([]any)
	return msgs[i].(map[string]any)["content"].([]any)
}

func partText(p any) string { return p.(map[string]any)["text"].(string) }
func partURL(p any) string {
	return p.(map[string]any)["image_url"].(map[string]any)["url"].(string)
}

func TestAskMCQSendsContextsThenScreenshot(t *testing.T) {
	or := &orServer{replies: []string{"B"}}
	cap := &fakeCapturer{jpeg: testJPEG(t, 200, 100)}
	s := testService(t, or.start(t), cap)
	s.contexts.Add(Image{Mime: "image/jpeg", B64: "Y3R4"})

	res, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"})
	if err != nil || res.Answer != "B" || res.Route != "direct (no proxy configured)" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	content := userContent(t, or.bodies[0], 0)
	if len(content) != 3 || partText(content[0]) != "U\n\nM" || partURL(content[1]) != "data:image/jpeg;base64,Y3R4" ||
		!strings.HasPrefix(partURL(content[2]), "data:image/jpeg;base64,") {
		t.Fatalf("content %v", content)
	}
}

func TestAskFRQWithCropAndAttachmentsUsesTwoTurns(t *testing.T) {
	or := &orServer{replies: []string{"  ", "answer"}}
	cap := &fakeCapturer{jpeg: testJPEG(t, 200, 100)}
	s := testService(t, or.start(t), cap)
	cfg, _ := s.loadConfig()
	cfg.ORReasoning = true
	s.loadConfig = func() (Config, error) { return cfg, nil }
	s.attachments.Put("notes.txt", strings.NewReader("hello"))

	res, err := s.Ask(context.Background(), AskRequest{Kind: "frq", Crop: &Crop{X: 0, Y: 0, W: 0.5, H: 0.5}})
	if err != nil || res.Answer != "answer" || len(or.bodies) != 2 {
		t.Fatalf("res=%+v err=%v calls=%d", res, err, len(or.bodies))
	}
	if _, ok := or.bodies[0]["reasoning"]; ok {
		t.Fatal("anchor turn must not reason")
	}
	if partText(userContent(t, or.bodies[0], 0)[0]) != ackPrompt {
		t.Fatalf("anchor text %v", userContent(t, or.bodies[0], 0)[0])
	}
	msgs := or.bodies[1]["messages"].([]any)
	if len(msgs) != 3 || msgs[1].(map[string]any)["content"] != "OK" {
		t.Fatalf("history %v", msgs)
	}
	want := "U\n\nF\n\nNow answer the question above." + "\n\nThe following text files were attached:\n\n--- notes.txt ---\nhello\n"
	if got := partText(userContent(t, or.bodies[1], 2)[0]); got != want {
		t.Fatalf("turn 2 text %q", got)
	}
	if _, ok := or.bodies[1]["reasoning"]; !ok {
		t.Fatal("answer turn must reason")
	}
}

func TestAskCustomUsesTextAndContextsOnly(t *testing.T) {
	or := &orServer{replies: []string{"reply"}}
	cap := &fakeCapturer{}
	s := testService(t, or.start(t), cap)
	s.contexts.Add(Image{Mime: "image/jpeg", B64: "Y3R4"})
	res, err := s.Ask(context.Background(), AskRequest{Kind: "custom", Text: "  why?  "})
	if err != nil || res.Answer != "reply" || cap.calls != 0 {
		t.Fatalf("res=%+v err=%v captures=%d", res, err, cap.calls)
	}
	content := userContent(t, or.bodies[0], 0)
	if partText(content[0]) != "why?\n\nU\n\nC" || len(content) != 2 {
		t.Fatalf("content %v", content)
	}
}

func TestAskOutcomes(t *testing.T) {
	or := &orServer{replies: []string{""}}
	s := testService(t, or.start(t), &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"}); !errors.Is(err, ErrNoAnswer) {
		t.Fatalf("empty choices: %v", err)
	}
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "custom", Text: "  "}); !errors.Is(err, ErrNoAnswer) {
		t.Fatalf("empty custom: %v", err)
	}
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "essay"}); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("kind: %v", err)
	}
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq", Crop: &Crop{X: 0.5, Y: 0.5}}); !errors.Is(err, ErrEmptyCrop) {
		t.Fatalf("empty crop: %v", err)
	}

	bad := &orServer{status: 401}
	s = testService(t, bad.start(t), &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"}); err == nil || err.Error() != "bad key" {
		t.Fatalf("provider error: %v", err)
	}

	cfg := defaultConfig()
	s.loadConfig = func() (Config, error) { return cfg, nil }
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled: %v", err)
	}
}

func TestAddContextAndReasoningPersist(t *testing.T) {
	useTestConfig(t)
	updateConfig(func(c Config) (Config, error) { c.Enabled = true; return c, nil })
	s := testService(t, "http://unused", &fakeCapturer{jpeg: testJPEG(t, 200, 100)})
	s.loadConfig = loadConfig
	if n, err := s.AddContext(context.Background(), &Crop{X: 0, Y: 0, W: 1, H: 1}); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	res, err := s.AdjustReasoning(true)
	if err != nil || res.Label != "Reasoning: low" || !res.Config.ORReasoning {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if cfg, _ := loadConfig(); !cfg.ORReasoning {
		t.Fatal("reasoning change not persisted")
	}
	_ = os.Remove // keep import for fixture helpers above
}
```

(Delete the `_ = os.Remove` line and the `os` import if nothing else uses them.)

- [ ] **Step 3: Run them and confirm they fail.** Run `GOTEST` with `-run 'Ask|Reasoning|AddContext' -v`.
- [ ] **Step 4: Implement `reasoning.go`**

```go
package assistant

import "fmt"

var orEfforts = []string{"low", "medium", "high"}

// adjustReasoning ports content.js nextGemini / nextOpenRouter and their labels.
func adjustReasoning(c Config, up bool) (Config, string, bool) {
	if c.Provider == "openrouter" {
		effort := c.ORReasoningEffort
		if effort == "" {
			effort = "low"
		}
		idx := -1
		if c.ORReasoning {
			idx = 0
			for i, e := range orEfforts {
				if e == effort {
					idx = i
				}
			}
		}
		next := idx - 1
		if up {
			next = idx + 1
		}
		switch {
		case next < 0:
			c.ORReasoning, c.ORReasoningEffort = false, effort
			return c, "Reasoning: off", true
		case next > len(orEfforts)-1:
			return c, "Reasoning: off (already)", false
		}
		c.ORReasoning, c.ORReasoningEffort = true, orEfforts[next]
		return c, "Reasoning: " + orEfforts[next], true
	}

	budget := c.ThinkingBudget
	if budget == 0 {
		budget = thinkingStep
	}
	switch {
	case up && c.GeminiThinking:
		c.ThinkingBudget = budget + thinkingStep
	case up:
		c.ThinkingBudget = thinkingStep
	case !c.GeminiThinking:
		return c, "Thinking: off (already)", false
	case budget <= thinkingStep:
		c.GeminiThinking, c.ThinkingBudget = false, thinkingStep
		return c, "Thinking: off", true
	default:
		c.ThinkingBudget = budget - thinkingStep
	}
	c.GeminiThinking = true
	return c, fmt.Sprintf("Thinking: %d", c.ThinkingBudget), true
}
```

- [ ] **Step 5: Implement `service.go`**

```go
package assistant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	ErrDisabled    = errors.New("assistant is disabled")
	ErrNoAnswer    = errors.New("no answer")
	ErrInvalidKind = errors.New("unknown ask kind")
)

// Verbatim from content.js askLLM.
const ackPrompt = `Above is the question and its context. Reply with only "OK" to confirm — ` +
	"do not answer yet. I will then send the remaining information and the " +
	"instructions, and you will answer."

type Service struct {
	capture     Capturer
	transport   *Transport
	contexts    *Contexts
	attachments *AttachmentStore
	prompts     func() (map[string]any, error)
	loadConfig  func() (Config, error)
}

func NewService(capture Capturer) *Service {
	return &Service{
		capture:     capture,
		transport:   NewTransport(),
		contexts:    &Contexts{},
		attachments: NewAttachmentStore(AttachmentsDir),
		prompts:     loadPrompts,
		loadConfig:  loadConfig,
	}
}

type AskRequest struct {
	Kind string `json:"kind"`
	Crop *Crop  `json:"crop"`
	Text string `json:"text"`
}

type AskResult struct {
	Answer string `json:"answer"`
	Route  string `json:"route"`
}

type ReasoningResult struct {
	Label   string       `json:"label"`
	Changed bool         `json:"changed"`
	Config  PublicConfig `json:"config"`
}

func (s *Service) enabledConfig() (Config, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return Config{}, err
	}
	if !cfg.Enabled {
		return Config{}, ErrDisabled
	}
	return cfg, nil
}

func (s *Service) captureJPEG(ctx context.Context, crop *Crop) ([]byte, error) {
	frame, err := s.capture.Capture(ctx)
	if err != nil {
		return nil, fmt.Errorf("capture: %w", err)
	}
	if crop == nil {
		return frame.JPEG, nil
	}
	return cropJPEG(frame.JPEG, *crop)
}

func jpegImage(data []byte) Image {
	return Image{Mime: "image/jpeg", B64: base64.StdEncoding.EncodeToString(data)}
}

// Ask runs content.js handleAnswer (mcq), handleFRQAnswer (frq) or
// handleCustomMessage (custom) up to the answer.
func (s *Service) Ask(ctx context.Context, req AskRequest) (AskResult, error) {
	cfg, err := s.enabledConfig()
	if err != nil {
		return AskResult{}, err
	}
	var questionText string
	var images []Image
	switch req.Kind {
	case "mcq", "frq":
		shot, err := s.captureJPEG(ctx, req.Crop)
		if err != nil {
			return AskResult{}, err
		}
		images = append(s.contexts.Snapshot(), jpegImage(shot))
	case "custom":
		if strings.TrimSpace(req.Text) == "" {
			return AskResult{}, ErrNoAnswer
		}
		questionText = req.Text
		images = s.contexts.Snapshot()
	default:
		return AskResult{}, ErrInvalidKind
	}
	p, err := s.prompts()
	if err != nil {
		return AskResult{}, err
	}
	answer, route, err := s.askLLM(ctx, cfg, questionText, images, getPrompt(p, req.Kind), s.attachments.Load())
	if err != nil {
		return AskResult{Route: route}, err
	}
	if answer == "" {
		return AskResult{Route: route}, ErrNoAnswer
	}
	return AskResult{Answer: answer, Route: route}, nil
}

// askLLM ports content.js askLLM: one turn, or with attachments a question
// anchor answered "OK" (thinking off) followed by the attachments and prompt.
func (s *Service) askLLM(ctx context.Context, cfg Config, questionText string, images []Image, answerPrompt string, atts []Attachment) (string, string, error) {
	provider := cfg.Provider
	qText := strings.TrimSpace(questionText)
	prompt := strings.TrimSpace(answerPrompt)
	lead := ""
	if qText != "" {
		lead = qText + "\n\n"
	}

	if len(atts) == 0 {
		turn := Turn{Role: "user", Text: lead + prompt, Images: images}
		data, route, err := s.apiSend(ctx, cfg, BuildConversation(provider, cfg, []Turn{turn}, nil), "API")
		if err != nil {
			return "", route, err
		}
		answer, _ := ParseResponse(provider, data)
		return answer, route, nil
	}

	turn1 := Turn{Role: "user", Text: lead + ackPrompt, Images: images}
	off := false
	data1, route, err := s.apiSend(ctx, cfg, BuildConversation(provider, cfg, []Turn{turn1}, &off), "Q-anchor")
	if err != nil {
		return "", route, err
	}
	ack, _ := ParseResponse(provider, data1)
	if strings.TrimSpace(ack) == "" {
		ack = "OK"
	}
	turn2 := Turn{Role: "user", Text: prompt + "\n\nNow answer the question above.", Attachments: atts}
	turns := []Turn{turn1, {Role: "assistant", Text: ack}, turn2}
	data2, route, err := s.apiSend(ctx, cfg, BuildConversation(provider, cfg, turns, nil), "API")
	if err != nil {
		return "", route, err
	}
	answer, _ := ParseResponse(provider, data2)
	return answer, route, nil
}

// apiSend ports content.js apiSend: any failure or non-2xx is an error whose
// text is the provider's error.message, else "HTTP n[: first 300 chars]".
func (s *Service) apiSend(ctx context.Context, cfg Config, req Request, label string) (json.RawMessage, string, error) {
	r, err := s.transport.Send(ctx, req, cfg.ProxyURL, cfg.ProxyPass)
	if err != nil {
		return nil, r.Route, err
	}
	log.Infof("assistant: %s route: %s", label, r.Route)
	if r.OK {
		return r.Data, r.Route, nil
	}
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if r.Data != nil && json.Unmarshal(r.Data, &e) == nil && e.Error.Message != "" {
		return nil, r.Route, errors.New(e.Error.Message)
	}
	if r.BodyText != "" {
		text := []rune(r.BodyText)
		if len(text) > 300 {
			text = text[:300]
		}
		return nil, r.Route, fmt.Errorf("HTTP %d: %s", r.Status, string(text))
	}
	return nil, r.Route, fmt.Errorf("HTTP %d", r.Status)
}

func (s *Service) AddContext(ctx context.Context, crop *Crop) (int, error) {
	if _, err := s.enabledConfig(); err != nil {
		return 0, err
	}
	shot, err := s.captureJPEG(ctx, crop)
	if err != nil {
		return s.contexts.Count(), err
	}
	return s.contexts.Add(jpegImage(shot))
}

func (s *Service) ClearContexts() error {
	if _, err := s.enabledConfig(); err != nil {
		return err
	}
	s.contexts.Clear()
	return nil
}

func (s *Service) ContextCount() (int, error) {
	if _, err := s.enabledConfig(); err != nil {
		return 0, err
	}
	return s.contexts.Count(), nil
}

func (s *Service) Screenshot(ctx context.Context) ([]byte, error) {
	if _, err := s.enabledConfig(); err != nil {
		return nil, err
	}
	return s.captureJPEG(ctx, nil)
}

func (s *Service) AdjustReasoning(up bool) (ReasoningResult, error) {
	var res ReasoningResult
	cfg, err := updateConfig(func(c Config) (Config, error) {
		if !c.Enabled {
			return c, ErrDisabled
		}
		next, label, changed := adjustReasoning(c, up)
		res.Label, res.Changed = label, changed
		return next, nil
	})
	if err != nil {
		return ReasoningResult{}, err
	}
	res.Config = publicConfig(cfg)
	return res, nil
}
```

- [ ] **Step 6: Run the whole package's tests and confirm they pass.** Run `GOTEST` with `-v`.
- [ ] **Step 7: Commit.** `git commit -m "assistant: ask flows and reasoning adjust from content.js"`

---

### Task 8: HTTP handlers and router

**Files:**
- Create: `server/service/assistant/handlers.go`, `server/router/assistant.go`
- Modify: `server/router/router.go` (call `assistantRouter(r)` right after `mcpRouter(r, control, picoclawService)`)
- Test: `server/service/assistant/handlers_test.go`

**Interfaces:**
- Consumes: `Service` (Task 7), `proto.Response`.
- Produces: `NewHandler(*Service) *Handler`, `(*Handler).Register(gin.IRoutes)`, and the codes `codeError = -1`, `codeDisabled = -2`, `codeNoAnswer = -3`. The routes are listed in the spec. `DELETE /api/assistant/attachments` takes `?name=`; the upload is multipart field `file`.

- [ ] **Step 1: Write the failing test**

```go
package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func testEngine(t *testing.T, s *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(s).Register(r.Group("/api/assistant"))
	return r
}

func call(t *testing.T, r *gin.Engine, method, path, body string) (int, map[string]any, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	code, _ := out["code"].(float64)
	return int(code), out, w
}

func TestConfigEndpointsMaskSecrets(t *testing.T) {
	useTestConfig(t)
	s := testService(t, "http://unused", &fakeCapturer{})
	s.loadConfig = loadConfig
	r := testEngine(t, s)

	code, out, _ := call(t, r, "POST", "/api/assistant/config", `{"enabled":true,"orApiKey":"sk-secret"}`)
	if code != 0 || strings.Contains(mustJSON(out), "sk-secret") || out["data"].(map[string]any)["hasOrApiKey"] != true {
		t.Fatalf("post %v", out)
	}
	code, out, _ = call(t, r, "POST", "/api/assistant/config", `{"provider":"nope"}`)
	if code != codeError {
		t.Fatalf("invalid provider accepted: %v", out)
	}
	_, out, _ = call(t, r, "GET", "/api/assistant/config", "")
	if strings.Contains(mustJSON(out), "sk-secret") || out["data"].(map[string]any)["enabled"] != true {
		t.Fatalf("get %v", out)
	}
}

func TestAskEndpointCodes(t *testing.T) {
	or := &orServer{replies: []string{"", "A"}}
	s := testService(t, or.start(t), &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	r := testEngine(t, s)
	if code, out, _ := call(t, r, "POST", "/api/assistant/ask", `{"kind":"mcq"}`); code != codeNoAnswer {
		t.Fatalf("no answer: %v", out)
	}
	code, out, _ := call(t, r, "POST", "/api/assistant/ask", `{"kind":"mcq"}`)
	if code != 0 || out["data"].(map[string]any)["answer"] != "A" {
		t.Fatalf("answer: %v", out)
	}
	cfg := defaultConfig()
	s.loadConfig = func() (Config, error) { return cfg, nil }
	if code, _, _ := call(t, r, "POST", "/api/assistant/ask", `{"kind":"mcq"}`); code != codeDisabled {
		t.Fatalf("disabled code %d", code)
	}
}

func TestContextScreenshotAndAttachmentEndpoints(t *testing.T) {
	s := testService(t, "http://unused", &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	r := testEngine(t, s)
	if code, out, _ := call(t, r, "POST", "/api/assistant/context", `{}`); code != 0 || out["data"].(map[string]any)["count"] != 1.0 {
		t.Fatalf("add context %v", out)
	}
	if _, out, _ := call(t, r, "DELETE", "/api/assistant/context", ""); out["data"].(map[string]any)["count"] != 0.0 {
		t.Fatalf("clear %v", out)
	}
	_, _, w := call(t, r, "GET", "/api/assistant/screenshot", "")
	if w.Header().Get("Content-Type") != "image/jpeg" || w.Body.Len() == 0 {
		t.Fatalf("screenshot %v", w.Header())
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "notes.txt")
	fw.Write([]byte("hi"))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/assistant/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_, out, _ := call(t, r, "GET", "/api/assistant/attachments", "")
	if list := out["data"].([]any); len(list) != 1 {
		t.Fatalf("list %v", out)
	}
	call(t, r, "DELETE", "/api/assistant/attachments?name=notes.txt", "")
	_, out, _ = call(t, r, "GET", "/api/assistant/attachments", "")
	if list := out["data"].([]any); len(list) != 0 {
		t.Fatalf("after delete %v", out)
	}
	_ = context.Background
	_ = http.StatusOK
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
```

(Remove the `_ = context.Background` and `_ = http.StatusOK` lines, and their imports, if they turn out to be unused.)

- [ ] **Step 2: Run it and confirm it fails.** Run `GOTEST` with `-run 'Endpoint' -v`.
- [ ] **Step 3: Implement `handlers.go`**

```go
package assistant

import (
	"errors"
	"net/http"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	codeError    = -1
	codeDisabled = -2
	codeNoAnswer = -3
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(g gin.IRoutes) {
	g.GET("/config", h.GetConfig)
	g.POST("/config", h.SetConfig)
	g.POST("/ask", h.Ask)
	g.GET("/context", h.ContextCount)
	g.POST("/context", h.AddContext)
	g.DELETE("/context", h.ClearContexts)
	g.GET("/screenshot", h.Screenshot)
	g.POST("/reasoning", h.Reasoning)
	g.GET("/attachments", h.ListAttachments)
	g.POST("/attachments", h.UploadAttachment)
	g.DELETE("/attachments", h.DeleteAttachment)
}

func respondErr(c *gin.Context, err error, data any) {
	var rsp proto.Response
	rsp.Data = data
	switch {
	case errors.Is(err, ErrDisabled):
		rsp.ErrRsp(c, codeDisabled, err.Error())
	case errors.Is(err, ErrNoAnswer):
		rsp.ErrRsp(c, codeNoAnswer, err.Error())
	default:
		rsp.ErrRsp(c, codeError, err.Error())
	}
}

func (h *Handler) GetConfig(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	cfg, err := h.svc.loadConfig()
	if err != nil {
		log.Errorf("assistant: load config: %v", err)
		rsp.ErrRsp(c, codeError, "get assistant config failed")
		return
	}
	rsp.OkRspWithData(c, publicConfig(cfg))
}

func (h *Handler) SetConfig(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	var u ConfigUpdate
	if err := c.ShouldBindJSON(&u); err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	cfg, err := updateConfig(func(cfg Config) (Config, error) { return applyUpdate(cfg, u) })
	if err != nil {
		if !errors.Is(err, errInvalidConfig) {
			log.Errorf("assistant: save config: %v", err)
		}
		rsp.ErrRsp(c, codeError, err.Error())
		return
	}
	rsp.OkRspWithData(c, publicConfig(cfg))
}

func (h *Handler) Ask(c *gin.Context) {
	var rsp proto.Response
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	res, err := h.svc.Ask(c.Request.Context(), req)
	if err != nil {
		log.Warnf("assistant: %s ask: %v (route: %s)", req.Kind, err, res.Route)
		respondErr(c, err, res)
		return
	}
	rsp.OkRspWithData(c, res)
}

type contextRequest struct {
	Crop *Crop `json:"crop"`
}

func (h *Handler) AddContext(c *gin.Context) {
	var rsp proto.Response
	var req contextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	n, err := h.svc.AddContext(c.Request.Context(), req.Crop)
	if err != nil {
		respondErr(c, err, gin.H{"count": n})
		return
	}
	rsp.OkRspWithData(c, gin.H{"count": n})
}

func (h *Handler) ClearContexts(c *gin.Context) {
	var rsp proto.Response
	if err := h.svc.ClearContexts(); err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, gin.H{"count": 0})
}

func (h *Handler) ContextCount(c *gin.Context) {
	var rsp proto.Response
	n, err := h.svc.ContextCount()
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, gin.H{"count": n})
}

func (h *Handler) Screenshot(c *gin.Context) {
	data, err := h.svc.Screenshot(c.Request.Context())
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/jpeg", data)
}

type reasoningRequest struct {
	Direction string `json:"direction"`
}

func (h *Handler) Reasoning(c *gin.Context) {
	var rsp proto.Response
	var req reasoningRequest
	if err := c.ShouldBindJSON(&req); err != nil || (req.Direction != "up" && req.Direction != "down") {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	res, err := h.svc.AdjustReasoning(req.Direction == "up")
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, res)
}

func (h *Handler) ListAttachments(c *gin.Context) {
	var rsp proto.Response
	list, err := h.svc.attachments.List()
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, list)
}

func (h *Handler) UploadAttachment(c *gin.Context) {
	var rsp proto.Response
	file, err := c.FormFile("file")
	if err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	f, err := file.Open()
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	defer f.Close()
	if err := h.svc.attachments.Put(file.Filename, f); err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRsp(c)
}

func (h *Handler) DeleteAttachment(c *gin.Context) {
	var rsp proto.Response
	if err := h.svc.attachments.Delete(c.Query("name")); err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRsp(c)
}
```

- [ ] **Step 4: Implement `server/router/assistant.go`**

```go
package router

import (
	"context"
	"errors"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/assistant"
	mcpservice "NanoKVM-Server/service/mcp"
	"NanoKVM-Server/service/mcp/capture/kvm"

	"github.com/gin-gonic/gin"
)

// assistantCapturer adapts the MCP capture backend (cgo) to the assistant
// package, which stays cgo-free so its tests run anywhere.
type assistantCapturer struct {
	snapshotter mcpservice.Snapshotter
}

func (a assistantCapturer) Capture(ctx context.Context) (assistant.Frame, error) {
	snap, err := a.snapshotter.Capture(ctx, mcpservice.SnapshotRequest{})
	if err != nil {
		return assistant.Frame{}, err
	}
	if !snap.OK {
		return assistant.Frame{}, errors.New(snap.Message)
	}
	return assistant.Frame{JPEG: snap.JPEG, Width: snap.Width, Height: snap.Height}, nil
}

func assistantRouter(r *gin.Engine) {
	handler := assistant.NewHandler(assistant.NewService(assistantCapturer{snapshotter: kvmcapture.New()}))
	handler.Register(r.Group("/api/assistant").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	))
}
```

In `router.go` `server()`, add `assistantRouter(r)` on the line after `mcpRouter(r, control, picoclawService)`.

- [ ] **Step 5: Run the package tests, then the full container build.** Run `GOTEST` with `-v` (all PASS). Then run `make app DOCKER_TTY=` (it must build; `go mod tidy` runs inside it). Commit any change `go mod tidy` makes to `go.mod`/`go.sum`. Also run `cd server && gofmt -l service/assistant router` inside the container shell or via GOTEST's docker line with `gofmt -l`; expect no output.
- [ ] **Step 6: Commit.** `git commit -m "assistant: admin API and router wiring"`

---

### Task 9: Web API client, atoms, keyboard-lock sources

**Files:**
- Create: `web/src/api/assistant.ts`, `web/src/jotai/assistant.ts`
- Modify: `web/src/jotai/keyboard.ts` (export the lock sources), `web/src/lib/localstorage.ts` (FRQ box keys)

**Interfaces:**
- Produces: the types `AssistantConfig`, `AssistantConfigUpdate`, `Crop`, `AskKind`, `AttachmentInfo`, `ReasoningResult`; the constants `ASSISTANT_DISABLED = -2` and `ASSISTANT_NO_ANSWER = -3`; the functions `getAssistantConfig`, `setAssistantConfig`, `ask`, `addContext`, `clearContexts`, `getContextCount`, `getScreenshot`, `adjustReasoning`, `listAttachments`, `uploadAttachment`, `deleteAttachment`; the atoms `assistantConfigAtom`, `assistantContextCountAtom`, `assistantBusyAtom`, `keyboardLockSourcesAtom`; and the helpers `getAssistantFrqText`/`setAssistantFrqText`/`getAssistantFrqRect`/`setAssistantFrqRect`.

- [ ] **Step 1: `web/src/api/assistant.ts`**

```ts
import { http } from '@/lib/http.ts';

export type Provider = 'gemini' | 'openrouter';
export type ReasoningEffort = 'low' | 'medium' | 'high';

export type AssistantConfig = {
  enabled: boolean;
  ui: boolean;
  provider: Provider;
  geminiBaseUrl: string;
  geminiModel: string;
  geminiThinking: boolean;
  thinkingBudget: number;
  orBaseUrl: string;
  orModel: string;
  orReasoning: boolean;
  orReasoningEffort: ReasoningEffort;
  proxyUrl: string;
  copyClipboard: boolean;
  answerSel: boolean;
  contextSel: boolean;
  frqSel: boolean;
  quadClickMCQ: boolean;
  hasGeminiApiKey: boolean;
  hasOrApiKey: boolean;
  hasProxyPass: boolean;
};

export type AssistantConfigUpdate = Partial<
  Omit<AssistantConfig, 'hasGeminiApiKey' | 'hasOrApiKey' | 'hasProxyPass'>
> & {
  geminiApiKey?: string;
  orApiKey?: string;
  proxyPass?: string;
  clearGeminiApiKey?: boolean;
  clearOrApiKey?: boolean;
  clearProxyPass?: boolean;
};

export type AskKind = 'mcq' | 'frq' | 'custom';
export type Crop = { x: number; y: number; w: number; h: number };
export type AttachmentInfo = { name: string; size: number };
export type ReasoningResult = { label: string; changed: boolean; config: AssistantConfig };

export const ASSISTANT_DISABLED = -2;
export const ASSISTANT_NO_ANSWER = -3;

export function getAssistantConfig() {
  return http.get('/api/assistant/config');
}

export function setAssistantConfig(update: AssistantConfigUpdate) {
  return http.post('/api/assistant/config', update);
}

// The server may spend 2 x 300 s on the relay, 5 s between, then 300 s direct.
export function ask(kind: AskKind, crop?: Crop | null, text?: string) {
  return http.post('/api/assistant/ask', { kind, crop: crop ?? undefined, text }, { timeout: 0 });
}

export function addContext(crop?: Crop | null) {
  return http.post('/api/assistant/context', { crop: crop ?? undefined });
}

export function clearContexts() {
  return http.delete('/api/assistant/context');
}

export function getContextCount() {
  return http.get('/api/assistant/context');
}

export async function getScreenshot(): Promise<Blob> {
  const blob = (await http.request({
    method: 'get',
    url: '/api/assistant/screenshot',
    responseType: 'blob'
  })) as unknown as Blob;
  if (blob.type !== 'image/jpeg') {
    throw new Error(await blob.text());
  }
  return blob;
}

export function adjustReasoning(direction: 'up' | 'down') {
  return http.post('/api/assistant/reasoning', { direction });
}

export function listAttachments() {
  return http.get('/api/assistant/attachments');
}

export function uploadAttachment(file: File) {
  const form = new FormData();
  form.append('file', file);
  return http.post('/api/assistant/attachments', form, { timeout: 0 });
}

export function deleteAttachment(name: string) {
  return http.delete(`/api/assistant/attachments?name=${encodeURIComponent(name)}`);
}
```

- [ ] **Step 2: `web/src/jotai/assistant.ts`**

```ts
import { atom } from 'jotai';

import type { AssistantConfig } from '@/api/assistant.ts';

// Loaded once on the desktop page for admins; the settings tab replaces it on save.
export const assistantConfigAtom = atom<AssistantConfig | null>(null);
export const assistantContextCountAtom = atom(0);
// The LLM loading dot.
export const assistantBusyAtom = atom(false);
```

- [ ] **Step 3: `web/src/jotai/keyboard.ts`.** Add this after `isKeyboardEnableAtom`:

```ts
// which sources hold the lock (the assistant lets its own FRQ box through)
export const keyboardLockSourcesAtom = atom((get) => get(keyboardLocksAtom));
```

- [ ] **Step 4: `web/src/lib/localstorage.ts`.** Add the keys next to the other `*_KEY` constants, and the functions at the end of the file:

```ts
const ASSISTANT_FRQ_TEXT_KEY = 'nano-kvm-assistant-frq-text';
const ASSISTANT_FRQ_RECT_KEY = 'nano-kvm-assistant-frq-rect';
```

```ts
export type AssistantFrqRect = { left: number; top: number; width: number; height: number };

export function getAssistantFrqText(): string {
  return localStorage.getItem(ASSISTANT_FRQ_TEXT_KEY) || '';
}

export function setAssistantFrqText(text: string): void {
  localStorage.setItem(ASSISTANT_FRQ_TEXT_KEY, text);
}

export function getAssistantFrqRect(): AssistantFrqRect | null {
  try {
    const rect = JSON.parse(localStorage.getItem(ASSISTANT_FRQ_RECT_KEY) || 'null');
    return rect && [rect.left, rect.top, rect.width, rect.height].every(Number.isFinite) ? rect : null;
  } catch {
    return null;
  }
}

export function setAssistantFrqRect(rect: AssistantFrqRect): void {
  localStorage.setItem(ASSISTANT_FRQ_RECT_KEY, JSON.stringify(rect));
}
```

- [ ] **Step 5: Type-check.** Run `cd web && pnpm exec tsc --noEmit`. Expected: no errors.
- [ ] **Step 6: Commit.** `git commit -m "web: assistant API client and state"`

---

### Task 10: Pure runtime logic (hotkeys, selection maths)

**Files:**
- Create: `web/src/pages/desktop/assistant/hotkeys.ts`, `web/src/pages/desktop/assistant/selection-math.ts`
- Test: `web/src/pages/desktop/assistant/hotkeys.test.ts`, `web/src/pages/desktop/assistant/selection-math.test.ts`

Neither module may import `@/…`, because `pnpm test` runs them under plain Node.

**Interfaces:**
- Produces:
  - `HOTKEY_CODES`, `type HotkeyCode`, and `class HotkeyMatcher { keydown({code, repeat}): { action: HotkeyCode | null; swallow: boolean }; keyup({code}): boolean; }`
  - `type Rect = {left, top, width, height}`, `selectionToCrop(sel: Rect, media: Rect): {x,y,w,h} | null`, `cropLockTransform(sel: Rect, element: Rect, area: Rect, layoutScale: number): string | null`

- [ ] **Step 1: Write the failing tests**

`hotkeys.test.ts`:

```ts
import assert from 'node:assert/strict';
import test from 'node:test';

import { HotkeyMatcher } from './hotkeys.ts';

function matcher() {
  let now = 0;
  const m = new HotkeyMatcher(() => now);
  return { m, advance: (ms: number) => (now += ms) };
}
const down = (code: string, repeat = false) => ({ code, repeat });

test('two Ctrl presses within 500 ms arm the action key', () => {
  const { m, advance } = matcher();
  m.keydown(down('ControlLeft'));
  advance(300);
  m.keydown(down('ControlRight'));
  assert.deepEqual(m.keydown(down('KeyA')), { action: 'KeyA', swallow: true });
  assert.deepEqual(m.keydown(down('KeyA')), { action: null, swallow: true });
});

test('Ctrl presses 500 ms apart do not arm', () => {
  const { m, advance } = matcher();
  m.keydown(down('ControlLeft'));
  advance(500);
  m.keydown(down('ControlLeft'));
  assert.deepEqual(m.keydown(down('KeyA')), { action: null, swallow: false });
});

test('an auto-repeated Ctrl does not count', () => {
  const { m } = matcher();
  m.keydown(down('ControlLeft'));
  m.keydown(down('ControlLeft', true));
  assert.equal(m.keydown(down('KeyA')).action, null);
});

test('held key repeats and keyup are swallowed', () => {
  const { m } = matcher();
  m.keydown(down('ControlLeft'));
  m.keydown(down('ControlLeft'));
  m.keydown(down('KeyC'));
  assert.equal(m.keydown(down('KeyC', true)).swallow, true);
  assert.equal(m.keyup({ code: 'KeyC' }), true);
  assert.equal(m.keyup({ code: 'KeyC' }), false);
  assert.deepEqual(m.keydown(down('KeyC')), { action: null, swallow: false });
});

test('a non-action key leaves the prefix armed (content.js)', () => {
  const { m } = matcher();
  m.keydown(down('ControlLeft'));
  m.keydown(down('ControlLeft'));
  assert.equal(m.keydown(down('KeyB')).swallow, false);
  assert.equal(m.keydown(down('ArrowUp')).action, 'ArrowUp');
});

test('Escape and D/Q are not action keys', () => {
  for (const code of ['Escape', 'KeyD', 'KeyQ']) {
    const { m } = matcher();
    m.keydown(down('ControlLeft'));
    m.keydown(down('ControlLeft'));
    assert.equal(m.keydown(down(code)).action, null);
  }
});
```

`selection-math.test.ts`:

```ts
import assert from 'node:assert/strict';
import test from 'node:test';

import { cropLockTransform, selectionToCrop } from './selection-math.ts';

const media = { left: 100, top: 0, width: 800, height: 600 };

test('selection maps to frame fractions', () => {
  assert.deepEqual(selectionToCrop({ left: 300, top: 150, width: 400, height: 300 }, media), {
    x: 0.25,
    y: 0.25,
    w: 0.5,
    h: 0.5
  });
});

test('selection over the letterbox is clamped to the picture', () => {
  assert.deepEqual(selectionToCrop({ left: 0, top: 0, width: 300, height: 600 }, media), {
    x: 0,
    y: 0,
    w: 0.25,
    h: 1
  });
});

test('selection outside the picture is null', () => {
  assert.equal(selectionToCrop({ left: 0, top: 0, width: 90, height: 600 }, media), null);
  assert.equal(selectionToCrop({ left: 300, top: 150, width: 0, height: 0 }, media), null);
});

test('crop-lock scales the selection to fill the area', () => {
  const el = { left: 0, top: 0, width: 800, height: 600 };
  assert.equal(cropLockTransform({ left: 0, top: 0, width: 400, height: 300 }, el, el, 1), 'translate(0px, 0px) scale(2)');
  assert.equal(
    cropLockTransform({ left: 400, top: 300, width: 400, height: 300 }, el, el, 1),
    'translate(-800px, -600px) scale(2)'
  );
});

test('crop-lock translate is divided by the layout scale', () => {
  const el = { left: 0, top: 0, width: 800, height: 600 };
  assert.equal(
    cropLockTransform({ left: 400, top: 300, width: 400, height: 300 }, el, el, 2),
    'translate(-400px, -300px) scale(2)'
  );
});
```

- [ ] **Step 2: Run them and confirm they fail.** Run `cd web && pnpm test`. Expected: the new files fail with module-not-found errors.
- [ ] **Step 3: Implement `hotkeys.ts`**

```ts
// Ctrl-Ctrl prefix from the chaice content script: two non-repeat Control
// keydowns within 500 ms, then an action key matched on `code`. The action
// key is swallowed from its keydown until its keyup so the target never sees it.

export const HOTKEY_CODES = [
  'KeyA',
  'KeyC',
  'KeyT',
  'KeyF',
  'KeyS',
  'KeyP',
  'KeyL',
  'KeyR',
  'KeyM',
  'ArrowUp',
  'ArrowDown'
] as const;

export type HotkeyCode = (typeof HOTKEY_CODES)[number];

const PREFIX_WINDOW_MS = 500;

export function isHotkeyCode(code: string): code is HotkeyCode {
  return (HOTKEY_CODES as readonly string[]).includes(code);
}

export class HotkeyMatcher {
  private ctrlTimes: number[] = [];
  private held: string | null = null;

  constructor(private readonly now: () => number = () => Date.now()) {}

  keydown(e: { code: string; repeat: boolean }): { action: HotkeyCode | null; swallow: boolean } {
    if (this.held !== null && e.code === this.held) {
      return { action: null, swallow: true };
    }
    if ((e.code === 'ControlLeft' || e.code === 'ControlRight') && !e.repeat) {
      const t = this.now();
      this.ctrlTimes.push(t);
      this.ctrlTimes = this.ctrlTimes.filter((x) => t - x < PREFIX_WINDOW_MS);
      if (this.ctrlTimes.length > 2) this.ctrlTimes.shift();
    }
    if (this.ctrlTimes.length === 2 && isHotkeyCode(e.code)) {
      this.ctrlTimes = [];
      this.held = e.code;
      return { action: e.code, swallow: true };
    }
    return { action: null, swallow: false };
  }

  keyup(e: { code: string }): boolean {
    if (this.held !== null && e.code === this.held) {
      this.held = null;
      return true;
    }
    return false;
  }
}
```

- [ ] **Step 4: Implement `selection-math.ts`**

```ts
export type Rect = { left: number; top: number; width: number; height: number };

// A selection in viewport CSS pixels -> fractions of the video frame, given
// where the frame is rendered. Null when it misses the picture.
export function selectionToCrop(sel: Rect, media: Rect) {
  if (media.width <= 0 || media.height <= 0) return null;
  const left = Math.max(sel.left, media.left);
  const top = Math.max(sel.top, media.top);
  const right = Math.min(sel.left + sel.width, media.left + media.width);
  const bottom = Math.min(sel.top + sel.height, media.top + media.height);
  if (right - left < 1 || bottom - top < 1) return null;
  return {
    x: (left - media.left) / media.width,
    y: (top - media.top) / media.height,
    w: (right - left) / media.width,
    h: (bottom - top) / media.height
  };
}

// L crop-lock: the transform (origin 0 0) that makes `sel` fill `area`,
// centred, as the extension did for the whole page. `element` is #screen's
// untransformed rect; `layoutScale` is its rendered/layout size ratio (the
// viewport's video scale), since translate is in the element's own units.
export function cropLockTransform(sel: Rect, element: Rect, area: Rect, layoutScale: number) {
  const w = Math.max(1, sel.width);
  const h = Math.max(1, sel.height);
  const scale = Math.min(area.width / w, area.height / h);
  if (!Number.isFinite(scale) || scale <= 0 || !(layoutScale > 0)) return null;
  const offX = area.left + (area.width - w * scale) / 2;
  const offY = area.top + (area.height - h * scale) / 2;
  const tx = (offX - element.left - (sel.left - element.left) * scale) / layoutScale;
  const ty = (offY - element.top - (sel.top - element.top) * scale) / layoutScale;
  return `translate(${tx}px, ${ty}px) scale(${scale})`;
}
```

- [ ] **Step 5: Run the tests and confirm they pass.** Run `cd web && pnpm test`. Expected: all tests pass, including the existing ones.
- [ ] **Step 6: Commit.** `git commit -m "web: assistant hotkey matcher and selection maths"`

---

### Task 11: Desktop runtime (the on-page UI)

**Files (all under `web/src/pages/desktop/assistant/`):**
- Create: `status.tsx`, `toast.tsx`, `frq-box.tsx`, `selection.tsx`, `freeze.tsx`, `crop-lock.ts`, `page-style.ts`, `quad-click.ts`, `actions.ts`, `use-hotkeys.ts`, `index.tsx`
- Modify: `web/src/pages/desktop/index.tsx` (mount it)

**Interfaces:**
- Consumes: Tasks 9 and 10. `getMediaSize` and `getRenderedMediaRect` from `../screen/geometry.ts`. `menuCloseSignalAtom` from `@/jotai/settings.ts`. `keyboardLockAtom` and `keyboardLockSourcesAtom`.
- Produces: `export const Assistant` (mounted by the desktop page), and `FRQ_LOCK_SOURCE = 'assistant-frq'`.

Every `store` below is `ReturnType<typeof useStore>` from jotai. Status and toast are functions of the store, so window listeners never hold stale React state.

- [ ] **Step 1: `status.tsx`**

```tsx
import type { CSSProperties } from 'react';
import { atom, useAtomValue, useStore } from 'jotai';

import { assistantBusyAtom, assistantConfigAtom, assistantContextCountAtom } from '@/jotai/assistant.ts';

import { freezeAtom } from './freeze.tsx';

type Store = ReturnType<typeof useStore>;

// Colours verbatim from the chaice content script.
export const STATUS = {
  idle: 'rgb(40,62,159)',
  loading: '#b3cfff',
  frq: '#e0cfff',
  context: '#0a1333',
  success: 'rgb(89, 105, 192)',
  error: 'red',
  warning: 'yellow',
  cleared: '#444',
  crop: '#285e9f',
  cropSelect: 'orange'
} as const;

const statusColorAtom = atom<string>(STATUS.idle);
let resetTimer: ReturnType<typeof setTimeout> | undefined;

export function setStatus(store: Store, color: string) {
  if (!store.get(assistantConfigAtom)?.ui) return;
  clearTimeout(resetTimer);
  store.set(statusColorAtom, color);
  resetTimer = setTimeout(() => store.set(statusColorAtom, STATUS.idle), 3000);
}

const dot = (background: string, top: string): CSSProperties => ({
  position: 'fixed',
  left: '4px',
  top,
  width: '8px',
  height: '8px',
  borderRadius: '50%',
  background,
  zIndex: 10051,
  boxShadow: `0 0 2px ${background}`,
  pointerEvents: 'none'
});

export const StatusBar = () => {
  const color = useAtomValue(statusColorAtom);
  const count = useAtomValue(assistantContextCountAtom);
  const busy = useAtomValue(assistantBusyAtom);
  const frozen = useAtomValue(freezeAtom);

  return (
    <>
      <div
        id="status-bar"
        style={{
          position: 'fixed',
          bottom: 0,
          left: 0,
          height: '2px',
          width: '100%',
          background: color,
          zIndex: 9999,
          transition: 'background 0.2s'
        }}
      />
      <div
        id="context-count"
        style={{
          position: 'fixed',
          left: '18px',
          bottom: '28px',
          color: '#800080',
          fontSize: '13px',
          fontFamily: 'sans-serif',
          fontWeight: 400,
          zIndex: 10003,
          pointerEvents: 'none',
          background: 'none',
          textAlign: 'left'
        }}
      >
        {count}
      </div>
      {busy && <div className="llm-loading-dot" style={dot('rgb(40,62,159)', '18px')} />}
      {frozen && <div className="freeze-dot" style={dot('#ffe600', '4px')} />}
    </>
  );
};
```

- [ ] **Step 2: `toast.tsx`**

```tsx
import { useEffect, useState } from 'react';
import { atom, useAtomValue, useStore } from 'jotai';

import { assistantConfigAtom } from '@/jotai/assistant.ts';

type Store = ReturnType<typeof useStore>;

const toastAtom = atom<{ text: string; id: number } | null>(null);
let toastId = 0;

// The answer toast: first 40 words, gone after 1.75 s (content.js showLLMAnswer).
export function showToast(store: Store, answer: string) {
  if (!store.get(assistantConfigAtom)?.ui) return;
  const text = answer.trim().split(/\s+/).slice(0, 40).join(' ');
  store.set(toastAtom, { text, id: ++toastId });
}

export const Toast = () => {
  const toast = useAtomValue(toastAtom);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!toast) return;
    setVisible(true);
    const timer = setTimeout(() => setVisible(false), 1750);
    return () => clearTimeout(timer);
  }, [toast]);

  if (!toast) return null;
  return (
    <div
      id="llm-answer"
      style={{
        position: 'fixed',
        left: '50%',
        bottom: '28px',
        transform: 'translateX(-50%)',
        color: '#800080',
        fontSize: '13px',
        fontFamily: 'sans-serif',
        fontWeight: 400,
        zIndex: 10003,
        pointerEvents: 'none',
        background: 'none',
        textAlign: 'center',
        opacity: visible ? 1 : 0
      }}
    >
      {toast.text}
    </div>
  );
};
```

- [ ] **Step 3: `freeze.tsx`**

```tsx
import { atom, useAtomValue } from 'jotai';

import type { Rect } from './selection-math.ts';

// P: a still of the target laid over the picture; pointer-events pass through.
export const freezeAtom = atom<{ src: string; rect: Rect } | null>(null);

export const FreezeOverlay = () => {
  const freeze = useAtomValue(freezeAtom);
  if (!freeze) return null;
  return (
    <div
      className="freeze-overlay"
      style={{
        position: 'fixed',
        left: `${freeze.rect.left}px`,
        top: `${freeze.rect.top}px`,
        width: `${freeze.rect.width}px`,
        height: `${freeze.rect.height}px`,
        zIndex: 10050,
        background: `rgba(255,255,255,0.01) url('${freeze.src}') center center / 100% 100% no-repeat`,
        pointerEvents: 'none',
        transition: 'opacity 0.2s'
      }}
    />
  );
};
```

- [ ] **Step 4: `selection.tsx`**

```tsx
import { useCallback, useEffect, useRef, useState } from 'react';
import { atom, useAtom, useAtomValue, useSetAtom } from 'jotai';

import { assistantConfigAtom } from '@/jotai/assistant.ts';

import type { Rect } from './selection-math.ts';

type Pending = { resolve: (rect: Rect | null) => void };

const pendingSelectionAtom = atom<Pending | null>(null);

// Resolves with the dragged rectangle (viewport CSS px), or null on Escape.
export function useRequestSelection() {
  const setPending = useSetAtom(pendingSelectionAtom);
  return useCallback(
    () => new Promise<Rect | null>((resolve) => setPending({ resolve })),
    [setPending]
  );
}

function swallowKeyupOnce(code: string) {
  const handler = (e: KeyboardEvent) => {
    if (e.code !== code) return;
    e.preventDefault();
    e.stopImmediatePropagation();
    window.removeEventListener('keyup', handler, true);
  };
  window.addEventListener('keyup', handler, true);
}

export const Selection = () => {
  const ui = useAtomValue(assistantConfigAtom)?.ui ?? true;
  const [pending, setPending] = useAtom(pendingSelectionAtom);
  const start = useRef<{ x: number; y: number } | null>(null);
  const [rect, setRect] = useState<Rect | null>(null);

  const finish = useCallback(
    (result: Rect | null) => {
      pending?.resolve(result);
      start.current = null;
      setRect(null);
      setPending(null);
    },
    [pending, setPending]
  );

  useEffect(() => {
    if (!pending) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      e.preventDefault();
      e.stopImmediatePropagation();
      swallowKeyupOnce(e.code);
      finish(null);
    };
    const oldUserSelect = document.body.style.userSelect;
    document.body.style.userSelect = 'none';
    window.addEventListener('keydown', onKeyDown, true);
    return () => {
      window.removeEventListener('keydown', onKeyDown, true);
      document.body.style.userSelect = oldUserSelect;
    };
  }, [pending, finish]);

  if (!pending) return null;

  return (
    <>
      <div
        className="chaice-overlay"
        style={{
          position: 'fixed',
          left: 0,
          top: 0,
          width: '100vw',
          height: '100vh',
          zIndex: 10000,
          background: 'rgba(0,0,0,0)',
          pointerEvents: 'all'
        }}
        onMouseDown={(e) => {
          if (e.button !== 0) return;
          e.preventDefault();
          e.stopPropagation();
          start.current = { x: e.clientX, y: e.clientY };
          setRect({ left: e.clientX, top: e.clientY, width: 0, height: 0 });
        }}
        onMouseMove={(e) => {
          const s = start.current;
          if (!s) return;
          e.preventDefault();
          const w = e.clientX - s.x;
          const h = e.clientY - s.y;
          setRect({
            left: w < 0 ? s.x + w : s.x,
            top: h < 0 ? s.y + h : s.y,
            width: Math.abs(w),
            height: Math.abs(h)
          });
        }}
        onMouseUp={(e) => {
          if (!start.current) return;
          e.preventDefault();
          e.stopPropagation();
          finish(rect ?? { left: e.clientX, top: e.clientY, width: 0, height: 0 });
        }}
      />
      {rect && (
        <div
          className="selection-rect"
          style={{
            position: 'fixed',
            left: `${rect.left}px`,
            top: `${rect.top}px`,
            width: `${rect.width}px`,
            height: `${rect.height}px`,
            pointerEvents: 'none',
            zIndex: 10001,
            background: 'none',
            // UI off: still laid out and dragged, never seen.
            border: ui ? '2px dashed #e0cfff' : 'none',
            opacity: ui ? 0.45 : 0
          }}
        />
      )}
    </>
  );
};
```

- [ ] **Step 5: `frq-box.tsx`**

```tsx
import { useEffect, useRef, useState } from 'react';
import { atom, useAtom, useAtomValue, useSetAtom } from 'jotai';

import * as ls from '@/lib/localstorage.ts';
import { keyboardLockAtom } from '@/jotai/keyboard.ts';

export const FRQ_LOCK_SOURCE = 'assistant-frq';

export const frqVisibleAtom = atom(false);
// rev bumps when the text is set from outside (an answer), so the editable is rewritten.
export const frqContentAtom = atom({ text: ls.getAssistantFrqText(), rev: 0 });

export const FrqBox = () => {
  const visible = useAtomValue(frqVisibleAtom);
  const [content, setContent] = useAtom(frqContentAtom);
  const setKeyboardLock = useSetAtom(keyboardLockAtom);
  const boxRef = useRef<HTMLDivElement>(null);
  const editableRef = useRef<HTMLDivElement>(null);
  const [initialRect] = useState(ls.getAssistantFrqRect);

  useEffect(() => {
    if (editableRef.current) editableRef.current.innerText = content.text;
    // Only external writes (rev) rewrite the editable; typing must keep the caret.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [content.rev]);

  useEffect(() => {
    const timer = setTimeout(() => ls.setAssistantFrqText(content.text), 500);
    return () => clearTimeout(timer);
  }, [content.text]);

  useEffect(() => {
    return () => setKeyboardLock({ source: FRQ_LOCK_SOURCE, locked: false });
  }, [setKeyboardLock]);

  // Drag on the box background (not the text, not the 8 px resize border).
  useEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    let dragging = false;
    let dx = 0;
    let dy = 0;
    const saveRect = () => {
      const r = el.getBoundingClientRect();
      ls.setAssistantFrqRect({ left: r.left, top: r.top, width: el.offsetWidth, height: el.offsetHeight });
    };
    const onDown = (e: MouseEvent) => {
      if (e.target !== el || e.button !== 0) return;
      const r = el.getBoundingClientRect();
      const border = 8;
      if (
        e.clientX - r.left < border ||
        r.right - e.clientX < border ||
        e.clientY - r.top < border ||
        r.bottom - e.clientY < border
      ) {
        return;
      }
      dragging = true;
      dx = e.clientX - r.left;
      dy = e.clientY - r.top;
      document.body.style.userSelect = 'none';
      e.preventDefault();
    };
    const onMove = (e: MouseEvent) => {
      if (!dragging) return;
      el.style.left = `${e.clientX - dx}px`;
      el.style.top = `${e.clientY - dy}px`;
      el.style.transform = 'none';
    };
    const onUp = () => {
      if (!dragging) return;
      dragging = false;
      document.body.style.userSelect = '';
      saveRect();
    };
    const resize = new ResizeObserver(() => {
      if (el.offsetWidth > 0) saveRect();
    });
    el.addEventListener('mousedown', onDown);
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    resize.observe(el);
    return () => {
      el.removeEventListener('mousedown', onDown);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      resize.disconnect();
    };
  }, []);

  return (
    <div
      id="frq-box"
      ref={boxRef}
      style={{
        position: 'fixed',
        left: initialRect ? `${initialRect.left}px` : '50%',
        top: initialRect ? `${initialRect.top}px` : '10%',
        transform: initialRect ? 'none' : 'translateX(-50%)',
        width: `${initialRect?.width ?? 340}px`,
        height: `${initialRect?.height ?? 120}px`,
        maxWidth: '80vw',
        maxHeight: '70vh',
        overflow: 'auto',
        resize: 'both',
        border: '2px solid rgba(40,62,159,0.25)',
        background: 'none',
        color: '#111',
        fontSize: '15px',
        fontFamily: 'sans-serif',
        zIndex: 10004,
        padding: '18px 18px 18px 18px',
        borderRadius: '10px',
        boxShadow: '0 2px 16px 0 rgba(40,62,159,0.07)',
        backdropFilter: 'none',
        pointerEvents: 'auto',
        userSelect: 'text',
        boxSizing: 'content-box',
        display: visible ? 'block' : 'none'
      }}
    >
      <div
        id="frq-editable"
        ref={editableRef}
        contentEditable
        suppressContentEditableWarning
        style={{
          width: '100%',
          height: '100%',
          outline: 'none',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          fontFamily: 'sans-serif'
        }}
        onFocus={() => setKeyboardLock({ source: FRQ_LOCK_SOURCE, locked: true })}
        onBlur={() => setKeyboardLock({ source: FRQ_LOCK_SOURCE, locked: false })}
        onInput={(e) => {
          const text = e.currentTarget.innerText || '';
          setContent((c) => ({ text, rev: c.rev }));
        }}
      />
    </div>
  );
};
```

- [ ] **Step 6: `crop-lock.ts`, `page-style.ts`, `quad-click.ts`**

`crop-lock.ts`:

```ts
import { atom } from 'jotai';

import { cropLockTransform, type Rect } from './selection-math.ts';

export const cropLockAtom = atom(false);

// L: scale the selected region of #screen to fill the screen viewport. Mouse
// mapping keeps working because it reads #screen's rendered rect.
export function applyCropLock(sel: Rect): boolean {
  const screen = document.getElementById('screen');
  const area = document.getElementById('screen-viewport');
  if (!screen || !area || screen.offsetWidth === 0) return false;
  screen.style.transform = 'none';
  const rect = screen.getBoundingClientRect();
  const transform = cropLockTransform(sel, rect, area.getBoundingClientRect(), rect.width / screen.offsetWidth);
  if (!transform) return false;
  screen.style.transformOrigin = '0 0';
  screen.style.transform = transform;
  return true;
}

export function resetCropLock() {
  const screen = document.getElementById('screen');
  if (!screen) return;
  screen.style.transform = 'none';
  screen.style.transformOrigin = '';
}
```

`page-style.ts`: the CSS is copied from `content.js` `hideScrollbarsStyle`. The only change is that the `#screen` rule is scoped away from the cropped input-region view (deviation 5).

```ts
import { useEffect, useRef } from 'react';
import { useSetAtom } from 'jotai';

import { menuCloseSignalAtom } from '@/jotai/settings.ts';

const PAGE_CSS = `
    * {
      scrollbar-width: none !important;
      -ms-overflow-style: none !important;
    }
    *::-webkit-scrollbar {
      display: none !important;
      width: 0 !important;
      height: 0 !important;
    }
    div.flex.h-screen.w-screen.items-start.justify-center.xl\\:items-center {
      background-color: white !important;
    }
    #screen-viewport[data-cropped="false"] #screen {
      height: 100vh;
      object-fit: contain !important;
    }
    div.fixed.left-1\\/2.top-\\[10px\\].z-\\[1000\\].-translate-x-1\\/2.react-draggable {
      background-color: white !important;
    }
    div.fixed.left-1\\/2.top-\\[10px\\].z-\\[1000\\].-translate-x-1\\/2.react-draggable .bg-neutral-800\\/80 {
      background-color: #fafafa !important;
    }
    div.fixed.left-1\\/2.top-\\[10px\\].z-\\[1000\\].-translate-x-1\\/2.react-draggable .bg-neutral-800\\/50 {
      background-color: #fafafa !important;
    }
  `;

// UI on: the extension's page CSS, and the menu bar closed once, 2 s after load.
export function usePageStyle(ui: boolean) {
  const closeMenu = useSetAtom(menuCloseSignalAtom);
  const closedOnce = useRef(false);

  useEffect(() => {
    if (!ui) return;
    const style = document.createElement('style');
    style.textContent = PAGE_CSS;
    document.head.appendChild(style);
    return () => style.remove();
  }, [ui]);

  useEffect(() => {
    if (!ui || closedOnce.current) return;
    const timer = setTimeout(() => {
      closedOnce.current = true;
      closeMenu((n) => n + 1);
    }, 2000);
    return () => clearTimeout(timer);
  }, [ui, closeMenu]);
}
```

`quad-click.ts`:

```ts
import { useEffect } from 'react';

// Four clicks within 2 s anywhere but the assistant's own UI -> MCQ.
export function useQuadClick(enabled: boolean, onQuad: () => void) {
  useEffect(() => {
    if (!enabled) return;
    let stamps: number[] = [];
    const handler = (e: MouseEvent) => {
      const target = e.target as Element | null;
      if (target?.closest?.('.chaice-overlay, #llm-answer, #frq-box, #context-count')) return;
      const now = Date.now();
      stamps.push(now);
      stamps = stamps.filter((t) => now - t < 2000);
      if (stamps.length >= 4) {
        stamps = [];
        onQuad();
      }
    };
    document.addEventListener('click', handler, true);
    return () => document.removeEventListener('click', handler, true);
  }, [enabled, onQuad]);
}
```

- [ ] **Step 7: `actions.ts`** (ports the handlers in `content.js`; the status colours match it step for step)

```ts
import { useMemo } from 'react';
import { useStore } from 'jotai';

import * as api from '@/api/assistant.ts';
import type { AskKind, Crop } from '@/api/assistant.ts';
import { assistantBusyAtom, assistantConfigAtom, assistantContextCountAtom } from '@/jotai/assistant.ts';

import { getMediaSize, getRenderedMediaRect } from '../screen/geometry.ts';
import { applyCropLock, cropLockAtom, resetCropLock } from './crop-lock.ts';
import { freezeAtom } from './freeze.tsx';
import { frqContentAtom, frqVisibleAtom } from './frq-box.tsx';
import type { HotkeyCode } from './hotkeys.ts';
import { selectionToCrop, type Rect } from './selection-math.ts';
import { STATUS, setStatus } from './status.tsx';
import { showToast } from './toast.tsx';

function renderedMediaRect(): Rect | null {
  const screen = document.getElementById('screen');
  const media = screen && getMediaSize(screen);
  return screen && media ? getRenderedMediaRect(screen.getBoundingClientRect(), media) : null;
}

export function useAssistantActions(requestSelection: () => Promise<Rect | null>) {
  const store = useStore();

  return useMemo(() => {
    const config = () => store.get(assistantConfigAtom)!;
    const status = (color: string) => setStatus(store, color);

    // Selection -> crop, or 'cancel' (Escape) / null (missed the picture).
    async function selectCrop(color: string): Promise<Crop | null | 'cancel'> {
      status(color);
      const sel = await requestSelection();
      if (!sel) return 'cancel';
      const media = renderedMediaRect();
      return media ? selectionToCrop(sel, media) : null;
    }

    function copyAnswer(answer: string) {
      if (!navigator.clipboard) {
        console.warn('[Assistant] Clipboard API not available');
      } else if (document.hasFocus()) {
        navigator.clipboard.writeText(answer).catch((err) => console.warn('[Assistant] Clipboard write failed:', err));
      } else {
        console.warn('[Assistant] Clipboard write skipped: document not focused');
      }
    }

    async function answer(kind: Exclude<AskKind, 'custom'>) {
      const useSel = kind === 'mcq' ? config().answerSel : config().frqSel;
      store.set(assistantBusyAtom, true);
      try {
        let crop: Crop | undefined;
        if (useSel) {
          const r = await selectCrop(kind === 'mcq' ? STATUS.loading : STATUS.frq);
          if (r === 'cancel') return status(STATUS.idle);
          if (r === null) return status(STATUS.error);
          crop = r;
        }
        const rsp = await api.ask(kind, crop);
        if (rsp.code === api.ASSISTANT_NO_ANSWER) return status(STATUS.warning);
        if (rsp.code !== 0) {
          console.error('[Assistant]', rsp.msg, rsp.data?.route);
          return status(STATUS.error);
        }
        const text: string = rsp.data.answer;
        if (kind === 'mcq') {
          showToast(store, text);
          status(STATUS.success);
          if (config().copyClipboard) copyAnswer(text);
        } else {
          store.set(frqContentAtom, (c) => ({ text, rev: c.rev + 1 }));
          status(STATUS.success);
        }
      } catch (err) {
        console.error('[Assistant]', err);
        status(STATUS.error);
      } finally {
        store.set(assistantBusyAtom, false);
      }
    }

    async function context() {
      let crop: Crop | undefined;
      if (config().contextSel) {
        const r = await selectCrop(STATUS.context);
        if (r === 'cancel') return status(STATUS.idle);
        if (r === null) return status(STATUS.error);
        crop = r;
      }
      try {
        const rsp = await api.addContext(crop);
        if (typeof rsp.data?.count === 'number') store.set(assistantContextCountAtom, rsp.data.count);
        if (rsp.code !== 0) status(STATUS.error);
      } catch {
        status(STATUS.error);
      }
    }

    async function clear() {
      try {
        const rsp = await api.clearContexts();
        if (rsp.code !== 0) return status(STATUS.error);
        store.set(assistantContextCountAtom, 0);
        status(STATUS.cleared);
      } catch {
        status(STATUS.error);
      }
    }

    async function custom() {
      if (!store.get(frqVisibleAtom)) store.set(frqVisibleAtom, true);
      store.set(assistantBusyAtom, true);
      status(STATUS.frq);
      try {
        const current = store.get(frqContentAtom).text;
        if (!current.trim()) return status(STATUS.warning);
        const rsp = await api.ask('custom', null, current);
        if (rsp.code === api.ASSISTANT_NO_ANSWER) return status(STATUS.warning);
        if (rsp.code !== 0) {
          console.error('[Assistant]', rsp.msg);
          return status(STATUS.error);
        }
        const updated = current + '\n\n' + rsp.data.answer;
        store.set(frqContentAtom, (c) => ({ text: updated, rev: c.rev + 1 }));
        status(STATUS.success);
      } catch (err) {
        console.error('[Assistant]', err);
        status(STATUS.error);
      } finally {
        store.set(assistantBusyAtom, false);
      }
    }

    async function freeze() {
      const current = store.get(freezeAtom);
      if (current) {
        URL.revokeObjectURL(current.src);
        store.set(freezeAtom, null);
        return;
      }
      try {
        const blob = await api.getScreenshot();
        const rect = renderedMediaRect();
        if (!config().ui || !rect) return;
        store.set(freezeAtom, { src: URL.createObjectURL(blob), rect });
      } catch (err) {
        console.error('[Assistant] freeze capture failed', err);
        status(STATUS.error);
      }
    }

    async function cropLock() {
      if (store.get(cropLockAtom)) {
        resetCropLock();
        store.set(cropLockAtom, false);
        return status(STATUS.crop);
      }
      status(STATUS.cropSelect);
      const sel = await requestSelection();
      if (!sel) return status(STATUS.idle);
      if (!config().ui) return;
      if (applyCropLock(sel)) {
        store.set(cropLockAtom, true);
        status(STATUS.crop);
      }
    }

    async function reasoning(up: boolean) {
      try {
        const rsp = await api.adjustReasoning(up ? 'up' : 'down');
        if (rsp.code !== 0) return status(STATUS.error);
        const result = rsp.data as api.ReasoningResult;
        showToast(store, result.label);
        if (result.changed) {
          store.set(assistantConfigAtom, result.config);
          status(STATUS.success);
        }
      } catch {
        status(STATUS.error);
      }
    }

    function run(code: HotkeyCode) {
      switch (code) {
        case 'KeyA':
          return void answer('mcq');
        case 'KeyF':
          return void answer('frq');
        case 'KeyC':
          return void context();
        case 'KeyT':
          return void clear();
        case 'KeyS':
          return store.set(frqVisibleAtom, (v) => !v);
        case 'KeyM':
          return void custom();
        case 'KeyP':
          return void freeze();
        case 'KeyL':
          return void cropLock();
        case 'KeyR':
          return location.reload();
        case 'ArrowUp':
        case 'ArrowDown':
          return void reasoning(code === 'ArrowUp');
      }
    }

    function quadClick() {
      status(STATUS.loading);
      void answer('mcq');
    }

    function teardown() {
      if (store.get(cropLockAtom)) resetCropLock();
      store.set(cropLockAtom, false);
      const current = store.get(freezeAtom);
      if (current) URL.revokeObjectURL(current.src);
      store.set(freezeAtom, null);
    }

    return { run, quadClick, teardown };
  }, [store, requestSelection]);
}

export type AssistantActions = ReturnType<typeof useAssistantActions>;
```

- [ ] **Step 8: `use-hotkeys.ts`**

```ts
import { useEffect } from 'react';

import type { AssistantActions } from './actions.ts';
import { HotkeyMatcher } from './hotkeys.ts';

// Window capture phase, so it runs before the desktop keyboard's document
// listener; a matched key never reaches the target. Ctrl itself still does.
export function useHotkeys(enabled: boolean, actions: AssistantActions) {
  useEffect(() => {
    if (!enabled) return;
    const matcher = new HotkeyMatcher();
    const onKeyDown = (e: KeyboardEvent) => {
      const result = matcher.keydown(e);
      if (!result.swallow) return;
      e.preventDefault();
      e.stopImmediatePropagation();
      if (result.action) actions.run(result.action);
    };
    const onKeyUp = (e: KeyboardEvent) => {
      if (!matcher.keyup(e)) return;
      e.preventDefault();
      e.stopImmediatePropagation();
    };
    window.addEventListener('keydown', onKeyDown, true);
    window.addEventListener('keyup', onKeyUp, true);
    return () => {
      window.removeEventListener('keydown', onKeyDown, true);
      window.removeEventListener('keyup', onKeyUp, true);
    };
  }, [enabled, actions]);
}
```

- [ ] **Step 9: `index.tsx`**

```tsx
import { useEffect } from 'react';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { createPortal } from 'react-dom';

import * as api from '@/api/assistant.ts';
import type { AssistantConfig } from '@/api/assistant.ts';
import { useAuth } from '@/contexts/auth.ts';
import { assistantConfigAtom, assistantContextCountAtom } from '@/jotai/assistant.ts';
import { keyboardLockSourcesAtom } from '@/jotai/keyboard.ts';

import { useAssistantActions } from './actions.ts';
import { FreezeOverlay } from './freeze.tsx';
import { FRQ_LOCK_SOURCE, FrqBox } from './frq-box.tsx';
import { usePageStyle } from './page-style.ts';
import { useQuadClick } from './quad-click.ts';
import { Selection, useRequestSelection } from './selection.tsx';
import { StatusBar } from './status.tsx';
import { Toast } from './toast.tsx';
import { useHotkeys } from './use-hotkeys.ts';

export const Assistant = () => {
  const { account } = useAuth();
  const isAdmin = account.role === 'admin';
  const [config, setConfig] = useAtom(assistantConfigAtom);
  const setContextCount = useSetAtom(assistantContextCountAtom);

  useEffect(() => {
    if (!isAdmin) return;
    api
      .getAssistantConfig()
      .then((rsp) => {
        if (rsp.code === 0) setConfig(rsp.data);
      })
      .catch(() => undefined);
  }, [isAdmin, setConfig]);

  useEffect(() => {
    if (!config?.enabled) return;
    api
      .getContextCount()
      .then((rsp) => {
        if (rsp.code === 0) setContextCount(rsp.data.count);
      })
      .catch(() => undefined);
  }, [config?.enabled, setContextCount]);

  if (!isAdmin || !config?.enabled) return null;
  return <AssistantRuntime config={config} />;
};

const AssistantRuntime = ({ config }: { config: AssistantConfig }) => {
  const requestSelection = useRequestSelection();
  const actions = useAssistantActions(requestSelection);
  const lockSources = useAtomValue(keyboardLockSourcesAtom);
  const hotkeysAllowed = [...lockSources].every((source) => source === FRQ_LOCK_SOURCE);

  useHotkeys(hotkeysAllowed, actions);
  useQuadClick(config.quadClickMCQ, actions.quadClick);
  usePageStyle(config.ui);
  useEffect(() => () => actions.teardown(), [actions]);

  return createPortal(
    <>
      {config.ui && <StatusBar />}
      <Toast />
      <FreezeOverlay />
      <FrqBox />
      <Selection />
    </>,
    document.body
  );
};
```

- [ ] **Step 10: Mount it in `web/src/pages/desktop/index.tsx`.** Add `import { Assistant } from './assistant';`. Inside the `videoMode && resolution` block, after the `<OverlayBoundary name="input">…</OverlayBoundary>` element, add:

```tsx
          <OverlayBoundary name="assistant">
            <Assistant />
          </OverlayBoundary>
```

- [ ] **Step 11: Run the checks.** Run `cd web && pnpm exec tsc --noEmit && pnpm test && pnpm lint`. Expected: no errors. Fix lint findings in the new files only.
- [ ] **Step 12: Commit.** `git commit -m "web: assistant desktop runtime ported from the chaice content script"`

---

### Task 12: Settings tab

**Files:**
- Create: `web/src/pages/desktop/menu/settings/assistant/index.tsx`
- Modify: `web/src/pages/desktop/menu/settings/index.tsx`, `web/src/i18n/locales/en.ts`

**Interfaces:**
- Consumes: Task 9 API/atoms.
- Produces: `export const AssistantSettings`

- [ ] **Step 1: i18n.** In `en.ts`, inside `settings`, add this sibling after the `mcp: { … },` block:

```ts
      assistant: {
        title: 'Assistant',
        enabled: 'Enable assistant',
        enabledDesc: 'Ctrl Ctrl + key on this page captures the remote screen and asks an LLM from the device',
        ui: 'Show UI',
        uiDesc: 'Status bar, context count, dots and overlays. Hotkeys keep working when off; the FRQ box still shows.',
        provider: 'Provider',
        baseUrl: 'Base URL',
        model: 'Model',
        apiKey: 'API key',
        secretSet: 'Saved. Leave empty to keep it.',
        clear: 'Clear',
        thinking: 'Enable thinking',
        thinkingBudget: 'Thinking budget',
        reasoning: 'Enable reasoning',
        reasoningEffort: 'Reasoning effort',
        proxy: 'Relay',
        proxyUrl: 'Relay URL (empty: direct)',
        proxyPass: 'Relay password',
        behaviour: 'Behaviour',
        copyClipboard: 'Copy answer to clipboard',
        answerSel: 'Default answer selection mode',
        contextSel: 'Default context selection mode',
        frqSel: 'Default FRQ selection mode',
        quadClickMCQ: 'Quad-click MCQ activation',
        attachments: 'Attachments',
        attachmentsDesc: 'Sent with every question (20 MB total)',
        upload: 'Upload',
        save: 'Save',
        saved: 'Saved',
        hotkeys: 'Hotkeys (press Ctrl twice, then the key)',
        hotkeyList:
          'A answer MCQ · F answer FRQ · C add context · T clear contexts · S show/hide FRQ box · M send FRQ box text · P freeze screen · L crop and lock · ↑/↓ thinking · R reload · Esc cancel selection'
      },
```

- [ ] **Step 2: The component**

```tsx
import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { Button, Divider, Input, InputNumber, message, Select, Switch, Upload } from 'antd';
import { useSetAtom } from 'jotai';
import { Trash2Icon, UploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/assistant.ts';
import type { AssistantConfig, AssistantConfigUpdate, AttachmentInfo } from '@/api/assistant.ts';
import { assistantConfigAtom } from '@/jotai/assistant.ts';

type SecretKey = 'geminiApiKey' | 'orApiKey' | 'proxyPass';
const emptySecrets: Record<SecretKey, string> = { geminiApiKey: '', orApiKey: '', proxyPass: '' };

const Row = ({ label, desc, children }: { label: string; desc?: string; children: ReactNode }) => (
  <div className="flex items-center justify-between gap-4 py-2">
    <div className="flex flex-col">
      <span>{label}</span>
      {desc && <span className="text-xs text-neutral-500">{desc}</span>}
    </div>
    <div className="flex shrink-0 items-center gap-2">{children}</div>
  </div>
);

export const AssistantSettings = () => {
  const { t } = useTranslation();
  const setRuntimeConfig = useSetAtom(assistantConfigAtom);
  const [config, setConfig] = useState<AssistantConfig | null>(null);
  const [secrets, setSecrets] = useState(emptySecrets);
  const [attachments, setAttachments] = useState<AttachmentInfo[]>([]);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    api.getAssistantConfig().then((rsp) => rsp.code === 0 && setConfig(rsp.data));
    loadAttachments();
  }, []);

  function loadAttachments() {
    api.listAttachments().then((rsp) => rsp.code === 0 && setAttachments(rsp.data));
  }

  function update<K extends keyof AssistantConfig>(key: K, value: AssistantConfig[K]) {
    setConfig((c) => (c ? { ...c, [key]: value } : c));
  }

  function applyResponse(rsp: { code: number; msg: string; data: AssistantConfig }) {
    if (rsp.code !== 0) {
      message.error(rsp.msg);
      return false;
    }
    setConfig(rsp.data);
    setRuntimeConfig(rsp.data);
    return true;
  }

  async function save() {
    if (!config) return;
    setIsSaving(true);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { hasGeminiApiKey, hasOrApiKey, hasProxyPass, ...rest } = config;
    const body: AssistantConfigUpdate = { ...rest };
    (Object.keys(secrets) as SecretKey[]).forEach((key) => {
      if (secrets[key]) body[key] = secrets[key];
    });
    try {
      if (applyResponse(await api.setAssistantConfig(body))) {
        setSecrets(emptySecrets);
        message.success(t('settings.assistant.saved'));
      }
    } finally {
      setIsSaving(false);
    }
  }

  async function clearSecret(flag: 'clearGeminiApiKey' | 'clearOrApiKey' | 'clearProxyPass') {
    applyResponse(await api.setAssistantConfig({ [flag]: true }));
  }

  async function upload(file: File) {
    const rsp = await api.uploadAttachment(file);
    if (rsp.code !== 0) message.error(rsp.msg);
    loadAttachments();
  }

  async function remove(name: string) {
    const rsp = await api.deleteAttachment(name);
    if (rsp.code !== 0) message.error(rsp.msg);
    loadAttachments();
  }

  if (!config) return null;

  const secretInput = (key: SecretKey, has: boolean, flag: Parameters<typeof clearSecret>[0]) => (
    <>
      <Input.Password
        className="w-[260px]"
        value={secrets[key]}
        placeholder={has ? t('settings.assistant.secretSet') : ''}
        onChange={(e) => setSecrets((s) => ({ ...s, [key]: e.target.value }))}
      />
      {has && <Button onClick={() => clearSecret(flag)}>{t('settings.assistant.clear')}</Button>}
    </>
  );

  return (
    <>
      <Row label={t('settings.assistant.enabled')} desc={t('settings.assistant.enabledDesc')}>
        <Switch checked={config.enabled} onChange={(v) => update('enabled', v)} />
      </Row>
      <Row label={t('settings.assistant.ui')} desc={t('settings.assistant.uiDesc')}>
        <Switch checked={config.ui} onChange={(v) => update('ui', v)} />
      </Row>

      <Divider />
      <Row label={t('settings.assistant.provider')}>
        <Select
          className="w-[260px]"
          value={config.provider}
          onChange={(v) => update('provider', v)}
          options={[
            { value: 'openrouter', label: 'OpenRouter' },
            { value: 'gemini', label: 'Gemini' }
          ]}
        />
      </Row>
      <div style={{ opacity: config.provider === 'gemini' ? 1 : 0.5 }}>
        <div className="pt-2 font-medium">Gemini</div>
        <Row label={t('settings.assistant.baseUrl')}>
          <Input className="w-[260px]" value={config.geminiBaseUrl} onChange={(e) => update('geminiBaseUrl', e.target.value)} />
        </Row>
        <Row label={t('settings.assistant.model')}>
          <Input className="w-[260px]" value={config.geminiModel} onChange={(e) => update('geminiModel', e.target.value)} />
        </Row>
        <Row label={t('settings.assistant.apiKey')}>
          {secretInput('geminiApiKey', config.hasGeminiApiKey, 'clearGeminiApiKey')}
        </Row>
        <Row label={t('settings.assistant.thinking')}>
          <Switch checked={config.geminiThinking} onChange={(v) => update('geminiThinking', v)} />
        </Row>
        <Row label={t('settings.assistant.thinkingBudget')}>
          <InputNumber min={0} step={512} value={config.thinkingBudget} onChange={(v) => update('thinkingBudget', v ?? 512)} />
        </Row>
      </div>
      <div style={{ opacity: config.provider === 'openrouter' ? 1 : 0.5 }}>
        <div className="pt-2 font-medium">OpenRouter</div>
        <Row label={t('settings.assistant.baseUrl')}>
          <Input className="w-[260px]" value={config.orBaseUrl} onChange={(e) => update('orBaseUrl', e.target.value)} />
        </Row>
        <Row label={t('settings.assistant.model')}>
          <Input className="w-[260px]" value={config.orModel} onChange={(e) => update('orModel', e.target.value)} />
        </Row>
        <Row label={t('settings.assistant.apiKey')}>{secretInput('orApiKey', config.hasOrApiKey, 'clearOrApiKey')}</Row>
        <Row label={t('settings.assistant.reasoning')}>
          <Switch checked={config.orReasoning} onChange={(v) => update('orReasoning', v)} />
        </Row>
        <Row label={t('settings.assistant.reasoningEffort')}>
          <Select
            className="w-[140px]"
            value={config.orReasoningEffort}
            onChange={(v) => update('orReasoningEffort', v)}
            options={['low', 'medium', 'high'].map((v) => ({ value: v, label: v }))}
          />
        </Row>
      </div>

      <Divider />
      <div className="font-medium">{t('settings.assistant.proxy')}</div>
      <Row label={t('settings.assistant.proxyUrl')}>
        <Input className="w-[260px]" value={config.proxyUrl} onChange={(e) => update('proxyUrl', e.target.value)} />
      </Row>
      <Row label={t('settings.assistant.proxyPass')}>{secretInput('proxyPass', config.hasProxyPass, 'clearProxyPass')}</Row>

      <Divider />
      <div className="font-medium">{t('settings.assistant.behaviour')}</div>
      {(['copyClipboard', 'answerSel', 'contextSel', 'frqSel', 'quadClickMCQ'] as const).map((key) => (
        <Row key={key} label={t(`settings.assistant.${key}`)}>
          <Switch checked={config[key]} onChange={(v) => update(key, v)} />
        </Row>
      ))}

      <div className="flex justify-end pt-4">
        <Button type="primary" loading={isSaving} onClick={save}>
          {t('settings.assistant.save')}
        </Button>
      </div>

      <Divider />
      <Row label={t('settings.assistant.attachments')} desc={t('settings.assistant.attachmentsDesc')}>
        <Upload
          showUploadList={false}
          customRequest={({ file, onSuccess }) => {
            upload(file as File).then(() => onSuccess?.(null));
          }}
        >
          <Button icon={<UploadIcon size={14} />}>{t('settings.assistant.upload')}</Button>
        </Upload>
      </Row>
      {attachments.map((a) => (
        <div key={a.name} className="flex items-center justify-between py-1 text-sm">
          <span>
            {a.name} <span className="text-neutral-500">({Math.ceil(a.size / 1024)} KB)</span>
          </span>
          <Button type="text" icon={<Trash2Icon size={14} />} onClick={() => remove(a.name)} />
        </div>
      ))}

      <Divider />
      <div className="font-medium">{t('settings.assistant.hotkeys')}</div>
      <div className="pt-1 text-sm text-neutral-400">{t('settings.assistant.hotkeyList')}</div>
    </>
  );
};
```

- [ ] **Step 3: Register the tab.** In `settings/index.tsx`, add `SparklesIcon` to the lucide import and `import { AssistantSettings } from './assistant';`. Insert this after the `mcp` tab entry:

```tsx
          { id: 'assistant', icon: <SparklesIcon size={16} />, component: <AssistantSettings /> },
```

- [ ] **Step 4: Run the checks.** Run `cd web && pnpm exec tsc --noEmit && pnpm lint && pnpm build`. Expected: no errors.
- [ ] **Step 5: Commit.** `git commit -m "web: assistant settings tab"`

---

### Task 13: Build, OTA to 192.168.8.210, end-to-end

**Files:** none new. Fix defects in the files of earlier tasks and commit each fix separately.

- [ ] **Step 1: Build.** Run `make release-build DOCKER_TTY=` and then `make web` (or `cd web && pnpm build`, whichever `package.sh` expects; read its header comment). Read the device's current version in its web UI (Settings → About or Update). Then run `scripts/package.sh <next patch version>`. Expected: `nanokvm_<version>.tar.gz` and `latest.json`.
- [ ] **Step 2: Install.** Use Playwright against `https://192.168.8.210`. Ask the user for the admin login if it is not already known. Go to Settings → Update → offline update, upload the tarball, and wait for the restart. Then confirm the new version via `/kvmapp/version` (the About tab shows it).
- [ ] **Step 3: Configure.** In Settings → Assistant, enable it, choose OpenRouter, and pick a vision model (e.g. `google/gemini-2.5-flash`). For the key, read `~/.config/tutor/openrouter.key` at test time and type it into the field. Never echo, log, screenshot or commit it; mask any screenshot of the field. Save. Then `GET /api/assistant/config` must show `hasOrApiKey: true` and contain no key.
- [ ] **Step 4: Target content.** Through the KVM, open a text editor on the target. Type this question with the KVM's paste: `What is 17 + 25?  A) 40  B) 42  C) 44  D) 38`. Leave the editor focused; it doubles as the target's key log.
- [ ] **Step 5: Check each behaviour.** Record the result of each:
  - Ctrl Ctrl A: the toast shows B/42 and the bar turns `rgb(89, 105, 192)`. No "a" appears in the target editor.
  - With `copyClipboard` on, A puts the answer in the clipboard (read it with Playwright `navigator.clipboard.readText` after granting permission).
  - F: the answer goes into the FRQ box; S shows it.
  - C: the count goes to 1. Then A: the server log (`/var/log`, or `journalctl` on the device via ssh) shows 2 images sent. T: the count goes to 0 and the bar flashes `#444`.
  - M with text in the FRQ box: the answer is appended after a blank line.
  - P freezes and P again unfreezes. L plus a drag zooms, and a mouse click still lands on the right target pixel. L again resets.
  - ↑/↓ show the reasoning toast and persist (check `GET config`).
  - With answerSel on: A shows the selection, Esc cancels (idle colour, no request), and a click without a drag turns the bar red.
  - Quad-click on the screen triggers an MCQ.
  - With `ui` off: nothing is rendered, and A still copies the answer to the clipboard.
  - No action key and no Escape appeared in the target editor at any point.
- [ ] **Step 6: Relay path.** On this machine run `cd ../../chaice/chaice/chaice-server && RELAY_PASS=<random> PORT=8787 bun run relay/relay.ts` (see its justfile). Set `proxyUrl` to `http://<this machine's LAN IP>:8787` with that password. Ask once; the device log must show `route: proxy http://…:8787 (attempt 1/2)`. Stop the relay and ask again; after about 5 s the log must show `direct (FALLBACK after 2 failed proxy attempts …)` and the answer must still arrive.
- [ ] **Step 7: Clean up.** Clear the key from the device config if the user wants that (ask), stop the relay, and commit any fixes. Do not push without asking.

---

## Self-review notes

- **Spec coverage:** every file the spec lists maps to a task, with two exceptions. `contexts.go`, `attachments.go` and the flows are Tasks 6–7, and `flows.go` is named `service.go` because it also holds the Service. The spec's prompts setting is intentionally dropped (see Deviations).
- **Types used across tasks:** `Crop` has the same `{x,y,w,h}` fraction shape in Go and TS. `ReasoningResult.config` is a `PublicConfig`, which matches the TS `AssistantConfig`. `FRQ_LOCK_SOURCE` is defined in `frq-box.tsx` and used in `index.tsx`.

---

### Task 14: Editable prompts in settings (added 2026-09-24 at the user's request)

The user now wants the prompts to be editable in the settings UI. This supersedes Deviation 1's "prompts are not a setting". The firmware default is still the bundled `prompts.toml`, and **nobody may open, read or print it**. That includes tests, logs, reports, and API responses captured into agent context. Tests may parse it and assert structure (counts, types), but never print values.

**Model.** The prompts are an ordered list of **entries**, one per TOML top-level table (for example `universal`, `mcq`). Each entry is an ordered list of **fields** (`key` → string `value`, for example `prompt`). The UI lists whatever entries and fields exist, with nothing hard-coded, and lets the admin add and remove both.

**Files:**
- Modify: `server/service/assistant/prompts.go`, `server/service/assistant/service.go` (only if needed), `server/service/assistant/handlers.go` (routes)
- Create: `server/service/assistant/prompts_store_test.go` (or extend `prompts_test.go`)
- Modify: `web/src/api/assistant.ts`, `web/src/pages/desktop/menu/settings/assistant/index.tsx` (or a new `prompts.tsx` beside it), `web/src/i18n/locales/en.ts`

**Server:**
- Storage for the override: `/etc/kvm/assistant/prompts.json` (mode 0600, `utils.WriteFileAtomic`). The path is a package var so tests can redirect it.
  - Shape: `{"entries":[{"name":"universal","fields":[{"key":"prompt","value":"..."}]}]}`.
  - JSON is used so the admin's order is kept.
  - If the file doesn't exist, the embedded `prompts.toml` is the source.
- Default order when converting the embedded TOML (TOML maps lose order): `universal` first, then the other tables alphabetically. Within a table, `prompt` first, then the other keys alphabetically.
- Non-string values in the embedded TOML: a test `TestEmbeddedPromptsFitEditableModel` asserts that every top-level value is a table and every field value is a string, without printing anything. If it fails, report NEEDS_CONTEXT. Do not work around it by reading the file.
- `loadPrompts()` returns `map[string]any` as before, built from the override if present, otherwise from the embedded default. `getPrompt` is unchanged, so the ask flows read whatever is current. Read the override on every ask (it's small). The embedded parse can stay behind `sync.Once`.
- Types:
  - `type PromptField struct{ Key string \`json:"key"\`; Value string \`json:"value"\` }`
  - `type PromptEntry struct{ Name string \`json:"name"\`; Fields []PromptField \`json:"fields"\` }`
  - `type PromptSet struct{ Entries []PromptEntry \`json:"entries"\`; IsDefault bool \`json:"isDefault"\` }` (`isDefault` is only on GET responses)
- Validation on save (return `errInvalidConfig`-style `-1` "invalid arguments" with a specific message):
  - Entry names are non-empty after trimming, unique, and contain no `.`, `[` or `]`.
  - Field keys are non-empty after trimming and unique within their entry.
  - Total JSON size is at most 256 KB.
  - Zero entries is allowed, and asks then use `"\n\n"` exactly as `getPrompt` already does.
- Routes, admin-only and on the same group:
  - `GET /api/assistant/prompts` → `PromptSet` (current prompts, with `isDefault`)
  - `POST /api/assistant/prompts` `{entries}` → saves and returns the new `PromptSet`
  - `DELETE /api/assistant/prompts` → removes the override and returns the default `PromptSet`
- Tests use inline TOML or inline JSON, never the embedded file's values:
  - store round trip keeps order
  - validation failures
  - override takes precedence in `loadPrompts`/`getPrompt`
  - reset falls back to the default
  - handler codes
  - `TestEmbeddedPromptsFitEditableModel` as above

**Web:**
- API: `getPrompts()`, `savePrompts(entries)`, `resetPrompts()`, plus the types above.
- A "Prompts" section in the Assistant settings tab, below Behaviour and above Attachments, with its own Save and "Reset to default" buttons (Reset asks for confirmation with an antd `Modal.confirm` or `Popconfirm`):
  - Each entry is a bordered card with an editable name `Input`, a "Remove entry" button, its fields, and an "Add field" button.
  - Each field has a key `Input` (narrow), a "Remove field" button, and a large value `Input.TextArea`. The textarea spans the full width, is monospace, uses `autoSize={{ minRows: 8, maxRows: 30 }}`, and can be resized.
  - An "Add entry" button at the bottom creates `{name: '', fields: [{key: 'prompt', value: ''}]}`.
  - A "Default" or "Customized" tag reflects `isDefault`.
  - Show server validation errors with `message.error(rsp.msg)`.
- i18n keys go in `en.ts` under `settings.assistant.prompts*`.
- The settings modal already holds the keyboard lock, so typing in the textareas doesn't reach the target.

**Verify:**
- Go: GOTEST on the full package, plus gofmt and vet.
- Web: `tsc --noEmit`, eslint on the touched files, `node --experimental-strip-types --test "src/**/*.test.ts"`, and `vite build`.
- Commit: `assistant: editable prompts, bundled prompts.toml as default`.
