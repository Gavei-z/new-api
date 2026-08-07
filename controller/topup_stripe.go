package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
	"github.com/thanhpk/randstr"
)

const (
	stripeMinimumTopUp             = model.StripeMinimumTopUpUSD
	stripeMaximumTopUp             = model.StripeMaximumTopUpUSD
	stripeWebhookBodyMaxSize int64 = 1 << 20
)

var (
	stripeAdaptor            = &StripeAdaptor{}
	stripeCheckoutSessionNew = session.New
)

// StripePayRequest represents an integer USD prepaid-credit purchase.
// The funding target is intentionally absent: it is resolved from the
// authenticated route and cannot be selected by the client.
type StripePayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	SuccessURL    string `json:"success_url,omitempty"`
	CancelURL     string `json:"cancel_url,omitempty"`
}

type StripeAdaptor struct{}

func stripeQuotaForAmount(amount int64) (int64, error) {
	if amount < stripeMinimumTopUp || amount > stripeMaximumTopUp {
		return 0, fmt.Errorf("Stripe top-up amount must be between %d and %d USD", stripeMinimumTopUp, stripeMaximumTopUp)
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return 0, errors.New("invalid quota unit configuration")
	}
	quota := decimal.NewFromInt(amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0)
	if quota.LessThanOrEqual(decimal.Zero) || quota.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, errors.New("top-up quota is out of range")
	}
	return quota.IntPart(), nil
}

func resolveStripeFundingTarget(c *gin.Context, targetType string) (int, error) {
	userID := c.GetInt("id")
	switch targetType {
	case model.StripeFundingTargetUser:
		if err := model.ValidateStripeTopUpTarget(userID, targetType, userID); err != nil {
			return 0, err
		}
		return userID, nil
	case model.StripeFundingTargetTeam:
		teamID := c.GetInt(middleware.TeamContextIdKey)
		if teamID <= 0 {
			return 0, model.ErrTeamAccessDenied
		}
		if err := model.ValidateStripeTopUpTarget(userID, targetType, teamID); err != nil {
			return 0, err
		}
		return teamID, nil
	default:
		return 0, errors.New("invalid Stripe funding target")
	}
}

func validateStripePayRequest(req *StripePayRequest) (int64, error) {
	if req == nil {
		return 0, errors.New("invalid payment request")
	}
	if req.PaymentMethod != "" && req.PaymentMethod != model.PaymentMethodStripe {
		return 0, errors.New("unsupported payment method")
	}
	return stripeQuotaForAmount(req.Amount)
}

func writeStripeRequestError(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{"message": "error", "data": message})
}

func (*StripeAdaptor) RequestAmount(c *gin.Context, req *StripePayRequest, targetType string) {
	if !isStripeTopUpEnabled() {
		writeStripeRequestError(c, "Stripe payment is unavailable")
		return
	}
	if _, err := validateStripePayRequest(req); err != nil {
		writeStripeRequestError(c, err.Error())
		return
	}
	if _, err := resolveStripeFundingTarget(c, targetType); err != nil {
		writeStripeRequestError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    strconv.FormatInt(req.Amount, 10) + ".00",
	})
}

func (*StripeAdaptor) RequestPay(c *gin.Context, req *StripePayRequest, targetType string) {
	if !isStripeTopUpEnabled() {
		writeStripeRequestError(c, "Stripe payment is unavailable")
		return
	}
	expectedQuota, err := validateStripePayRequest(req)
	if err != nil {
		writeStripeRequestError(c, err.Error())
		return
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = model.PaymentMethodStripe
	}
	if req.SuccessURL != "" && common.ValidateRedirectURL(req.SuccessURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "error", "data": "payment success redirect URL is not trusted"})
		return
	}
	if req.CancelURL != "" && common.ValidateRedirectURL(req.CancelURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "error", "data": "payment cancel redirect URL is not trusted"})
		return
	}

	userID := c.GetInt("id")
	targetID, err := resolveStripeFundingTarget(c, targetType)
	if err != nil {
		writeStripeRequestError(c, err.Error())
		return
	}
	user, err := model.GetUserById(userID, false)
	if err != nil || user == nil {
		writeStripeRequestError(c, "user does not exist")
		return
	}

	reference := fmt.Sprintf("new-api-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceID := "ref_" + common.Sha1([]byte(reference))
	expectedLivemode := stripeSecretIsLivemode(setting.GetStripeApiSecret())
	topUp := &model.TopUp{
		UserId:              userID,
		Amount:              req.Amount,
		Money:               float64(req.Amount),
		TradeNo:             referenceID,
		PaymentMethod:       model.PaymentMethodStripe,
		PaymentProvider:     model.PaymentProviderStripe,
		CreateTime:          time.Now().Unix(),
		Status:              common.TopUpStatusPending,
		FundingTarget:       targetType,
		FundingTargetId:     targetID,
		ExpectedAmountMinor: req.Amount * 100,
		ExpectedCurrency:    "usd",
		ExpectedLivemode:    expectedLivemode,
		ExpectedQuota:       expectedQuota,
	}
	if err := model.InsertStripeTopUpOrder(topUp); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Stripe order creation failed trade_no=%s error=%q", referenceID, err.Error()))
		writeStripeRequestError(c, "failed to create payment order")
		return
	}

	successURL, cancelURL := stripeReturnURLs(targetType, req.SuccessURL, req.CancelURL)
	checkoutSession, err := createStripeCheckoutSession(
		referenceID,
		user.StripeCustomer,
		user.Email,
		req.Amount,
		successURL,
		cancelURL,
	)
	if err != nil {
		_ = model.FailPendingStripeTopUp(referenceID, common.TopUpStatusFailed)
		logger.LogError(c.Request.Context(), fmt.Sprintf("Stripe Checkout Session creation failed trade_no=%s error=%q", referenceID, err.Error()))
		writeStripeRequestError(c, "failed to start payment")
		return
	}
	paymentIntentID := ""
	if checkoutSession.PaymentIntent != nil {
		paymentIntentID = checkoutSession.PaymentIntent.ID
	}
	if checkoutSession.ID == "" ||
		checkoutSession.URL == "" ||
		checkoutSession.Livemode != expectedLivemode ||
		checkoutSession.AmountTotal != topUp.ExpectedAmountMinor ||
		strings.ToLower(string(checkoutSession.Currency)) != topUp.ExpectedCurrency {
		_ = model.FlagStripeTopUpReconciliation(referenceID, "checkout session response did not match expected amount, currency, or mode")
		logger.LogError(c.Request.Context(), fmt.Sprintf("Stripe Checkout Session response invalid trade_no=%s", referenceID))
		writeStripeRequestError(c, "failed to start payment")
		return
	}
	if err := model.BindStripeCheckoutSession(referenceID, checkoutSession.ID, paymentIntentID, checkoutSession.Livemode); err != nil {
		// The signed webhook can safely bind an empty session ID later. Keep the
		// order pending rather than creating a second payable order.
		_ = model.FlagStripeTopUpReconciliation(referenceID, "checkout session could not be persisted")
		logger.LogError(c.Request.Context(), fmt.Sprintf("Stripe Checkout Session persistence failed trade_no=%s error=%q", referenceID, err.Error()))
		writeStripeRequestError(c, "payment was created but could not be persisted; contact support")
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf(
		"Stripe Checkout Session created trade_no=%s target=%s target_id=%d amount_minor=%d",
		referenceID,
		targetType,
		targetID,
		topUp.ExpectedAmountMinor,
	))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": checkoutSession.URL,
		},
	})
}

func RequestStripeAmount(c *gin.Context) {
	var req StripePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeStripeRequestError(c, "invalid request")
		return
	}
	stripeAdaptor.RequestAmount(c, &req, model.StripeFundingTargetUser)
}

func RequestStripePay(c *gin.Context) {
	var req StripePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeStripeRequestError(c, "invalid request")
		return
	}
	stripeAdaptor.RequestPay(c, &req, model.StripeFundingTargetUser)
}

func RequestTeamStripeAmount(c *gin.Context) {
	var req StripePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeStripeRequestError(c, "invalid request")
		return
	}
	stripeAdaptor.RequestAmount(c, &req, model.StripeFundingTargetTeam)
}

func RequestTeamStripePay(c *gin.Context) {
	var req StripePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeStripeRequestError(c, "invalid request")
		return
	}
	stripeAdaptor.RequestPay(c, &req, model.StripeFundingTargetTeam)
}

func StripeWebhook(c *gin.Context) {
	ctx := c.Request.Context()
	if !isStripeWebhookEnabled() {
		logger.LogWarn(ctx, "Stripe webhook rejected because it is not configured")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, stripeWebhookBodyMaxSize)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("Stripe webhook body read failed error=%q", err.Error()))
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}
	event, err := webhook.ConstructEventWithOptions(
		payload,
		c.GetHeader("Stripe-Signature"),
		setting.GetStripeWebhookSecret(),
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("Stripe webhook signature verification failed error=%q", err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := processStripeWebhookEvent(ctx, event); err != nil {
		logger.LogError(ctx, fmt.Sprintf(
			"Stripe webhook processing failed event_id=%s event_type=%s error=%q",
			event.ID,
			event.Type,
			err.Error(),
		))
		// A non-2xx response is intentional: Stripe must retry transient
		// database failures instead of silently losing a paid order.
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	logger.LogInfo(ctx, fmt.Sprintf("Stripe webhook processed event_id=%s event_type=%s", event.ID, event.Type))
	c.Status(http.StatusOK)
}

func processStripeWebhookEvent(ctx context.Context, event stripe.Event) error {
	if event.Data == nil {
		return errors.New("Stripe event has no data")
	}
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted, stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		var checkoutSession stripe.CheckoutSession
		if err := common.Unmarshal(event.Data.Raw, &checkoutSession); err != nil {
			return fmt.Errorf("decode Checkout Session: %w", err)
		}
		return fulfillStripeCheckout(ctx, event, &checkoutSession)
	case stripe.EventTypeCheckoutSessionExpired:
		var checkoutSession stripe.CheckoutSession
		if err := common.Unmarshal(event.Data.Raw, &checkoutSession); err != nil {
			return fmt.Errorf("decode expired Checkout Session: %w", err)
		}
		return expireStripeCheckout(ctx, event, &checkoutSession)
	case stripe.EventTypeCheckoutSessionAsyncPaymentFailed:
		var checkoutSession stripe.CheckoutSession
		if err := common.Unmarshal(event.Data.Raw, &checkoutSession); err != nil {
			return fmt.Errorf("decode failed Checkout Session: %w", err)
		}
		return failStripeCheckout(ctx, event, &checkoutSession)
	case stripe.EventTypeRefundCreated, stripe.EventTypeRefundUpdated, stripe.EventTypeRefundFailed:
		var refund stripe.Refund
		if err := common.Unmarshal(event.Data.Raw, &refund); err != nil {
			return fmt.Errorf("decode refund: %w", err)
		}
		return processStripeRefund(ctx, event, &refund)
	case stripe.EventTypeChargeDisputeCreated, stripe.EventTypeChargeDisputeClosed:
		var dispute stripe.Dispute
		if err := common.Unmarshal(event.Data.Raw, &dispute); err != nil {
			return fmt.Errorf("decode dispute: %w", err)
		}
		return processStripeDispute(ctx, event, &dispute)
	default:
		return nil
	}
}

func isUniRoutersStripeCheckoutReference(reference string) bool {
	return strings.HasPrefix(reference, "ref_") ||
		strings.HasPrefix(reference, "sub_ref_")
}

func isUniRoutersStripeTopUpReference(reference string) bool {
	return strings.HasPrefix(reference, "ref_")
}

func isUniRoutersStripeSubscriptionReference(reference string) bool {
	return strings.HasPrefix(reference, "sub_ref_")
}

func fulfillStripeCheckout(ctx context.Context, event stripe.Event, checkoutSession *stripe.CheckoutSession) error {
	if checkoutSession == nil {
		return errors.New("Checkout Session is missing")
	}
	// The Stripe account may create unrelated Checkout Sessions (including
	// deployment preflight checks). They have no local order and must not cause
	// a permanently retrying webhook.
	if !isUniRoutersStripeCheckoutReference(checkoutSession.ClientReferenceID) {
		return nil
	}
	if checkoutSession.Status != stripe.CheckoutSessionStatusComplete {
		return fmt.Errorf("Checkout Session status is %s", checkoutSession.Status)
	}
	if checkoutSession.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		// checkout.session.completed may precede a delayed payment. Returning
		// success here is safe because async_payment_succeeded will be delivered.
		if event.Type == stripe.EventTypeCheckoutSessionCompleted &&
			checkoutSession.PaymentStatus == stripe.CheckoutSessionPaymentStatusUnpaid {
			return nil
		}
		return fmt.Errorf("Checkout Session payment status is %s", checkoutSession.PaymentStatus)
	}

	paymentIntentID := ""
	if checkoutSession.PaymentIntent != nil {
		paymentIntentID = checkoutSession.PaymentIntent.ID
	}
	customerID := ""
	if checkoutSession.Customer != nil {
		customerID = checkoutSession.Customer.ID
	}
	if isUniRoutersStripeTopUpReference(checkoutSession.ClientReferenceID) {
		topUp, findErr := model.FindTopUpByTradeNo(checkoutSession.ClientReferenceID)
		if findErr != nil {
			if errors.Is(findErr, model.ErrTopUpNotFound) {
				// Prepaid Checkout is created only after its local order. A ref_
				// without an order therefore belongs outside this deployment (or
				// to data that requires offline recovery); retries cannot repair it.
				return nil
			}
			return findErr
		}
		if topUp.FundingTarget != model.StripeFundingTargetUser &&
			topUp.FundingTarget != model.StripeFundingTargetTeam {
			// Legacy Stripe top-ups did not snapshot enough immutable data to
			// fulfill under the new financial contract. Persist a review marker
			// and handle the paid event instead of retrying forever.
			return model.FlagStripeTopUpReconciliation(
				checkoutSession.ClientReferenceID,
				"Paid legacy Stripe Checkout requires manual reconciliation",
			)
		}
		result, err := model.CompleteStripeTopUp(model.StripeCheckoutCompletion{
			EventId:          event.ID,
			TradeNo:          checkoutSession.ClientReferenceID,
			SessionId:        checkoutSession.ID,
			PaymentIntentId:  paymentIntentID,
			CustomerId:       customerID,
			Status:           string(checkoutSession.Status),
			PaymentStatus:    string(checkoutSession.PaymentStatus),
			AmountTotalMinor: checkoutSession.AmountTotal,
			Currency:         strings.ToLower(string(checkoutSession.Currency)),
			Livemode:         event.Livemode,
			ObjectLivemode:   checkoutSession.Livemode,
		})
		if err != nil {
			return err
		}
		logger.LogInfo(ctx, fmt.Sprintf(
			"Stripe top-up credited trade_no=%s target=%s target_id=%d quota=%d",
			checkoutSession.ClientReferenceID,
			result.FundingTarget,
			result.FundingTargetId,
			result.CreditedQuota,
		))
		return nil
	}

	if !isUniRoutersStripeSubscriptionReference(checkoutSession.ClientReferenceID) {
		return nil
	}
	// Subscription Checkout shares this endpoint. Preserve its established
	// fulfillment path while keeping the provider payload minimal and free of
	// customer data.
	payload := map[string]any{
		"session_id":     checkoutSession.ID,
		"payment_intent": paymentIntentID,
		"amount_total":   checkoutSession.AmountTotal,
		"currency":       strings.ToLower(string(checkoutSession.Currency)),
		"livemode":       checkoutSession.Livemode,
		"event_id":       event.ID,
	}
	if err := model.CompleteSubscriptionOrder(
		checkoutSession.ClientReferenceID,
		common.GetJsonString(payload),
		model.PaymentProviderStripe,
		"",
	); err != nil {
		return err
	}
	return nil
}

func expireStripeCheckout(ctx context.Context, event stripe.Event, checkoutSession *stripe.CheckoutSession) error {
	if checkoutSession == nil {
		return errors.New("expired Checkout Session is missing")
	}
	if !isUniRoutersStripeCheckoutReference(checkoutSession.ClientReferenceID) {
		return nil
	}
	if checkoutSession.Status != stripe.CheckoutSessionStatusExpired {
		return fmt.Errorf("Checkout Session expiration status is %s", checkoutSession.Status)
	}
	if event.Livemode != checkoutSession.Livemode {
		return errors.New("expired Checkout Session mode mismatch")
	}
	if isUniRoutersStripeTopUpReference(checkoutSession.ClientReferenceID) {
		err := model.ExpireStripeTopUp(checkoutSession.ClientReferenceID, checkoutSession.ID, checkoutSession.Livemode)
		if err == nil ||
			errors.Is(err, model.ErrTopUpStatusInvalid) ||
			errors.Is(err, model.ErrTopUpNotFound) {
			return nil
		}
		return err
	}
	if !isUniRoutersStripeSubscriptionReference(checkoutSession.ClientReferenceID) {
		return nil
	}
	err := model.ExpireSubscriptionOrder(checkoutSession.ClientReferenceID, model.PaymentProviderStripe)
	if errors.Is(err, model.ErrSubscriptionOrderStatusInvalid) {
		return nil
	}
	if err == nil {
		logger.LogInfo(ctx, fmt.Sprintf("Stripe subscription order expired trade_no=%s", checkoutSession.ClientReferenceID))
	}
	return err
}

func failStripeCheckout(_ context.Context, event stripe.Event, checkoutSession *stripe.CheckoutSession) error {
	if checkoutSession == nil {
		return errors.New("failed Checkout Session is missing")
	}
	if !isUniRoutersStripeCheckoutReference(checkoutSession.ClientReferenceID) {
		return nil
	}
	if event.Livemode != checkoutSession.Livemode {
		return errors.New("failed Checkout Session mode mismatch")
	}
	if !isUniRoutersStripeTopUpReference(checkoutSession.ClientReferenceID) {
		// Subscription Checkout currently has no failed terminal-state helper.
		// Preserve its existing pending state for operator review without making
		// Stripe retry an event that cannot be applied.
		return nil
	}
	topUp, err := model.FindTopUpByTradeNo(checkoutSession.ClientReferenceID)
	if err != nil && !errors.Is(err, model.ErrTopUpNotFound) {
		return err
	}
	if topUp == nil {
		return nil
	}
	if topUp.FundingTarget != model.StripeFundingTargetUser &&
		topUp.FundingTarget != model.StripeFundingTargetTeam {
		err = model.UpdatePendingTopUpStatus(
			checkoutSession.ClientReferenceID,
			model.PaymentProviderStripe,
			common.TopUpStatusFailed,
		)
		if errors.Is(err, model.ErrTopUpStatusInvalid) {
			return nil
		}
		return err
	}
	err = model.FailStripeTopUp(checkoutSession.ClientReferenceID, checkoutSession.ID, checkoutSession.Livemode)
	if errors.Is(err, model.ErrTopUpStatusInvalid) {
		return nil
	}
	return err
}

func processStripeRefund(ctx context.Context, event stripe.Event, refund *stripe.Refund) error {
	if refund == nil || refund.ID == "" || refund.PaymentIntent == nil || refund.PaymentIntent.ID == "" {
		return errors.New("refund is missing identifiers")
	}
	result, err := model.ProcessStripeRefund(model.StripeRefundAdjustment{
		EventId:         event.ID,
		EventCreated:    event.Created,
		EventType:       string(event.Type),
		RefundId:        refund.ID,
		PaymentIntentId: refund.PaymentIntent.ID,
		AmountMinor:     refund.Amount,
		Currency:        strings.ToLower(string(refund.Currency)),
		Status:          string(refund.Status),
		Livemode:        event.Livemode,
	})
	if err != nil {
		return err
	}
	if result == nil {
		logger.LogInfo(ctx, fmt.Sprintf(
			"Stripe refund queued pending PaymentIntent linkage refund_id=%s",
			refund.ID,
		))
		return nil
	}
	logger.LogInfo(ctx, fmt.Sprintf(
		"Stripe refund reconciled trade_no=%s refund_id=%s status=%s reversed_quota=%d reconciliation_required=%t",
		result.TradeNo,
		refund.ID,
		refund.Status,
		result.ReversedQuota,
		result.ReconciliationRequired,
	))
	return nil
}

func processStripeDispute(ctx context.Context, event stripe.Event, dispute *stripe.Dispute) error {
	if dispute == nil || dispute.ID == "" || dispute.PaymentIntent == nil || dispute.PaymentIntent.ID == "" {
		return errors.New("dispute is missing identifiers")
	}
	result, err := model.ProcessStripeDispute(model.StripeDisputeAdjustment{
		EventId:         event.ID,
		EventCreated:    event.Created,
		EventType:       string(event.Type),
		DisputeId:       dispute.ID,
		PaymentIntentId: dispute.PaymentIntent.ID,
		AmountMinor:     dispute.Amount,
		Currency:        strings.ToLower(string(dispute.Currency)),
		Status:          string(dispute.Status),
		Livemode:        event.Livemode,
	})
	if err != nil {
		return err
	}
	if result == nil {
		logger.LogInfo(ctx, fmt.Sprintf(
			"Stripe dispute queued pending PaymentIntent linkage dispute_id=%s",
			dispute.ID,
		))
		return nil
	}
	logger.LogInfo(ctx, fmt.Sprintf(
		"Stripe dispute reconciled trade_no=%s dispute_id=%s status=%s reversed_quota=%d reconciliation_required=%t",
		result.TradeNo,
		dispute.ID,
		dispute.Status,
		result.ReversedQuota,
		result.ReconciliationRequired,
	))
	return nil
}

func stripeReturnURLs(targetType string, customSuccessURL string, customCancelURL string) (string, string) {
	successURL := customSuccessURL
	cancelURL := customCancelURL
	if targetType == model.StripeFundingTargetTeam {
		if successURL == "" {
			successURL = paymentReturnPath("/team-management?stripe=success")
		}
		if cancelURL == "" {
			cancelURL = paymentReturnPath("/team-management?stripe=cancel")
		}
		return successURL, cancelURL
	}
	if successURL == "" {
		successURL = paymentReturnPath("/wallet?stripe=success")
	}
	if cancelURL == "" {
		cancelURL = paymentReturnPath("/wallet?stripe=cancel")
	}
	return successURL, cancelURL
}

func stripeSecretIsLivemode(secret string) bool {
	secret = strings.TrimSpace(secret)
	return strings.HasPrefix(secret, "sk_live_") || strings.HasPrefix(secret, "rk_live_")
}

func createStripeCheckoutSession(
	referenceID string,
	customerID string,
	email string,
	amount int64,
	successURL string,
	cancelURL string,
) (*stripe.CheckoutSession, error) {
	secret := setting.GetStripeApiSecret()
	if !strings.HasPrefix(secret, "sk_") && !strings.HasPrefix(secret, "rk_") {
		return nil, errors.New("invalid Stripe API key")
	}
	priceID := setting.GetStripePriceId()
	if priceID == "" {
		return nil, errors.New("Stripe Price is not configured")
	}
	stripe.Key = secret

	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(referenceID),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(amount),
			},
		},
		Mode:                stripe.String(string(stripe.CheckoutSessionModePayment)),
		AllowPromotionCodes: stripe.Bool(false),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{"topup_reference": referenceID},
		},
		Metadata: map[string]string{"topup_reference": referenceID},
	}
	params.SetIdempotencyKey("stripe-topup:" + referenceID)
	// UniRouters sells a fixed amount of prepaid USD credit. Managed Payments
	// can add location-dependent tax and alter the charged/refunded amount, so
	// keep this Checkout on Stripe's standard payment contract.
	params.AddExtra("managed_payments[enabled]", "false")

	if customerID == "" {
		if email != "" {
			params.CustomerEmail = stripe.String(email)
		}
		params.CustomerCreation = stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways))
	} else {
		params.Customer = stripe.String(customerID)
	}
	return stripeCheckoutSessionNew(params)
}

// genStripeLink is retained for subscription/top-up compatibility callers.
// New prepaid-credit orders use createStripeCheckoutSession so the returned
// Stripe identifiers can be persisted and verified.
func genStripeLink(referenceID string, customerID string, email string, amount int64, successURL string, cancelURL string) (string, error) {
	if successURL == "" || cancelURL == "" {
		successURL, cancelURL = stripeReturnURLs(model.StripeFundingTargetUser, successURL, cancelURL)
	}
	checkoutSession, err := createStripeCheckoutSession(referenceID, customerID, email, amount, successURL, cancelURL)
	if err != nil {
		return "", err
	}
	return checkoutSession.URL, nil
}

func GetChargedAmount(count float64, _ model.User) float64 {
	return count
}

func getStripePayMoney(amount float64, _ string) float64 {
	return amount
}

func getStripeMinTopup() int64 {
	return stripeMinimumTopUp
}
