package picoclaw

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/service/picoclaw/modelconfig"
)

func TestUpdatePicoclawModelConfigInitializesMissingConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	previousOnboard := runPicoclawOnboardForConfig
	onboardCalled := false
	runPicoclawOnboardForConfig = func() (string, *PicoclawError) {
		onboardCalled = true
		configPath := filepath.Join(home, "config.json")
		err := os.WriteFile(configPath, []byte(`{
  "agents": {
    "defaults": {}
  },
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790
  },
  "model_list": [],
  "channel_list": {}
}`), 0o600)
		if err != nil {
			return "", newPicoclawError(CodeRuntimeUnavailable, err.Error())
		}
		return "initialized", nil
	}
	t.Cleanup(func() {
		runPicoclawOnboardForConfig = previousOnboard
	})

	if err := savePicoclawModelConfig(mustResolveModel(t, "openai/test-model", "https://api.example.invalid", "secret-key")); err != nil {
		t.Fatal(err)
	}
	if !onboardCalled {
		t.Fatal("missing config did not trigger PicoClaw onboard")
	}

	doc, err := loadPicoclawConfigDocument()
	if err != nil {
		t.Fatal(err)
	}
	if doc.config.Agents.Defaults.ModelName != "test-model" {
		t.Fatalf("default model = %q, want test-model", doc.config.Agents.Defaults.ModelName)
	}
	if version, ok := doc.raw["version"].(float64); !ok || int(version) != currentPicoclawConfigVersion {
		t.Fatalf("config version = %v, want %d", doc.raw["version"], currentPicoclawConfigVersion)
	}
	if !isPicoclawModelConfigured(doc.config, doc.security, "test-model") {
		t.Fatalf("model was not configured: config=%+v security=%+v", doc.config.ModelList, doc.security.ModelList)
	}
	if len(doc.config.ModelList) != 1 || doc.config.ModelList[0].APIKey != "" || len(doc.config.ModelList[0].APIKeys) != 0 {
		t.Fatalf("model API key leaked into config.json: %+v", doc.config.ModelList)
	}
}

func TestUpdatePicoclawModelConfigReportsOnboardFailure(t *testing.T) {
	t.Setenv("PICOCLAW_HOME", t.TempDir())

	previousOnboard := runPicoclawOnboardForConfig
	runPicoclawOnboardForConfig = func() (string, *PicoclawError) {
		return "", newPicoclawError(CodeRuntimeUnavailable, "onboard boom")
	}
	t.Cleanup(func() {
		runPicoclawOnboardForConfig = previousOnboard
	})

	err := savePicoclawModelConfig(mustResolveModel(t, "openai/test-model", "https://api.example.invalid", "secret-key"))
	if err == nil {
		t.Fatal("expected onboard failure")
	}
	if !strings.Contains(err.Error(), "failed to initialize PicoClaw config before saving model config") {
		t.Fatalf("error = %v, want initialization context", err)
	}
}

func TestUpdatePicoclawModelConfigMigratesVersionAndKeepsUnknownFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	configPath := filepath.Join(home, "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "version": 4,
  "unknown_top_level": {
    "keep": true
  },
  "agents": {
    "defaults": {
      "model_name": "old-model"
    }
  },
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790
  },
  "model_list": [
    {
      "model_name": "old-model",
      "model": "openai/old-model",
      "api_base": "https://api.example.invalid"
    }
  ],
  "channel_list": {}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := savePicoclawModelConfig(mustResolveModel(t, "openai/new-model", "https://api.example.invalid", "secret-key")); err != nil {
		t.Fatal(err)
	}

	doc, err := loadPicoclawConfigDocument()
	if err != nil {
		t.Fatal(err)
	}
	if version, ok := doc.raw["version"].(float64); !ok || int(version) != currentPicoclawConfigVersion {
		t.Fatalf("config version = %v, want %d", doc.raw["version"], currentPicoclawConfigVersion)
	}
	unknown, ok := doc.raw["unknown_top_level"].(map[string]any)
	if !ok || unknown["keep"] != true {
		t.Fatalf("unknown fields were not preserved: %#v", doc.raw["unknown_top_level"])
	}
}

func mustResolveModel(t *testing.T, model string, apiBase string, apiKey string) modelconfig.Resolved {
	t.Helper()
	resolved, err := modelconfig.Resolve(modelconfig.Request{Model: model, APIBase: apiBase, APIKey: apiKey})
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// TestSavePicoclawModelConfigReplacesOnboardTemplate saves the panel's
// placeholder identifier over the configuration PicoClaw's onboard writes,
// whose "gpt-5.4" template routes through an explicit provider. The saved
// entry must be the only one under the alias and carry no provider, or
// PicoClaw sends "openai/gpt-5.4" to the endpoint as the model name. Secrets
// NanoKVM does not manage must survive the rewrite.
func TestSavePicoclawModelConfigReplacesOnboardTemplate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(`{
  "version": 3,
  "agents": {"defaults": {"model_name": ""}},
  "gateway": {"host": "127.0.0.1", "port": 18790},
  "model_list": [
    {"model_name": "glm-4.7", "model": "glm-4.7", "provider": "zhipu", "api_base": "https://open.bigmodel.cn/api/paas/v4"},
    {"model_name": "gpt-5.4", "model": "gpt-5.4", "provider": "openai", "api_base": "https://api.openai.com/v1"},
    {"model_name": "openrouter-gpt-5.4", "model": "openai/gpt-5.4", "provider": "openrouter", "api_base": "https://openrouter.ai/api/v1"}
  ],
  "channel_list": {}
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".security.yml"), []byte(`model_list:
  glm-4.7:0: {}
  gpt-5.4:0: {}
  gpt-5.4: {api_keys: [stale-bare-key]}
  openrouter-gpt-5.4:0: {}
channel_list:
  pico:
    settings:
      token: pico-token
  slack:
    settings:
      bot_token: xoxb-keep
web:
  brave:
    api_keys: [brave-keep]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := savePicoclawModelConfig(mustResolveModel(t, "openai/gpt-5.4", "", "sk-new")); err != nil {
		t.Fatal(err)
	}

	doc, err := loadPicoclawConfigDocument()
	if err != nil {
		t.Fatal(err)
	}
	var entries []map[string]any
	for _, item := range doc.raw["model_list"].([]any) {
		entry := item.(map[string]any)
		if entry["model_name"] == "gpt-5.4" {
			entries = append(entries, entry)
		}
	}
	if len(entries) != 1 {
		t.Fatalf("gpt-5.4 entries = %v, want one", entries)
	}
	if _, ok := entries[0]["provider"]; ok || entries[0]["model"] != "openai/gpt-5.4" || entries[0]["api_base"] != "https://api.openai.com/v1" {
		t.Fatalf("gpt-5.4 entry = %v, want model openai/gpt-5.4 with the default base and no provider", entries[0])
	}
	if doc.config.Agents.Defaults.ModelName != "gpt-5.4" {
		t.Fatalf("default model = %q, want gpt-5.4", doc.config.Agents.Defaults.ModelName)
	}

	if keys := doc.security.ModelList["gpt-5.4:0"].APIKeys; len(keys) != 1 || keys[0] != "sk-new" {
		t.Fatalf("gpt-5.4:0 keys = %v, want [sk-new]", keys)
	}
	if _, ok := doc.security.ModelList["gpt-5.4"]; ok {
		t.Fatal("stale bare-alias key was kept")
	}
	if _, ok := doc.security.ModelList["openrouter-gpt-5.4:0"]; !ok {
		t.Fatal("another alias's security entry was removed")
	}
	if !isPicoclawModelConfigured(doc.config, doc.security, "gpt-5.4") {
		t.Fatal("saved model is not reported as configured")
	}

	data, err := os.ReadFile(filepath.Join(home, ".security.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"pico-token", "xoxb-keep", "brave-keep"} {
		if !strings.Contains(string(data), secret) {
			t.Fatalf(".security.yml lost %s:\n%s", secret, data)
		}
	}
}

func TestSavePicoclawModelConfigAllowsKeylessLocalServer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(`{"version": 3, "agents": {"defaults": {}}, "model_list": [], "channel_list": {}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := savePicoclawModelConfig(mustResolveModel(t, "ollama/llama3.1:8b", "http://192.168.1.20:11434/v1", "")); err != nil {
		t.Fatal(err)
	}
	doc, err := loadPicoclawConfigDocument()
	if err != nil {
		t.Fatal(err)
	}
	if !isPicoclawModelConfigured(doc.config, doc.security, "llama3.1:8b") {
		t.Fatalf("keyless ollama model is not configured: %+v", doc.config.ModelList)
	}
}

func TestIsPicoclawModelConfiguredFollowsPicoclawKeyLookup(t *testing.T) {
	var cfg picoclawConfigFile
	if err := json.Unmarshal([]byte(`{"model_list": [
    {"model_name": "m", "model": "openai/m", "api_base": "https://a.example/v1"},
    {"model_name": "m", "model": "openai/m", "api_base": "https://b.example/v1"}
  ]}`), &cfg); err != nil {
		t.Fatal(err)
	}
	withKeys := func(keys ...string) picoclawModelSecurityEntry {
		return picoclawModelSecurityEntry{APIKeys: keys}
	}

	tests := []struct {
		name       string
		security   map[string]picoclawModelSecurityEntry
		configured bool
	}{
		{"both indexed", map[string]picoclawModelSecurityEntry{"m:0": withKeys("a"), "m:1": withKeys("b")}, true},
		{"bare alias covers both", map[string]picoclawModelSecurityEntry{"m": withKeys("a")}, true},
		{"second balanced entry has no key", map[string]picoclawModelSecurityEntry{"m:0": withKeys("a")}, false},
		{"empty indexed entry hides the bare alias", map[string]picoclawModelSecurityEntry{"m:0": {}, "m:1": {}, "m": withKeys("a")}, false},
		{"no keys", map[string]picoclawModelSecurityEntry{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isPicoclawModelConfigured(cfg, picoclawSecurityConfig{ModelList: test.security}, "m")
			if got != test.configured {
				t.Fatalf("configured = %v, want %v", got, test.configured)
			}
		})
	}
}
