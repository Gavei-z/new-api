package service

import (
	"errors"
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func calibratedKiroRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: "claude-test",
		StartTime:       time.Now(),
		PriceData: types.PriceData{
			ModelRatio:           0.5,
			CompletionRatio:      5,
			CacheRatio:           0.1,
			CacheCreationRatio:   1.25,
			CacheCreation5mRatio: 1.25,
			CacheCreation1hRatio: 2,
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeAnthropic,
			ChannelOtherSettings: dto.ChannelOtherSettings{
				KiroCreditBillingMode: dto.KiroCreditBillingModeActualCalibrated,
				PricePerCredit:        0.10,
			},
		},
	}
}

func kiroUsageWithCredits(credits float64) *dto.Usage {
	return &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
		TotalTokens:      120,
		UsageSemantic:    dto.BillingUsageSemanticAnthropic,
		KiroMetering: &dto.KiroMeteringUsage{
			Valid:       true,
			CreditsUsed: credits,
			EventCount:  1,
		},
	}
}

func TestCalibrateKiroUsageResidualInputAndExactTarget(t *testing.T) {
	relayInfo := calibratedKiroRelayInfo()
	usage := kiroUsageWithCredits(0.02)

	calibration := CalibrateKiroUsage(relayInfo, usage)

	require.NotNil(t, calibration)
	assert.True(t, calibration.Applied)
	assert.Equal(t, kiroCalibrationMethodResidualInput, calibration.Method)
	assert.Equal(t, dto.KiroUsageBuckets{InputTokens: 100, OutputTokens: 20}, calibration.RawUsage)
	assert.Equal(t, 20, calibration.CalibratedUsage.OutputTokens)
	assert.Equal(t, 1900, calibration.CalibratedUsage.InputTokens)
	assert.Equal(t, 1000, calibration.TargetQuota)
	assert.Equal(t, 1000, calibration.ReplayedQuota)
	assert.Zero(t, calibration.RoundingResidualQuota)
	assert.Equal(t, calibration.CalibratedUsage.InputTokens, usage.PromptTokens)
	require.NotNil(t, usage.BillingUsage)
	require.NotNil(t, usage.BillingUsage.ClaudeUsage)
	assert.Equal(t, calibration.CalibratedUsage.InputTokens, usage.BillingUsage.ClaudeUsage.InputTokens)

	finalQuota, settled := settleKiroCalibratedQuota(calibration.ReplayedQuota, calibration)
	assert.True(t, settled)
	assert.Equal(t, calibration.TargetQuota, finalQuota)
}

func TestCalibrateKiroUsagePreservesKnownOutputAndCacheBuckets(t *testing.T) {
	relayInfo := calibratedKiroRelayInfo()
	usage := kiroUsageWithCredits(0.02)
	usage.PromptTokensDetails.CachedTokens = 100
	usage.PromptTokensDetails.CachedCreationTokens = 60
	usage.ClaudeCacheCreation5mTokens = 20
	usage.ClaudeCacheCreation1hTokens = 30

	calibration := CalibrateKiroUsage(relayInfo, usage)

	require.NotNil(t, calibration)
	require.True(t, calibration.Applied)
	assert.Equal(t, 10, calibration.RawUsage.CacheCreationTokens)
	assert.Equal(t, 20, calibration.RawUsage.CacheCreation5mTokens)
	assert.Equal(t, 30, calibration.RawUsage.CacheCreation1hTokens)
	assert.Equal(t, calibration.RawUsage.OutputTokens, calibration.CalibratedUsage.OutputTokens)
	assert.Equal(t, calibration.RawUsage.CacheReadInputTokens, calibration.CalibratedUsage.CacheReadInputTokens)
	assert.Equal(t, calibration.RawUsage.CacheCreationTokens, calibration.CalibratedUsage.CacheCreationTokens)
	assert.Equal(t, calibration.RawUsage.CacheCreation5mTokens, calibration.CalibratedUsage.CacheCreation5mTokens)
	assert.Equal(t, calibration.RawUsage.CacheCreation1hTokens, calibration.CalibratedUsage.CacheCreation1hTokens)
	assert.Equal(t, calibration.ReplayedQuota, calculateTextQuotaSummary(newKiroBillingContext(), relayInfo, effectiveBillingUsage(usage)).Quota)

	rawSnapshot := KiroRawUsageSnapshot(usage)
	require.NotNil(t, rawSnapshot)
	assert.Equal(t, 100, rawSnapshot.PromptTokens)
	assert.Equal(t, 20, rawSnapshot.CompletionTokens)
	assert.Equal(t, 100, rawSnapshot.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 60, rawSnapshot.PromptTokensDetails.CacheCreationTokensTotal())
}

func TestCalibrateKiroUsageScalesAllBucketsWhenKnownCostExceedsTarget(t *testing.T) {
	relayInfo := calibratedKiroRelayInfo()
	usage := kiroUsageWithCredits(0.001)
	usage.CompletionTokens = 100
	usage.PromptTokensDetails.CachedTokens = 100
	usage.PromptTokensDetails.CachedCreationTokens = 20
	usage.ClaudeCacheCreation5mTokens = 20

	calibration := CalibrateKiroUsage(relayInfo, usage)

	require.NotNil(t, calibration)
	assert.True(t, calibration.Applied)
	assert.Equal(t, kiroCalibrationMethodProportionalScale, calibration.Method)
	assert.Less(t, calibration.CalibratedUsage.OutputTokens, calibration.RawUsage.OutputTokens)
	assert.Less(t, calibration.CalibratedUsage.CacheReadInputTokens, calibration.RawUsage.CacheReadInputTokens)
	assert.LessOrEqual(t, int(math.Abs(float64(calibration.RoundingResidualQuota))), 1)
}

func TestCalibrateKiroUsageZeroCreditsAndUnreachableIntegerReplay(t *testing.T) {
	t.Run("zero credits scales all billable buckets to zero", func(t *testing.T) {
		relayInfo := calibratedKiroRelayInfo()
		usage := kiroUsageWithCredits(0)

		calibration := CalibrateKiroUsage(relayInfo, usage)

		require.NotNil(t, calibration)
		assert.True(t, calibration.Applied)
		assert.Zero(t, calibration.TargetQuota)
		assert.Equal(t, dto.KiroUsageBuckets{}, calibration.CalibratedUsage)
		assert.Zero(t, usage.PromptTokens)
		assert.Zero(t, usage.CompletionTokens)
	})

	t.Run("target remains authoritative when integer usage cannot replay it", func(t *testing.T) {
		relayInfo := calibratedKiroRelayInfo()
		relayInfo.PriceData.ModelRatio = 2
		relayInfo.PriceData.GroupRatioInfo.GroupRatio = 3
		usage := kiroUsageWithCredits(0.00001)
		usage.PromptTokens = 0
		usage.CompletionTokens = 0

		calibration := CalibrateKiroUsage(relayInfo, usage)

		require.NotNil(t, calibration)
		assert.True(t, calibration.Applied)
		assert.Equal(t, 2, calibration.TargetQuota)
		assert.NotEqual(t, calibration.TargetQuota, calibration.ReplayedQuota)
		finalQuota, settled := settleKiroCalibratedQuota(calibration.ReplayedQuota, calibration)
		assert.True(t, settled)
		assert.Equal(t, 2, finalQuota)
	})
}

func TestCalibrateKiroUsageFallbacks(t *testing.T) {
	tests := []struct {
		name       string
		mutateInfo func(*relaycommon.RelayInfo)
		usage      *dto.Usage
		wantStatus string
		wantReason string
	}{
		{name: "missing", usage: &dto.Usage{PromptTokens: 7}, wantStatus: kiroCalibrationStatusFallbackMissing, wantReason: "x_kiro_metering_missing"},
		{name: "negative", usage: kiroUsageWithCredits(-1), wantStatus: kiroCalibrationStatusFallbackInvalid, wantReason: "credits_used_invalid"},
		{name: "NaN", usage: kiroUsageWithCredits(math.NaN()), wantStatus: kiroCalibrationStatusFallbackInvalid, wantReason: "credits_used_invalid"},
		{name: "Inf", usage: kiroUsageWithCredits(math.Inf(1)), wantStatus: kiroCalibrationStatusFallbackInvalid, wantReason: "credits_used_invalid"},
		{name: "overflow", usage: kiroUsageWithCredits(math.MaxFloat64), wantStatus: kiroCalibrationStatusFallbackOverflow, wantReason: "target_quota_out_of_range"},
		{name: "use price", usage: kiroUsageWithCredits(1), mutateInfo: func(info *relaycommon.RelayInfo) { info.PriceData.UsePrice = true }, wantStatus: kiroCalibrationStatusFallbackUnsupported, wantReason: "use_price_not_supported"},
		{name: "tiered", usage: kiroUsageWithCredits(1), mutateInfo: func(info *relaycommon.RelayInfo) { info.TieredBillingSnapshot = &billingexpr.BillingSnapshot{} }, wantStatus: kiroCalibrationStatusFallbackUnsupported, wantReason: "tiered_billing_not_supported"},
		{name: "other ratio", usage: kiroUsageWithCredits(1), mutateInfo: func(info *relaycommon.RelayInfo) { info.PriceData.AddOtherRatio("priority", 2) }, wantStatus: kiroCalibrationStatusFallbackUnsupported, wantReason: "other_ratios_not_supported"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			relayInfo := calibratedKiroRelayInfo()
			if test.mutateInfo != nil {
				test.mutateInfo(relayInfo)
			}
			originalPrompt := test.usage.PromptTokens

			calibration := CalibrateKiroUsage(relayInfo, test.usage)

			require.NotNil(t, calibration)
			assert.False(t, calibration.Applied)
			assert.Equal(t, test.wantStatus, calibration.Status)
			assert.Equal(t, test.wantReason, calibration.FallbackReason)
			assert.Equal(t, originalPrompt, test.usage.PromptTokens)
		})
	}
}

func TestCalibrateKiroUsageDisabledLeavesTokenBillingUntouched(t *testing.T) {
	relayInfo := calibratedKiroRelayInfo()
	relayInfo.ChannelOtherSettings.KiroCreditBillingMode = ""
	relayInfo.ChannelOtherSettings.PricePerCredit = 0
	relayInfo.ChannelOtherSettings.ClaudeInputBillingMode = dto.ClaudeInputBillingModeUpstream
	usage := kiroUsageWithCredits(1)
	original := *usage

	calibration := CalibrateKiroUsage(relayInfo, usage)

	assert.Nil(t, calibration)
	assert.Equal(t, original.PromptTokens, usage.PromptTokens)
	assert.Equal(t, original.CompletionTokens, usage.CompletionTokens)
	assert.Nil(t, usage.KiroMetering.Calibration)
}

func TestSettleKiroCalibratedQuotaKeepsCreditsTargetOnReplayMismatch(t *testing.T) {
	calibration := &dto.KiroCreditCalibration{
		Applied:       true,
		Status:        kiroCalibrationStatusApplied,
		TargetQuota:   100,
		ReplayedQuota: 99,
	}

	quota, settled := settleKiroCalibratedQuota(101, calibration)

	assert.True(t, settled)
	assert.Equal(t, 100, quota)
	assert.True(t, calibration.Applied)
	assert.Equal(t, kiroCalibrationStatusApplied, calibration.Status)
}

func TestAppendKiroCreditBillingAuditForLogSeparatesUserReceiptAndAdminDetail(t *testing.T) {
	calibration := &dto.KiroCreditCalibration{
		Applied:               true,
		Status:                kiroCalibrationStatusApplied,
		Method:                kiroCalibrationMethodResidualInput,
		CreditsUsed:           0.02,
		PricePerCredit:        0.10,
		ChargedUSD:            0.002,
		TargetUSD:             0.002,
		TargetQuota:           1000,
		ReplayedQuota:         999,
		RoundingResidualQuota: 1,
		ChargedQuota:          1000,
		SettlementStatus:      kiroSettlementStatusSettled,
		RawUsage:              dto.KiroUsageBuckets{InputTokens: 100},
		CalibratedUsage:       dto.KiroUsageBuckets{InputTokens: 1998},
	}
	other := map[string]interface{}{}

	appendKiroCreditBillingAuditForLog(other, calibration)

	assert.Equal(t, "kiro_credits", other["billing_basis"])
	assert.Equal(t, "billing_equivalent", other["usage_type"])
	assert.Equal(t, 0.02, other["credits_used"])
	assert.Equal(t, 0.10, other["price_per_credit"])
	assert.Equal(t, 0.002, other["charged_usd"])
	assert.Equal(t, kiroSettlementStatusSettled, other["billing_status"])
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	adminBilling, ok := adminInfo["kiro_credit_billing"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, adminBilling, "raw_usage")
	assert.Contains(t, adminBilling, "calibrated_usage")
	assert.Equal(t, 1, adminBilling["rounding_residual_quota"])
	assert.Equal(t, 1000, adminBilling["charged_quota"])
	assert.Equal(t, kiroSettlementStatusSettled, adminBilling["settlement_status"])
}

type countingBillingSettler struct {
	preConsumed  int
	reserveCalls int
	reserveErr   error
	settleCalls  int
	actualQuota  int
	settleErr    error
}

func (b *countingBillingSettler) Settle(actualQuota int) error {
	b.settleCalls++
	b.actualQuota = actualQuota
	return b.settleErr
}

func (b *countingBillingSettler) Refund(*gin.Context) {}
func (b *countingBillingSettler) NeedsRefund() bool   { return false }
func (b *countingBillingSettler) GetPreConsumedQuota() int {
	return b.preConsumed
}
func (b *countingBillingSettler) Reserve(targetQuota int) error {
	b.reserveCalls++
	if b.reserveErr != nil {
		return b.reserveErr
	}
	b.preConsumed = targetQuota
	return nil
}

func newKiroBillingContext() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	return ctx
}

func TestPostTextConsumeQuotaSettlesCalibratedTargetExactlyOnce(t *testing.T) {
	for _, test := range []struct {
		name             string
		reserveErr       error
		wantChargedQuota int
		wantSettlement   string
	}{
		{name: "success", wantChargedQuota: 1000, wantSettlement: kiroSettlementStatusSettled},
		{name: "insufficient balance records only reserved amount", reserveErr: errors.New("insufficient balance"), wantChargedQuota: 500, wantSettlement: kiroSettlementStatusPartialReserveFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			billing := &countingBillingSettler{preConsumed: 500, reserveErr: test.reserveErr}
			relayInfo := calibratedKiroRelayInfo()
			relayInfo.Billing = billing
			relayInfo.ChannelOtherSettings.ClaudeInputBillingMode = dto.ClaudeInputBillingModeLocalEstimate
			relayInfo.SetEstimatePromptTokens(99999)
			usage := kiroUsageWithCredits(0.02)

			PostTextConsumeQuota(newKiroBillingContext(), relayInfo, usage, nil)

			assert.Equal(t, 1, billing.reserveCalls)
			assert.Equal(t, 1, billing.settleCalls)
			assert.Equal(t, test.wantChargedQuota, billing.actualQuota)
			require.NotNil(t, usage.KiroMetering.Calibration)
			assert.True(t, usage.KiroMetering.Calibration.Status == kiroCalibrationStatusApplied)
			assert.Equal(t, test.wantChargedQuota, usage.KiroMetering.Calibration.ChargedQuota)
			assert.Equal(t, test.wantSettlement, usage.KiroMetering.Calibration.SettlementStatus)
			assert.NotEqual(t, 99999, usage.PromptTokens)
		})
	}
}

func TestPostTextConsumeQuotaKeepsCfjwlUpstreamTokenBilling(t *testing.T) {
	billing := &countingBillingSettler{}
	relayInfo := calibratedKiroRelayInfo()
	relayInfo.Billing = billing
	relayInfo.ChannelOtherSettings.KiroCreditBillingMode = ""
	relayInfo.ChannelOtherSettings.PricePerCredit = 0
	relayInfo.ChannelOtherSettings.ClaudeInputBillingMode = dto.ClaudeInputBillingModeUpstream
	usage := kiroUsageWithCredits(1)

	PostTextConsumeQuota(newKiroBillingContext(), relayInfo, usage, nil)

	assert.Equal(t, 1, billing.settleCalls)
	assert.Zero(t, billing.reserveCalls)
	// 100 input * 0.5 + 20 output * (0.5 * 5) = 100 quota.
	assert.Equal(t, 100, billing.actualQuota)
	assert.Nil(t, usage.KiroMetering.Calibration)
}
