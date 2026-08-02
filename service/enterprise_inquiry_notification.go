package service

import (
	"bytes"
	"fmt"
	"html/template"
	"net/mail"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const EnterpriseInquiryNotificationEmailEnv = "ENTERPRISE_INQUIRY_NOTIFICATION_EMAIL"

type enterpriseInquiryEmailData struct {
	Id          int
	CompanyName string
	ContactName string
	Email       string
	Phone       string
	WeChat      string
	TeamSize    string
	Message     string
	SubmittedAt string
}

var enterpriseInquiryEmailTemplate = template.Must(template.New("enterprise-inquiry").Parse(`
<div style="font-family:Arial,sans-serif;line-height:1.6;color:#1f2937;max-width:680px;margin:0 auto">
  <h2 style="margin-bottom:8px">新的企业询价 / New enterprise inquiry</h2>
  <p style="margin-top:0;color:#6b7280">询价编号 / Inquiry ID: #{{.Id}}</p>
  <table style="width:100%;border-collapse:collapse">
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:600">公司 / Company</td><td style="padding:8px;border:1px solid #e5e7eb">{{.CompanyName}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:600">联系人 / Contact</td><td style="padding:8px;border:1px solid #e5e7eb">{{.ContactName}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:600">邮箱 / Email</td><td style="padding:8px;border:1px solid #e5e7eb">{{.Email}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:600">电话 / Phone</td><td style="padding:8px;border:1px solid #e5e7eb">{{.Phone}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:600">微信 / WeChat</td><td style="padding:8px;border:1px solid #e5e7eb">{{.WeChat}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:600">团队规模 / Team size</td><td style="padding:8px;border:1px solid #e5e7eb">{{.TeamSize}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:600">提交时间 / Submitted at</td><td style="padding:8px;border:1px solid #e5e7eb">{{.SubmittedAt}}</td></tr>
  </table>
  <h3 style="margin-bottom:6px">备注 / Notes</h3>
  <div style="padding:12px;background:#f3f4f6;border-radius:6px;white-space:pre-wrap">{{.Message}}</div>
  <p style="margin-top:20px;color:#6b7280">请登录 UniRouters 管理后台，在“团队管理 → 企业询价”中更新处理状态。</p>
</div>
`))

func EnterpriseInquiryNotificationReceiver() string {
	return strings.TrimSpace(common.GetEnvOrDefaultString(EnterpriseInquiryNotificationEmailEnv, ""))
}

func SendEnterpriseInquiryNotification(inquiry model.EnterpriseInquiry, receiver string) error {
	receiver = strings.TrimSpace(receiver)
	address, err := mail.ParseAddress(receiver)
	if err != nil || address.Address != receiver {
		return fmt.Errorf("invalid enterprise inquiry notification receiver")
	}

	content, err := buildEnterpriseInquiryEmail(inquiry)
	if err != nil {
		return fmt.Errorf("build enterprise inquiry notification: %w", err)
	}
	subject := fmt.Sprintf("[%s] 新企业询价 / New enterprise inquiry #%d", common.SystemName, inquiry.Id)
	if err := common.SendEmail(subject, receiver, content); err != nil {
		return fmt.Errorf("send enterprise inquiry notification: %w", err)
	}
	return nil
}

func buildEnterpriseInquiryEmail(inquiry model.EnterpriseInquiry) (string, error) {
	displayValue := func(value string) string {
		if strings.TrimSpace(value) == "" {
			return "—"
		}
		return value
	}
	submittedAt := time.Unix(inquiry.CreatedAt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	data := enterpriseInquiryEmailData{
		Id:          inquiry.Id,
		CompanyName: displayValue(inquiry.CompanyName),
		ContactName: displayValue(inquiry.ContactName),
		Email:       displayValue(inquiry.Email),
		Phone:       displayValue(inquiry.Phone),
		WeChat:      displayValue(inquiry.WeChat),
		TeamSize:    displayValue(inquiry.TeamSize),
		Message:     displayValue(inquiry.Message),
		SubmittedAt: submittedAt,
	}
	var content bytes.Buffer
	if err := enterpriseInquiryEmailTemplate.Execute(&content, data); err != nil {
		return "", err
	}
	return content.String(), nil
}
