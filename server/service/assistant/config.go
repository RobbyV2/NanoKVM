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
	defaultORModel       = "openai/gpt-6-astra"
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
		ORModel:           defaultORModel,
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
	setTrimmed(&c.ORModel, u.ORModel, defaultORModel)
	setTrimmed(&c.ProxyURL, u.ProxyURL, "")
	setSecret(&c.GeminiAPIKey, u.GeminiAPIKey, u.ClearGeminiAPIKey)
	setSecret(&c.ORAPIKey, u.ORAPIKey, u.ClearORAPIKey)
	setSecret(&c.ProxyPass, u.ProxyPass, u.ClearProxyPass)
	return c, nil
}
