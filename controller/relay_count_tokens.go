package controller

import (
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// RelayClaudeCountTokens forwards Anthropic's free token-counting endpoint to
// a same-protocol upstream. It intentionally skips generation pre-consumption
// and settlement, while retaining the normal token authentication, model ACL,
// request-rate limiting and channel selection middleware.
func RelayClaudeCountTokens(c *gin.Context) {
	requestID := c.GetString(common.RequestIdKey)
	var newAPIError *types.NewAPIError
	defer func() {
		if newAPIError == nil {
			return
		}
		logger.LogError(c, fmt.Sprintf("Claude count_tokens relay error: %s", common.LocalLogPreview(newAPIError.Error())))
		newAPIError.SetMessage(common.MessageWithRequestId(newAPIError.Error(), requestID))
		if c.Writer.Written() {
			return
		}
		c.JSON(newAPIError.StatusCode, gin.H{
			"type":  "error",
			"error": newAPIError.ToClaudeError(),
		})
	}()

	info := relaycommon.GenRelayInfoClaude(c, nil)
	info.InitChannelMeta(c)
	if !relaycommon.IsAnthropicDirectPassthrough(info) {
		newAPIError = types.NewErrorWithStatusCode(
			fmt.Errorf("selected channel does not support direct Anthropic count_tokens passthrough"),
			types.ErrorCodeInvalidApiType,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
		return
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		status := http.StatusBadRequest
		if common.IsRequestBodyTooLargeError(err) {
			status = http.StatusRequestEntityTooLarge
		}
		newAPIError = types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, status, types.ErrOptionWithSkipRetry())
		return
	}
	info.UpstreamRequestBodySize = storage.Size()

	adaptor := relay.GetAdaptor(info.ApiType)
	if adaptor == nil {
		newAPIError = types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
		return
	}
	adaptor.Init(info)
	response, err := adaptor.DoRequest(c, info, common.ReaderOnly(storage))
	if err != nil {
		newAPIError = types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
		return
	}
	httpResponse, ok := response.(*http.Response)
	if !ok || httpResponse == nil {
		newAPIError = types.NewError(fmt.Errorf("invalid count_tokens upstream response: %T", response), types.ErrorCodeBadResponse)
		return
	}
	if httpResponse.StatusCode != http.StatusOK {
		newAPIError = claude.DirectPassthroughErrorHandler(c, httpResponse)
		return
	}
	defer service.CloseResponseBodyGracefully(httpResponse)

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		newAPIError = types.NewError(err, types.ErrorCodeReadResponseBodyFailed)
		return
	}
	service.IOCopyBytesGracefully(c, httpResponse, body)
}
