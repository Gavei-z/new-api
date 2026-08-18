package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func claudeBillingRelayInfo(mode dto.ClaudeInputBillingMode, estimate int) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeAnthropic,
			ChannelOtherSettings: dto.ChannelOtherSettings{
				ClaudeInputBillingMode: mode,
			},
		},
	}
	info.SetEstimatePromptTokens(estimate)
	return info
}

func TestResolveClaudeInputBillingUsesSameLocalEstimateAcrossSources(t *testing.T) {
	tests := []struct {
		name          string
		upstreamInput int
	}{
		{name: "upstream injects hidden input", upstreamInput: 6480},
		{name: "upstream reports client input", upstreamInput: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origin := &dto.Usage{
				PromptTokens:     tt.upstreamInput,
				CompletionTokens: 2,
				UsageSemantic:    dto.BillingUsageSemanticAnthropic,
				BillingUsage: dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{
					InputTokens:  tt.upstreamInput,
					OutputTokens: 2,
				}),
			}
			result, audit := resolveChannelTextBillingUsage(
				claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 5),
				origin,
			)

			require.NotNil(t, audit)
			assert.Equal(t, "applied", audit.Status)
			assert.Equal(t, 5, result.PromptTokens)
			assert.Equal(t, 5, result.InputTokens)
			assert.Equal(t, 7, result.TotalTokens)
			require.NotNil(t, result.BillingUsage)
			require.NotNil(t, result.BillingUsage.ClaudeUsage)
			assert.Equal(t, 5, result.BillingUsage.ClaudeUsage.InputTokens)

			assert.Equal(t, tt.upstreamInput, origin.PromptTokens)
			assert.Equal(t, tt.upstreamInput, origin.BillingUsage.ClaudeUsage.InputTokens)
			assert.NotSame(t, origin, result)
			assert.NotSame(t, origin.BillingUsage, result.BillingUsage)
		})
	}
}

func TestResolveClaudeInputBillingSynchronizesNestedCacheUsageWithoutMutatingOrigin(t *testing.T) {
	origin := &dto.Usage{
		PromptTokens:     7,
		CompletionTokens: 1,
		UsageSemantic:    dto.BillingUsageSemanticAnthropic,
		BillingUsage: dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{
			InputTokens:              7,
			CacheReadInputTokens:     38,
			CacheCreationInputTokens: 60,
			OutputTokens:             1,
			CacheCreation: &dto.ClaudeCacheCreationUsage{
				Ephemeral5mInputTokens: 20,
				Ephemeral1hInputTokens: 40,
			},
		}),
	}
	upstream := effectiveBillingUsage(origin)
	result, audit := resolveChannelTextBillingUsage(
		claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 5),
		upstream,
	)

	require.NotNil(t, audit)
	assert.Equal(t, claudeInputBuckets{CacheRead: 2, CacheWrite5m: 1, CacheWrite1h: 2}, audit.Billed)
	assert.Equal(t, 0, result.PromptTokens)
	assert.Equal(t, 2, result.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 3, result.PromptTokensDetails.CachedCreationTokens)
	assert.Equal(t, 1, result.ClaudeCacheCreation5mTokens)
	assert.Equal(t, 2, result.ClaudeCacheCreation1hTokens)
	require.NotNil(t, result.BillingUsage)
	require.NotNil(t, result.BillingUsage.ClaudeUsage)
	assert.Equal(t, 0, result.BillingUsage.ClaudeUsage.InputTokens)
	assert.Equal(t, 2, result.BillingUsage.ClaudeUsage.CacheReadInputTokens)
	assert.Equal(t, 3, result.BillingUsage.ClaudeUsage.CacheCreationInputTokens)
	assert.Equal(t, 1, result.BillingUsage.ClaudeUsage.GetCacheCreation5mTokens())
	assert.Equal(t, 2, result.BillingUsage.ClaudeUsage.GetCacheCreation1hTokens())

	tiered := BuildTieredTokenParams(result, true, map[string]bool{
		"cr": true, "cc": true, "cc1h": true,
	})
	assert.Equal(t, float64(0), tiered.P)
	assert.Equal(t, float64(2), tiered.CR)
	assert.Equal(t, float64(1), tiered.CC)
	assert.Equal(t, float64(2), tiered.CC1h)
	assert.Equal(t, float64(5), tiered.Len)

	assert.Equal(t, 7, origin.BillingUsage.ClaudeUsage.InputTokens)
	assert.Equal(t, 38, origin.BillingUsage.ClaudeUsage.CacheReadInputTokens)
	assert.Equal(t, 60, origin.BillingUsage.ClaudeUsage.CacheCreationInputTokens)
	assert.Equal(t, 20, origin.BillingUsage.ClaudeUsage.GetCacheCreation5mTokens())
	assert.Equal(t, 40, origin.BillingUsage.ClaudeUsage.GetCacheCreation1hTokens())
}

func TestBuildTieredTokenParamsUsesExplicitClaudeSemanticForCacheTiers(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:                2,
		UsageSemantic:               "",
		ClaudeCacheCreation5mTokens: 3,
		ClaudeCacheCreation1hTokens: 4,
		PromptTokensDetails:         dto.InputTokenDetails{CachedCreationTokens: 7},
	}

	params := BuildTieredTokenParams(usage, true, map[string]bool{"cc": true, "cc1h": true})

	assert.Equal(t, float64(3), params.CC)
	assert.Equal(t, float64(4), params.CC1h)
	assert.Equal(t, float64(9), params.Len)
}

func TestNormalizeClaudeInputBucketsPreservesAndScalesCacheTiers(t *testing.T) {
	tests := []struct {
		name     string
		local    int
		upstream claudeInputBuckets
		want     claudeInputBuckets
	}{
		{
			name:  "cache below local estimate leaves uncached remainder",
			local: 10,
			upstream: claudeInputBuckets{
				Uncached: 999, CacheRead: 2, CacheWrite5m: 1,
			},
			want: claudeInputBuckets{Uncached: 7, CacheRead: 2, CacheWrite5m: 1},
		},
		{
			name:  "cache equal to local estimate",
			local: 10,
			upstream: claudeInputBuckets{
				Uncached: 999, CacheRead: 5, CacheWrite5m: 3, CacheWrite1h: 2,
			},
			want: claudeInputBuckets{CacheRead: 5, CacheWrite5m: 3, CacheWrite1h: 2},
		},
		{
			name:  "largest remainder scales oversized cache exactly",
			local: 10,
			upstream: claudeInputBuckets{
				Uncached: 999, CacheRead: 8, CacheWrite5m: 4, CacheWrite1h: 2,
			},
			want: claudeInputBuckets{CacheRead: 6, CacheWrite5m: 3, CacheWrite1h: 1},
		},
		{
			name:  "equal remainders favor expensive write tiers",
			local: 2,
			upstream: claudeInputBuckets{
				CacheRead: 1, CacheWrite5m: 1, CacheWrite1h: 1,
			},
			want: claudeInputBuckets{CacheWrite5m: 1, CacheWrite1h: 1},
		},
		{
			name:  "negative cache counts cannot lower billing",
			local: 5,
			upstream: claudeInputBuckets{
				Uncached: 999, CacheRead: -2, CacheWrite5m: -3, CacheWrite1h: -4,
			},
			want: claudeInputBuckets{Uncached: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeClaudeInputBuckets(tt.local, tt.upstream))
		})
	}
}

func TestScaleClaudeCacheBucketsDoesNotOverflowAtIntLimit(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	cacheRead, cacheWrite5m, cacheWrite1h := scaleClaudeCacheBuckets(maxInt, maxInt, maxInt, maxInt)

	require.GreaterOrEqual(t, cacheRead, 0)
	require.GreaterOrEqual(t, cacheWrite5m, 0)
	require.GreaterOrEqual(t, cacheWrite1h, 0)
	require.Equal(t, maxInt, cacheRead+cacheWrite5m+cacheWrite1h)
}

func TestCanonicalClaudeInputBucketsReconcilesAggregateCacheCreation(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens: -1,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         -2,
			CachedCreationTokens: 7,
		},
		ClaudeCacheCreation5mTokens: 2,
		ClaudeCacheCreation1hTokens: 1,
	}

	assert.Equal(t, claudeInputBuckets{
		CacheWrite5m: 6,
		CacheWrite1h: 1,
	}, canonicalClaudeInputBuckets(usage))
}

func TestResolveClaudeInputBillingAuditAndFallbackDoNotChangeCharge(t *testing.T) {
	upstream := &dto.Usage{
		PromptTokens:  6480,
		UsageSemantic: dto.BillingUsageSemanticAnthropic,
	}

	auditResult, audit := resolveChannelTextBillingUsage(
		claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimateAudit, 5),
		upstream,
	)
	assert.Same(t, upstream, auditResult)
	require.NotNil(t, audit)
	assert.Equal(t, "audit_only", audit.Status)
	assert.Equal(t, 5, audit.Candidate.Uncached)
	assert.Equal(t, 6480, audit.Billed.Uncached)

	fallbackResult, fallbackAudit := resolveChannelTextBillingUsage(
		claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 0),
		upstream,
	)
	assert.Same(t, upstream, fallbackResult)
	require.NotNil(t, fallbackAudit)
	assert.Equal(t, "fallback_upstream", fallbackAudit.Status)
	assert.Equal(t, "estimate_unavailable", fallbackAudit.FallbackReason)
}

func TestResolveClaudeInputBillingAuditsExistingNilUsageFallbackAccurately(t *testing.T) {
	result, audit := resolveChannelTextBillingUsage(
		claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 5),
		nil,
	)

	assert.Nil(t, result)
	require.NotNil(t, audit)
	assert.Equal(t, "existing_local_fallback", audit.Status)
	assert.Equal(t, "usage_unavailable", audit.FallbackReason)
	assert.Equal(t, claudeInputBuckets{Uncached: 5}, audit.Candidate)
	assert.Equal(t, claudeInputBuckets{Uncached: 5}, audit.Billed)
}

func TestResolveClaudeInputBillingClampsNegativeOutputOnlyInBillingClone(t *testing.T) {
	origin := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: -9,
		OutputTokens:     -9,
		UsageSemantic:    dto.BillingUsageSemanticAnthropic,
		BillingUsage: dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{
			InputTokens:  100,
			OutputTokens: -9,
		}),
	}
	result, audit := resolveChannelTextBillingUsage(
		claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 5),
		origin,
	)

	require.NotNil(t, audit)
	assert.Equal(t, 0, result.CompletionTokens)
	assert.Equal(t, 0, result.OutputTokens)
	assert.Equal(t, 5, result.TotalTokens)
	require.NotNil(t, result.BillingUsage)
	require.NotNil(t, result.BillingUsage.ClaudeUsage)
	assert.Equal(t, 0, result.BillingUsage.ClaudeUsage.OutputTokens)
	assert.Equal(t, -9, origin.CompletionTokens)
	assert.Equal(t, -9, origin.OutputTokens)
	assert.Equal(t, -9, origin.BillingUsage.ClaudeUsage.OutputTokens)
}

func TestResolveClaudeInputBillingIsScopedToAnthropicChannels(t *testing.T) {
	upstream := &dto.Usage{PromptTokens: 100, UsageSemantic: dto.BillingUsageSemanticAnthropic}
	info := claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 5)
	info.ChannelType = constant.ChannelTypeOpenAI

	result, audit := resolveChannelTextBillingUsage(info, upstream)
	assert.Same(t, upstream, result)
	assert.Nil(t, audit)
}

func TestClaudeUsageSemanticComparisonIsCaseInsensitiveAndCanonical(t *testing.T) {
	info := claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 5)
	usage := &dto.Usage{PromptTokens: 100, UsageSemantic: " Anthropic "}

	result, audit := resolveChannelTextBillingUsage(info, usage)
	require.NotNil(t, audit)
	assert.Equal(t, "applied", audit.Status)
	assert.Equal(t, 5, result.PromptTokens)
	assert.Equal(t, dto.BillingUsageSemanticAnthropic, result.UsageSemantic)
	assert.Equal(t, dto.BillingUsageSemanticAnthropic, usageSemanticFromUsage(info, result))
}

func TestAppendClaudeInputBillingAuditKeepsDetailsAdminOnly(t *testing.T) {
	other := map[string]interface{}{}
	audit := &claudeInputBillingAudit{
		Mode:          dto.ClaudeInputBillingModeLocalEstimate,
		Status:        "applied",
		LocalEstimate: 5,
		Upstream:      claudeInputBuckets{Uncached: 6480},
		Candidate:     claudeInputBuckets{Uncached: 5},
		Billed:        claudeInputBuckets{Uncached: 5},
	}

	appendClaudeInputBillingAuditForLog(other, audit)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, usageBillingPathAnthropicLocal, adminInfo["usage_billing_path"])
	details, ok := adminInfo["claude_input_billing"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 5, details["billed_total_input_tokens"])
	assert.Equal(t, -6475, details["delta_total_input_tokens"])
	_, leakedAtTopLevel := other["claude_input_billing"]
	assert.False(t, leakedAtTopLevel)
}

func TestClaudeLocalEstimateFeedsTheActualQuotaSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := claudeBillingRelayInfo(dto.ClaudeInputBillingModeLocalEstimate, 5)
	info.OriginModelName = "claude-opus-5"
	info.StartTime = time.Now()
	info.PriceData = types.PriceData{
		ModelRatio:           0.5,
		CompletionRatio:      5,
		CacheRatio:           0.1,
		CacheCreationRatio:   1.25,
		CacheCreation5mRatio: 1.25,
		CacheCreation1hRatio: 2,
		GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
	}
	origin := &dto.Usage{
		PromptTokens:     6480,
		CompletionTokens: 1,
		BillingUsage: dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{
			InputTokens:  6480,
			OutputTokens: 1,
		}),
	}

	upstream := effectiveBillingUsage(origin)
	billing, audit := resolveChannelTextBillingUsage(info, upstream)
	summary := calculateTextQuotaSummary(ctx, info, billing)

	require.NotNil(t, audit)
	assert.Equal(t, "applied", audit.Status)
	assert.Equal(t, 5, summary.PromptTokens)
	assert.Equal(t, 1, summary.CompletionTokens)
	assert.Equal(t, 5, summary.Quota)
	assert.Equal(t, 6480, origin.BillingUsage.ClaudeUsage.InputTokens)
}
