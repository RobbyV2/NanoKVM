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
	if _, err := embeddedPrompts(); err != nil {
		t.Fatalf("embedded prompts.toml: %v", err)
	}
}
