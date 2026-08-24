package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdaptorAdvertisesSupportedClaudeOpusModels(t *testing.T) {
	models := (&Adaptor{}).GetModelList()

	for _, modelName := range []string{
		"claude-opus-4-5",
		"claude-opus-4-5-20251101",
		"claude-opus-4-6",
		"claude-opus-4-7",
		"claude-opus-4-8",
		"claude-opus-5",
	} {
		assert.Contains(t, models, modelName)
	}
}

func TestAdaptorAdvertisesSupportedClaudeSonnetModels(t *testing.T) {
	models := (&Adaptor{}).GetModelList()

	for _, modelName := range []string{
		"claude-sonnet-4-5",
		"claude-sonnet-4-5-20250929",
		"claude-sonnet-4-6",
		"claude-sonnet-5",
	} {
		assert.Contains(t, models, modelName)
	}
}

func TestAdaptorAdvertisesSupportedClaudeHaikuModels(t *testing.T) {
	models := (&Adaptor{}).GetModelList()

	for _, modelName := range []string{
		"claude-haiku-4-5",
		"claude-haiku-4-5-20251001",
	} {
		assert.Contains(t, models, modelName)
	}
}

func TestAdaptorAdvertisesSupportedClaudeFableModels(t *testing.T) {
	assert.Contains(t, (&Adaptor{}).GetModelList(), "claude-fable-5")
}
