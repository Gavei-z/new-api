package claude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func directPassthroughTestContext(path string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return c, recorder
}

func directPassthroughRelayInfo(format types.RelayFormat, path string) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		RelayFormat:    format,
		RequestURLPath: path,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAnthropic,
			UpstreamModelName: "claude-haiku-4-5",
		},
	}
	info.ChannelOtherSettings.AnthropicDirectPassthroughEnabled = true
	if format == types.RelayFormatOpenAI {
		info.RelayMode = relayconstant.RelayModeChatCompletions
	}
	info.SetEstimatePromptTokens(17)
	return info
}

func TestDirectPassthroughHandlerPreservesClaudeJSON(t *testing.T) {
	body := `{"id":"msg_exact","type":"message","role":"assistant","model":"claude-haiku-4-5-20251001","content":[{"type":"thinking","thinking":"secret","signature":"encrypted-signature"},{"type":"text","text":"OK","future_extension":{"kept":true}}],"stop_reason":"end_turn","usage":{"input_tokens":41,"output_tokens":7,"cache_read_input_tokens":3}}`
	c, recorder := directPassthroughTestContext("/v1/messages")
	info := directPassthroughRelayInfo(types.RelayFormatClaude, "/v1/messages")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Upstream": []string{"cfjwl"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, relayErr := DirectPassthroughHandler(c, resp, info)

	require.Nil(t, relayErr)
	require.Equal(t, body, recorder.Body.String())
	require.Equal(t, "cfjwl", recorder.Header().Get("X-Upstream"))
	require.Equal(t, 41, usage.PromptTokens)
	require.Equal(t, 7, usage.CompletionTokens)
	require.Equal(t, 3, usage.PromptTokensDetails.CachedTokens)
}

func TestDirectPassthroughStreamHandlerPreservesClaudeSSEBytes(t *testing.T) {
	body := "event: message_start\r\n" +
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_exact\",\"model\":\"claude-haiku-4-5-20251001\",\"usage\":{\"input_tokens\":41,\"output_tokens\":1}}}\r\n\r\n" +
		": upstream-heartbeat\r\n\r\n" +
		"event: content_block_delta\r\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"signature_delta\",\"signature\":\"encrypted-signature\"}}\r\n\r\n" +
		"event: message_delta\r\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":7}}\r\n\r\n"
	c, recorder := directPassthroughTestContext("/v1/messages")
	info := directPassthroughRelayInfo(types.RelayFormatClaude, "/v1/messages")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "X-Upstream": []string{"cfjwl"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, relayErr := DirectPassthroughStreamHandler(c, resp, info)

	require.Nil(t, relayErr)
	require.Equal(t, body, recorder.Body.String())
	require.Equal(t, "cfjwl", recorder.Header().Get("X-Upstream"))
	require.Equal(t, 41, usage.PromptTokens)
	require.Equal(t, 7, usage.CompletionTokens)
}

func TestDirectPassthroughStreamHandlerPreservesOpenAIChatSSEBytes(t *testing.T) {
	body := "data: {\"id\":\"chat_exact\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-haiku-4-5-20251001\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"OK\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chat_exact\",\"object\":\"chat.completion.chunk\",\"choices\":[],\"usage\":{\"prompt_tokens\":41,\"completion_tokens\":7,\"total_tokens\":48}}\n\n" +
		"data: [DONE]\n\n"
	c, recorder := directPassthroughTestContext("/v1/chat/completions")
	info := directPassthroughRelayInfo(types.RelayFormatOpenAI, "/v1/chat/completions")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, relayErr := DirectPassthroughStreamHandler(c, resp, info)

	require.Nil(t, relayErr)
	require.Equal(t, body, recorder.Body.String())
	require.Equal(t, &dto.Usage{PromptTokens: 41, CompletionTokens: 7, TotalTokens: 48}, usage)
}

func TestDirectPassthroughStreamHandlerRejectsIncompleteClaudeUsage(t *testing.T) {
	body := "event: message_start\n" +
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_incomplete\",\"usage\":{\"input_tokens\":41,\"output_tokens\":1}}}\n\n"
	c, recorder := directPassthroughTestContext("/v1/messages")
	info := directPassthroughRelayInfo(types.RelayFormatClaude, "/v1/messages")
	billing := &directPassthroughBillingSpy{preConsumed: 321}
	info.Billing = billing
	info.FinalPreConsumedQuota = 321
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, relayErr := DirectPassthroughStreamHandler(c, resp, info)

	require.Nil(t, usage)
	require.NotNil(t, relayErr)
	require.Equal(t, http.StatusBadGateway, relayErr.StatusCode)
	require.Equal(t, body, recorder.Body.String())
	require.Equal(t, 1, billing.settleCalls)
	require.Equal(t, 321, billing.settledQuota)
}

func TestDirectPassthroughHandlerRejectsMissingUpstreamUsageWithoutEstimating(t *testing.T) {
	body := `{"id":"msg_no_usage","type":"message","content":[{"type":"text","text":"OK"}]}`
	c, recorder := directPassthroughTestContext("/v1/messages")
	info := directPassthroughRelayInfo(types.RelayFormatClaude, "/v1/messages")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, relayErr := DirectPassthroughHandler(c, resp, info)

	require.Nil(t, usage)
	require.NotNil(t, relayErr)
	require.Equal(t, http.StatusBadGateway, relayErr.StatusCode)
	require.Empty(t, recorder.Body.String())
	require.False(t, c.GetBool(string(constant.ContextKeyLocalCountTokens)))
}

func TestDirectPassthroughHandlerRejectsIncompleteOrInvalidUsage(t *testing.T) {
	for _, body := range []string{
		`{"id":"msg_empty_usage","type":"message","usage":{}}`,
		`{"id":"msg_missing_output","type":"message","usage":{"input_tokens":41}}`,
		`{"id":"msg_negative_usage","type":"message","usage":{"input_tokens":-1,"output_tokens":7}}`,
	} {
		c, recorder := directPassthroughTestContext("/v1/messages")
		info := directPassthroughRelayInfo(types.RelayFormatClaude, "/v1/messages")
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		usage, relayErr := DirectPassthroughHandler(c, resp, info)

		require.Nil(t, usage)
		require.NotNil(t, relayErr)
		require.Equal(t, http.StatusBadGateway, relayErr.StatusCode)
		require.Empty(t, recorder.Body.String())
	}
}

type directPassthroughBillingSpy struct {
	preConsumed  int
	settledQuota int
	settleCalls  int
}

func (b *directPassthroughBillingSpy) Settle(actualQuota int) error {
	b.settleCalls++
	b.settledQuota = actualQuota
	return nil
}

func (b *directPassthroughBillingSpy) Refund(*gin.Context) {}

func (b *directPassthroughBillingSpy) NeedsRefund() bool { return false }

func (b *directPassthroughBillingSpy) GetPreConsumedQuota() int { return b.preConsumed }

func (b *directPassthroughBillingSpy) Reserve(targetQuota int) error {
	b.preConsumed = targetQuota
	return nil
}

func TestDirectPassthroughErrorHandlerPreservesStatusHeadersAndBody(t *testing.T) {
	body := `{"type":"error","error":{"type":"invalid_request_error","message":"provider error"}}`
	c, recorder := directPassthroughTestContext("/v1/messages")
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Upstream": []string{"cfjwl"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	relayErr := DirectPassthroughErrorHandler(c, resp)

	require.NotNil(t, relayErr)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, body, recorder.Body.String())
	require.Equal(t, "cfjwl", recorder.Header().Get("X-Upstream"))
}

func TestDirectAdaptorPreservesNativeQueryAndUserAgent(t *testing.T) {
	c, _ := directPassthroughTestContext("/v1/messages?beta=true&client=veridrop")
	c.Request.Header.Set("User-Agent", "python-httpx/veridrop")
	info := directPassthroughRelayInfo(types.RelayFormatClaude, "/v1/messages?beta=true&client=veridrop")
	info.ChannelBaseUrl = "https://cfjwl.example"
	info.ApiKey = "provider-key"
	adaptor := &Adaptor{}

	requestURL, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://cfjwl.example/v1/messages?beta=true&client=veridrop", requestURL)

	header := http.Header{}
	require.NoError(t, adaptor.SetupRequestHeader(c, &header, info))
	require.Equal(t, "python-httpx/veridrop", header.Get("User-Agent"))
	require.Equal(t, "provider-key", header.Get("x-api-key"))
}
