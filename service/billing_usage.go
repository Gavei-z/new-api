package service

import (
	"math/big"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const (
	usageBillingPathLocal              = "local"
	usageBillingPathUpstream           = "upstream"
	usageBillingPathOpenAI             = "billing-usage-openai"
	usageBillingPathOpenAIEstimated    = "billing-usage-openai-estimated"
	usageBillingPathAnthropic          = "billing-usage-anthropic"
	usageBillingPathAnthropicEstimated = "billing-usage-anthropic-estimated"
	usageBillingPathAnthropicLocal     = "billing-usage-anthropic-local-estimate"
	usageBillingPathGemini             = "billing-usage-gemini"
	usageBillingPathGeminiEstimated    = "billing-usage-gemini-estimated"
)

type claudeInputBuckets struct {
	Uncached     int
	CacheRead    int
	CacheWrite5m int
	CacheWrite1h int
}

type claudeInputBillingAudit struct {
	Mode           dto.ClaudeInputBillingMode
	Status         string
	FallbackReason string
	LocalEstimate  int
	Upstream       claudeInputBuckets
	Candidate      claudeInputBuckets
	Billed         claudeInputBuckets
}

func resolveChannelTextBillingUsage(info *relaycommon.RelayInfo, upstream *dto.Usage) (*dto.Usage, *claudeInputBillingAudit) {
	if info == nil || info.ChannelMeta == nil || info.ChannelType != constant.ChannelTypeAnthropic {
		return upstream, nil
	}

	mode := info.ChannelOtherSettings.ClaudeInputBillingMode
	if mode != dto.ClaudeInputBillingModeLocalEstimate && mode != dto.ClaudeInputBillingModeLocalEstimateAudit {
		return upstream, nil
	}

	audit := &claudeInputBillingAudit{
		Mode:   mode,
		Status: "fallback_upstream",
	}
	if upstream == nil {
		audit.LocalEstimate = info.GetEstimatePromptTokens()
		if audit.LocalEstimate > 0 {
			audit.Status = "existing_local_fallback"
			audit.FallbackReason = "usage_unavailable"
			audit.Candidate = claudeInputBuckets{Uncached: audit.LocalEstimate}
			audit.Billed = audit.Candidate
			return upstream, audit
		}
		audit.FallbackReason = "usage_and_estimate_unavailable"
		return upstream, audit
	}
	if !strings.EqualFold(usageSemanticFromUsage(info, upstream), dto.BillingUsageSemanticAnthropic) {
		audit.FallbackReason = "usage_semantic_mismatch"
		return upstream, audit
	}

	audit.Upstream = canonicalClaudeInputBuckets(upstream)
	audit.Billed = audit.Upstream
	audit.LocalEstimate = info.GetEstimatePromptTokens()
	if audit.LocalEstimate <= 0 {
		audit.FallbackReason = "estimate_unavailable"
		return upstream, audit
	}

	audit.Candidate = normalizeClaudeInputBuckets(audit.LocalEstimate, audit.Upstream)
	if mode == dto.ClaudeInputBillingModeLocalEstimateAudit {
		audit.Status = "audit_only"
		return upstream, audit
	}

	audit.Status = "applied"
	audit.Billed = audit.Candidate
	return cloneUsageWithClaudeInputBuckets(upstream, audit.LocalEstimate, audit.Candidate), audit
}

func canonicalClaudeInputBuckets(usage *dto.Usage) claudeInputBuckets {
	if usage == nil {
		return claudeInputBuckets{}
	}

	buckets := claudeInputBuckets{
		Uncached:     nonNegativeTokenCount(usage.PromptTokens),
		CacheRead:    nonNegativeTokenCount(usage.PromptTokensDetails.CachedTokens),
		CacheWrite5m: nonNegativeTokenCount(usage.ClaudeCacheCreation5mTokens),
		CacheWrite1h: nonNegativeTokenCount(usage.ClaudeCacheCreation1hTokens),
	}
	aggregateCreation := nonNegativeTokenCount(usage.PromptTokensDetails.CacheCreationTokensTotal())
	splitCreation := new(big.Int).Add(
		big.NewInt(int64(buckets.CacheWrite5m)),
		big.NewInt(int64(buckets.CacheWrite1h)),
	)
	if splitCreation.Sign() == 0 {
		buckets.CacheWrite5m = aggregateCreation
		return buckets
	}
	if splitCreation.Cmp(big.NewInt(int64(aggregateCreation))) < 0 {
		difference := aggregateCreation - int(splitCreation.Int64())
		buckets.CacheWrite5m += difference
	}
	return buckets
}

func normalizeClaudeInputBuckets(localEstimate int, upstream claudeInputBuckets) claudeInputBuckets {
	if localEstimate <= 0 {
		return upstream
	}

	cacheRead, cacheWrite5m, cacheWrite1h := scaleClaudeCacheBuckets(
		localEstimate,
		nonNegativeTokenCount(upstream.CacheRead),
		nonNegativeTokenCount(upstream.CacheWrite5m),
		nonNegativeTokenCount(upstream.CacheWrite1h),
	)
	cacheTotal := cacheRead + cacheWrite5m + cacheWrite1h
	return claudeInputBuckets{
		Uncached:     localEstimate - cacheTotal,
		CacheRead:    cacheRead,
		CacheWrite5m: cacheWrite5m,
		CacheWrite1h: cacheWrite1h,
	}
}

func scaleClaudeCacheBuckets(localEstimate int, cacheRead int, cacheWrite5m int, cacheWrite1h int) (int, int, int) {
	values := []int{cacheRead, cacheWrite5m, cacheWrite1h}
	total := new(big.Int)
	for _, value := range values {
		total.Add(total, big.NewInt(int64(value)))
	}
	local := big.NewInt(int64(localEstimate))
	if total.Cmp(local) <= 0 {
		return cacheRead, cacheWrite5m, cacheWrite1h
	}

	type allocationRemainder struct {
		index     int
		remainder *big.Int
	}
	allocations := make([]int, len(values))
	remainders := make([]allocationRemainder, 0, len(values))
	allocated := 0
	for index, value := range values {
		numerator := new(big.Int).Mul(big.NewInt(int64(value)), local)
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(numerator, total, remainder)
		allocations[index] = int(quotient.Int64())
		allocated += allocations[index]
		remainders = append(remainders, allocationRemainder{index: index, remainder: remainder})
	}

	// Equal fractional remainders prefer the more expensive write tiers:
	// 1-hour write, then 5-minute write, then cache read.
	tiePriority := []int{2, 1, 0}
	sort.SliceStable(remainders, func(i, j int) bool {
		comparison := remainders[i].remainder.Cmp(remainders[j].remainder)
		if comparison != 0 {
			return comparison > 0
		}
		return tiePriority[remainders[i].index] < tiePriority[remainders[j].index]
	})
	for index := 0; index < localEstimate-allocated; index++ {
		allocations[remainders[index].index]++
	}
	return allocations[0], allocations[1], allocations[2]
}

func cloneUsageWithClaudeInputBuckets(upstream *dto.Usage, localEstimate int, buckets claudeInputBuckets) *dto.Usage {
	clone := *upstream
	if upstream.InputTokensDetails != nil {
		details := *upstream.InputTokensDetails
		clone.InputTokensDetails = &details
	}
	clone.BillingUsage = dto.CloneBillingUsage(upstream.BillingUsage)

	cacheCreation := buckets.CacheWrite5m + buckets.CacheWrite1h
	outputTokens := nonNegativeTokenCount(clone.CompletionTokens)
	if clone.OutputTokens > outputTokens {
		outputTokens = clone.OutputTokens
	}
	clone.PromptTokens = buckets.Uncached
	clone.CompletionTokens = outputTokens
	clone.InputTokens = localEstimate
	clone.OutputTokens = outputTokens
	clone.TotalTokens = saturatedPositiveTokenSum(localEstimate, outputTokens)
	clone.UsageSemantic = dto.BillingUsageSemanticAnthropic
	clone.PromptTokensDetails.CachedTokens = buckets.CacheRead
	clone.PromptTokensDetails.CachedCreationTokens = cacheCreation
	clone.PromptTokensDetails.CacheWriteTokens = 0
	clone.ClaudeCacheCreation5mTokens = buckets.CacheWrite5m
	clone.ClaudeCacheCreation1hTokens = buckets.CacheWrite1h
	if clone.InputTokensDetails != nil {
		clone.InputTokensDetails.CachedTokens = buckets.CacheRead
		clone.InputTokensDetails.CachedCreationTokens = cacheCreation
		clone.InputTokensDetails.CacheWriteTokens = 0
	}

	if clone.BillingUsage != nil && clone.BillingUsage.ClaudeUsage != nil {
		clone.BillingUsage.Semantic = dto.BillingUsageSemanticAnthropic
		claudeUsage := clone.BillingUsage.ClaudeUsage
		claudeUsage.InputTokens = buckets.Uncached
		claudeUsage.OutputTokens = outputTokens
		claudeUsage.CacheReadInputTokens = buckets.CacheRead
		claudeUsage.CacheCreationInputTokens = cacheCreation
		claudeUsage.ClaudeCacheCreation5mTokens = buckets.CacheWrite5m
		claudeUsage.ClaudeCacheCreation1hTokens = buckets.CacheWrite1h
		claudeUsage.CacheCreation = &dto.ClaudeCacheCreationUsage{
			Ephemeral5mInputTokens: buckets.CacheWrite5m,
			Ephemeral1hInputTokens: buckets.CacheWrite1h,
		}
	}
	return &clone
}

func nonNegativeTokenCount(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func saturatedPositiveTokenSum(left int, right int) int {
	maxInt := int(^uint(0) >> 1)
	if left > maxInt-right {
		return maxInt
	}
	return left + right
}

func effectiveBillingUsage(usage *dto.Usage) *dto.Usage {
	if billingUsage, ok := usageFromBillingUsage(usage); ok {
		return billingUsage
	}
	return usage
}

func usageBillingPathForLog(isLocalCountTokens bool, usage *dto.Usage) string {
	if isLocalCountTokens {
		return usageBillingPathLocal
	}
	if usage == nil || usage.BillingUsage == nil {
		return usageBillingPathUpstream
	}
	source := strings.TrimSpace(usage.BillingUsage.Source)
	semantic := strings.TrimSpace(usage.BillingUsage.Semantic)
	if strings.EqualFold(source, dto.BillingUsageSourceOAIChat) ||
		strings.EqualFold(source, dto.BillingUsageSourceOAIResponses) ||
		strings.EqualFold(semantic, dto.BillingUsageSemanticOpenAI) {
		if usage.BillingUsage.Estimated {
			return usageBillingPathOpenAIEstimated
		}
		return usageBillingPathOpenAI
	}
	if strings.EqualFold(source, dto.BillingUsageSourceClaudeMessages) ||
		strings.EqualFold(semantic, dto.BillingUsageSemanticAnthropic) {
		if usage.BillingUsage.Estimated {
			return usageBillingPathAnthropicEstimated
		}
		return usageBillingPathAnthropic
	}
	if strings.EqualFold(source, dto.BillingUsageSourceGeminiChat) ||
		strings.EqualFold(semantic, dto.BillingUsageSemanticGemini) {
		if usage.BillingUsage.Estimated {
			return usageBillingPathGeminiEstimated
		}
		return usageBillingPathGemini
	}
	return usageBillingPathUpstream
}

func appendUsageBillingPathForLog(other map[string]interface{}, isLocalCountTokens bool, usage *dto.Usage) {
	if other == nil {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = make(map[string]interface{})
		other["admin_info"] = adminInfo
	}
	adminInfo["usage_billing_path"] = usageBillingPathForLog(isLocalCountTokens, usage)
}

func appendClaudeInputBillingAuditForLog(other map[string]interface{}, audit *claudeInputBillingAudit) {
	if other == nil || audit == nil {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = make(map[string]interface{})
		other["admin_info"] = adminInfo
	}
	if audit.Status == "applied" || audit.Status == "existing_local_fallback" {
		adminInfo["usage_billing_path"] = usageBillingPathAnthropicLocal
	}

	upstreamTotal := inputBucketTotal(audit.Upstream)
	candidateTotal := inputBucketTotal(audit.Candidate)
	billedTotal := inputBucketTotal(audit.Billed)
	adminInfo["claude_input_billing"] = map[string]interface{}{
		"mode":                            string(audit.Mode),
		"status":                          audit.Status,
		"fallback_reason":                 audit.FallbackReason,
		"local_estimate_tokens":           audit.LocalEstimate,
		"upstream_uncached_tokens":        audit.Upstream.Uncached,
		"upstream_cache_read_tokens":      audit.Upstream.CacheRead,
		"upstream_cache_write_5m_tokens":  audit.Upstream.CacheWrite5m,
		"upstream_cache_write_1h_tokens":  audit.Upstream.CacheWrite1h,
		"upstream_total_input_tokens":     upstreamTotal,
		"candidate_uncached_tokens":       audit.Candidate.Uncached,
		"candidate_cache_read_tokens":     audit.Candidate.CacheRead,
		"candidate_cache_write_5m_tokens": audit.Candidate.CacheWrite5m,
		"candidate_cache_write_1h_tokens": audit.Candidate.CacheWrite1h,
		"candidate_total_input_tokens":    candidateTotal,
		"billed_uncached_tokens":          audit.Billed.Uncached,
		"billed_cache_read_tokens":        audit.Billed.CacheRead,
		"billed_cache_write_5m_tokens":    audit.Billed.CacheWrite5m,
		"billed_cache_write_1h_tokens":    audit.Billed.CacheWrite1h,
		"billed_total_input_tokens":       billedTotal,
		"delta_total_input_tokens":        billedTotal - upstreamTotal,
	}
}

func inputBucketTotal(buckets claudeInputBuckets) int {
	total := saturatedPositiveTokenSum(buckets.Uncached, buckets.CacheRead)
	total = saturatedPositiveTokenSum(total, buckets.CacheWrite5m)
	return saturatedPositiveTokenSum(total, buckets.CacheWrite1h)
}

func usageFromBillingUsage(usage *dto.Usage) (*dto.Usage, bool) {
	if usage == nil || usage.BillingUsage == nil {
		return nil, false
	}
	billingUsage := usage.BillingUsage
	source := strings.TrimSpace(billingUsage.Source)
	semantic := strings.TrimSpace(billingUsage.Semantic)

	if billingUsage.OpenAIUsage != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceOAIChat) ||
			strings.EqualFold(source, dto.BillingUsageSourceOAIResponses) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticOpenAI)) {
		return usageFromOpenAIBillingUsage(billingUsage), true
	}

	if billingUsage.ClaudeUsage != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceClaudeMessages) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticAnthropic)) {
		return usageFromClaudeBillingUsage(billingUsage), true
	}

	if billingUsage.GeminiUsageMetadata != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceGeminiChat) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticGemini)) {
		return usageFromGeminiBillingUsage(billingUsage), true
	}

	return nil, false
}

func usageFromOpenAIBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	usage := *billingUsage.OpenAIUsage
	if usage.PromptTokens == 0 && usage.InputTokens > 0 {
		usage.PromptTokens = usage.InputTokens
	}
	if usage.CompletionTokens == 0 && usage.OutputTokens > 0 {
		usage.CompletionTokens = usage.OutputTokens
	}
	if usage.InputTokens == 0 && usage.PromptTokens > 0 {
		usage.InputTokens = usage.PromptTokens
	}
	if usage.OutputTokens == 0 && usage.CompletionTokens > 0 {
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	usage.UsageSemantic = dto.BillingUsageSemanticOpenAI
	usage.UsageSource = billingUsage.Source
	usage.BillingUsage = dto.CloneBillingUsage(billingUsage)
	return &usage
}

func usageFromClaudeBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	claudeUsage := billingUsage.ClaudeUsage
	cacheCreation5m := claudeUsage.GetCacheCreation5mTokens()
	if cacheCreation5m == 0 {
		cacheCreation5m = claudeUsage.ClaudeCacheCreation5mTokens
	}
	cacheCreation1h := claudeUsage.GetCacheCreation1hTokens()
	if cacheCreation1h == 0 {
		cacheCreation1h = claudeUsage.ClaudeCacheCreation1hTokens
	}

	usage := &dto.Usage{
		PromptTokens:                claudeUsage.InputTokens,
		CompletionTokens:            claudeUsage.OutputTokens,
		TotalTokens:                 claudeUsage.InputTokens + claudeUsage.OutputTokens,
		InputTokens:                 claudeUsage.InputTokens + claudeUsage.CacheReadInputTokens + claudeUsage.CacheCreationInputTokens,
		OutputTokens:                claudeUsage.OutputTokens,
		UsageSemantic:               dto.BillingUsageSemanticAnthropic,
		UsageSource:                 dto.BillingUsageSourceClaudeMessages,
		BillingUsage:                dto.CloneBillingUsage(billingUsage),
		ClaudeCacheCreation5mTokens: cacheCreation5m,
		ClaudeCacheCreation1hTokens: cacheCreation1h,
	}
	usage.PromptTokensDetails.CachedTokens = claudeUsage.CacheReadInputTokens
	usage.PromptTokensDetails.CachedCreationTokens = claudeUsage.CacheCreationInputTokens
	return usage
}

func usageFromGeminiBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	metadata := *billingUsage.GeminiUsageMetadata
	promptTokens := metadata.PromptTokenCount + metadata.ToolUsePromptTokenCount
	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: metadata.CandidatesTokenCount + metadata.ThoughtsTokenCount,
		TotalTokens:      metadata.TotalTokenCount,
		UsageSemantic:    dto.BillingUsageSemanticGemini,
		UsageSource:      dto.BillingUsageSourceGeminiChat,
		BillingUsage:     dto.CloneBillingUsage(billingUsage),
	}
	usage.CompletionTokenDetails.ReasoningTokens = metadata.ThoughtsTokenCount
	usage.PromptTokensDetails.CachedTokens = metadata.CachedContentTokenCount

	for _, detail := range metadata.PromptTokensDetails {
		addGeminiInputTokenDetail(&usage.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.ToolUsePromptTokensDetails {
		addGeminiInputTokenDetail(&usage.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.CandidatesTokensDetails {
		switch detail.Modality {
		case "IMAGE":
			usage.CompletionTokenDetails.ImageTokens += detail.TokenCount
		case "AUDIO":
			usage.CompletionTokenDetails.AudioTokens += detail.TokenCount
		case "TEXT":
			usage.CompletionTokenDetails.TextTokens += detail.TokenCount
		}
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	} else if usage.CompletionTokens <= 0 {
		usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
	}
	if usage.PromptTokens > 0 && usage.PromptTokensDetails.TextTokens == 0 && usage.PromptTokensDetails.AudioTokens == 0 {
		usage.PromptTokensDetails.TextTokens = usage.PromptTokens
	}
	return usage
}

func addGeminiInputTokenDetail(details *dto.InputTokenDetails, detail dto.GeminiPromptTokensDetails) {
	switch detail.Modality {
	case "AUDIO":
		details.AudioTokens += detail.TokenCount
	case "IMAGE":
		details.ImageTokens += detail.TokenCount
	case "TEXT":
		details.TextTokens += detail.TokenCount
	}
}
