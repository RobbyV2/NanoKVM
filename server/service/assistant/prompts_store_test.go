package assistant

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useTestPrompts redirects the override file and swaps the embedded default
// for inline TOML, so no test depends on the bundled file's values.
func useTestPrompts(t *testing.T, defaultTOML string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "assistant", "prompts.json")
	oldPath, oldDefault := promptsFilePath, defaultPrompts
	promptsFilePath = path
	defaultPrompts = func() (map[string]any, error) { return parsePrompts([]byte(defaultTOML)) }
	t.Cleanup(func() { promptsFilePath, defaultPrompts = oldPath, oldDefault })
	return path
}

const inlineDefaultTOML = "[mcq]\nprompt = \"M\"\n[universal]\nzeta = \"Z\"\nprompt = \"U\"\nalpha = \"A\"\n[frq]\nprompt = \"F\"\n"

func entryNames(set PromptSet) []string {
	var out []string
	for _, e := range set.Entries {
		out = append(out, e.Name)
	}
	return out
}

func fieldKeys(e PromptEntry) []string {
	var out []string
	for _, f := range e.Fields {
		out = append(out, f.Key)
	}
	return out
}

func TestDefaultPromptSetOrder(t *testing.T) {
	useTestPrompts(t, inlineDefaultTOML)
	set, err := loadPromptSet()
	if err != nil {
		t.Fatal(err)
	}
	if !set.IsDefault {
		t.Fatal("expected isDefault")
	}
	if got := strings.Join(entryNames(set), ","); got != "universal,frq,mcq" {
		t.Fatalf("entry order %s", got)
	}
	if got := strings.Join(fieldKeys(set.Entries[0]), ","); got != "prompt,alpha,zeta" {
		t.Fatalf("field order %s", got)
	}
	if set.Entries[0].Fields[0].Value != "U" {
		t.Fatalf("value %q", set.Entries[0].Fields[0].Value)
	}
}

func TestSavePromptsRoundTripKeepsOrder(t *testing.T) {
	path := useTestPrompts(t, inlineDefaultTOML)
	entries := []PromptEntry{
		{Name: "zz", Fields: []PromptField{{Key: "b", Value: "2"}, {Key: "a", Value: "1"}}},
		{Name: "universal", Fields: []PromptField{{Key: "prompt", Value: "U2"}}},
		{Name: "aa", Fields: nil},
	}
	saved, err := savePrompts(entries)
	if err != nil {
		t.Fatal(err)
	}
	if saved.IsDefault {
		t.Fatal("saved set reported as default")
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode().Perm())
	}
	got, err := loadPromptSet()
	if err != nil {
		t.Fatal(err)
	}
	if got.IsDefault {
		t.Fatal("loaded override reported as default")
	}
	if names := strings.Join(entryNames(got), ","); names != "zz,universal,aa" {
		t.Fatalf("entry order %s", names)
	}
	if keys := strings.Join(fieldKeys(got.Entries[0]), ","); keys != "b,a" {
		t.Fatalf("field order %s", keys)
	}
	if got.Entries[2].Fields == nil {
		t.Fatal("empty fields should round trip as a list")
	}
}

func TestSavePromptsValidation(t *testing.T) {
	useTestPrompts(t, inlineDefaultTOML)
	f := []PromptField{{Key: "prompt", Value: "x"}}
	cases := map[string][]PromptEntry{
		"empty name":     {{Name: "  ", Fields: f}},
		"duplicate name": {{Name: "a", Fields: f}, {Name: " a ", Fields: f}},
		"dot":            {{Name: "a.b", Fields: f}},
		"bracket open":   {{Name: "a[", Fields: f}},
		"bracket close":  {{Name: "a]", Fields: f}},
		"empty key":      {{Name: "a", Fields: []PromptField{{Key: " ", Value: "x"}}}},
		"duplicate key":  {{Name: "a", Fields: []PromptField{{Key: "k", Value: "1"}, {Key: "k", Value: "2"}}}},
		"too large":      {{Name: "a", Fields: []PromptField{{Key: "k", Value: strings.Repeat("x", 256<<10)}}}},
	}
	for name, entries := range cases {
		_, err := savePrompts(entries)
		if !errors.Is(err, errInvalidConfig) {
			t.Errorf("%s: err = %v", name, err)
			continue
		}
		if err.Error() == errInvalidConfig.Error() {
			t.Errorf("%s: message not specific", name)
		}
	}
	if _, err := os.Stat(promptsFilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid save wrote a file: %v", err)
	}
	// Zero entries is allowed.
	set, err := savePrompts(nil)
	if err != nil {
		t.Fatal(err)
	}
	if set.Entries == nil || len(set.Entries) != 0 || set.IsDefault {
		t.Fatalf("empty save %+v", set)
	}
	p, err := loadPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if got := getPrompt(p, "mcq"); got != "\n\n" {
		t.Fatalf("empty prompts ask = %q", got)
	}
}

func TestSavePromptsTrimsNamesAndKeys(t *testing.T) {
	useTestPrompts(t, inlineDefaultTOML)
	set, err := savePrompts([]PromptEntry{{Name: " mcq ", Fields: []PromptField{{Key: " prompt ", Value: " v "}}}})
	if err != nil {
		t.Fatal(err)
	}
	if set.Entries[0].Name != "mcq" || set.Entries[0].Fields[0].Key != "prompt" || set.Entries[0].Fields[0].Value != " v " {
		t.Fatalf("trim %+v", set)
	}
}

func TestOverrideTakesPrecedence(t *testing.T) {
	useTestPrompts(t, inlineDefaultTOML)
	p, err := loadPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if got := getPrompt(p, "mcq"); got != "U\n\nM" {
		t.Fatalf("default mcq = %q", got)
	}
	if _, err := savePrompts([]PromptEntry{
		{Name: "universal", Fields: []PromptField{{Key: "prompt", Value: "U2"}}},
		{Name: "mcq", Fields: []PromptField{{Key: "prompt", Value: "M2"}}},
	}); err != nil {
		t.Fatal(err)
	}
	p, err = loadPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if got := getPrompt(p, "mcq"); got != "U2\n\nM2" {
		t.Fatalf("override mcq = %q", got)
	}
	if got := getPrompt(p, "frq"); got != "U2\n\n" {
		t.Fatalf("override frq = %q", got)
	}
}

func TestResetPromptsFallsBackToDefault(t *testing.T) {
	path := useTestPrompts(t, inlineDefaultTOML)
	if _, err := savePrompts([]PromptEntry{{Name: "mcq", Fields: []PromptField{{Key: "prompt", Value: "M2"}}}}); err != nil {
		t.Fatal(err)
	}
	set, err := resetPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if !set.IsDefault || strings.Join(entryNames(set), ",") != "universal,frq,mcq" {
		t.Fatalf("reset %+v", entryNames(set))
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("override still present: %v", err)
	}
	p, _ := loadPrompts()
	if got := getPrompt(p, "mcq"); got != "U\n\nM" {
		t.Fatalf("after reset mcq = %q", got)
	}
	// Resetting with no override is fine.
	if _, err := resetPrompts(); err != nil {
		t.Fatal(err)
	}
}

func TestPromptSetFromMapRejectsNonStrings(t *testing.T) {
	for _, src := range []string{"top = \"x\"\n", "[a]\nn = 1\n", "[a]\n[a.b]\nc = \"d\"\n"} {
		m, err := parsePrompts([]byte(src))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := promptSetFromMap(m); err == nil {
			t.Errorf("accepted %q", src)
		}
	}
}

// Structure only: never print anything parsed from the bundled file.
func TestEmbeddedPromptsFitEditableModel(t *testing.T) {
	m, err := parsePrompts(promptsTOML)
	if err != nil {
		t.Fatal("embedded prompts.toml does not parse")
	}
	if len(m) == 0 {
		t.Fatal("embedded prompts.toml has no tables")
	}
	bad := 0
	for _, v := range m {
		table, ok := v.(map[string]any)
		if !ok {
			bad++
			continue
		}
		for _, fv := range table {
			if _, ok := fv.(string); !ok {
				bad++
			}
		}
	}
	if bad != 0 {
		t.Fatalf("%d top-level values or fields do not fit the entries/fields model", bad)
	}
	if _, err := promptSetFromMap(m); err != nil {
		t.Fatal("embedded prompts do not convert to a PromptSet")
	}
}
