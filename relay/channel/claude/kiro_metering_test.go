package claude

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const validKiroMetering = `"x_kiro_metering":{"schema_version":1,"credits_used":2.75}`

func newClaudeKiroTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return ctx, recorder
}

func newClaudeKiroRelayInfo(format types.RelayFormat) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayFormat: format,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-opus-5",
		},
	}
}

func newCalibratedClaudeKiroRelayInfo(format types.RelayFormat) *relaycommon.RelayInfo {
	info := newClaudeKiroRelayInfo(format)
	info.ChannelMeta.ChannelType = constant.ChannelTypeAnthropic
	info.ChannelMeta.ChannelOtherSettings = dto.ChannelOtherSettings{
		KiroCreditBillingMode: dto.KiroCreditBillingModeActualCalibrated,
		PricePerCredit:        0.10,
	}
	info.PriceData = types.PriceData{
		ModelRatio:           0.5,
		CompletionRatio:      5,
		CacheRatio:           0.1,
		CacheCreationRatio:   1.25,
		CacheCreation5mRatio: 1.25,
		CacheCreation1hRatio: 2,
		GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
	}
	return info
}

func streamJSONPayloads(body string) []string {
	payloads := make([]string, 0)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload != "" && payload != "[DONE]" {
			payloads = append(payloads, payload)
		}
	}
	return payloads
}

func assertCalibratedKiroUsagePayload(t *testing.T, format types.RelayFormat, payload string) {
	t.Helper()
	assert.NotContains(t, payload, kiroMeteringExtensionPath)
	if format == types.RelayFormatClaude {
		assert.EqualValues(t, 1792, gjson.Get(payload, "usage.input_tokens").Int())
		assert.EqualValues(t, 20, gjson.Get(payload, "usage.output_tokens").Int())
		assert.EqualValues(t, 100, gjson.Get(payload, "usage.cache_read_input_tokens").Int())
		assert.EqualValues(t, 60, gjson.Get(payload, "usage.cache_creation_input_tokens").Int())
		assert.EqualValues(t, 20, gjson.Get(payload, "usage.cache_creation.ephemeral_5m_input_tokens").Int())
		assert.EqualValues(t, 30, gjson.Get(payload, "usage.cache_creation.ephemeral_1h_input_tokens").Int())
		return
	}

	// OpenAI prompt_tokens includes uncached input plus cache reads and writes.
	assert.EqualValues(t, 1952, gjson.Get(payload, "usage.prompt_tokens").Int())
	assert.EqualValues(t, 20, gjson.Get(payload, "usage.completion_tokens").Int())
	assert.EqualValues(t, 1972, gjson.Get(payload, "usage.total_tokens").Int())
	assert.EqualValues(t, 1952, gjson.Get(payload, "usage.input_tokens").Int())
	assert.EqualValues(t, 20, gjson.Get(payload, "usage.output_tokens").Int())
	assert.EqualValues(t, 100, gjson.Get(payload, "usage.prompt_tokens_details.cached_tokens").Int())
	assert.EqualValues(t, 60, gjson.Get(payload, "usage.prompt_tokens_details.cached_creation_tokens").Int())
	assert.EqualValues(t, 60, gjson.Get(payload, "usage.prompt_tokens_details.cache_write_tokens").Int())
}

const calibratedKiroRawUsage = `"usage":{"input_tokens":100,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":60,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":30}}`

const calibratedKiroMetering = `"x_kiro_metering":{"schema_version":1,"credits_used":0.02}`

func TestStripKiroMeteringExtensionValidation(t *testing.T) {
	tests := []struct {
		name       string
		extension  string
		accept     bool
		wantValid  bool
		wantReason string
		wantNil    bool
	}{
		{name: "valid terminal", extension: validKiroMetering, accept: true, wantValid: true},
		{name: "nonterminal is stripped but ignored", extension: validKiroMetering, wantNil: true},
		{name: "missing extension", extension: `"vendor_meta":{"trace":"x"}`, accept: true, wantNil: true},
		{name: "wrong schema", extension: `"x_kiro_metering":{"schema_version":2,"credits_used":2.75}`, accept: true, wantReason: "schema_version_invalid"},
		{name: "string credits", extension: `"x_kiro_metering":{"schema_version":1,"credits_used":"2.75"}`, accept: true, wantReason: "credits_used_not_number"},
		{name: "negative credits", extension: `"x_kiro_metering":{"schema_version":1,"credits_used":-1}`, accept: true, wantReason: "credits_used_invalid"},
		{name: "overflow credits", extension: `"x_kiro_metering":{"schema_version":1,"credits_used":1e999}`, accept: true, wantReason: "credits_used_invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := `{"type":"message_delta","usage":{"output_tokens":3},` + tt.extension + `}`
			cleaned, observation := stripKiroMeteringExtension(raw, tt.accept)

			assert.False(t, gjson.Get(cleaned, kiroMeteringExtensionPath).Exists())
			assert.Equal(t, int64(3), gjson.Get(cleaned, "usage.output_tokens").Int())
			if tt.wantNil {
				assert.Nil(t, observation)
				return
			}
			require.NotNil(t, observation)
			assert.Equal(t, tt.wantValid, observation.Valid)
			assert.Equal(t, tt.wantReason, observation.InvalidReason)
			if tt.wantValid {
				assert.Equal(t, 2.75, observation.CreditsUsed)
			}
		})
	}
}

func TestMergeKiroMeteringObservationDoesNotSumDuplicateEvents(t *testing.T) {
	usage := &dto.Usage{}
	mergeKiroMeteringObservation(usage, &dto.KiroMeteringUsage{Valid: true, CreditsUsed: 2.75, EventCount: 1})
	mergeKiroMeteringObservation(usage, &dto.KiroMeteringUsage{Valid: true, CreditsUsed: 2.75, EventCount: 1})
	mergeKiroMeteringObservation(usage, &dto.KiroMeteringUsage{Valid: true, CreditsUsed: 9, EventCount: 1})

	require.NotNil(t, usage.KiroMetering)
	assert.Equal(t, 2.75, usage.KiroMetering.CreditsUsed)
	assert.Equal(t, 3, usage.KiroMetering.EventCount)
	assert.True(t, usage.KiroMetering.Duplicate)
	assert.True(t, usage.KiroMetering.Conflict)
}

func TestKiroMeteringStreamIsReadAndStrippedForNativeAndOpenAI(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			ctx, recorder := newClaudeKiroTestContext()
			info := newClaudeKiroRelayInfo(format)
			claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}, ResponseText: strings.Builder{}}
			data := `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3},` + validKiroMetering + `}`

			err := HandleStreamResponseData(ctx, info, claudeInfo, data)

			require.Nil(t, err)
			require.NotNil(t, claudeInfo.Usage.KiroMetering)
			assert.True(t, claudeInfo.Usage.KiroMetering.Valid)
			assert.Equal(t, 2.75, claudeInfo.Usage.KiroMetering.CreditsUsed)
			assert.NotContains(t, recorder.Body.String(), "x_kiro_metering")
			assert.NotEmpty(t, recorder.Body.String())
			if format == types.RelayFormatClaude {
				assert.Contains(t, recorder.Body.String(), "message_delta")
			} else {
				assert.Contains(t, recorder.Body.String(), "chat.completion.chunk")
			}
		})
	}
}

func TestKiroMeteringNonStreamIsReadAndStrippedForNativeAndOpenAI(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			ctx, recorder := newClaudeKiroTestContext()
			info := newClaudeKiroRelayInfo(format)
			claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}}
			upstream := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}}
			data := []byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":2,"output_tokens":3},` + validKiroMetering + `}`)

			err := HandleClaudeResponseData(ctx, info, claudeInfo, upstream, data)

			require.Nil(t, err)
			require.NotNil(t, claudeInfo.Usage.KiroMetering)
			assert.True(t, claudeInfo.Usage.KiroMetering.Valid)
			assert.Equal(t, 2.75, claudeInfo.Usage.KiroMetering.CreditsUsed)
			assert.NotContains(t, recorder.Body.String(), "x_kiro_metering")
			assert.Contains(t, recorder.Body.String(), "msg_1")
		})
	}
}

func TestActualCalibratedKiroUsageIsReturnedForNativeAndOpenAINonStream(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			ctx, recorder := newClaudeKiroTestContext()
			info := newCalibratedClaudeKiroRelayInfo(format)
			claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}}
			upstream := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}}
			data := []byte(`{"id":"msg_calibrated","type":"message","role":"assistant","model":"claude-opus-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn",` +
				calibratedKiroRawUsage + `,` + calibratedKiroMetering + `,"vendor_meta":{"trace_id":"trace_1"}}`)

			err := HandleClaudeResponseData(ctx, info, claudeInfo, upstream, data)

			require.Nil(t, err)
			require.NotNil(t, claudeInfo.Usage.KiroMetering)
			require.NotNil(t, claudeInfo.Usage.KiroMetering.Calibration)
			assert.True(t, claudeInfo.Usage.KiroMetering.Calibration.Applied)
			assert.Equal(t, 1792, claudeInfo.Usage.PromptTokens)
			assert.Equal(t, 20, claudeInfo.Usage.CompletionTokens)
			assertCalibratedKiroUsagePayload(t, format, recorder.Body.String())
			if format == types.RelayFormatClaude {
				assert.Equal(t, "trace_1", gjson.Get(recorder.Body.String(), "vendor_meta.trace_id").String())
			}
		})
	}
}

func TestActualCalibratedKiroUsageIsReturnedForNativeAndOpenAIStream(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			ctx, recorder := newClaudeKiroTestContext()
			info := newCalibratedClaudeKiroRelayInfo(format)
			info.ShouldIncludeUsage = true
			claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}, ResponseText: strings.Builder{}}
			messageStart := `{"type":"message_start","message":{"id":"msg_stream","type":"message","role":"assistant","model":"claude-opus-5",` + calibratedKiroRawUsage + `}}`
			messageDelta := `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},` +
				calibratedKiroRawUsage + `,` + calibratedKiroMetering + `,"vendor_meta":{"trace_id":"trace_stream"}}`

			require.Nil(t, HandleStreamResponseData(ctx, info, claudeInfo, messageStart))
			require.Nil(t, HandleStreamResponseData(ctx, info, claudeInfo, messageDelta))
			HandleStreamFinalResponse(ctx, info, claudeInfo)

			require.NotNil(t, claudeInfo.Usage.KiroMetering)
			require.NotNil(t, claudeInfo.Usage.KiroMetering.Calibration)
			assert.True(t, claudeInfo.Usage.KiroMetering.Calibration.Applied)
			assert.Equal(t, 1792, claudeInfo.Usage.PromptTokens)
			assert.Equal(t, 20, claudeInfo.Usage.CompletionTokens)
			payloads := streamJSONPayloads(recorder.Body.String())
			require.Len(t, payloads, 2)
			assertCalibratedKiroUsagePayload(t, format, payloads[1])
			assert.NotContains(t, recorder.Body.String(), kiroMeteringExtensionPath)
			if format == types.RelayFormatClaude {
				assert.Equal(t, "trace_stream", gjson.Get(payloads[1], "vendor_meta.trace_id").String())
			}
		})
	}
}

func TestActualCalibratedNativeStreamWritesExplicitZeroUsageInPassThroughMode(t *testing.T) {
	ctx, recorder := newClaudeKiroTestContext()
	info := newCalibratedClaudeKiroRelayInfo(types.RelayFormatClaude)
	info.ChannelMeta.ChannelSetting.PassThroughBodyEnabled = true
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{
			PromptTokens:     7,
			CompletionTokens: 11,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         3,
				CachedCreationTokens: 9,
			},
			ClaudeCacheCreation5mTokens: 4,
			ClaudeCacheCreation1hTokens: 5,
		},
		ResponseText: strings.Builder{},
	}
	data := `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":11},"x_kiro_metering":{"schema_version":1,"credits_used":0},"vendor_meta":{"trace_id":"trace_zero"}}`

	require.True(t, shouldSkipClaudeMessageDeltaUsagePatch(info))
	require.Nil(t, HandleStreamResponseData(ctx, info, claudeInfo, data))

	payloads := streamJSONPayloads(recorder.Body.String())
	require.Len(t, payloads, 1)
	payload := payloads[0]
	for _, path := range []string{
		"usage.input_tokens",
		"usage.output_tokens",
		"usage.cache_read_input_tokens",
		"usage.cache_creation_input_tokens",
		"usage.cache_creation.ephemeral_5m_input_tokens",
		"usage.cache_creation.ephemeral_1h_input_tokens",
	} {
		result := gjson.Get(payload, path)
		assert.True(t, result.Exists(), "%s must be explicitly present", path)
		assert.Zero(t, result.Int(), "%s must be calibrated to zero", path)
	}
	require.NotNil(t, claudeInfo.Usage.KiroMetering)
	require.NotNil(t, claudeInfo.Usage.KiroMetering.Calibration)
	assert.True(t, claudeInfo.Usage.KiroMetering.Calibration.Applied)
	assert.Equal(t, "trace_zero", gjson.Get(payload, "vendor_meta.trace_id").String())
	assert.NotContains(t, payload, kiroMeteringExtensionPath)
}

func TestActualCalibratedMissingMeteringFallsBackWithoutFabricatingUsage(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			ctx, recorder := newClaudeKiroTestContext()
			info := newCalibratedClaudeKiroRelayInfo(format)
			claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}}
			upstream := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}}
			data := []byte(`{"id":"msg_missing","type":"message","role":"assistant","model":"claude-opus-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn",` + calibratedKiroRawUsage + `}`)

			err := HandleClaudeResponseData(ctx, info, claudeInfo, upstream, data)

			require.Nil(t, err)
			require.NotNil(t, claudeInfo.Usage.KiroMetering)
			require.NotNil(t, claudeInfo.Usage.KiroMetering.Calibration)
			assert.False(t, claudeInfo.Usage.KiroMetering.Calibration.Applied)
			assert.Equal(t, 100, claudeInfo.Usage.PromptTokens)
			assert.Equal(t, 20, claudeInfo.Usage.CompletionTokens)
			assert.NotContains(t, recorder.Body.String(), kiroMeteringExtensionPath)
			assert.NotContains(t, recorder.Body.String(), "kiro_credits_calibrated")
			if format == types.RelayFormatClaude {
				assert.EqualValues(t, 100, gjson.Get(recorder.Body.String(), "usage.input_tokens").Int())
				assert.EqualValues(t, 20, gjson.Get(recorder.Body.String(), "usage.output_tokens").Int())
			} else {
				assert.EqualValues(t, 260, gjson.Get(recorder.Body.String(), "usage.prompt_tokens").Int())
				assert.EqualValues(t, 20, gjson.Get(recorder.Body.String(), "usage.completion_tokens").Int())
				assert.EqualValues(t, 280, gjson.Get(recorder.Body.String(), "usage.total_tokens").Int())
			}
		})
	}
}

func TestLaterValidTerminalMeteringRecoversFromMissingFallback(t *testing.T) {
	ctx, _ := newClaudeKiroTestContext()
	info := newCalibratedClaudeKiroRelayInfo(types.RelayFormatClaude)
	claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}, ResponseText: strings.Builder{}}
	missing := `{"type":"message_delta","delta":{"stop_reason":"end_turn"},` + calibratedKiroRawUsage + `}`
	valid := `{"type":"message_delta","delta":{"stop_reason":"end_turn"},` + calibratedKiroRawUsage + `,` + calibratedKiroMetering + `}`

	require.Nil(t, HandleStreamResponseData(ctx, info, claudeInfo, missing))
	require.NotNil(t, claudeInfo.Usage.KiroMetering)
	require.NotNil(t, claudeInfo.Usage.KiroMetering.Calibration)
	assert.False(t, claudeInfo.Usage.KiroMetering.Calibration.Applied)

	require.Nil(t, HandleStreamResponseData(ctx, info, claudeInfo, valid))
	require.NotNil(t, claudeInfo.Usage.KiroMetering.Calibration)
	assert.True(t, claudeInfo.Usage.KiroMetering.Calibration.Applied)
	assert.Equal(t, 1792, claudeInfo.Usage.PromptTokens)
}

func TestKiroMeteringErrorResponseDoesNotReachUsage(t *testing.T) {
	ctx, recorder := newClaudeKiroTestContext()
	info := newCalibratedClaudeKiroRelayInfo(types.RelayFormatClaude)
	claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}}
	upstream := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
	data := []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"limited"},` + validKiroMetering + `}`)

	err := HandleClaudeResponseData(ctx, info, claudeInfo, upstream, data)

	require.NotNil(t, err)
	assert.Nil(t, claudeInfo.Usage.KiroMetering)
	assert.Empty(t, recorder.Body.String())
}

func TestKiroMeteringStreamErrorDoesNotFabricateCalibration(t *testing.T) {
	ctx, recorder := newClaudeKiroTestContext()
	info := newCalibratedClaudeKiroRelayInfo(types.RelayFormatClaude)
	claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}, ResponseText: strings.Builder{}}
	data := `{"type":"error","error":{"type":"rate_limit_error","message":"limited"},` + validKiroMetering + `}`

	err := HandleStreamResponseData(ctx, info, claudeInfo, data)

	require.NotNil(t, err)
	assert.Nil(t, claudeInfo.Usage.KiroMetering)
	assert.Empty(t, recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), kiroMeteringExtensionPath)
}
