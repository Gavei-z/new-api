package common

import "strings"

// ClaudeRouteModelName returns the canonical model ID used to match channel
// abilities for a Claude request. Provider namespaces are routing metadata and
// are not part of the model ID stored by ordinary Anthropic channels.
func ClaudeRouteModelName(modelName string) string {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	if slash := strings.LastIndexByte(modelName, '/'); slash >= 0 {
		modelName = modelName[slash+1:]
	}
	return modelName
}

// ShouldForceClaudeToCFJWL reports whether the default-closed Kiro routing gate
// requires this request to use the dedicated cfjwl Claude channel. Token counts
// always stay on cfjwl because the Kiro endpoint can fall back to a local token
// estimate. Namespaced model IDs are supported by inspecting the final path
// component.
func ShouldForceClaudeToCFJWL(modelName string, requestPath string) bool {
	if requestPath == "/v1/messages/count_tokens" {
		return true
	}
	if !strings.HasPrefix(ClaudeRouteModelName(modelName), "claude-") {
		return false
	}
	return !ClaudeKiroRoutingEnabled
}
