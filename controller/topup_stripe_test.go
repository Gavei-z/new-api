package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"gorm.io/gorm"
)

func setupStripeWebhookControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	dsn := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared",
		strings.ReplaceAll(t.Name(), "/", "_"),
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Team{},
		&model.TeamMember{},
		&model.TopUp{},
		&model.StripeFXDailyRate{},
		&model.StripeFXRateHead{},
		&model.StripePaymentIntentLink{},
		&model.StripeAdjustmentInbox{},
	))
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
	return db
}

func insertStripeFXControllerTestRate(t *testing.T) *model.StripeFXDailyRate {
	t.Helper()
	now := time.Now()
	rate, err := model.CreateStripeFXDailyRate(&model.StripeFXDailyRate{
		PricingDate:       model.StripeFXPricingDate(now),
		Source:            model.StripeFXSourceBOCSpotSelling,
		BaseCurrency:      model.StripeFXBaseCurrencyUSD,
		QuoteCurrency:     model.StripeFXQuoteCurrencyCNY,
		RateE4:            67_655,
		SourcePublishedAt: now.Add(-time.Minute).Unix(),
		FetchedAt:         now.Unix(),
		SourceURL:         model.StripeFXBOCSourceURL,
	})
	require.NoError(t, err)
	return rate
}

func TestCreateStripeCheckoutUsesOneDollarPriceQuantityAndIdempotency(t *testing.T) {
	t.Setenv("STRIPE_API_SECRET", "")
	t.Setenv("STRIPE_PRICE_ID", "")
	originalSecret := setting.StripeApiSecret
	originalPrice := setting.StripePriceId
	originalNew := stripeCheckoutSessionNew
	t.Cleanup(func() {
		setting.StripeApiSecret = originalSecret
		setting.StripePriceId = originalPrice
		stripeCheckoutSessionNew = originalNew
	})
	setting.StripeApiSecret = "rk_test_topup"
	setting.StripePriceId = "price_one_usd"

	var captured *stripe.CheckoutSessionParams
	stripeCheckoutSessionNew = func(params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		captured = params
		return &stripe.CheckoutSession{
			ID:          "cs_test_created",
			URL:         "https://checkout.stripe.test/session",
			AmountTotal: 50000,
			Currency:    stripe.CurrencyUSD,
		}, nil
	}
	terms, err := resolveStripeCheckoutTerms(stripeCheckoutMethodStandard, 500, "", false)
	require.NoError(t, err)

	result, err := createStripeCheckoutSession(
		"ref_order",
		"",
		"payer@example.test",
		500,
		"https://unirouters.cc/wallet?stripe=success",
		"https://unirouters.cc/wallet?stripe=cancel",
		terms,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, captured)
	require.Len(t, captured.LineItems, 1)
	require.NotNil(t, captured.LineItems[0].Price)
	require.NotNil(t, captured.LineItems[0].Quantity)
	assert.Equal(t, "price_one_usd", *captured.LineItems[0].Price)
	assert.EqualValues(t, 500, *captured.LineItems[0].Quantity)
	require.NotNil(t, captured.IdempotencyKey)
	assert.Equal(t, "stripe-topup:ref_order", *captured.IdempotencyKey)
	require.NotNil(t, captured.Extra)
	assert.Equal(t, "false", captured.Extra.Values.Get("managed_payments[enabled]"))
	require.NotNil(t, captured.AllowPromotionCodes)
	assert.False(t, *captured.AllowPromotionCodes)
	assert.Equal(t, "ref_order", captured.Metadata["topup_reference"])
	assert.Equal(t, stripeCheckoutMethodStandard, captured.Metadata["checkout_method"])
	require.NotNil(t, captured.PaymentIntentData)
	assert.Equal(t, "ref_order", captured.PaymentIntentData.Metadata["topup_reference"])
	assert.Equal(t, stripeCheckoutMethodStandard, captured.PaymentIntentData.Metadata["checkout_method"])
	assert.Nil(t, captured.PaymentMethodTypes)
	assert.Nil(t, captured.PaymentMethodOptions)
	assert.Nil(t, captured.AdaptivePricing)
	assert.Nil(t, captured.Currency)
}

func TestCreateStripeCheckoutUsesDynamicCNYPriceForWeChatPayOnlyWhenRequested(t *testing.T) {
	setupStripeWebhookControllerTestDB(t)
	rate := insertStripeFXControllerTestRate(t)
	t.Setenv("STRIPE_API_SECRET", "")
	t.Setenv("STRIPE_PRICE_ID", "")
	t.Setenv("STRIPE_PRODUCT_ID", "prod_wallet_credit")
	originalSecret := setting.StripeApiSecret
	originalPrice := setting.StripePriceId
	originalNew := stripeCheckoutSessionNew
	t.Cleanup(func() {
		setting.StripeApiSecret = originalSecret
		setting.StripePriceId = originalPrice
		stripeCheckoutSessionNew = originalNew
	})
	setting.StripeApiSecret = "rk_test_wechat"
	setting.StripePriceId = "price_one_usd"

	var captured *stripe.CheckoutSessionParams
	stripeCheckoutSessionNew = func(params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		captured = params
		return &stripe.CheckoutSession{
			ID:          "cs_test_wechat",
			URL:         "https://checkout.stripe.test/wechat",
			AmountTotal: 1354,
			Currency:    stripe.CurrencyCNY,
		}, nil
	}
	terms, err := resolveStripeCheckoutTerms(stripeCheckoutMethodWeChatPay, 2, "", false)
	require.NoError(t, err)
	assert.Equal(t, rate.QuoteVersion(), terms.FXQuoteVersion)

	_, err = createStripeCheckoutSession(
		"ref_wechat",
		"",
		"payer@example.test",
		2,
		"https://unirouters.cc/wallet?stripe=success",
		"https://unirouters.cc/wallet?stripe=cancel",
		terms,
	)
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Len(t, captured.LineItems, 1)
	assert.Nil(t, captured.LineItems[0].Price)
	require.NotNil(t, captured.LineItems[0].PriceData)
	require.NotNil(t, captured.LineItems[0].PriceData.Product)
	require.NotNil(t, captured.LineItems[0].PriceData.Currency)
	require.NotNil(t, captured.LineItems[0].PriceData.UnitAmount)
	assert.Equal(t, "prod_wallet_credit", *captured.LineItems[0].PriceData.Product)
	assert.Equal(t, "cny", *captured.LineItems[0].PriceData.Currency)
	assert.EqualValues(t, 1354, *captured.LineItems[0].PriceData.UnitAmount)
	assert.EqualValues(t, 1, *captured.LineItems[0].Quantity)
	require.NotNil(t, captured.Currency)
	assert.Equal(t, "cny", *captured.Currency)
	require.Len(t, captured.PaymentMethodTypes, 1)
	require.NotNil(t, captured.PaymentMethodTypes[0])
	assert.Equal(t, string(stripe.PaymentMethodTypeWeChatPay), *captured.PaymentMethodTypes[0])
	require.NotNil(t, captured.PaymentMethodOptions)
	require.NotNil(t, captured.PaymentMethodOptions.WeChatPay)
	require.NotNil(t, captured.PaymentMethodOptions.WeChatPay.Client)
	assert.Equal(t, string(stripe.PaymentIntentPaymentMethodOptionsWeChatPayClientWeb), *captured.PaymentMethodOptions.WeChatPay.Client)
	require.NotNil(t, captured.AdaptivePricing)
	require.NotNil(t, captured.AdaptivePricing.Enabled)
	assert.False(t, *captured.AdaptivePricing.Enabled)
	assert.Equal(t, stripeCheckoutMethodWeChatPay, captured.Metadata["checkout_method"])
	assert.Equal(t, stripeCheckoutMethodWeChatPay, captured.PaymentIntentData.Metadata["checkout_method"])
	assert.Equal(t, rate.QuoteVersion(), captured.Metadata["fx_quote_version"])
	assert.Equal(t, rate.QuoteVersion(), captured.PaymentIntentData.Metadata["fx_quote_version"])
	assert.Equal(t, "false", captured.Extra.Values.Get("managed_payments[enabled]"))
}

func TestStripeCheckoutMethodValidationIsStrictAndBackwardCompatible(t *testing.T) {
	for _, checkoutMethod := range []string{"", stripeCheckoutMethodStandard, stripeCheckoutMethodWeChatPay} {
		actual, err := normalizeStripeCheckoutMethod(checkoutMethod)
		require.NoError(t, err)
		if checkoutMethod == "" {
			assert.Equal(t, stripeCheckoutMethodStandard, actual)
		} else {
			assert.Equal(t, checkoutMethod, actual)
		}
	}

	for _, checkoutMethod := range []string{"wechat", "card", "WECHAT_PAY", " wechat_pay", "wechat_pay "} {
		_, err := normalizeStripeCheckoutMethod(checkoutMethod)
		require.Error(t, err)
	}
}

func TestStripeWeChatPayCheckoutRequiresExplicitCapabilityFlag(t *testing.T) {
	setupStripeWebhookControllerTestDB(t)
	insertStripeFXControllerTestRate(t)
	t.Setenv("STRIPE_API_SECRET", "")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	t.Setenv("STRIPE_PRICE_ID", "")
	t.Setenv("STRIPE_PRODUCT_ID", "")
	confirmPaymentComplianceForTest(t)
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalPrice := setting.StripePriceId
	t.Cleanup(func() {
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripePriceId = originalPrice
	})
	setting.StripeApiSecret = "rk_test_wechat_capability"
	setting.StripeWebhookSecret = "whsec_wechat_capability"
	setting.StripePriceId = "price_wechat_capability"

	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "")
	require.NoError(t, requireStripeCheckoutMethodEnabled(stripeCheckoutMethodStandard))
	require.Error(t, requireStripeCheckoutMethodEnabled(stripeCheckoutMethodWeChatPay))

	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "true")
	require.Error(t, requireStripeCheckoutMethodEnabled(stripeCheckoutMethodWeChatPay))

	t.Setenv("STRIPE_PRODUCT_ID", "prod_wechat_cny")
	require.NoError(t, requireStripeCheckoutMethodEnabled(stripeCheckoutMethodWeChatPay))
}

func TestStripePayRequestValidatesAmountProviderAndCheckoutMethodTogether(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	testCases := []struct {
		name    string
		request StripePayRequest
		wantErr bool
	}{
		{
			name:    "legacy standard minimum",
			request: StripePayRequest{Amount: 2},
		},
		{
			name: "wechat maximum",
			request: StripePayRequest{
				Amount:         50_000,
				PaymentMethod:  model.PaymentMethodStripe,
				CheckoutMethod: stripeCheckoutMethodWeChatPay,
			},
		},
		{
			name:    "below minimum",
			request: StripePayRequest{Amount: 1, CheckoutMethod: stripeCheckoutMethodStandard},
			wantErr: true,
		},
		{
			name:    "above maximum",
			request: StripePayRequest{Amount: 50_001, CheckoutMethod: stripeCheckoutMethodWeChatPay},
			wantErr: true,
		},
		{
			name:    "wrong provider",
			request: StripePayRequest{Amount: 2, PaymentMethod: "wxpay"},
			wantErr: true,
		},
		{
			name:    "unknown checkout method",
			request: StripePayRequest{Amount: 2, CheckoutMethod: "card"},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			quota, err := validateStripePayRequest(&testCase.request)
			if testCase.wantErr {
				require.Error(t, err)
				assert.Zero(t, quota)
				return
			}
			require.NoError(t, err)
			assert.Positive(t, quota)
		})
	}
}

func TestStripeAmountQuoteKeepsLegacyUSDAndReturnsConfiguredCNYForWeChat(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	rate := insertStripeFXControllerTestRate(t)
	confirmPaymentComplianceForTest(t)
	t.Setenv("STRIPE_API_SECRET", "rk_test_quote")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_quote")
	t.Setenv("STRIPE_PRICE_ID", "price_usd_quote")
	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "true")
	t.Setenv("STRIPE_PRODUCT_ID", "prod_cny_quote")
	user := model.User{
		Username: "stripe-quote-user",
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "stripe-quote-aff",
	}
	require.NoError(t, db.Create(&user).Error)

	tests := []struct {
		name           string
		checkoutMethod string
		want           string
	}{
		{name: "legacy request is USD", want: "2.00"},
		{name: "explicit standard is USD", checkoutMethod: stripeCheckoutMethodStandard, want: "2.00"},
		{name: "wechat uses daily CNY rate", checkoutMethod: stripeCheckoutMethodWeChatPay, want: "13.54"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Set("id", user.Id)
			stripeAdaptor.RequestAmount(context, &StripePayRequest{
				Amount:         2,
				CheckoutMethod: test.checkoutMethod,
			}, model.StripeFundingTargetUser)

			var body struct {
				Message string                 `json:"message"`
				Data    string                 `json:"data"`
				Quote   *stripeFXQuoteResponse `json:"quote"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
			assert.Equal(t, "success", body.Message)
			assert.Equal(t, test.want, body.Data)
			if test.checkoutMethod == stripeCheckoutMethodWeChatPay {
				require.NotNil(t, body.Quote)
				assert.EqualValues(t, 1_354, body.Quote.PayAmountMinor)
				assert.Equal(t, "CNY", body.Quote.Currency)
				assert.Equal(t, "6.7655", body.Quote.ExchangeRate)
				assert.Equal(t, rate.QuoteVersion(), body.Quote.QuoteVersion)
			} else {
				assert.Nil(t, body.Quote)
			}
		})
	}
}

func TestStripeWeChatInvalidConfigurationCreatesNoOrderAndCallsNoStripeAPI(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	confirmPaymentComplianceForTest(t)
	t.Setenv("STRIPE_API_SECRET", "rk_test_invalid_wechat")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_invalid_wechat")
	t.Setenv("STRIPE_PRICE_ID", "price_usd_valid")
	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "true")
	t.Setenv("STRIPE_PRODUCT_ID", "")

	originalNew := stripeCheckoutSessionNew
	stripeCalls := 0
	stripeCheckoutSessionNew = func(*stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		stripeCalls++
		return nil, errors.New("must not be called")
	}
	t.Cleanup(func() { stripeCheckoutSessionNew = originalNew })

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("id", 123)
	stripeAdaptor.RequestPay(context, &StripePayRequest{
		Amount:         2,
		CheckoutMethod: stripeCheckoutMethodWeChatPay,
	}, model.StripeFundingTargetUser)

	assert.Zero(t, stripeCalls)
	var orderCount int64
	require.NoError(t, db.Model(&model.TopUp{}).Count(&orderCount).Error)
	assert.Zero(t, orderCount)
}

func TestStripeWeChatPayRejectsMissingOrForgedQuoteBeforeCreatingOrder(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	rate := insertStripeFXControllerTestRate(t)
	confirmPaymentComplianceForTest(t)
	t.Setenv("STRIPE_API_SECRET", "rk_test_quote_contract")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_quote_contract")
	t.Setenv("STRIPE_PRICE_ID", "price_usd_quote_contract")
	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "true")
	t.Setenv("STRIPE_PRODUCT_ID", "prod_quote_contract")

	originalNew := stripeCheckoutSessionNew
	stripeCalls := 0
	stripeCheckoutSessionNew = func(*stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		stripeCalls++
		return nil, errors.New("must not be called")
	}
	t.Cleanup(func() { stripeCheckoutSessionNew = originalNew })

	for name, quoteVersion := range map[string]string{
		"missing": "",
		"forged":  rate.QuoteVersion() + "tampered",
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Set("id", 123)
			context.Request = httptest.NewRequest(http.MethodPost, "/api/user/stripe/pay", nil)
			stripeAdaptor.RequestPay(context, &StripePayRequest{
				Amount:         2,
				CheckoutMethod: stripeCheckoutMethodWeChatPay,
				QuoteVersion:   quoteVersion,
			}, model.StripeFundingTargetUser)

			var body struct {
				Message string `json:"message"`
				Code    string `json:"code"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
			assert.Equal(t, "error", body.Message)
			assert.Equal(t, "stripe_fx_quote_expired", body.Code)
		})
	}

	assert.Zero(t, stripeCalls)
	var orderCount int64
	require.NoError(t, db.Model(&model.TopUp{}).Count(&orderCount).Error)
	assert.Zero(t, orderCount)
}

func TestStripeCheckoutResponseMismatchExpiresSessionAndPreservesCNYSnapshot(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	rate := insertStripeFXControllerTestRate(t)
	confirmPaymentComplianceForTest(t)
	t.Setenv("STRIPE_API_SECRET", "rk_test_mismatch")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_mismatch")
	t.Setenv("STRIPE_PRICE_ID", "price_usd_mismatch")
	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "true")
	t.Setenv("STRIPE_PRODUCT_ID", "prod_cny_mismatch")
	user := model.User{
		Username: "stripe-mismatch-user",
		Email:    "payer@example.test",
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "stripe-mismatch-aff",
	}
	require.NoError(t, db.Create(&user).Error)

	originalNew := stripeCheckoutSessionNew
	originalExpire := stripeCheckoutSessionExpire
	expiredSessionID := ""
	stripeCheckoutSessionNew = func(*stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		return &stripe.CheckoutSession{
			ID:          "cs_cny_mismatch",
			URL:         "https://checkout.stripe.test/cny-mismatch",
			AmountTotal: 1353,
			Currency:    stripe.CurrencyCNY,
			Livemode:    false,
		}, nil
	}
	stripeCheckoutSessionExpire = func(id string, _ *stripe.CheckoutSessionExpireParams) (*stripe.CheckoutSession, error) {
		expiredSessionID = id
		return &stripe.CheckoutSession{ID: id, Status: stripe.CheckoutSessionStatusExpired}, nil
	}
	t.Cleanup(func() {
		stripeCheckoutSessionNew = originalNew
		stripeCheckoutSessionExpire = originalExpire
	})

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("id", user.Id)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/stripe/pay", nil)
	stripeAdaptor.RequestPay(context, &StripePayRequest{
		Amount:         2,
		CheckoutMethod: stripeCheckoutMethodWeChatPay,
		QuoteVersion:   rate.QuoteVersion(),
	}, model.StripeFundingTargetUser)

	assert.Equal(t, "cs_cny_mismatch", expiredSessionID)
	var stored model.TopUp
	require.NoError(t, db.First(&stored).Error)
	assert.Equal(t, model.StripeCheckoutMethodWeChatPay, stored.StripeCheckoutMethod)
	assert.Equal(t, model.StripePricingModeInline, stored.StripePricingMode)
	assert.Empty(t, stored.StripePriceId)
	assert.Equal(t, "prod_cny_mismatch", stored.StripeProductId)
	assert.EqualValues(t, 1, stored.StripeLineItemQuantity)
	assert.EqualValues(t, 1354, stored.ExpectedUnitAmountMinor)
	assert.EqualValues(t, 1354, stored.ExpectedAmountMinor)
	assert.Equal(t, "cny", stored.ExpectedCurrency)
	assert.Equal(t, rate.Id, stored.StripeFXRateId)
	assert.Equal(t, rate.RateE4, stored.StripeFXRateE4)
	assert.True(t, stored.ReconciliationRequired)
}

func TestStripeSDKUsesManagedPaymentsCompatibleAPIVersion(t *testing.T) {
	assert.True(t, strings.HasSuffix(stripe.APIVersion, ".basil"))
	assert.GreaterOrEqual(t, stripe.APIVersion, "2025-03-31.basil")
}

func TestStripeReturnURLsDistinguishPersonalAndTeamCheckout(t *testing.T) {
	originalAddress := system_setting.ServerAddress
	t.Cleanup(func() {
		system_setting.ServerAddress = originalAddress
	})
	system_setting.ServerAddress = "https://unirouters.cc"

	personalSuccess, personalCancel := stripeReturnURLs("user", "", "")
	assert.Equal(t, "https://unirouters.cc/wallet?stripe=success", personalSuccess)
	assert.Equal(t, "https://unirouters.cc/wallet?stripe=cancel", personalCancel)

	teamSuccess, teamCancel := stripeReturnURLs("team", "", "")
	assert.Equal(t, "https://unirouters.cc/team-management?stripe=success", teamSuccess)
	assert.Equal(t, "https://unirouters.cc/team-management?stripe=cancel", teamCancel)
}

func TestTopUpInfoPublishesStripeUSDContract(t *testing.T) {
	setupStripeWebhookControllerTestDB(t)
	insertStripeFXControllerTestRate(t)
	t.Setenv("STRIPE_API_SECRET", "")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	t.Setenv("STRIPE_PRICE_ID", "")
	t.Setenv("STRIPE_WECHAT_PAY_ENABLED", "true")
	t.Setenv("STRIPE_PRODUCT_ID", "prod_info_cny")
	confirmPaymentComplianceForTest(t)
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalPrice := setting.StripePriceId
	t.Cleanup(func() {
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripePriceId = originalPrice
	})
	setting.StripeApiSecret = "rk_test_info"
	setting.StripeWebhookSecret = "whsec_info"
	setting.StripePriceId = "price_info"

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	GetTopUpInfo(context)
	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			StripeMinTopUp      int64 `json:"stripe_min_topup"`
			StripeMaxTopUp      int64 `json:"stripe_max_topup"`
			StripeAmountOptions []int `json:"stripe_amount_options"`
			EnableWeChatPay     bool  `json:"enable_stripe_wechat_pay"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	require.True(t, body.Success)
	assert.EqualValues(t, 2, body.Data.StripeMinTopUp)
	assert.EqualValues(t, 50000, body.Data.StripeMaxTopUp)
	assert.Equal(t, []int{2, 5, 10, 50, 200, 500}, body.Data.StripeAmountOptions)
	assert.True(t, body.Data.EnableWeChatPay)
}

func TestTopUpInfoHidesEpayMethodsWhenGatewayIsUnconfigured(t *testing.T) {
	t.Setenv("STRIPE_API_SECRET", "")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	t.Setenv("STRIPE_PRICE_ID", "")
	confirmPaymentComplianceForTest(t)

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalPrice := setting.StripePriceId
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripePriceId = originalPrice
	})

	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{{
		"name": "支付宝",
		"type": "alipay",
	}}
	setting.StripeApiSecret = ""
	setting.StripeWebhookSecret = ""
	setting.StripePriceId = ""

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	GetTopUpInfo(context)
	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			EnableOnlineTopUp bool                `json:"enable_online_topup"`
			PayMethods        []map[string]string `json:"pay_methods"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	require.True(t, body.Success)
	assert.False(t, body.Data.EnableOnlineTopUp)
	assert.Empty(t, body.Data.PayMethods)
}

func TestStripeWebhookRequiresSignatureOverExactRawBody(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	originalSecret := setting.StripeWebhookSecret
	t.Cleanup(func() {
		setting.StripeWebhookSecret = originalSecret
	})
	setting.StripeWebhookSecret = "whsec_exact_raw_body"
	payload, err := common.Marshal(map[string]any{
		"id":          "evt_signature_test",
		"object":      "event",
		"api_version": stripe.APIVersion,
		"created":     time.Now().Unix(),
		"livemode":    false,
		"type":        "customer.created",
		"data": map[string]any{
			"object": map[string]any{"id": "cus_signature_test"},
		},
	})
	require.NoError(t, err)
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload: payload,
		Secret:  setting.StripeWebhookSecret,
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/stripe/webhook", StripeWebhook)

	request := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", strings.NewReader(string(signed.Payload)))
	request.Header.Set("Stripe-Signature", signed.Header)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)

	tamperedRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/stripe/webhook",
		strings.NewReader(string(signed.Payload)+" "),
	)
	tamperedRequest.Header.Set("Stripe-Signature", signed.Header)
	tamperedResponse := httptest.NewRecorder()
	router.ServeHTTP(tamperedResponse, tamperedRequest)
	assert.Equal(t, http.StatusBadRequest, tamperedResponse.Code)
}

func TestStripeWebhookIgnoresCheckoutSessionsOutsideLocalOrderPrefixes(t *testing.T) {
	sessionObject := stripe.CheckoutSession{
		ID:                "cs_test_preflight",
		ClientReferenceID: "unirouters_preflight",
		Status:            stripe.CheckoutSessionStatusComplete,
		PaymentStatus:     stripe.CheckoutSessionPaymentStatusPaid,
	}
	raw, err := common.Marshal(sessionObject)
	require.NoError(t, err)

	for _, eventType := range []stripe.EventType{
		stripe.EventTypeCheckoutSessionCompleted,
		stripe.EventTypeCheckoutSessionExpired,
		stripe.EventTypeCheckoutSessionAsyncPaymentFailed,
	} {
		event := stripe.Event{
			ID:   "evt_preflight",
			Type: eventType,
			Data: &stripe.EventData{Raw: raw},
		}
		require.NoError(t, processStripeWebhookEvent(context.Background(), event))
	}
}

func TestStripeWebhookHandlesUnknownTopUpReferencesAndFinancialObjects(t *testing.T) {
	setupStripeWebhookControllerTestDB(t)

	checkout := &stripe.CheckoutSession{
		ID:                "cs_test_unknown_local_ref",
		ClientReferenceID: "ref_unknown_local_order",
		Status:            stripe.CheckoutSessionStatusComplete,
		PaymentStatus:     stripe.CheckoutSessionPaymentStatusPaid,
		PaymentIntent:     &stripe.PaymentIntent{ID: "pi_unknown_local_order"},
		AmountTotal:       200,
		Currency:          stripe.CurrencyUSD,
	}
	require.NoError(t, fulfillStripeCheckout(
		context.Background(),
		stripe.Event{ID: "evt_unknown_checkout"},
		checkout,
	))

	refund := &stripe.Refund{
		ID:            "re_unknown_local_order",
		PaymentIntent: &stripe.PaymentIntent{ID: "pi_unknown_refund"},
		Amount:        200,
		Currency:      stripe.CurrencyUSD,
		Status:        stripe.RefundStatusSucceeded,
	}
	require.NoError(t, processStripeRefund(
		context.Background(),
		stripe.Event{
			ID:      "evt_unknown_refund",
			Created: 100,
			Type:    stripe.EventTypeRefundCreated,
		},
		refund,
	))

	dispute := &stripe.Dispute{
		ID:            "dp_unknown_local_order",
		PaymentIntent: &stripe.PaymentIntent{ID: "pi_unknown_dispute"},
		Amount:        200,
		Currency:      stripe.CurrencyUSD,
		Status:        stripe.DisputeStatusNeedsResponse,
	}
	require.NoError(t, processStripeDispute(
		context.Background(),
		stripe.Event{
			ID:      "evt_unknown_dispute",
			Created: 100,
			Type:    stripe.EventTypeChargeDisputeCreated,
		},
		dispute,
	))

	var pendingCount int64
	require.NoError(t, model.DB.Model(&model.StripeAdjustmentInbox{}).
		Where("state = ?", "pending").
		Count(&pendingCount).Error)
	assert.EqualValues(t, 2, pendingCount)
}
