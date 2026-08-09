package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripeEnvironmentCredentialsOverrideDatabaseFallback(t *testing.T) {
	originalAPISecret := StripeApiSecret
	originalWebhookSecret := StripeWebhookSecret
	originalPriceID := StripePriceId
	t.Cleanup(func() {
		StripeApiSecret = originalAPISecret
		StripeWebhookSecret = originalWebhookSecret
		StripePriceId = originalPriceID
	})

	StripeApiSecret = "database-api-secret"
	StripeWebhookSecret = "database-webhook-secret"
	StripePriceId = "database-price"
	t.Setenv("STRIPE_API_SECRET", " environment-api-secret ")
	t.Setenv("STRIPE_WEBHOOK_SECRET", " environment-webhook-secret ")
	t.Setenv("STRIPE_PRICE_ID", " environment-price ")

	assert.Equal(t, "environment-api-secret", GetStripeApiSecret())
	assert.Equal(t, "environment-webhook-secret", GetStripeWebhookSecret())
	assert.Equal(t, "environment-price", GetStripePriceId())
}

func TestStripeCredentialGettersFallBackToDatabaseOptions(t *testing.T) {
	originalAPISecret := StripeApiSecret
	originalWebhookSecret := StripeWebhookSecret
	originalPriceID := StripePriceId
	t.Cleanup(func() {
		StripeApiSecret = originalAPISecret
		StripeWebhookSecret = originalWebhookSecret
		StripePriceId = originalPriceID
	})

	StripeApiSecret = " database-api-secret "
	StripeWebhookSecret = " database-webhook-secret "
	StripePriceId = " database-price "
	t.Setenv("STRIPE_API_SECRET", "")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	t.Setenv("STRIPE_PRICE_ID", "")

	assert.Equal(t, "database-api-secret", GetStripeApiSecret())
	assert.Equal(t, "database-webhook-secret", GetStripeWebhookSecret())
	assert.Equal(t, "database-price", GetStripePriceId())
}

func TestStripeWeChatPayRequiresExplicitEnvironmentEnablement(t *testing.T) {
	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "")
	assert.False(t, IsStripeWeChatPayEnabled())

	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "true")
	assert.True(t, IsStripeWeChatPayEnabled())

	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "invalid")
	assert.False(t, IsStripeWeChatPayEnabled())
}
