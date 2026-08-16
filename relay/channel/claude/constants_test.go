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
