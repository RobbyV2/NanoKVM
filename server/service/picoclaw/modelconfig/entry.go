package modelconfig

import (
	"fmt"
	"strings"
)

// routingFields decide which client PicoClaw builds for an entry and how it
// authenticates. A saved entry never inherits them from the entry it replaces:
// PicoClaw's "provider" field takes precedence over the protocol prefix in
// "model", so a provider left behind by an onboarding template that shares the
// alias would send the prefixed identifier to the wrong endpoint.
var routingFields = map[string]struct{}{
	"model_name":   {},
	"model":        {},
	"provider":     {},
	"api_base":     {},
	"api_key":      {},
	"api_keys":     {},
	"auth_method":  {},
	"connect_mode": {},
	"workspace":    {},
	"enabled":      {},
}

// ApplyEntry returns modelList with every entry named r.ModelName replaced by
// one entry built from r, placed where the first of them was. Other fields of
// the replaced entry, such as a proxy or request timeout, are kept only when
// it used the same protocol. PicoClaw load-balances across entries sharing an
// alias, so leaving a second one behind would route some requests to it.
func ApplyEntry(modelList []any, r Resolved) []any {
	entry := map[string]any{}
	position := -1
	updated := make([]any, 0, len(modelList)+1)
	for _, item := range modelList {
		existing, ok := item.(map[string]any)
		if !ok || EntryName(existing) != r.ModelName {
			updated = append(updated, item)
			continue
		}
		if position < 0 {
			position = len(updated)
			if EntryProtocol(stringField(existing, "provider"), stringField(existing, "model")) == r.Protocol {
				for key, value := range existing {
					if _, routing := routingFields[key]; !routing {
						entry[key] = value
					}
				}
			}
		}
	}

	entry["model_name"] = r.ModelName
	entry["model"] = r.Model
	entry["api_base"] = r.APIBase
	entry["enabled"] = true

	if position < 0 {
		return append(updated, entry)
	}
	updated = append(updated, nil)
	copy(updated[position+1:], updated[position:])
	updated[position] = entry
	return updated
}

// EntryName returns the alias of a raw model_list entry.
func EntryName(entry map[string]any) string {
	return stringField(entry, "model_name")
}

func stringField(entry map[string]any, key string) string {
	value, ok := entry[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

// SecurityKey is the .security.yml model_list key PicoClaw reads for the
// index-th entry named modelName.
func SecurityKey(modelName string, index int) string {
	return fmt.Sprintf("%s:%d", modelName, index)
}

// IsSecurityKeyFor reports whether a .security.yml model_list key holds
// credentials for an entry named modelName: the bare alias, which PicoClaw
// falls back to, or the alias followed by an entry index. A model ID may itself
// contain a colon, so "llama3.1:8b:0" belongs to "llama3.1:8b", not "llama3.1".
func IsSecurityKeyFor(key string, modelName string) bool {
	if key == modelName {
		return true
	}
	index, found := strings.CutPrefix(key, modelName+":")
	if !found || index == "" {
		return false
	}
	return strings.IndexFunc(index, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

// Usable reports whether PicoClaw can call an entry: it has an endpoint, its
// own or the protocol default, and a key unless the protocol needs none.
func Usable(provider string, model string, apiBase string, hasKey bool) bool {
	protocol := EntryProtocol(provider, model)
	if strings.TrimSpace(apiBase) == "" && protocols[protocol].defaultAPIBase == "" {
		return false
	}
	return hasKey || protocols[protocol].keyOptional
}
