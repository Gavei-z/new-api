package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestEpayPaymentEndpointsFailClosedWhenGatewayIsUnconfigured(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
	})

	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}}

	testCases := []struct {
		name    string
		body    string
		handler gin.HandlerFunc
	}{
		{
			name:    "wallet top-up",
			body:    `{"amount":2,"payment_method":"alipay"}`,
			handler: RequestEpay,
		},
		{
			name:    "subscription purchase",
			body:    `{"plan_id":1,"payment_method":"alipay"}`,
			handler: SubscriptionRequestEpay,
		},
	}

	gin.SetMode(gin.TestMode)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(testCase.body),
			)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = request

			testCase.handler(context)

			assert.Equal(t, http.StatusOK, response.Code)
			assert.Contains(t, response.Body.String(), "支付通道未配置")
		})
	}
}
