package setting

import (
	"os"
	"strconv"
	"strings"
)

var StripeApiSecret = ""
var StripeWebhookSecret = ""
var StripePriceId = ""
var StripeUnitPrice = 8.0
var StripeMinTopUp = 1
var StripePromotionCodesEnabled = false

func stripeEnvironmentValue(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

// GetStripeApiSecret returns the environment-managed secret when present.
// Database options remain a backward-compatible fallback, but production
// deployments should keep payment credentials out of the options table.
func GetStripeApiSecret() string {
	return stripeEnvironmentValue("STRIPE_API_SECRET", StripeApiSecret)
}

func GetStripeWebhookSecret() string {
	return stripeEnvironmentValue("STRIPE_WEBHOOK_SECRET", StripeWebhookSecret)
}

func GetStripePriceId() string {
	return stripeEnvironmentValue("STRIPE_PRICE_ID", StripePriceId)
}

// IsStripeWeChatPayEnabled is an explicit deployment-level kill switch for
// the WeChat-only Checkout variant. Stripe credentials alone do not prove
// that the connected account has an active WeChat Pay capability.
func IsStripeWeChatPayEnabled() bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("STRIPE_WECHAT_PAY_ENABLED")))
	return err == nil && enabled
}
