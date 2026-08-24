package claude

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// DirectPassthroughHandler observes usage for billing while returning the
// upstream body byte-for-byte. It deliberately avoids DTO re-marshalling so
// provider extensions, encrypted thinking signatures, document blocks and
// message identifiers remain untouched.
func DirectPassthroughHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeReadResponseBodyFailed)
	}

	usage, ok := observeDirectJSONResponse(c, info, body)
	if !ok {
		return nil, types.NewErrorWithStatusCode(
			errors.New("cfjwl response is missing upstream usage"),
			types.ErrorCodeBadResponseBody,
			http.StatusBadGateway,
			types.ErrOptionWithSkipRetry(),
		)
	}
	service.IOCopyBytesGracefully(c, resp, body)
	return usage, nil
}

// DirectPassthroughErrorHandler writes an upstream error unchanged while also
// returning an internal error so pre-consumed quota can be refunded. Callers
// must avoid appending a second JSON envelope after the response is written.
func DirectPassthroughErrorHandler(c *gin.Context, resp *http.Response) *types.NewAPIError {
	defer service.CloseResponseBodyGracefully(resp)
	statusCode := http.StatusBadGateway
	if resp != nil && resp.StatusCode > 0 {
		statusCode = resp.StatusCode
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeReadResponseBodyFailed, statusCode, types.ErrOptionWithSkipRetry())
	}
	service.IOCopyBytesGracefully(c, resp, body)
	return types.NewErrorWithStatusCode(
		fmt.Errorf("cfjwl upstream returned HTTP %d", statusCode),
		types.ErrorCodeBadResponse,
		statusCode,
		types.ErrOptionWithSkipRetry(),
	)
}

// DirectPassthroughStreamHandler relays the complete upstream SSE framing,
// including event lines, comments and JSON whitespace. Usage is parsed from a
// side channel only and is never written back into the response stream.
func DirectPassthroughStreamHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	copyDirectResponseHeaders(c, resp)
	c.Writer.WriteHeader(resp.StatusCode)

	observer := newDirectStreamUsageObserver(info)
	idleTimeout := time.Duration(constant.StreamingTimeout) * time.Second
	idleTimer := time.AfterFunc(idleTimeout, func() {
		_ = resp.Body.Close()
	})
	defer idleTimer.Stop()
	reader := bufio.NewReader(resp.Body)
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			idleTimer.Reset(idleTimeout)
			observer.Observe(line)
			if _, writeErr := c.Writer.Write([]byte(line)); writeErr != nil {
				logger.LogError(c, "direct passthrough stream write failed: "+writeErr.Error())
				break
			}
			c.Writer.Flush()
		}

		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				logger.LogError(c, "direct passthrough stream read failed: "+readErr.Error())
			}
			break
		}
		if c.Request != nil && c.Request.Context().Err() != nil {
			break
		}
	}

	usage, ok := observer.Usage()
	if !ok {
		logger.LogError(c, "cfjwl direct stream ended without upstream usage; local token estimation is disabled")
		finalizeIncompleteDirectStreamBilling(c, info)
		return nil, types.NewErrorWithStatusCode(
			errors.New("cfjwl stream ended without complete upstream usage"),
			types.ErrorCodeBadResponseBody,
			http.StatusBadGateway,
			types.ErrOptionWithSkipRetry(),
		)
	}
	if observer.webSearchRequests > 0 {
		c.Set("claude_web_search_requests", observer.webSearchRequests)
	}
	return usage, nil
}

func copyDirectResponseHeaders(c *gin.Context, resp *http.Response) {
	if c == nil || c.Writer == nil || resp == nil {
		return
	}
	for name, values := range resp.Header {
		if !service.ShouldCopyUpstreamHeader(c, name, values) {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
}

func observeDirectJSONResponse(c *gin.Context, info *relaycommon.RelayInfo, body []byte) (*dto.Usage, bool) {
	if info != nil && info.RelayFormat == types.RelayFormatClaude {
		if !hasNonNegativeIntegerField(body, "usage.input_tokens") ||
			!hasNonNegativeIntegerField(body, "usage.output_tokens") {
			return nil, false
		}
		var response dto.ClaudeResponse
		if err := common.Unmarshal(body, &response); err == nil && response.Usage != nil {
			recordClaudeServerToolUsage(c, response.Usage)
			return usageFromClaudeUsage(response.Usage), true
		}
		return nil, false
	}

	if !hasNonNegativeIntegerField(body, "usage.prompt_tokens") ||
		!hasNonNegativeIntegerField(body, "usage.completion_tokens") {
		return nil, false
	}
	var response struct {
		Usage *dto.Usage `json:"usage"`
	}
	if err := common.Unmarshal(body, &response); err != nil || response.Usage == nil {
		return nil, false
	}
	return response.Usage, true
}

func hasNonNegativeIntegerField(body []byte, path string) bool {
	value := gjson.GetBytes(body, path)
	if !value.Exists() || value.Type != gjson.Number || value.Int() < 0 {
		return false
	}
	return value.Float() == float64(value.Int())
}

func hasNonNegativeIntegerFieldString(body string, path string) bool {
	value := gjson.Get(body, path)
	if !value.Exists() || value.Type != gjson.Number || value.Int() < 0 {
		return false
	}
	return value.Float() == float64(value.Int())
}

func finalizeIncompleteDirectStreamBilling(c *gin.Context, info *relaycommon.RelayInfo) {
	if info == nil {
		return
	}
	reservedQuota := info.FinalPreConsumedQuota
	if info.Billing != nil {
		reservedQuota = info.Billing.GetPreConsumedQuota()
	}
	if err := service.SettleBilling(c, info, reservedQuota); err != nil {
		logger.LogError(c, fmt.Sprintf("failed to finalize incomplete direct stream at reserved quota %d: %s", reservedQuota, err.Error()))
	}
}

func usageFromClaudeUsage(upstream *dto.ClaudeUsage) *dto.Usage {
	if upstream == nil {
		return &dto.Usage{UsageSemantic: "anthropic"}
	}
	usage := &dto.Usage{
		PromptTokens:     upstream.InputTokens,
		CompletionTokens: upstream.OutputTokens,
		TotalTokens:      upstream.InputTokens + upstream.OutputTokens,
		UsageSemantic:    "anthropic",
	}
	usage.PromptTokensDetails.CachedTokens = upstream.CacheReadInputTokens
	usage.PromptTokensDetails.CachedCreationTokens = upstream.CacheCreationInputTokens
	usage.ClaudeCacheCreation5mTokens = upstream.GetCacheCreation5mTokens()
	usage.ClaudeCacheCreation1hTokens = upstream.GetCacheCreation1hTokens()
	usage.BillingUsage = dto.NewClaudeMessagesBillingUsage(upstream)
	return usage
}

func recordClaudeServerToolUsage(c *gin.Context, usage *dto.ClaudeUsage) {
	if c == nil || usage == nil || usage.ServerToolUse == nil {
		return
	}
	if usage.ServerToolUse.WebSearchRequests > 0 {
		c.Set("claude_web_search_requests", usage.ServerToolUse.WebSearchRequests)
	}
}

type directStreamUsageObserver struct {
	info               *relaycommon.RelayInfo
	claudeInfo         *ClaudeResponseInfo
	openAI             *dto.Usage
	sawPromptUsage     bool
	sawCompletionUsage bool
	webSearchRequests  int
}

func newDirectStreamUsageObserver(info *relaycommon.RelayInfo) *directStreamUsageObserver {
	observer := &directStreamUsageObserver{info: info, openAI: &dto.Usage{}}
	if info != nil && info.RelayFormat == types.RelayFormatClaude {
		observer.claudeInfo = &ClaudeResponseInfo{
			Model:        info.UpstreamModelName,
			ResponseText: strings.Builder{},
			Usage:        &dto.Usage{},
		}
	}
	return observer
}

func (o *directStreamUsageObserver) Observe(line string) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "data:") {
		return
	}
	payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if payload == "" || payload == "[DONE]" {
		return
	}
	if o.info != nil {
		o.info.SetFirstResponseTime()
		o.info.ReceivedResponseCount++
	}

	if o.claudeInfo != nil {
		var response dto.ClaudeResponse
		if err := common.UnmarshalJsonStr(payload, &response); err == nil {
			if response.Type == "message_start" && response.Message != nil && response.Message.Usage != nil &&
				hasNonNegativeIntegerFieldString(payload, "message.usage.input_tokens") {
				o.sawPromptUsage = true
			}
			if response.Type == "message_delta" && response.Usage != nil &&
				hasNonNegativeIntegerFieldString(payload, "usage.output_tokens") {
				o.sawCompletionUsage = true
			}
			if response.Usage != nil {
				if response.Usage.ServerToolUse != nil && response.Usage.ServerToolUse.WebSearchRequests > 0 {
					o.webSearchRequests = response.Usage.ServerToolUse.WebSearchRequests
				}
			}
			FormatClaudeResponseInfo(&response, nil, o.claudeInfo)
		}
		return
	}

	var response dto.ChatCompletionsStreamResponse
	if err := common.UnmarshalJsonStr(payload, &response); err != nil {
		return
	}
	if response.Usage != nil &&
		hasNonNegativeIntegerFieldString(payload, "usage.prompt_tokens") &&
		hasNonNegativeIntegerFieldString(payload, "usage.completion_tokens") {
		o.sawPromptUsage = true
		o.sawCompletionUsage = true
		*o.openAI = *response.Usage
	}
}

func (o *directStreamUsageObserver) Usage() (*dto.Usage, bool) {
	if !o.sawPromptUsage || !o.sawCompletionUsage {
		return nil, false
	}
	if o.claudeInfo != nil {
		if o.claudeInfo.Usage != nil && o.claudeInfo.Usage.BillingUsage == nil {
			o.claudeInfo.Usage.BillingUsage = dto.NewClaudeMessagesBillingUsage(buildMessageDeltaPatchUsage(nil, o.claudeInfo))
		}
		return o.claudeInfo.Usage, true
	}
	return o.openAI, true
}
