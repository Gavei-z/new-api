package common

import "testing"

func TestClaudeRouteModelName(t *testing.T) {
	tests := map[string]string{
		"claude-sonnet-5":                    "claude-sonnet-5",
		"anthropic/claude-opus-5":            "claude-opus-5",
		" Anthropic/CLAUDE-OPUS-4-6 ":        "claude-opus-4-6",
		"provider/nested/claude-sonnet-5":    "claude-sonnet-5",
		"provider/nested/not-a-claude-model": "not-a-claude-model",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := ClaudeRouteModelName(input); got != want {
				t.Fatalf("ClaudeRouteModelName(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestShouldForceClaudeToCFJWL(t *testing.T) {
	previous := ClaudeForceCFJWL
	t.Cleanup(func() { ClaudeForceCFJWL = previous })

	ClaudeForceCFJWL = true
	tests := []struct {
		model string
		want  bool
	}{
		{model: "claude-sonnet-5", want: true},
		{model: "anthropic/claude-opus-5", want: true},
		{model: " CLAUDE-OPUS-4-6-thinking ", want: true},
		{model: "gpt-5.6-sol", want: false},
		{model: "not-claude-sonnet", want: false},
		{model: "", want: false},
	}
	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			if got := ShouldForceClaudeToCFJWL(test.model); got != test.want {
				t.Fatalf("ShouldForceClaudeToCFJWL(%q) = %t, want %t", test.model, got, test.want)
			}
		})
	}

	ClaudeForceCFJWL = false
	if ShouldForceClaudeToCFJWL("claude-sonnet-5") {
		t.Fatal("disabled route lock must not force Claude")
	}
}
