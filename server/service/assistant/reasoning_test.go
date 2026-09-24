package assistant

import "testing"

func TestAdjustReasoning(t *testing.T) {
	g := func(on bool, b int) Config {
		c := defaultConfig()
		c.Provider = "gemini"
		c.GeminiThinking = on
		c.ThinkingBudget = b
		return c
	}
	o := func(on bool, e string) Config {
		c := defaultConfig()
		c.ORReasoning = on
		c.ORReasoningEffort = e
		return c
	}
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
