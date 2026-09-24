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
