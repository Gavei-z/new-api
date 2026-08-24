package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
			require.Equal(t, want, ClaudeRouteModelName(input))
		})
	}
}

func TestShouldForceClaudeToCFJWL(t *testing.T) {
	previous := ClaudeKiroRoutingEnabled
	t.Cleanup(func() { ClaudeKiroRoutingEnabled = previous })

	ClaudeKiroRoutingEnabled = false
	tests := []struct {
		model       string
		requestPath string
		want        bool
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
			require.Equal(t, test.want, ShouldForceClaudeToCFJWL(test.model, test.requestPath))
		})
	}

	ClaudeKiroRoutingEnabled = true
	require.False(t, ShouldForceClaudeToCFJWL("claude-sonnet-5", "/v1/messages"))
	require.True(t, ShouldForceClaudeToCFJWL("claude-sonnet-5", "/v1/messages/count_tokens"))
	require.True(t, ShouldForceClaudeToCFJWL("provider-model-alias", "/v1/messages/count_tokens"))
}
