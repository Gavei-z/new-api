package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeOpusModelsExposeOfficialRelativePricing(t *testing.T) {
	InitRatioSettings()

	for _, modelName := range []string{"claude-opus-4-5", "claude-opus-5"} {
		modelRatio, ok, matchedModel := GetModelRatio(modelName)
		require.True(t, ok)
		assert.Equal(t, modelName, matchedModel)
		assert.Equal(t, 2.5, modelRatio)
		assert.Equal(t, 5.0, GetCompletionRatio(modelName))

		cacheRatio, ok := GetCacheRatio(modelName)
		require.True(t, ok)
		assert.Equal(t, 0.1, cacheRatio)

		createCacheRatio, ok := GetCreateCacheRatio(modelName)
		require.True(t, ok)
		assert.Equal(t, 1.25, createCacheRatio)
	}
}
