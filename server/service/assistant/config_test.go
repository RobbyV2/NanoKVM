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
	if cfg.ORModel != "openai/gpt-6-astra" {
		t.Fatalf("default ORModel=%q", cfg.ORModel)
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
	cfg, _ := applyUpdate(base, ConfigUpdate{ORBaseURL: ptr("  "), ORModel: ptr("  "), ProxyURL: ptr(" http://r:1 ")})
	if cfg.ORBaseURL != defaultORBaseURL || cfg.ORModel != defaultORModel || cfg.ProxyURL != "http://r:1" {
		t.Fatalf("normalise wrong: %+v", cfg)
	}
}
