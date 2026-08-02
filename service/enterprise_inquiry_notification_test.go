package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnterpriseInquiryEmailIncludesContactAndEscapesUntrustedContent(t *testing.T) {
	inquiry := model.EnterpriseInquiry{
		Id:          42,
		CompanyName: `<script>alert("company")</script>`,
		ContactName: "Alice",
		Email:       "alice@example.com",
		Phone:       "+1 555 0100",
		WeChat:      "alice-wechat",
		TeamSize:    "11-50",
		Message:     `<img src=x onerror="alert(1)">`,
		CreatedAt:   1_786_000_000,
	}

	content, err := buildEnterpriseInquiryEmail(inquiry)

	require.NoError(t, err)
	assert.Contains(t, content, "alice@example.com")
	assert.Contains(t, content, "555 0100")
	assert.Contains(t, content, "&lt;script&gt;")
	assert.Contains(t, content, "&lt;img")
	assert.NotContains(t, content, "<script>")
	assert.NotContains(t, content, "<img src=x")
}

func TestEnterpriseInquiryNotificationReceiverUsesTrimmedEnvironmentValue(t *testing.T) {
	t.Setenv(EnterpriseInquiryNotificationEmailEnv, "  admin@example.com  ")

	assert.Equal(t, "admin@example.com", EnterpriseInquiryNotificationReceiver())
}

func TestSendEnterpriseInquiryNotificationRejectsInvalidReceiverBeforeSMTP(t *testing.T) {
	err := SendEnterpriseInquiryNotification(model.EnterpriseInquiry{}, "not-an-email")

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid enterprise inquiry notification receiver"))
}
