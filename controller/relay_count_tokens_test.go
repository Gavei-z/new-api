package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayClaudeCountTokensForwardsSamePathBodyHeadersAndResponse(t *testing.T) {
	service.InitHttpClient()
	requestBody := `{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"JVBERi0="}},{"type":"text","text":"count"}]}],"thinking":{"type":"adaptive"},"future_field":{"kept":true}}`
	responseBody := "{ \"input_tokens\" : 321, \"future_usage\" : {\"kept\":true} }\n"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages/count_tokens", r.URL.Path)
		require.Equal(t, "provider-key", r.Header.Get("x-api-key"))
		require.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		require.Equal(t, "pdfs-2024-09-25", r.Header.Get("anthropic-beta"))
		require.Equal(t, "python-httpx/veridrop", r.Header.Get("User-Agent"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, requestBody, string(body))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Upstream", "cfjwl")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer upstream.Close()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("anthropic-version", "2023-06-01")
	c.Request.Header.Set("anthropic-beta", "pdfs-2024-09-25")
	c.Request.Header.Set("User-Agent", "python-httpx/veridrop")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "claude-opus-5")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "provider-key")
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, dto.ChannelOtherSettings{AnthropicDirectPassthroughEnabled: true})

	RelayClaudeCountTokens(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, responseBody, recorder.Body.String())
	require.Equal(t, "cfjwl", recorder.Header().Get("X-Upstream"))
}
