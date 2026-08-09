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

// GetStripeWeChatCNYPriceId returns the Price used only for WeChat Pay.
// The Price can be a dedicated CNY Price or the same multi-currency Price ID
// used by standard Checkout, provided it has a CNY currency option.
func GetStripeWeChatCNYPriceId() string {
	return strings.TrimSpace(os.Getenv("STRIPE_WECHAT_CNY_PRICE_ID"))
}

// GetStripeWeChatCNYUnitAmountMinor returns the CNY minor-unit charge for one
// USD of UniRouters wallet credit. Invalid, missing, and non-positive values
// are deliberately unavailable rather than falling back to an exchange rate.
func GetStripeWeChatCNYUnitAmountMinor() int64 {
	unitAmountMinor, err := strconv.ParseInt(
		strings.TrimSpace(os.Getenv("STRIPE_WECHAT_CNY_UNIT_AMOUNT_MINOR")),
		10,
		64,
	)
	if err != nil || unitAmountMinor <= 0 {
		return 0
	}
	return unitAmountMinor
}

// IsStripeWeChatPayEnabled is an explicit deployment-level kill switch for
// the WeChat-only Checkout variant. Stripe credentials alone do not prove
// that the connected account has an active WeChat Pay capability.
func IsStripeWeChatPayEnabled() bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("STRIPE_WECHAT_PAY_ENABLED")))
	return err == nil && enabled
}
