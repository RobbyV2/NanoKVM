// Package modelconfig turns the PicoClaw panel's model form into the
// model_list entry and credential that PicoClaw reads.
//
// PicoClaw resolves a model_list entry in two steps: an explicit "provider"
// field wins, otherwise the first slash-separated segment of "model" is the
// protocol and the remainder is the model ID sent upstream. The protocol then
// selects a client, its default API base and whether an API key is required.
// This package mirrors that resolution for the PicoClaw release NanoKVM
// installs, so that every form it accepts is a model PicoClaw can call and
// every identifier PicoClaw would reject is refused with a message the panel
// can show next to the offending field.
//
// The package has no dependency on the native libraries the rest of the
// PicoClaw service links against, so its tests run on any host.
package modelconfig

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unicode"
)

// Field names a form input. Validation errors carry one so the panel can mark
// the input that needs to change.
type Field string

const (
	FieldModel   Field = "model"
	FieldAPIBase Field = "api_base"
	FieldAPIKey  Field = "api_key"
)

// Error is a validation failure for one form field.
type Error struct {
	Field   Field
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func fieldError(field Field, format string, args ...any) *Error {
	return &Error{Field: field, Message: fmt.Sprintf(format, args...)}
}

// Request is the model form as submitted.
type Request struct {
	Model   string
	APIBase string
	APIKey  string
}

// Resolved is a validated form, normalized to what is written to PicoClaw's
// configuration.
type Resolved struct {
	// Protocol is PicoClaw's canonical protocol name, after alias folding.
	Protocol string
	// Model is the value written to the model_list entry's "model" field:
	// the protocol prefix as entered (lowercased) and the model ID.
	Model string
	// ModelID is the identifier PicoClaw sends to the provider.
	ModelID string
	// ModelName is the model_list alias used by agents.defaults.model_name.
	ModelName string
	// APIBase is the endpoint root, with the protocol default filled in.
	APIBase string
	// APIKey is empty only for protocols that accept unauthenticated calls.
	APIKey string
}

type protocolInfo struct {
	// defaultAPIBase is used when the form leaves the API base empty. An
	// empty value means the user must provide one.
	defaultAPIBase string
	// keyOptional marks local servers PicoClaw calls without a key.
	keyOptional bool
	// unsupported explains why the form cannot configure the protocol even
	// though PicoClaw knows it: its credentials do not come from an API key.
	unsupported string
}

// protocols lists the canonical protocols PicoClaw v0.2.8 creates a client for
// (pkg/providers.CreateProviderFromConfig), with the defaults from its
// protocolMetaByName table. Update it together with the PicoClaw release
// pinned in runtime_constants.go.
var protocols = map[string]protocolInfo{
	"openai":                {defaultAPIBase: "https://api.openai.com/v1"},
	"anthropic":             {defaultAPIBase: "https://api.anthropic.com/v1"},
	"anthropic-messages":    {defaultAPIBase: "https://api.anthropic.com/v1"},
	"azure":                 {},
	"azure-openai":          {},
	"gemini":                {defaultAPIBase: "https://generativelanguage.googleapis.com/v1beta"},
	"minimax":               {defaultAPIBase: "https://api.minimaxi.com/v1"},
	"litellm":               {defaultAPIBase: "http://localhost:4000/v1"},
	"lmstudio":              {defaultAPIBase: "http://localhost:1234/v1", keyOptional: true},
	"ollama":                {defaultAPIBase: "http://localhost:11434/v1", keyOptional: true},
	"vllm":                  {defaultAPIBase: "http://localhost:8000/v1", keyOptional: true},
	"openrouter":            {defaultAPIBase: "https://openrouter.ai/api/v1"},
	"groq":                  {defaultAPIBase: "https://api.groq.com/openai/v1"},
	"zhipu":                 {defaultAPIBase: "https://open.bigmodel.cn/api/paas/v4"},
	"nvidia":                {defaultAPIBase: "https://integrate.api.nvidia.com/v1"},
	"venice":                {defaultAPIBase: "https://api.venice.ai/api/v1"},
	"moonshot":              {defaultAPIBase: "https://api.moonshot.cn/v1"},
	"shengsuanyun":          {defaultAPIBase: "https://router.shengsuanyun.com/api/v1"},
	"deepseek":              {defaultAPIBase: "https://api.deepseek.com/v1"},
	"cerebras":              {defaultAPIBase: "https://api.cerebras.ai/v1"},
	"vivgrid":               {defaultAPIBase: "https://api.vivgrid.com/v1"},
	"volcengine":            {defaultAPIBase: "https://ark.cn-beijing.volces.com/api/v3"},
	"qwen-portal":           {defaultAPIBase: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
	"qwen-intl":             {defaultAPIBase: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
	"qwen-us":               {defaultAPIBase: "https://dashscope-us.aliyuncs.com/compatible-mode/v1"},
	"coding-plan":           {defaultAPIBase: "https://coding-intl.dashscope.aliyuncs.com/v1"},
	"coding-plan-anthropic": {defaultAPIBase: "https://coding-intl.dashscope.aliyuncs.com/apps/anthropic"},
	"zai":                   {defaultAPIBase: "https://api.z.ai/api/coding/paas/v4"},
	"mistral":               {defaultAPIBase: "https://api.mistral.ai/v1"},
	"avian":                 {defaultAPIBase: "https://api.avian.io/v1"},
	"longcat":               {defaultAPIBase: "https://api.longcat.chat/openai"},
	"modelscope":            {defaultAPIBase: "https://api-inference.modelscope.cn/v1"},
	"novita":                {defaultAPIBase: "https://api.novita.ai/openai"},
	"mimo":                  {defaultAPIBase: "https://api.xiaomimimo.com/v1"},
	"bedrock":               {unsupported: "uses AWS credentials from the environment"},
	"antigravity":           {unsupported: "signs in with OAuth through `picoclaw auth login`"},
	"github-copilot":        {unsupported: "connects through the GitHub Copilot bridge"},
	"claude-cli":            {unsupported: "runs the local Claude CLI"},
	"codex-cli":             {unsupported: "runs the local Codex CLI"},
}

// protocolAliases folds the spellings PicoClaw accepts onto the canonical names
// above (pkg/providers.NormalizeProvider and the alias cases of
// CreateProviderFromConfig).
var protocolAliases = map[string]string{
	"gpt":                      "openai",
	"claude":                   "anthropic",
	"glm":                      "zhipu",
	"google":                   "gemini",
	"qwen":                     "qwen-portal",
	"qwen-international":       "qwen-intl",
	"dashscope-intl":           "qwen-intl",
	"dashscope-us":             "qwen-us",
	"alibaba-coding":           "coding-plan",
	"qwen-coding":              "coding-plan",
	"alibaba-coding-anthropic": "coding-plan-anthropic",
	"z.ai":                     "zai",
	"z-ai":                     "zai",
	"copilot":                  "github-copilot",
	"claudecli":                "claude-cli",
	"codexcli":                 "codex-cli",
}

// NormalizeProtocol returns PicoClaw's canonical protocol name for a prefix.
func NormalizeProtocol(prefix string) string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if canonical, ok := protocolAliases[prefix]; ok {
		return canonical
	}
	return prefix
}

// KeyOptional reports whether PicoClaw calls the protocol without an API key.
func KeyOptional(protocol string) bool {
	return protocols[NormalizeProtocol(protocol)].keyOptional
}

// EntryProtocol returns the protocol PicoClaw uses for a model_list entry.
func EntryProtocol(provider string, model string) string {
	if provider = strings.TrimSpace(provider); provider != "" {
		return NormalizeProtocol(provider)
	}
	prefix, _, found := strings.Cut(strings.TrimSpace(model), "/")
	if !found {
		return "openai"
	}
	return NormalizeProtocol(prefix)
}

// Resolve validates the form and fills in protocol defaults.
func Resolve(req Request) (Resolved, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return Resolved{}, fieldError(FieldModel, "Enter a model identifier, for example openai/gpt-5.4")
	}
	if containsSpace(model) {
		return Resolved{}, fieldError(FieldModel, "The model identifier must not contain spaces")
	}

	prefix, modelID, found := strings.Cut(model, "/")
	if !found {
		return Resolved{}, fieldError(FieldModel,
			"Start the model identifier with its provider, for example openai/%s", model)
	}
	prefix = strings.ToLower(prefix)
	if prefix == "" {
		return Resolved{}, fieldError(FieldModel, "The model identifier is missing its provider before the first /")
	}
	if modelID == "" {
		return Resolved{}, fieldError(FieldModel, "The model identifier is missing the model after %s/", prefix)
	}
	for _, segment := range strings.Split(modelID, "/") {
		if segment == "" {
			return Resolved{}, fieldError(FieldModel, "The model identifier %q contains an empty path segment", model)
		}
	}

	protocol := NormalizeProtocol(prefix)
	info, known := protocols[protocol]
	if !known {
		message := fmt.Sprintf("PicoClaw does not support the provider %q", prefix)
		if suggestion := suggestProtocol(prefix); suggestion != "" {
			message += fmt.Sprintf("; did you mean %s?", suggestion)
		} else {
			message += fmt.Sprintf(". For another OpenAI-compatible service, enter openai/%s with its API base URL. Supported providers: %s",
				modelID, strings.Join(SupportedPrefixes(), ", "))
		}
		return Resolved{}, &Error{Field: FieldModel, Message: message}
	}
	if info.unsupported != "" {
		return Resolved{}, fieldError(FieldModel,
			"The %s provider %s and cannot be configured with an API key here", prefix, info.unsupported)
	}

	apiBase, err := resolveAPIBase(req.APIBase, prefix, info)
	if err != nil {
		return Resolved{}, err
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if containsSpace(apiKey) {
		return Resolved{}, fieldError(FieldAPIKey, "The API key must not contain spaces or line breaks")
	}
	if apiKey == "" && !info.keyOptional {
		return Resolved{}, fieldError(FieldAPIKey, "Enter the API key for %s", prefix)
	}

	segments := strings.Split(modelID, "/")
	return Resolved{
		Protocol:  protocol,
		Model:     prefix + "/" + modelID,
		ModelID:   modelID,
		ModelName: segments[len(segments)-1],
		APIBase:   apiBase,
		APIKey:    apiKey,
	}, nil
}

func resolveAPIBase(raw string, prefix string, info protocolInfo) (string, error) {
	apiBase := strings.TrimSpace(raw)
	if apiBase == "" {
		if info.defaultAPIBase == "" {
			return "", fieldError(FieldAPIBase, "Enter the API base URL for %s", prefix)
		}
		return info.defaultAPIBase, nil
	}
	if containsSpace(apiBase) {
		return "", fieldError(FieldAPIBase, "The API base URL must not contain spaces")
	}

	parsed, err := url.Parse(apiBase)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fieldError(FieldAPIBase,
			"Enter the API base URL with http:// or https://, for example https://api.openai.com/v1")
	}

	apiBase = strings.TrimRight(apiBase, "/")
	lowerPath := strings.ToLower(strings.TrimRight(parsed.Path, "/"))
	for _, endpoint := range []string{"/chat/completions", "/messages"} {
		if strings.HasSuffix(lowerPath, endpoint) {
			return "", fieldError(FieldAPIBase,
				"Enter the API base URL without %s; PicoClaw adds the endpoint itself", endpoint)
		}
	}
	return apiBase, nil
}

func containsSpace(value string) bool {
	return strings.IndexFunc(value, unicode.IsSpace) >= 0
}

// SupportedPrefixes lists the canonical prefixes the form accepts.
func SupportedPrefixes() []string {
	prefixes := make([]string, 0, len(protocols))
	for name, info := range protocols {
		if info.unsupported == "" {
			prefixes = append(prefixes, name)
		}
	}
	sort.Strings(prefixes)
	return prefixes
}

// suggestProtocol returns a supported prefix one edit away from a mistyped
// one. Short prefixes get no suggestion: "xai" and "grok" are other vendors,
// not typos of "zai" and "groq".
func suggestProtocol(prefix string) string {
	if len(prefix) < 5 {
		return ""
	}
	candidates := make([]string, 0, len(protocols)+len(protocolAliases))
	for name := range protocols {
		candidates = append(candidates, name)
	}
	for alias := range protocolAliases {
		candidates = append(candidates, alias)
	}
	sort.Strings(candidates)
	for _, candidate := range candidates {
		if protocols[NormalizeProtocol(candidate)].unsupported == "" && editDistance(prefix, candidate) == 1 {
			return candidate
		}
	}
	return ""
}

func editDistance(a string, b string) int {
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}
