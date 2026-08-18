package helper

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAnthropicMinimumMaxTokensExactModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	models := []string{
		"claude-opus-4-5-20251101",
		"claude-opus-4-6",
		"claude-opus-4-7",
		"claude-opus-4-8",
		"claude-opus-5",
		"claude-sonnet-4-5-20250929",
		"claude-sonnet-4-6",
		"claude-sonnet-5",
	}
	formats := []struct {
		name  string
		path  string
		parse func(*gin.Context) error
	}{
		{
			name: "native messages",
			path: "/v1/messages",
			parse: func(c *gin.Context) error {
				_, err := GetAndValidateClaudeRequest(c)
				return err
			},
		},
		{
			name: "OpenAI chat conversion",
			path: "/v1/chat/completions",
			parse: func(c *gin.Context) error {
				_, err := GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
				return err
			},
		},
	}

	for _, model := range models {
		for _, format := range formats {
			t.Run(model+"/"+format.name, func(t *testing.T) {
				body := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hi"}],"max_tokens":1023}`, model)
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, format.path, bytes.NewBufferString(body))
				c.Request.Header.Set("Content-Type", "application/json")
				common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)

				err := format.parse(c)
				require.Error(t, err)
				require.ErrorContains(t, err, "max_tokens must be 0 or at least 1024")

				var apiErr *types.NewAPIError
				require.True(t, errors.As(err, &apiErr))
				require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
				require.Equal(t, types.ErrorCodeInvalidRequest, apiErr.GetErrorCode())
				require.True(t, types.IsSkipRetryError(apiErr))
			})
		}
	}
}

func TestAnthropicMinimumMaxTokensBoundariesAndScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		path        string
		channelType int
		model       string
		field       string
		value       string
		wantError   bool
	}{
		{name: "native omitted accepted", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6"},
		{name: "native zero accepted", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "0"},
		{name: "native one rejected", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "1", wantError: true},
		{name: "native legacy one rejected", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens_to_sample", value: "1", wantError: true},
		{name: "native minimum accepted", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "1024"},
		{name: "chat omitted accepted", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6"},
		{name: "chat zero accepted", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "0"},
		{name: "chat one rejected", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "1", wantError: true},
		{name: "chat minimum accepted", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "1024"},
		{name: "chat max completion rejected", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_completion_tokens", value: "1023", wantError: true},
		{name: "other channel accepted", path: "/v1/chat/completions", channelType: constant.ChannelTypeOpenAI, model: "claude-opus-4-6", field: "max_tokens", value: "1023"},
		{name: "non exact alias accepted", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-5", field: "max_tokens", value: "1023"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			optionalField := ""
			if test.field != "" {
				optionalField = fmt.Sprintf(`,%q:%s`, test.field, test.value)
			}
			body := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hi"}]%s}`, test.model, optionalField)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, test.path, bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(c, constant.ContextKeyChannelType, test.channelType)

			var err error
			if test.path == "/v1/messages" {
				_, err = GetAndValidateClaudeRequest(c)
			} else {
				_, err = GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
			}

			if test.wantError {
				require.ErrorContains(t, err, "max_tokens must be 0 or at least 1024")
				return
			}
			require.NoError(t, err)
		})
	}
}
