package service

import (
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/shopspring/decimal"
)

const (
	kiroCalibrationStatusApplied             = "applied"
	kiroCalibrationStatusFallbackMissing     = "fallback_missing"
	kiroCalibrationStatusFallbackInvalid     = "fallback_invalid"
	kiroCalibrationStatusFallbackUnsupported = "fallback_unsupported"
	kiroCalibrationStatusFallbackOverflow    = "fallback_overflow"
	kiroCalibrationMethodResidualInput       = "residual_to_uncached_input"
	kiroCalibrationMethodProportionalScale   = "proportional_scale_known_cost_exceeds_target"
	kiroSettlementStatusPending              = "pending"
	kiroSettlementStatusReserved             = "reserved"
	kiroSettlementStatusSettled              = "settled"
	kiroSettlementStatusPartialReserveFailed = "partial_reserve_failed"
	kiroSettlementStatusSettlementFailed     = "settlement_failed"
)

type kiroLinearPrices struct {
	input        decimal.Decimal
	output       decimal.Decimal
	cacheRead    decimal.Decimal
	cacheWrite   decimal.Decimal
	cacheWrite5m decimal.Decimal
	cacheWrite1h decimal.Decimal
	group        decimal.Decimal
}

func finiteNonNegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func newKiroFallback(raw dto.KiroUsageBuckets, status string, reason string) *dto.KiroCreditCalibration {
	return &dto.KiroCreditCalibration{
		Status:         status,
		FallbackReason: reason,
		RawUsage:       raw,
	}
}

func normalizedKiroUsageBuckets(usage *dto.Usage) (dto.KiroUsageBuckets, string) {
	if usage == nil {
		return dto.KiroUsageBuckets{}, ""
	}
	values := []int{
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.PromptTokensDetails.CachedTokens,
		usage.PromptTokensDetails.CachedCreationTokens,
		usage.PromptTokensDetails.CacheWriteTokens,
		usage.ClaudeCacheCreation5mTokens,
		usage.ClaudeCacheCreation1hTokens,
	}
	for _, value := range values {
		if value < 0 {
			return dto.KiroUsageBuckets{}, "negative_usage_bucket"
		}
	}

	cacheCreationTotal := usage.PromptTokensDetails.CacheCreationTokensTotal()
	cache5m := usage.ClaudeCacheCreation5mTokens
	cache1h := usage.ClaudeCacheCreation1hTokens
	if cache5m > math.MaxInt-cache1h {
		return dto.KiroUsageBuckets{}, "cache_usage_overflow"
	}
	splitTotal := cache5m + cache1h
	cacheGeneric := 0
	if cacheCreationTotal > splitTotal {
		cacheGeneric = cacheCreationTotal - splitTotal
	}

	return dto.KiroUsageBuckets{
		InputTokens:           usage.PromptTokens,
		OutputTokens:          usage.CompletionTokens,
		CacheReadInputTokens:  usage.PromptTokensDetails.CachedTokens,
		CacheCreationTokens:   cacheGeneric,
		CacheCreation5mTokens: cache5m,
		CacheCreation1hTokens: cache1h,
	}, ""
}

func kiroLinearPricesForRelay(relayInfo *relaycommon.RelayInfo) (kiroLinearPrices, string) {
	if relayInfo == nil || relayInfo.ChannelMeta == nil {
		return kiroLinearPrices{}, "missing_relay_info"
	}
	settings := relayInfo.ChannelOtherSettings
	if relayInfo.ChannelType != constant.ChannelTypeAnthropic {
		return kiroLinearPrices{}, "channel_type_not_anthropic"
	}
	if settings.KiroCreditBillingMode != dto.KiroCreditBillingModeActualCalibrated {
		return kiroLinearPrices{}, "unsupported_mode"
	}
	if !settings.HasValidKiroCreditPrice() {
		return kiroLinearPrices{}, "invalid_price_per_credit"
	}
	if relayInfo.PriceData.UsePrice {
		return kiroLinearPrices{}, "use_price_not_supported"
	}
	if relayInfo.TieredBillingSnapshot != nil {
		return kiroLinearPrices{}, "tiered_billing_not_supported"
	}
	if len(relayInfo.PriceData.OtherRatios()) > 0 {
		return kiroLinearPrices{}, "other_ratios_not_supported"
	}
	if !finitePositive(common.QuotaPerUnit) {
		return kiroLinearPrices{}, "invalid_quota_per_unit"
	}
	if !finitePositive(relayInfo.PriceData.ModelRatio) ||
		!finiteNonNegative(relayInfo.PriceData.CompletionRatio) ||
		!finiteNonNegative(relayInfo.PriceData.CacheRatio) ||
		!finiteNonNegative(relayInfo.PriceData.CacheCreationRatio) ||
		!finiteNonNegative(relayInfo.PriceData.CacheCreation5mRatio) ||
		!finiteNonNegative(relayInfo.PriceData.CacheCreation1hRatio) ||
		!finiteNonNegative(relayInfo.PriceData.GroupRatioInfo.GroupRatio) {
		return kiroLinearPrices{}, "invalid_linear_price_ratio"
	}

	modelRatio := decimal.NewFromFloat(relayInfo.PriceData.ModelRatio)
	return kiroLinearPrices{
		input:        modelRatio,
		output:       modelRatio.Mul(decimal.NewFromFloat(relayInfo.PriceData.CompletionRatio)),
		cacheRead:    modelRatio.Mul(decimal.NewFromFloat(relayInfo.PriceData.CacheRatio)),
		cacheWrite:   modelRatio.Mul(decimal.NewFromFloat(relayInfo.PriceData.CacheCreationRatio)),
		cacheWrite5m: modelRatio.Mul(decimal.NewFromFloat(relayInfo.PriceData.CacheCreation5mRatio)),
		cacheWrite1h: modelRatio.Mul(decimal.NewFromFloat(relayInfo.PriceData.CacheCreation1hRatio)),
		group:        decimal.NewFromFloat(relayInfo.PriceData.GroupRatioInfo.GroupRatio),
	}, ""
}

func kiroQuotaBeforeGroup(buckets dto.KiroUsageBuckets, prices kiroLinearPrices) decimal.Decimal {
	return decimal.NewFromInt(int64(buckets.InputTokens)).Mul(prices.input).
		Add(decimal.NewFromInt(int64(buckets.OutputTokens)).Mul(prices.output)).
		Add(decimal.NewFromInt(int64(buckets.CacheReadInputTokens)).Mul(prices.cacheRead)).
		Add(decimal.NewFromInt(int64(buckets.CacheCreationTokens)).Mul(prices.cacheWrite)).
		Add(decimal.NewFromInt(int64(buckets.CacheCreation5mTokens)).Mul(prices.cacheWrite5m)).
		Add(decimal.NewFromInt(int64(buckets.CacheCreation1hTokens)).Mul(prices.cacheWrite1h))
}

func kiroHasTokens(buckets dto.KiroUsageBuckets) bool {
	return buckets.InputTokens > 0 || buckets.OutputTokens > 0 || buckets.CacheReadInputTokens > 0 ||
		buckets.CacheCreationTokens > 0 || buckets.CacheCreation5mTokens > 0 || buckets.CacheCreation1hTokens > 0
}

func replayKiroQuota(buckets dto.KiroUsageBuckets, prices kiroLinearPrices) (int, bool) {
	quota, clamp := common.QuotaFromDecimalChecked(kiroQuotaBeforeGroup(buckets, prices).Mul(prices.group))
	if clamp != nil {
		return 0, false
	}
	if quota == 0 && kiroHasTokens(buckets) && !prices.input.Mul(prices.group).IsZero() {
		quota = 1
	}
	return quota, true
}

func nearbyTokenCandidates(value decimal.Decimal) ([]int, bool) {
	if value.IsNegative() || value.GreaterThan(decimal.NewFromInt(int64(math.MaxInt))) {
		return nil, false
	}
	floor := value.Floor().IntPart()
	ceil := value.Ceil().IntPart()
	result := []int{int(floor)}
	if ceil != floor {
		result = append(result, int(ceil))
	}
	return result, true
}

func chooseNearestKiroCandidate(targetQuota int, prices kiroLinearPrices, ideals [6]decimal.Decimal) (dto.KiroUsageBuckets, int, bool) {
	candidates := [6][]int{}
	for index, ideal := range ideals {
		values, ok := nearbyTokenCandidates(ideal)
		if !ok {
			return dto.KiroUsageBuckets{}, 0, false
		}
		candidates[index] = values
	}

	bestDiff := math.MaxInt
	bestReplay := 0
	best := dto.KiroUsageBuckets{}
	found := false
	for _, input := range candidates[0] {
		for _, output := range candidates[1] {
			for _, cacheRead := range candidates[2] {
				for _, cacheGeneric := range candidates[3] {
					for _, cache5m := range candidates[4] {
						for _, cache1h := range candidates[5] {
							bucket := dto.KiroUsageBuckets{
								InputTokens:           input,
								OutputTokens:          output,
								CacheReadInputTokens:  cacheRead,
								CacheCreationTokens:   cacheGeneric,
								CacheCreation5mTokens: cache5m,
								CacheCreation1hTokens: cache1h,
							}
							replayed, ok := replayKiroQuota(bucket, prices)
							if !ok {
								continue
							}
							diff := replayed - targetQuota
							if diff < 0 {
								diff = -diff
							}
							if !found || diff < bestDiff {
								found = true
								bestDiff = diff
								bestReplay = replayed
								best = bucket
							}
						}
					}
				}
			}
		}
	}
	return best, bestReplay, found
}

func calibratedKiroCandidate(raw dto.KiroUsageBuckets, targetBeforeGroup decimal.Decimal, targetQuota int, prices kiroLinearPrices) (dto.KiroUsageBuckets, int, string, bool) {
	known := dto.KiroUsageBuckets{
		OutputTokens:          raw.OutputTokens,
		CacheReadInputTokens:  raw.CacheReadInputTokens,
		CacheCreationTokens:   raw.CacheCreationTokens,
		CacheCreation5mTokens: raw.CacheCreation5mTokens,
		CacheCreation1hTokens: raw.CacheCreation1hTokens,
	}
	knownCost := kiroQuotaBeforeGroup(known, prices)
	if knownCost.LessThanOrEqual(targetBeforeGroup) {
		idealInput := targetBeforeGroup.Sub(knownCost).Div(prices.input)
		ideals := [6]decimal.Decimal{
			idealInput,
			decimal.NewFromInt(int64(raw.OutputTokens)),
			decimal.NewFromInt(int64(raw.CacheReadInputTokens)),
			decimal.NewFromInt(int64(raw.CacheCreationTokens)),
			decimal.NewFromInt(int64(raw.CacheCreation5mTokens)),
			decimal.NewFromInt(int64(raw.CacheCreation1hTokens)),
		}
		buckets, replayed, ok := chooseNearestKiroCandidate(targetQuota, prices, ideals)
		return buckets, replayed, kiroCalibrationMethodResidualInput, ok
	}

	rawCost := kiroQuotaBeforeGroup(raw, prices)
	if !rawCost.IsPositive() {
		return dto.KiroUsageBuckets{}, 0, kiroCalibrationMethodProportionalScale, false
	}
	scale := targetBeforeGroup.Div(rawCost)
	ideals := [6]decimal.Decimal{
		decimal.NewFromInt(int64(raw.InputTokens)).Mul(scale),
		decimal.NewFromInt(int64(raw.OutputTokens)).Mul(scale),
		decimal.NewFromInt(int64(raw.CacheReadInputTokens)).Mul(scale),
		decimal.NewFromInt(int64(raw.CacheCreationTokens)).Mul(scale),
		decimal.NewFromInt(int64(raw.CacheCreation5mTokens)).Mul(scale),
		decimal.NewFromInt(int64(raw.CacheCreation1hTokens)).Mul(scale),
	}
	buckets, replayed, ok := chooseNearestKiroCandidate(targetQuota, prices, ideals)
	return buckets, replayed, kiroCalibrationMethodProportionalScale, ok
}

func applyKiroBucketsToUsage(usage *dto.Usage, buckets dto.KiroUsageBuckets) bool {
	if usage == nil || buckets.CacheCreation5mTokens > math.MaxInt-buckets.CacheCreation1hTokens ||
		buckets.InputTokens > math.MaxInt-buckets.OutputTokens {
		return false
	}
	cacheCreationTotal := buckets.CacheCreation5mTokens + buckets.CacheCreation1hTokens
	if buckets.CacheCreationTokens > math.MaxInt-cacheCreationTotal {
		return false
	}
	cacheCreationTotal += buckets.CacheCreationTokens
	if buckets.InputTokens > math.MaxInt-buckets.CacheReadInputTokens {
		return false
	}
	inputTokensTotal := buckets.InputTokens + buckets.CacheReadInputTokens
	if inputTokensTotal > math.MaxInt-cacheCreationTotal {
		return false
	}
	inputTokensTotal += cacheCreationTotal
	usage.PromptTokens = buckets.InputTokens
	usage.CompletionTokens = buckets.OutputTokens
	usage.TotalTokens = buckets.InputTokens + buckets.OutputTokens
	usage.InputTokens = inputTokensTotal
	usage.OutputTokens = buckets.OutputTokens
	usage.PromptTokensDetails.CachedTokens = buckets.CacheReadInputTokens
	usage.PromptTokensDetails.CachedCreationTokens = cacheCreationTotal
	usage.PromptTokensDetails.CacheWriteTokens = cacheCreationTotal
	usage.ClaudeCacheCreation5mTokens = buckets.CacheCreation5mTokens
	usage.ClaudeCacheCreation1hTokens = buckets.CacheCreation1hTokens
	usage.UsageSemantic = dto.BillingUsageSemanticAnthropic
	usage.UsageSource = "kiro_credits_calibrated"
	claudeUsage := &dto.ClaudeUsage{
		InputTokens:              buckets.InputTokens,
		OutputTokens:             buckets.OutputTokens,
		CacheReadInputTokens:     buckets.CacheReadInputTokens,
		CacheCreationInputTokens: cacheCreationTotal,
		CacheCreation: &dto.ClaudeCacheCreationUsage{
			Ephemeral5mInputTokens: buckets.CacheCreation5mTokens,
			Ephemeral1hInputTokens: buckets.CacheCreation1hTokens,
		},
	}
	usage.BillingUsage = dto.NewClaudeMessagesBillingUsage(claudeUsage)
	return true
}

// KiroRawUsageSnapshot rebuilds the pre-calibration usage for internal
// affinity/performance observations. Billing and client responses continue to
// use the calibrated usage stored on the original value.
func KiroRawUsageSnapshot(usage *dto.Usage) *dto.Usage {
	if usage == nil || usage.KiroMetering == nil || usage.KiroMetering.Calibration == nil {
		return usage
	}
	clone := *usage
	clone.BillingUsage = dto.CloneBillingUsage(usage.BillingUsage)
	if !applyKiroBucketsToUsage(&clone, usage.KiroMetering.Calibration.RawUsage) {
		return usage
	}
	clone.UsageSource = "kiro_raw_usage"
	return &clone
}

// CalibrateKiroUsage applies the configured credits-based billing-equivalent
// usage exactly once. Repeated calls reapply the frozen calibrated buckets so
// duplicate terminal stream events cannot replace or compound them.
func CalibrateKiroUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) *dto.KiroCreditCalibration {
	if relayInfo == nil || relayInfo.ChannelMeta == nil ||
		relayInfo.ChannelOtherSettings.KiroCreditBillingMode == "" || usage == nil {
		return nil
	}
	if usage.KiroMetering != nil && usage.KiroMetering.Calibration != nil {
		calibration := usage.KiroMetering.Calibration
		if calibration.Applied {
			_ = applyKiroBucketsToUsage(usage, calibration.CalibratedUsage)
			return calibration
		}
		// A malformed or missing first terminal event must not permanently freeze
		// token fallback when a later duplicate terminal carries valid cumulative
		// metering. Unsupported configuration remains frozen for this request.
		if usage.KiroMetering.Valid && finiteNonNegative(usage.KiroMetering.CreditsUsed) &&
			(calibration.Status == kiroCalibrationStatusFallbackMissing || calibration.Status == kiroCalibrationStatusFallbackInvalid) {
			usage.KiroMetering.Calibration = nil
		} else {
			return calibration
		}
	}

	raw, rawReason := normalizedKiroUsageBuckets(usage)
	if usage.KiroMetering == nil {
		usage.KiroMetering = &dto.KiroMeteringUsage{}
	}
	metering := usage.KiroMetering
	setCalibration := func(calibration *dto.KiroCreditCalibration) *dto.KiroCreditCalibration {
		metering.Calibration = calibration
		return calibration
	}
	if rawReason != "" {
		return setCalibration(newKiroFallback(raw, kiroCalibrationStatusFallbackInvalid, rawReason))
	}
	prices, setupReason := kiroLinearPricesForRelay(relayInfo)
	if setupReason != "" {
		return setCalibration(newKiroFallback(raw, kiroCalibrationStatusFallbackUnsupported, setupReason))
	}
	if metering.EventCount == 0 {
		return setCalibration(newKiroFallback(raw, kiroCalibrationStatusFallbackMissing, "x_kiro_metering_missing"))
	}
	if !metering.Valid || !finiteNonNegative(metering.CreditsUsed) {
		reason := metering.InvalidReason
		if reason == "" {
			reason = "credits_used_invalid"
		}
		return setCalibration(newKiroFallback(raw, kiroCalibrationStatusFallbackInvalid, reason))
	}

	settings := relayInfo.ChannelOtherSettings
	targetUSD := decimal.NewFromFloat(metering.CreditsUsed).Mul(decimal.NewFromFloat(settings.PricePerCredit))
	targetBeforeGroup := targetUSD.Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	targetQuota, clamp := common.QuotaFromDecimalChecked(targetBeforeGroup.Mul(prices.group))
	if clamp != nil {
		return setCalibration(newKiroFallback(raw, kiroCalibrationStatusFallbackOverflow, "target_quota_out_of_range"))
	}
	calibrated, replayed, method, ok := calibratedKiroCandidate(raw, targetBeforeGroup, targetQuota, prices)
	if !ok {
		return setCalibration(newKiroFallback(raw, kiroCalibrationStatusFallbackOverflow, "calibrated_usage_out_of_range"))
	}
	if !applyKiroBucketsToUsage(usage, calibrated) {
		return setCalibration(newKiroFallback(raw, kiroCalibrationStatusFallbackOverflow, "calibrated_usage_total_overflow"))
	}

	targetUSDFloat, _ := targetUSD.Float64()
	targetBeforeGroupFloat, _ := targetBeforeGroup.Float64()
	calibration := &dto.KiroCreditCalibration{
		Applied:                true,
		Status:                 kiroCalibrationStatusApplied,
		Method:                 method,
		RawUsage:               raw,
		CalibratedUsage:        calibrated,
		CreditsUsed:            metering.CreditsUsed,
		PricePerCredit:         settings.PricePerCredit,
		TargetUSD:              targetUSDFloat,
		ChargedUSD:             float64(targetQuota) / common.QuotaPerUnit,
		GroupRatio:             relayInfo.PriceData.GroupRatioInfo.GroupRatio,
		TargetQuotaBeforeGroup: targetBeforeGroupFloat,
		TargetQuota:            targetQuota,
		ReplayedQuota:          replayed,
		RoundingResidualQuota:  targetQuota - replayed,
		ChargedQuota:           targetQuota,
		SettlementStatus:       kiroSettlementStatusPending,
	}
	return setCalibration(calibration)
}

// settleKiroCalibratedQuota selects the authoritative credits target without
// issuing a second settlement. ReplayedQuota is explanatory only: integer
// billing-equivalent usage cannot represent every decimal credits charge.
func settleKiroCalibratedQuota(tokenQuota int, calibration *dto.KiroCreditCalibration) (int, bool) {
	if calibration == nil || !calibration.Applied {
		return tokenQuota, false
	}
	return calibration.TargetQuota, true
}

// reserveKiroCalibratedQuota makes the late-bound credits target the active
// pre-consumed amount before final settlement. If the caller cannot reserve
// the full target, return the amount already held so logs and used-quota
// counters never claim money that was not collected.
func reserveKiroCalibratedQuota(relayInfo *relaycommon.RelayInfo, calibration *dto.KiroCreditCalibration) (int, error) {
	if calibration == nil || !calibration.Applied {
		return 0, nil
	}
	targetQuota := calibration.TargetQuota
	calibration.ChargedQuota = targetQuota
	if relayInfo == nil || relayInfo.Billing == nil {
		return targetQuota, nil
	}
	if err := relayInfo.Billing.Reserve(targetQuota); err != nil {
		chargedQuota := relayInfo.Billing.GetPreConsumedQuota()
		if chargedQuota < 0 {
			chargedQuota = 0
		}
		calibration.ChargedQuota = chargedQuota
		calibration.ChargedUSD = float64(chargedQuota) / common.QuotaPerUnit
		calibration.SettlementStatus = kiroSettlementStatusPartialReserveFailed
		return chargedQuota, err
	}
	calibration.SettlementStatus = kiroSettlementStatusReserved
	return targetQuota, nil
}

func kiroBucketsAuditMap(buckets dto.KiroUsageBuckets) map[string]interface{} {
	return map[string]interface{}{
		"input_tokens":                   buckets.InputTokens,
		"output_tokens":                  buckets.OutputTokens,
		"cache_read_input_tokens":        buckets.CacheReadInputTokens,
		"cache_creation_input_tokens":    buckets.CacheCreationTokens,
		"cache_creation_5m_input_tokens": buckets.CacheCreation5mTokens,
		"cache_creation_1h_input_tokens": buckets.CacheCreation1hTokens,
	}
}

func appendKiroCreditBillingAuditForLog(other map[string]interface{}, calibration *dto.KiroCreditCalibration) {
	if other == nil || calibration == nil {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = map[string]interface{}{}
		other["admin_info"] = adminInfo
	}
	audit := map[string]interface{}{
		"mode":                    dto.KiroCreditBillingModeActualCalibrated,
		"status":                  calibration.Status,
		"method":                  calibration.Method,
		"target_quota":            calibration.TargetQuota,
		"replayed_quota":          calibration.ReplayedQuota,
		"rounding_residual_quota": calibration.RoundingResidualQuota,
		"charged_quota":           calibration.ChargedQuota,
		"settlement_status":       calibration.SettlementStatus,
		"raw_usage":               kiroBucketsAuditMap(calibration.RawUsage),
		"calibrated_usage":        kiroBucketsAuditMap(calibration.CalibratedUsage),
	}
	if calibration.FallbackReason != "" {
		audit["fallback_reason"] = calibration.FallbackReason
	}
	if finiteNonNegative(calibration.CreditsUsed) && calibration.Status == kiroCalibrationStatusApplied {
		audit["credits_used"] = calibration.CreditsUsed
	}
	if finitePositive(calibration.PricePerCredit) {
		audit["price_per_credit"] = calibration.PricePerCredit
	}
	if finiteNonNegative(calibration.TargetUSD) {
		audit["target_usd"] = calibration.TargetUSD
	}
	if finiteNonNegative(calibration.TargetQuotaBeforeGroup) {
		audit["target_quota_before_group"] = calibration.TargetQuotaBeforeGroup
	}
	if finiteNonNegative(calibration.GroupRatio) {
		audit["group_ratio"] = calibration.GroupRatio
	}
	adminInfo["kiro_credit_billing"] = audit

	if calibration.Status == kiroCalibrationStatusApplied {
		other["billing_basis"] = "kiro_credits"
		other["credits_used"] = calibration.CreditsUsed
		other["price_per_credit"] = calibration.PricePerCredit
		other["charged_usd"] = calibration.ChargedUSD
		other["usage_type"] = "billing_equivalent"
		other["billing_status"] = calibration.SettlementStatus
	}
}
