package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayReturnsBadRequestForAnthropicMinimumMaxTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		path        string
		relayFormat types.RelayFormat
	}{
		{name: "native messages", path: "/v1/messages", relayFormat: types.RelayFormatClaude},
		{name: "OpenAI chat conversion", path: "/v1/chat/completions", relayFormat: types.RelayFormatOpenAI},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(
				http.MethodPost,
				test.path,
				bytes.NewBufferString(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"hi"}],"max_tokens":1023}`),
			)
			c.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)

			Relay(c, test.relayFormat)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "max_tokens must be 0 or at least 1024")
		})
	}
}
