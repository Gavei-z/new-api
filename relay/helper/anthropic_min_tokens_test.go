package helper

import (
	"bytes"
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
	"github.com/tidwall/gjson"
)

func TestAnthropicMinimumMaxTokensNormalizesExactModels(t *testing.T) {
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
		parse func(*gin.Context) (*uint, error)
	}{
		{
			name: "native messages",
			path: "/v1/messages",
			parse: func(c *gin.Context) (*uint, error) {
				request, err := GetAndValidateClaudeRequest(c)
				if err != nil {
					return nil, err
				}
				return request.MaxTokens, nil
			},
		},
		{
			name: "OpenAI chat conversion",
			path: "/v1/chat/completions",
			parse: func(c *gin.Context) (*uint, error) {
				request, err := GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
				if err != nil {
					return nil, err
				}
				return request.MaxTokens, nil
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

				maxTokens, err := format.parse(c)
				require.NoError(t, err)
				require.NotNil(t, maxTokens)
				require.Equal(t, AnthropicMinimumMaxTokens, *maxTokens)
			})
		}
	}
}

func TestAnthropicMinimumMaxTokensNormalizationBoundariesAndScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                  string
		path                  string
		channelType           int
		model                 string
		field                 string
		value                 string
		wantMaxPresent        bool
		wantMax               uint
		wantLegacyPresent     bool
		wantLegacy            uint
		wantCompletionPresent bool
		wantCompletion        uint
	}{
		{name: "native omitted accepted", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6"},
		{name: "native zero preserved", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "0", wantMaxPresent: true},
		{name: "native one normalized", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "1", wantMaxPresent: true, wantMax: 1024},
		{name: "native legacy normalized and canonicalized", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens_to_sample", value: "1", wantMaxPresent: true, wantMax: 1024},
		{name: "native minimum preserved", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "1024", wantMaxPresent: true, wantMax: 1024},
		{name: "native legacy outside scope unchanged", path: "/v1/messages", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-5", field: "max_tokens_to_sample", value: "1023", wantLegacyPresent: true, wantLegacy: 1023},
		{name: "chat omitted accepted", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6"},
		{name: "chat zero preserved", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "0", wantMaxPresent: true},
		{name: "chat one normalized", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_tokens", value: "1", wantMaxPresent: true, wantMax: 1024},
		{name: "chat completion normalized and canonicalized", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-6", field: "max_completion_tokens", value: "1023", wantMaxPresent: true, wantMax: 1024},
		{name: "other channel unchanged", path: "/v1/chat/completions", channelType: constant.ChannelTypeOpenAI, model: "claude-opus-4-6", field: "max_completion_tokens", value: "1023", wantCompletionPresent: true, wantCompletion: 1023},
		{name: "non exact alias unchanged", path: "/v1/chat/completions", channelType: constant.ChannelTypeAnthropic, model: "claude-opus-4-5", field: "max_tokens", value: "1023", wantMaxPresent: true, wantMax: 1023},
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

			var maxTokens *uint
			var legacyMaxTokens *uint
			var maxCompletionTokens *uint
			if test.path == "/v1/messages" {
				request, err := GetAndValidateClaudeRequest(c)
				require.NoError(t, err)
				maxTokens = request.MaxTokens
				legacyMaxTokens = request.MaxTokensToSample
			} else {
				request, err := GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
				require.NoError(t, err)
				maxTokens = request.MaxTokens
				maxCompletionTokens = request.MaxCompletionTokens
			}

			require.Equal(t, test.wantMaxPresent, maxTokens != nil)
			if maxTokens != nil {
				require.Equal(t, test.wantMax, *maxTokens)
			}
			require.Equal(t, test.wantLegacyPresent, legacyMaxTokens != nil)
			if legacyMaxTokens != nil {
				require.Equal(t, test.wantLegacy, *legacyMaxTokens)
			}
			require.Equal(t, test.wantCompletionPresent, maxCompletionTokens != nil)
			if maxCompletionTokens != nil {
				require.Equal(t, test.wantCompletion, *maxCompletionTokens)
			}
		})
	}
}

func TestAnthropicMinimumMaxTokensCanonicalizesEffectiveLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("native max_tokens takes precedence", func(t *testing.T) {
		body := `{"model":"claude-sonnet-5","messages":[{"role":"user","content":"hi"}],"max_tokens":2048,"max_tokens_to_sample":1}`
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)

		request, err := GetAndValidateClaudeRequest(c)
		require.NoError(t, err)
		require.Equal(t, uint(2048), *request.MaxTokens)
		require.Nil(t, request.MaxTokensToSample)
	})

	t.Run("OpenAI max_completion_tokens takes precedence", func(t *testing.T) {
		body := `{"model":"claude-sonnet-5","messages":[{"role":"user","content":"hi"}],"max_tokens":100000,"max_completion_tokens":1}`
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)

		request, err := GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
		require.NoError(t, err)
		require.Equal(t, AnthropicMinimumMaxTokens, *request.MaxTokens)
		require.Nil(t, request.MaxCompletionTokens)
		require.Equal(t, int(AnthropicMinimumMaxTokens), request.GetTokenCountMeta().MaxTokens)
	})
}

func TestNormalizeAnthropicMinimumMaxTokensJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                  string
		channelType           int
		body                  string
		mappedModel           string
		inputFormat           types.RelayFormat
		wantMaxTokens         int64
		wantModel             string
		wantChanged           bool
		wantCompletionPresent bool
		wantLegacyPresent     bool
	}{
		{name: "small canonical limit", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-sonnet-5","max_tokens":1}`, wantMaxTokens: 1024, wantModel: "claude-sonnet-5", wantChanged: true},
		{name: "native canonical limit takes precedence", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-sonnet-5","max_tokens":2048,"max_completion_tokens":1}`, wantMaxTokens: 2048, wantModel: "claude-sonnet-5", wantChanged: true},
		{name: "OpenAI completion limit takes precedence", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-sonnet-5","max_tokens":2048,"max_completion_tokens":1}`, inputFormat: types.RelayFormatOpenAI, wantMaxTokens: 1024, wantModel: "claude-sonnet-5", wantChanged: true},
		{name: "legacy limit becomes canonical", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-sonnet-5","max_tokens_to_sample":2048}`, wantMaxTokens: 2048, wantModel: "claude-sonnet-5", wantChanged: true},
		{name: "missing limit receives minimum", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-sonnet-5"}`, wantMaxTokens: 1024, wantChanged: true},
		{name: "zero limit receives minimum", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-sonnet-5","max_tokens":0}`, wantMaxTokens: 1024, wantChanged: true},
		{name: "mapped model rewrites raw alias", channelType: constant.ChannelTypeAnthropic, body: `{"model":"public-sonnet-alias","max_tokens":1}`, mappedModel: "claude-sonnet-5", wantMaxTokens: 1024, wantModel: "claude-sonnet-5", wantChanged: true},
		{name: "valid limit unchanged", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-sonnet-5","max_tokens":2048}`, wantMaxTokens: 2048},
		{name: "other channel unchanged", channelType: constant.ChannelTypeOpenAI, body: `{"model":"claude-sonnet-5","max_tokens":1}`, wantMaxTokens: 1},
		{name: "other model unchanged", channelType: constant.ChannelTypeAnthropic, body: `{"model":"claude-opus-4-5","max_tokens":1}`, wantMaxTokens: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			common.SetContextKey(c, constant.ContextKeyChannelType, test.channelType)

			result, changed, err := NormalizeAnthropicMinimumMaxTokensJSON(c, []byte(test.body), test.mappedModel, test.inputFormat)
			require.NoError(t, err)
			require.Equal(t, test.wantChanged, changed)
			require.Equal(t, test.wantMaxTokens, gjson.GetBytes(result, "max_tokens").Int())
			if test.wantModel != "" {
				require.Equal(t, test.wantModel, gjson.GetBytes(result, "model").String())
			}
			require.Equal(t, test.wantCompletionPresent, gjson.GetBytes(result, "max_completion_tokens").Exists())
			require.Equal(t, test.wantLegacyPresent, gjson.GetBytes(result, "max_tokens_to_sample").Exists())
		})
	}
}
