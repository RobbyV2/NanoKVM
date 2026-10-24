package modelconfig

import (
	"errors"
	"strings"
	"testing"
)

// picoclawExtractProtocol is PicoClaw v0.2.8's pkg/providers.ExtractProtocol:
// an explicit provider wins and the model is then sent unchanged; otherwise
// the first slash-separated segment is the protocol and the rest is sent.
func picoclawExtractProtocol(provider string, model string) (string, string) {
	model = strings.TrimSpace(model)
	if provider = strings.TrimSpace(provider); provider != "" {
		return NormalizeProtocol(provider), model
	}
	protocol, rest, found := strings.Cut(model, "/")
	if !found {
		return "openai", model
	}
	return NormalizeProtocol(protocol), strings.TrimSpace(rest)
}

func mustResolve(t *testing.T, req Request) Resolved {
	t.Helper()
	resolved, err := Resolve(req)
	if err != nil {
		t.Fatalf("Resolve(%+v) error = %v", req, err)
	}
	return resolved
}

func TestResolveAcceptsIdentifiersPicoclawSupports(t *testing.T) {
	tests := []struct {
		name      string
		req       Request
		protocol  string
		model     string
		modelID   string
		modelName string
		apiBase   string
	}{
		{
			name:      "placeholder with default base",
			req:       Request{Model: "openai/gpt-5.4", APIKey: "sk-test"},
			protocol:  "openai",
			model:     "openai/gpt-5.4",
			modelID:   "gpt-5.4",
			modelName: "gpt-5.4",
			apiBase:   "https://api.openai.com/v1",
		},
		{
			name:      "openrouter keeps the vendor path",
			req:       Request{Model: "openrouter/anthropic/claude-sonnet-4.6", APIKey: "sk-or", APIBase: "https://openrouter.ai/api/v1/"},
			protocol:  "openrouter",
			model:     "openrouter/anthropic/claude-sonnet-4.6",
			modelID:   "anthropic/claude-sonnet-4.6",
			modelName: "claude-sonnet-4.6",
			apiBase:   "https://openrouter.ai/api/v1",
		},
		{
			name:      "nvidia nested model",
			req:       Request{Model: "nvidia/z-ai/glm-5.1", APIKey: "nv"},
			protocol:  "nvidia",
			model:     "nvidia/z-ai/glm-5.1",
			modelID:   "z-ai/glm-5.1",
			modelName: "glm-5.1",
			apiBase:   "https://integrate.api.nvidia.com/v1",
		},
		{
			name:      "minimax",
			req:       Request{Model: "minimax/MiniMax-M2.5", APIKey: "mm"},
			protocol:  "minimax",
			model:     "minimax/MiniMax-M2.5",
			modelID:   "MiniMax-M2.5",
			modelName: "MiniMax-M2.5",
			apiBase:   "https://api.minimaxi.com/v1",
		},
		{
			name:      "alias and surrounding whitespace",
			req:       Request{Model: "  Google/gemini-2.5-flash \n", APIKey: " AIza-test\n", APIBase: " "},
			protocol:  "gemini",
			model:     "google/gemini-2.5-flash",
			modelID:   "gemini-2.5-flash",
			modelName: "gemini-2.5-flash",
			apiBase:   "https://generativelanguage.googleapis.com/v1beta",
		},
		{
			name:      "qwen folds to qwen-portal",
			req:       Request{Model: "qwen/qwen-plus", APIKey: "ds"},
			protocol:  "qwen-portal",
			model:     "qwen/qwen-plus",
			modelID:   "qwen-plus",
			modelName: "qwen-plus",
			apiBase:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
		},
		{
			name:      "custom OpenAI-compatible endpoint",
			req:       Request{Model: "openai/my-model", APIKey: "k", APIBase: "http://192.168.1.20:8080/v1"},
			protocol:  "openai",
			model:     "openai/my-model",
			modelID:   "my-model",
			modelName: "my-model",
			apiBase:   "http://192.168.1.20:8080/v1",
		},
		{
			name:      "local ollama without a key",
			req:       Request{Model: "ollama/llama3.1:8b", APIBase: "http://192.168.1.20:11434/v1"},
			protocol:  "ollama",
			model:     "ollama/llama3.1:8b",
			modelID:   "llama3.1:8b",
			modelName: "llama3.1:8b",
			apiBase:   "http://192.168.1.20:11434/v1",
		},
		{
			name:      "azure with its endpoint",
			req:       Request{Model: "azure/my-gpt5-deployment", APIKey: "az", APIBase: "https://res.openai.azure.com"},
			protocol:  "azure",
			model:     "azure/my-gpt5-deployment",
			modelID:   "my-gpt5-deployment",
			modelName: "my-gpt5-deployment",
			apiBase:   "https://res.openai.azure.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := mustResolve(t, test.req)
			if resolved.Protocol != test.protocol || resolved.Model != test.model ||
				resolved.ModelID != test.modelID || resolved.ModelName != test.modelName ||
				resolved.APIBase != test.apiBase {
				t.Fatalf("Resolve = %+v, want protocol=%s model=%s id=%s name=%s base=%s",
					resolved, test.protocol, test.model, test.modelID, test.modelName, test.apiBase)
			}
			if strings.TrimSpace(test.req.APIKey) != resolved.APIKey {
				t.Fatalf("API key = %q, want trimmed %q", resolved.APIKey, test.req.APIKey)
			}
			if protocol, modelID := picoclawExtractProtocol("", resolved.Model); protocol != resolved.Protocol || modelID != resolved.ModelID {
				t.Fatalf("PicoClaw would resolve %q to (%s, %s), want (%s, %s)",
					resolved.Model, protocol, modelID, resolved.Protocol, resolved.ModelID)
			}
		})
	}
}

func TestResolveRejectsFormsPicoclawCannotUse(t *testing.T) {
	tests := []struct {
		name  string
		req   Request
		field Field
		want  string
	}{
		{"empty model", Request{APIKey: "k"}, FieldModel, "Enter a model identifier"},
		{"missing provider", Request{Model: "gpt-4o-mini", APIKey: "k"}, FieldModel, "openai/gpt-4o-mini"},
		{"typo provider", Request{Model: "openao/deepseek-v4-flash", APIKey: "k"}, FieldModel, "did you mean openai"},
		{"unknown provider", Request{Model: "xai/grok-4", APIKey: "k"}, FieldModel, "enter openai/grok-4 with its API base URL"},
		{"no suggestion for another vendor", Request{Model: "grok/grok-4", APIKey: "k"}, FieldModel, "Supported providers: anthropic"},
		{"oauth provider", Request{Model: "github-copilot/gpt-5.4", APIKey: "k"}, FieldModel, "Copilot bridge"},
		{"missing model id", Request{Model: "openai/", APIKey: "k"}, FieldModel, "missing the model"},
		{"empty segment", Request{Model: "openrouter//gpt-4o", APIKey: "k"}, FieldModel, "empty path segment"},
		{"space in model", Request{Model: "openai/gpt 4o", APIKey: "k"}, FieldModel, "spaces"},
		{"base without scheme", Request{Model: "openai/gpt-4o", APIKey: "k", APIBase: "api.openai.com/v1"}, FieldAPIBase, "http://"},
		{"completions endpoint", Request{Model: "openai/gpt-4o", APIKey: "k", APIBase: "https://api.openai.com/v1/chat/completions"}, FieldAPIBase, "/chat/completions"},
		{"azure without base", Request{Model: "azure/deployment", APIKey: "k"}, FieldAPIBase, "API base URL for azure"},
		{"missing key", Request{Model: "deepseek/deepseek-chat"}, FieldAPIKey, "API key for deepseek"},
		{"key with line break", Request{Model: "openai/gpt-4o", APIKey: "sk-a\nsk-b"}, FieldAPIKey, "line breaks"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Resolve(test.req)
			var fieldErr *Error
			if !errors.As(err, &fieldErr) {
				t.Fatalf("Resolve(%+v) error = %v, want field error", test.req, err)
			}
			if fieldErr.Field != test.field || !strings.Contains(fieldErr.Message, test.want) {
				t.Fatalf("error = %s: %q, want %s containing %q", fieldErr.Field, fieldErr.Message, test.field, test.want)
			}
		})
	}
}

func TestEveryAliasNamesAKnownProtocol(t *testing.T) {
	for alias, canonical := range protocolAliases {
		if _, ok := protocols[canonical]; !ok {
			t.Errorf("alias %q maps to unknown protocol %q", alias, canonical)
		}
	}
}

// onboardTemplate has the shape PicoClaw v0.2.8's onboard writes: aliases
// that a user's identifier can collide with, routed by "provider".
func onboardTemplate() []any {
	return []any{
		map[string]any{"model_name": "glm-4.7", "model": "glm-4.7", "provider": "zhipu", "api_base": "https://open.bigmodel.cn/api/paas/v4"},
		map[string]any{"model_name": "gpt-5.4", "model": "gpt-5.4", "provider": "openai", "api_base": "https://api.openai.com/v1"},
		map[string]any{"model_name": "claude-sonnet-4.6", "model": "claude-sonnet-4.6", "provider": "anthropic", "api_base": "https://api.anthropic.com/v1"},
		map[string]any{"model_name": "copilot-gpt-5.4", "model": "gpt-5.4", "provider": "github-copilot", "auth_method": "oauth", "api_base": "http://localhost:4321"},
	}
}

func entriesNamed(list []any, name string) []map[string]any {
	var matches []map[string]any
	for _, item := range list {
		if entry, ok := item.(map[string]any); ok && EntryName(entry) == name {
			matches = append(matches, entry)
		}
	}
	return matches
}

func TestApplyEntryRoutesTheSavedModelAsEntered(t *testing.T) {
	for _, req := range []Request{
		{Model: "openai/gpt-5.4", APIKey: "k"},
		{Model: "openrouter/openai/gpt-5.4", APIKey: "k"},
		{Model: "openrouter/anthropic/claude-sonnet-4.6", APIKey: "k"},
		{Model: "zhipu/glm-4.7", APIKey: "k"},
	} {
		t.Run(req.Model, func(t *testing.T) {
			resolved := mustResolve(t, req)
			list := ApplyEntry(onboardTemplate(), resolved)

			matches := entriesNamed(list, resolved.ModelName)
			if len(matches) != 1 {
				t.Fatalf("entries named %q = %d, want 1: %v", resolved.ModelName, len(matches), list)
			}
			entry := matches[0]
			protocol, modelID := picoclawExtractProtocol(stringField(entry, "provider"), stringField(entry, "model"))
			if protocol != resolved.Protocol || modelID != resolved.ModelID {
				t.Fatalf("PicoClaw would call %s with %q, want %s with %q (entry %v)",
					protocol, modelID, resolved.Protocol, resolved.ModelID, entry)
			}
			if entry["api_base"] != resolved.APIBase || entry["enabled"] != true {
				t.Fatalf("entry = %v, want api_base %s and enabled", entry, resolved.APIBase)
			}
			if len(list) != len(onboardTemplate()) {
				t.Fatalf("list length = %d, want the template entry replaced in place", len(list))
			}
		})
	}
}

func TestApplyEntryKeepsTuningOnlyForTheSameProtocol(t *testing.T) {
	existing := []any{
		map[string]any{"model_name": "gpt-5.4", "model": "openai/gpt-5.4", "api_base": "https://old.example/v1", "request_timeout": 90, "proxy": "http://proxy:3128"},
		map[string]any{"model_name": "claude-sonnet-4.6", "provider": "anthropic", "model": "claude-sonnet-4.6", "extra_body": map[string]any{"x": 1}, "auth_method": "oauth"},
	}

	same := ApplyEntry(existing, mustResolve(t, Request{Model: "openai/gpt-5.4", APIKey: "k"}))
	entry := entriesNamed(same, "gpt-5.4")[0]
	if entry["request_timeout"] != 90 || entry["proxy"] != "http://proxy:3128" {
		t.Fatalf("same-protocol entry lost its tuning: %v", entry)
	}
	if entry["api_base"] != "https://api.openai.com/v1" {
		t.Fatalf("api_base = %v, want the submitted default", entry["api_base"])
	}

	other := ApplyEntry(existing, mustResolve(t, Request{Model: "openrouter/anthropic/claude-sonnet-4.6", APIKey: "k"}))
	entry = entriesNamed(other, "claude-sonnet-4.6")[0]
	for _, key := range []string{"provider", "extra_body", "auth_method"} {
		if _, ok := entry[key]; ok {
			t.Fatalf("entry for a different protocol kept %s: %v", key, entry)
		}
	}
}

func TestApplyEntryCollapsesDuplicatesAndAppendsNewAliases(t *testing.T) {
	existing := []any{
		map[string]any{"model_name": "a", "model": "openai/a"},
		map[string]any{"model_name": "gpt-4o", "model": "openai/gpt-4o", "api_base": "https://one.example/v1"},
		map[string]any{"model_name": "b", "model": "openai/b"},
		map[string]any{"model_name": "gpt-4o", "model": "openai/gpt-4o", "api_base": "https://two.example/v1"},
	}
	list := ApplyEntry(existing, mustResolve(t, Request{Model: "openai/gpt-4o", APIKey: "k"}))
	if len(list) != 3 || EntryName(list[1].(map[string]any)) != "gpt-4o" {
		t.Fatalf("list = %v, want one gpt-4o entry at the first duplicate's position", list)
	}

	list = ApplyEntry(existing[:1], mustResolve(t, Request{Model: "deepseek/deepseek-chat", APIKey: "k"}))
	if len(list) != 2 || EntryName(list[1].(map[string]any)) != "deepseek-chat" {
		t.Fatalf("list = %v, want the new alias appended", list)
	}
	if list := ApplyEntry(nil, mustResolve(t, Request{Model: "openai/gpt-4o", APIKey: "k"})); len(list) != 1 {
		t.Fatalf("list = %v, want a single entry", list)
	}
}

func TestIsSecurityKeyFor(t *testing.T) {
	tests := []struct {
		key   string
		name  string
		match bool
	}{
		{"gpt-5.4:0", "gpt-5.4", true},
		{"gpt-5.4:12", "gpt-5.4", true},
		{"gpt-5.4", "gpt-5.4", true},
		{"gpt-5.4:", "gpt-5.4", false},
		{"gpt-5.4:x", "gpt-5.4", false},
		{"openrouter-gpt-5.4:0", "gpt-5.4", false},
		{"llama3.1:8b:0", "llama3.1", false},
		{"llama3.1:8b:0", "llama3.1:8b", true},
	}
	for _, test := range tests {
		if got := IsSecurityKeyFor(test.key, test.name); got != test.match {
			t.Errorf("IsSecurityKeyFor(%q, %q) = %v, want %v", test.key, test.name, got, test.match)
		}
	}
}

func TestUsable(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		apiBase  string
		hasKey   bool
		usable   bool
	}{
		{"default base with key", "", "openai/gpt-5.4", "", true, true},
		{"no key", "", "openai/gpt-5.4", "https://api.openai.com/v1", false, false},
		{"azure needs a base", "", "azure/deployment", "", true, false},
		{"local ollama without key", "", "ollama/llama3", "http://localhost:11434/v1", false, true},
		{"provider field decides", "ollama", "llama3", "", false, true},
		{"unknown protocol needs a base", "", "custom/model", "", true, false},
		{"unknown protocol with base and key", "", "custom/model", "https://example.invalid", true, true},
	}
	for _, test := range tests {
		if got := Usable(test.provider, test.model, test.apiBase, test.hasKey); got != test.usable {
			t.Errorf("%s: Usable = %v, want %v", test.name, got, test.usable)
		}
	}
}
