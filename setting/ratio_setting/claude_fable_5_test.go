package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeFable5ExposesOfficialRelativePricing(t *testing.T) {
	InitRatioSettings()

	modelRatio, ok, matchedModel := GetModelRatio("claude-fable-5")
	require.True(t, ok)
	assert.Equal(t, "claude-fable-5", matchedModel)
	assert.Equal(t, 5.0, modelRatio)
	assert.Equal(t, 5.0, GetCompletionRatio("claude-fable-5"))

	cacheRatio, ok := GetCacheRatio("claude-fable-5")
	require.True(t, ok)
	assert.Equal(t, 0.1, cacheRatio)

	createCacheRatio, ok := GetCreateCacheRatio("claude-fable-5")
	require.True(t, ok)
	assert.Equal(t, 1.25, createCacheRatio)
}
