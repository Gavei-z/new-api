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

func TestStripeWeChatCNYConfigurationIsEnvironmentManagedAndStrict(t *testing.T) {
	t.Setenv("STRIPE_WECHAT_CNY_PRICE_ID", " price_cny_or_multicurrency ")
	t.Setenv("STRIPE_WECHAT_CNY_UNIT_AMOUNT_MINOR", " 725 ")

	assert.Equal(t, "price_cny_or_multicurrency", GetStripeWeChatCNYPriceId())
	assert.EqualValues(t, 725, GetStripeWeChatCNYUnitAmountMinor())

	for _, invalid := range []string{"", "0", "-1", "7.25", "not-a-number", "9223372036854775808"} {
		t.Run(invalid, func(t *testing.T) {
			t.Setenv("STRIPE_WECHAT_CNY_UNIT_AMOUNT_MINOR", invalid)
			assert.Zero(t, GetStripeWeChatCNYUnitAmountMinor())
		})
	}
}
