package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeSonnetModelsExposeOfficialRelativePricing(t *testing.T) {
	InitRatioSettings()

	tests := []struct {
		modelName  string
		modelRatio float64
	}{
		{modelName: "claude-sonnet-4-5", modelRatio: 1.5},
		{modelName: "claude-sonnet-4-5-20250929", modelRatio: 1.5},
		{modelName: "claude-sonnet-4-6", modelRatio: 1.5},
		{modelName: "claude-sonnet-5", modelRatio: 1.5},
	}
	for _, test := range tests {
		t.Run(test.modelName, func(t *testing.T) {
			modelRatio, ok, matchedModel := GetModelRatio(test.modelName)
			require.True(t, ok)
			assert.Equal(t, test.modelName, matchedModel)
			assert.Equal(t, test.modelRatio, modelRatio)
			assert.Equal(t, 5.0, GetCompletionRatio(test.modelName))

			cacheRatio, ok := GetCacheRatio(test.modelName)
			require.True(t, ok)
			assert.Equal(t, 0.1, cacheRatio)

			createCacheRatio, ok := GetCreateCacheRatio(test.modelName)
			require.True(t, ok)
			assert.Equal(t, 1.25, createCacheRatio)
		})
	}
}
