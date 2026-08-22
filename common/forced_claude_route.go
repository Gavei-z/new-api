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

// ShouldForceClaudeToCFJWL reports whether startup configuration requires this
// model to use the dedicated cfjwl Claude channel. Namespaced model IDs are
// supported by inspecting the final path component.
func ShouldForceClaudeToCFJWL(modelName string) bool {
	if !ClaudeForceCFJWL {
		return false
	}

	return strings.HasPrefix(ClaudeRouteModelName(modelName), "claude-")
}
