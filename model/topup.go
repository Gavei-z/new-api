package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TopUp struct {
	Id              int     `json:"id"`
	UserId          int     `json:"user_id" gorm:"index"`
	Amount          int64   `json:"amount"`
	Money           float64 `json:"money"`
	TradeNo         string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string  `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
	Status          string  `json:"status"`

	// Stripe prepaid-credit orders snapshot every value used for fulfillment.
	// The target is resolved from authentication state, never from request JSON.
	FundingTarget           string  `json:"funding_target,omitempty" gorm:"type:varchar(16);index"`
	FundingTargetId         int     `json:"funding_target_id,omitempty" gorm:"index"`
	StripeSessionId         *string `json:"-" gorm:"type:varchar(255);uniqueIndex"`
	StripePaymentIntent     *string `json:"-" gorm:"type:varchar(255);uniqueIndex"`
	StripeCompleteEvent     string  `json:"-" gorm:"type:varchar(255)"`
	StripeCheckoutMethod    string  `json:"-" gorm:"type:varchar(16)"`
	StripePriceId           string  `json:"-" gorm:"type:varchar(255)"`
	ExpectedUnitAmountMinor int64   `json:"-" gorm:"type:bigint"`
	ExpectedAmountMinor     int64   `json:"-" gorm:"type:bigint"`
	ExpectedCurrency        string  `json:"-" gorm:"type:varchar(8)"`
	ExpectedLivemode        bool    `json:"-"`
	ExpectedQuota           int64   `json:"-" gorm:"type:bigint"`
	CreditedQuota           int64   `json:"credited_quota,omitempty" gorm:"type:bigint"`

	// ReversedQuota is the total valid refund/dispute liability. Applied quota
	// may be lower when credits were already consumed; that gap requires manual
	// reconciliation and the target is disabled without ever going negative.
	ReversedQuota               int64  `json:"reversed_quota,omitempty" gorm:"type:bigint"`
	AppliedReversedQuota        int64  `json:"-" gorm:"type:bigint"`
	StripeAdjustmentState       string `json:"-" gorm:"type:text"`
	ReconciliationRequired      bool   `json:"reconciliation_required,omitempty"`
	ReconciliationNote          string `json:"-" gorm:"type:varchar(255)"`
	DisabledByStripeAdjustment  bool   `json:"-"`
	StripeAdjustmentQuarantined bool   `json:"-"`
}

const (
	PaymentMethodStripe       = "stripe"
	PaymentMethodCreem        = "creem"
	PaymentMethodWaffo        = "waffo"
	PaymentMethodWaffoPancake = "waffo_pancake"
	PaymentMethodBalance      = "balance"
)

const (
	PaymentProviderEpay         = "epay"
	PaymentProviderStripe       = "stripe"
	PaymentProviderCreem        = "creem"
	PaymentProviderWaffo        = "waffo"
	PaymentProviderWaffoPancake = "waffo_pancake"
	PaymentProviderBalance      = "balance"
)

const (
	StripeFundingTargetUser             = PrepaidTargetUser
	StripeFundingTargetTeam             = PrepaidTargetTeam
	StripeMinimumTopUpUSD         int64 = 2
	StripeMaximumTopUpUSD         int64 = 50000
	StripeCheckoutMethodStandard        = "standard"
	StripeCheckoutMethodWeChatPay       = "wechat_pay"
)

var (
	ErrPaymentMethodMismatch        = errors.New("payment method mismatch")
	ErrTopUpNotFound                = errors.New("topup not found")
	ErrTopUpStatusInvalid           = errors.New("topup status invalid")
	ErrStripeTopUpVerification      = errors.New("Stripe top-up verification failed")
	ErrStripeReconciliationRequired = errors.New("Stripe top-up requires reconciliation")
)

type StripeCheckoutCompletion struct {
	EventId          string
	TradeNo          string
	SessionId        string
	PaymentIntentId  string
	CustomerId       string
	Status           string
	PaymentStatus    string
	AmountTotalMinor int64
	Currency         string
	Livemode         bool
	ObjectLivemode   bool
}

type StripeRefundAdjustment struct {
	EventId         string
	EventCreated    int64
	EventType       string
	RefundId        string
	PaymentIntentId string
	AmountMinor     int64
	Currency        string
	Status          string
	Livemode        bool
}

type StripeDisputeAdjustment struct {
	EventId         string
	EventCreated    int64
	EventType       string
	DisputeId       string
	PaymentIntentId string
	AmountMinor     int64
	Currency        string
	Status          string
	Livemode        bool
}

type StripeTopUpResult struct {
	TradeNo                     string
	UserId                      int
	FundingTarget               string
	FundingTargetId             int
	CreditedQuota               int64
	ReversedQuota               int64
	ReconciliationRequired      bool
	DisabledByStripeAdjustment  bool
	StripeAdjustmentQuarantined bool
}

// StripePaymentIntentLink is a database-backed serialization point for all
// completion, refund, and dispute events concerning one PaymentIntent. The
// row lock prevents a refund/dispute that arrives concurrently with Checkout
// completion from being acknowledged before either side can observe the
// other.
type StripePaymentIntentLink struct {
	PaymentIntentId string `json:"-" gorm:"primaryKey;type:varchar(255)"`
	TradeNo         string `json:"-" gorm:"type:varchar(255);index"`
	CreatedAt       int64  `json:"-" gorm:"autoCreateTime"`
	UpdatedAt       int64  `json:"-" gorm:"autoUpdateTime"`
}

// StripeAdjustmentInbox contains only the minimum signed financial fields
// required to replay an out-of-order refund or dispute. Raw Stripe payloads
// and customer data are deliberately not retained.
type StripeAdjustmentInbox struct {
	EventId         string `json:"-" gorm:"primaryKey;type:varchar(255)"`
	EventCreated    int64  `json:"-" gorm:"index"`
	Kind            string `json:"-" gorm:"type:varchar(16);not null"`
	EventType       string `json:"-" gorm:"type:varchar(64);not null"`
	ObjectId        string `json:"-" gorm:"type:varchar(255);not null;index"`
	PaymentIntentId string `json:"-" gorm:"type:varchar(255);not null;index"`
	AmountMinor     int64  `json:"-" gorm:"type:bigint;not null"`
	Currency        string `json:"-" gorm:"type:varchar(8);not null"`
	Status          string `json:"-" gorm:"type:varchar(64);not null"`
	Livemode        bool   `json:"-"`
	State           string `json:"-" gorm:"type:varchar(16);not null;index"`
	TradeNo         string `json:"-" gorm:"type:varchar(255);index"`
	CreatedAt       int64  `json:"-" gorm:"autoCreateTime"`
	ProcessedAt     int64  `json:"-"`
}

type stripeAdjustmentRecord struct {
	EventId      string `json:"event_id"`
	EventCreated int64  `json:"event_created"`
	AmountMinor  int64  `json:"amount_minor"`
	Status       string `json:"status"`
	Active       bool   `json:"active"`
	Terminal     bool   `json:"terminal"`
}

type stripeAdjustmentState struct {
	Refunds  map[string]stripeAdjustmentRecord `json:"refunds,omitempty"`
	Disputes map[string]stripeAdjustmentRecord `json:"disputes,omitempty"`
}

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	topUp, err := FindTopUpByTradeNo(tradeNo)
	if err != nil {
		return nil
	}
	return topUp
}

func FindTopUpByTradeNo(tradeNo string) (*TopUp, error) {
	if strings.TrimSpace(tradeNo) == "" {
		return nil, ErrTopUpNotFound
	}
	var topUp TopUp
	if err := DB.Where("trade_no = ?", tradeNo).First(&topUp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTopUpNotFound
		}
		return nil, err
	}
	return &topUp, nil
}

func stripeExpectedQuotaForAmount(amount int64) (int64, error) {
	if amount < StripeMinimumTopUpUSD || amount > StripeMaximumTopUpUSD {
		return 0, errors.New("Stripe top-up amount is out of range")
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return 0, errors.New("invalid quota unit configuration")
	}
	quota := decimal.NewFromInt(amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0)
	if quota.LessThanOrEqual(decimal.Zero) || quota.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, errors.New("Stripe top-up quota is out of range")
	}
	return quota.IntPart(), nil
}

func validateStripeTopUpTargetTx(tx *gorm.DB, userId int, targetType string, targetId int) error {
	if userId <= 0 || targetId <= 0 {
		return ErrTeamAccessDenied
	}
	var user User
	if err := lockForUpdate(tx).Select("id", "status").Where("id = ?", userId).First(&user).Error; err != nil {
		return err
	}
	if user.Status != common.UserStatusEnabled {
		return errors.New("user is disabled")
	}

	switch targetType {
	case StripeFundingTargetUser:
		if targetId != userId {
			return ErrTeamAccessDenied
		}
		// A team identity always uses the team wallet. This includes disabled
		// memberships so a member cannot bypass tenant accounting through the
		// personal endpoint.
		var memberCount int64
		if err := tx.Model(&TeamMember{}).Where("user_id = ?", userId).Count(&memberCount).Error; err != nil {
			return err
		}
		if memberCount > 0 {
			return ErrTeamMemberPersonalQuota
		}
		return nil
	case StripeFundingTargetTeam:
		var member TeamMember
		if err := lockForUpdate(tx).
			Where("team_id = ? AND user_id = ?", targetId, userId).
			First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeamAccessDenied
			}
			return err
		}
		if member.Status != TeamMemberStatusEnabled || member.Role < TeamMemberRoleAdmin {
			return ErrTeamAccessDenied
		}
		var team Team
		if err := lockForUpdate(tx).Select("id", "status").Where("id = ?", targetId).First(&team).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeamNotFound
			}
			return err
		}
		if team.Status != TeamStatusEnabled {
			return ErrTeamDisabled
		}
		return nil
	default:
		return errors.New("invalid Stripe funding target")
	}
}

func ValidateStripeTopUpTarget(userId int, targetType string, targetId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return validateStripeTopUpTargetTx(tx, userId, targetType, targetId)
	})
}

func InsertStripeTopUpOrder(topUp *TopUp) error {
	if topUp == nil || strings.TrimSpace(topUp.TradeNo) == "" {
		return errors.New("invalid Stripe top-up order")
	}
	if topUp.PaymentMethod != PaymentMethodStripe ||
		topUp.PaymentProvider != PaymentProviderStripe ||
		topUp.Status != common.TopUpStatusPending {
		return ErrPaymentMethodMismatch
	}
	if topUp.Money != float64(topUp.Amount) {
		return ErrStripeTopUpVerification
	}
	expectedQuota, err := stripeExpectedQuotaForAmount(topUp.Amount)
	if err != nil {
		return err
	}
	if expectedQuota != topUp.ExpectedQuota {
		return ErrStripeTopUpVerification
	}
	checkoutMethod := topUp.StripeCheckoutMethod
	unitAmountMinor := topUp.ExpectedUnitAmountMinor
	switch checkoutMethod {
	case "":
		// Rows created before checkout snapshots were introduced are legacy USD
		// orders. Keep accepting that exact shape for rolling upgrades and tests;
		// all new controller-created orders populate the explicit fields below.
		if strings.TrimSpace(topUp.StripePriceId) != "" ||
			unitAmountMinor != 0 ||
			topUp.ExpectedCurrency != "usd" ||
			topUp.ExpectedAmountMinor != topUp.Amount*100 {
			return ErrStripeTopUpVerification
		}
	case StripeCheckoutMethodStandard:
		if strings.TrimSpace(topUp.StripePriceId) == "" ||
			unitAmountMinor != 100 ||
			topUp.ExpectedCurrency != "usd" {
			return ErrStripeTopUpVerification
		}
	case StripeCheckoutMethodWeChatPay:
		if strings.TrimSpace(topUp.StripePriceId) == "" ||
			unitAmountMinor <= 0 ||
			topUp.ExpectedCurrency != "cny" {
			return ErrStripeTopUpVerification
		}
	default:
		return ErrStripeTopUpVerification
	}
	if len(topUp.StripePriceId) > 255 {
		return ErrStripeTopUpVerification
	}
	if checkoutMethod != "" {
		if topUp.Amount > math.MaxInt64/unitAmountMinor ||
			topUp.ExpectedAmountMinor != topUp.Amount*unitAmountMinor {
			return ErrStripeTopUpVerification
		}
	}
	if len(topUp.TradeNo) > 255 {
		return errors.New("Stripe order reference is too long")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := validateStripeTopUpTargetTx(
			tx,
			topUp.UserId,
			topUp.FundingTarget,
			topUp.FundingTargetId,
		); err != nil {
			return err
		}
		return tx.Create(topUp).Error
	})
}

func setStripeIdentifiersTx(
	tx *gorm.DB,
	topUp *TopUp,
	sessionId string,
	paymentIntentId string,
	livemode bool,
) error {
	if topUp == nil || strings.TrimSpace(sessionId) == "" || len(sessionId) > 255 {
		return ErrStripeTopUpVerification
	}
	if livemode != topUp.ExpectedLivemode {
		return ErrStripeTopUpVerification
	}
	if topUp.StripeSessionId != nil && *topUp.StripeSessionId != sessionId {
		return ErrStripeTopUpVerification
	}
	var duplicateCount int64
	if err := tx.Model(&TopUp{}).
		Where("stripe_session_id = ? AND id <> ?", sessionId, topUp.Id).
		Count(&duplicateCount).Error; err != nil {
		return err
	}
	if duplicateCount > 0 {
		return ErrStripeTopUpVerification
	}
	topUp.StripeSessionId = &sessionId

	if paymentIntentId != "" {
		if len(paymentIntentId) > 255 {
			return ErrStripeTopUpVerification
		}
		if topUp.StripePaymentIntent != nil && *topUp.StripePaymentIntent != paymentIntentId {
			return ErrStripeTopUpVerification
		}
		duplicateCount = 0
		if err := tx.Model(&TopUp{}).
			Where("stripe_payment_intent = ? AND id <> ?", paymentIntentId, topUp.Id).
			Count(&duplicateCount).Error; err != nil {
			return err
		}
		if duplicateCount > 0 {
			return ErrStripeTopUpVerification
		}
		topUp.StripePaymentIntent = &paymentIntentId
	}
	return nil
}

func BindStripeCheckoutSession(tradeNo string, sessionId string, paymentIntentId string, livemode bool) error {
	if tradeNo == "" {
		return ErrTopUpNotFound
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var topUp TopUp
		if err := lockForUpdate(tx).Where("trade_no = ?", tradeNo).First(&topUp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopUpNotFound
			}
			return err
		}
		if topUp.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending && topUp.Status != common.TopUpStatusSuccess {
			return ErrTopUpStatusInvalid
		}
		if err := setStripeIdentifiersTx(tx, &topUp, sessionId, paymentIntentId, livemode); err != nil {
			return err
		}
		return tx.Save(&topUp).Error
	})
}

func setStripeReconciliation(topUp *TopUp, note string) {
	topUp.ReconciliationRequired = true
	note = strings.TrimSpace(note)
	noteRunes := []rune(note)
	if len(noteRunes) > 255 {
		note = string(noteRunes[:255])
	}
	topUp.ReconciliationNote = note
}

func FlagStripeTopUpReconciliation(tradeNo string, note string) error {
	if tradeNo == "" {
		return ErrTopUpNotFound
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var topUp TopUp
		if err := lockForUpdate(tx).Where("trade_no = ?", tradeNo).First(&topUp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopUpNotFound
			}
			return err
		}
		if topUp.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}
		setStripeReconciliation(&topUp, note)
		return tx.Save(&topUp).Error
	})
}

func FailPendingStripeTopUp(tradeNo string, targetStatus string) error {
	if targetStatus != common.TopUpStatusFailed && targetStatus != common.TopUpStatusExpired {
		return ErrTopUpStatusInvalid
	}
	return UpdatePendingTopUpStatus(tradeNo, PaymentProviderStripe, targetStatus)
}

func updateStripeTopUpTerminalStatus(tradeNo string, sessionId string, livemode bool, targetStatus string) error {
	if tradeNo == "" || sessionId == "" {
		return ErrStripeTopUpVerification
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var topUp TopUp
		if err := lockForUpdate(tx).Where("trade_no = ?", tradeNo).First(&topUp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopUpNotFound
			}
			return err
		}
		if topUp.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}
		if err := setStripeIdentifiersTx(tx, &topUp, sessionId, "", livemode); err != nil {
			return err
		}
		topUp.Status = targetStatus
		return tx.Save(&topUp).Error
	})
}

func ExpireStripeTopUp(tradeNo string, sessionId string, livemode bool) error {
	return updateStripeTopUpTerminalStatus(tradeNo, sessionId, livemode, common.TopUpStatusExpired)
}

func FailStripeTopUp(tradeNo string, sessionId string, livemode bool) error {
	return updateStripeTopUpTerminalStatus(tradeNo, sessionId, livemode, common.TopUpStatusFailed)
}

func stripeTopUpResult(topUp *TopUp) *StripeTopUpResult {
	if topUp == nil {
		return nil
	}
	return &StripeTopUpResult{
		TradeNo:                     topUp.TradeNo,
		UserId:                      topUp.UserId,
		FundingTarget:               topUp.FundingTarget,
		FundingTargetId:             topUp.FundingTargetId,
		CreditedQuota:               topUp.CreditedQuota,
		ReversedQuota:               topUp.ReversedQuota,
		ReconciliationRequired:      topUp.ReconciliationRequired,
		DisabledByStripeAdjustment:  topUp.DisabledByStripeAdjustment,
		StripeAdjustmentQuarantined: topUp.StripeAdjustmentQuarantined,
	}
}

func finalizeStripeTopUpTarget(result *StripeTopUpResult) error {
	if result == nil {
		return nil
	}
	var finalizationErrors []error
	if err := RefreshPrepaidTargetCache(result.FundingTarget, result.FundingTargetId); err != nil {
		finalizationErrors = append(finalizationErrors, err)
	}
	if result.FundingTarget != StripeFundingTargetUser || !result.DisabledByStripeAdjustment {
		return errors.Join(finalizationErrors...)
	}
	if err := PublishUserAuthCache(result.FundingTargetId); err != nil {
		finalizationErrors = append(finalizationErrors, err)
	}
	if err := InvalidateUserTokensCache(result.FundingTargetId); err != nil {
		finalizationErrors = append(finalizationErrors, err)
	}
	if _, err := RevokeAllUserSessions(result.FundingTargetId, "stripe_adjustment_shortfall"); err != nil {
		finalizationErrors = append(finalizationErrors, err)
	}
	return errors.Join(finalizationErrors...)
}

func lockStripePaymentIntentLinkTx(
	tx *gorm.DB,
	paymentIntentId string,
) (*StripePaymentIntentLink, error) {
	if tx == nil || strings.TrimSpace(paymentIntentId) == "" || len(paymentIntentId) > 255 {
		return nil, ErrStripeTopUpVerification
	}
	link := StripePaymentIntentLink{PaymentIntentId: paymentIntentId}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
		return nil, err
	}
	if err := lockForUpdate(tx).
		Where("payment_intent_id = ?", paymentIntentId).
		First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func bindStripePaymentIntentLinkTx(
	tx *gorm.DB,
	link *StripePaymentIntentLink,
	tradeNo string,
) error {
	if tx == nil ||
		link == nil ||
		strings.TrimSpace(link.PaymentIntentId) == "" ||
		strings.TrimSpace(tradeNo) == "" ||
		len(tradeNo) > 255 {
		return ErrStripeTopUpVerification
	}
	if link.TradeNo != "" && link.TradeNo != tradeNo {
		return ErrStripeTopUpVerification
	}
	if link.TradeNo == tradeNo {
		return nil
	}
	link.TradeNo = tradeNo
	return tx.Save(link).Error
}

func stripeAdjustmentInboxMatches(
	stored *StripeAdjustmentInbox,
	candidate *StripeAdjustmentInbox,
) bool {
	return stored != nil &&
		candidate != nil &&
		stored.EventId == candidate.EventId &&
		stored.EventCreated == candidate.EventCreated &&
		stored.Kind == candidate.Kind &&
		stored.EventType == candidate.EventType &&
		stored.ObjectId == candidate.ObjectId &&
		stored.PaymentIntentId == candidate.PaymentIntentId &&
		stored.AmountMinor == candidate.AmountMinor &&
		stored.Currency == candidate.Currency &&
		stored.Status == candidate.Status &&
		stored.Livemode == candidate.Livemode
}

func storeStripeAdjustmentInboxTx(
	tx *gorm.DB,
	event *StripeAdjustmentInbox,
) (*StripeAdjustmentInbox, error) {
	if tx == nil ||
		event == nil ||
		strings.TrimSpace(event.EventId) == "" ||
		strings.TrimSpace(event.ObjectId) == "" ||
		strings.TrimSpace(event.PaymentIntentId) == "" ||
		(event.Kind != stripeAdjustmentKindRefund &&
			event.Kind != stripeAdjustmentKindDispute) ||
		len(event.EventId) > 255 ||
		len(event.EventType) > 64 ||
		len(event.ObjectId) > 255 ||
		len(event.PaymentIntentId) > 255 ||
		len(event.Currency) > 8 ||
		len(event.Status) > 64 {
		return nil, ErrStripeTopUpVerification
	}
	event.Currency = strings.ToLower(strings.TrimSpace(event.Currency))
	event.Status = strings.TrimSpace(event.Status)
	event.State = stripeAdjustmentInboxPending
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(event).Error; err != nil {
		return nil, err
	}
	var stored StripeAdjustmentInbox
	if err := lockForUpdate(tx).
		Where("event_id = ?", event.EventId).
		First(&stored).Error; err != nil {
		return nil, err
	}
	if !stripeAdjustmentInboxMatches(&stored, event) {
		return nil, ErrStripeTopUpVerification
	}
	if stored.State != stripeAdjustmentInboxPending &&
		stored.State != stripeAdjustmentInboxApplied {
		return nil, ErrStripeTopUpVerification
	}
	return &stored, nil
}

func markStripeAdjustmentInboxAppliedTx(
	tx *gorm.DB,
	event *StripeAdjustmentInbox,
	tradeNo string,
) error {
	if tx == nil ||
		event == nil ||
		event.EventId == "" ||
		tradeNo == "" {
		return ErrStripeTopUpVerification
	}
	if event.State == stripeAdjustmentInboxApplied {
		if event.TradeNo != tradeNo {
			return ErrStripeTopUpVerification
		}
		return nil
	}
	event.State = stripeAdjustmentInboxApplied
	event.TradeNo = tradeNo
	event.ProcessedAt = common.GetTimestamp()
	return tx.Save(event).Error
}

func CompleteStripeTopUp(completion StripeCheckoutCompletion) (*StripeTopUpResult, error) {
	if completion.TradeNo == "" ||
		completion.SessionId == "" ||
		completion.PaymentIntentId == "" ||
		completion.EventId == "" ||
		len(completion.EventId) > 255 {
		return nil, ErrStripeTopUpVerification
	}

	var result *StripeTopUpResult
	var newlyCredited bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		paymentIntentLink, err := lockStripePaymentIntentLinkTx(tx, completion.PaymentIntentId)
		if err != nil {
			return err
		}
		var topUp TopUp
		if err := lockForUpdate(tx).Where("trade_no = ?", completion.TradeNo).First(&topUp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopUpNotFound
			}
			return err
		}
		if topUp.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}

		verificationFailed := completion.Status != "complete" ||
			completion.PaymentStatus != "paid" ||
			completion.AmountTotalMinor != topUp.ExpectedAmountMinor ||
			strings.ToLower(strings.TrimSpace(completion.Currency)) != topUp.ExpectedCurrency ||
			completion.Livemode != topUp.ExpectedLivemode ||
			completion.ObjectLivemode != topUp.ExpectedLivemode
		if !verificationFailed {
			verifiedTopUp := topUp
			if err := setStripeIdentifiersTx(
				tx,
				&verifiedTopUp,
				completion.SessionId,
				completion.PaymentIntentId,
				completion.ObjectLivemode,
			); err != nil {
				verificationFailed = true
			} else if err := bindStripePaymentIntentLinkTx(
				tx,
				paymentIntentLink,
				topUp.TradeNo,
			); err != nil {
				verificationFailed = true
			} else {
				topUp = verifiedTopUp
			}
		}
		if verificationFailed {
			if topUp.Status == common.TopUpStatusSuccess {
				if err := failClosedStripeAdjustmentTx(
					tx,
					&topUp,
					completion.EventId,
					"Signed Stripe Checkout no longer matches a credited order; target frozen for review",
				); err != nil {
					return err
				}
			} else {
				setStripeReconciliation(&topUp, "Stripe Checkout amount, currency, mode, status, or identifier mismatch")
				if err := tx.Save(&topUp).Error; err != nil {
					return err
				}
			}
			result = stripeTopUpResult(&topUp)
			return nil
		}

		if topUp.Status == common.TopUpStatusSuccess {
			if topUp.CreditedQuota != topUp.ExpectedQuota {
				if err := failClosedStripeAdjustmentTx(
					tx,
					&topUp,
					completion.EventId,
					"Successful Stripe order has an unexpected credited quota; target frozen for review",
				); err != nil {
					return err
				}
			} else {
				if _, err := applyPendingStripeAdjustmentsTx(tx, &topUp); err != nil {
					return err
				}
			}
			result = stripeTopUpResult(&topUp)
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			setStripeReconciliation(&topUp, "Paid Stripe Checkout reached a non-pending local order")
			if err := tx.Save(&topUp).Error; err != nil {
				return err
			}
			result = stripeTopUpResult(&topUp)
			return nil
		}
		if topUp.ExpectedQuota <= 0 || topUp.CreditedQuota != 0 {
			setStripeReconciliation(&topUp, "Stripe order has invalid quota accounting state")
			if err := tx.Save(&topUp).Error; err != nil {
				return err
			}
			result = stripeTopUpResult(&topUp)
			return nil
		}

		if err := ApplyPrepaidCreditTx(
			tx,
			topUp.FundingTarget,
			topUp.FundingTargetId,
			topUp.UserId,
			topUp.ExpectedQuota,
			"stripe-credit:"+topUp.TradeNo,
			"Stripe prepaid credit",
		); err != nil {
			return err
		}
		topUp.CreditedQuota = topUp.ExpectedQuota
		topUp.StripeCompleteEvent = completion.EventId
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if !topUp.DisabledByStripeAdjustment && topUp.AppliedReversedQuota == topUp.ReversedQuota {
			topUp.ReconciliationRequired = false
			topUp.ReconciliationNote = ""
		}
		if completion.CustomerId != "" &&
			len(completion.CustomerId) <= 64 &&
			strings.HasPrefix(completion.CustomerId, "cus_") {
			if err := tx.Model(&User{}).
				Where("id = ?", topUp.UserId).
				Update("stripe_customer", completion.CustomerId).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(&topUp).Error; err != nil {
			return err
		}
		if _, err := applyPendingStripeAdjustmentsTx(tx, &topUp); err != nil {
			return err
		}
		newlyCredited = true
		result = stripeTopUpResult(&topUp)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result != nil {
		if err := finalizeStripeTopUpTarget(result); err != nil {
			return result, err
		}
	}
	if newlyCredited && result != nil {
		RecordLog(
			result.UserId,
			LogTypeTopup,
			fmt.Sprintf("Stripe充值成功，充值额度单位: %d", result.CreditedQuota),
		)
	}
	return result, nil
}

const stripeAdjustmentRecordLimit = 128

const (
	stripeAdjustmentKindRefund  = "refund"
	stripeAdjustmentKindDispute = "dispute"

	stripeAdjustmentInboxPending = "pending"
	stripeAdjustmentInboxApplied = "applied"
)

func loadStripeAdjustmentState(topUp *TopUp) (*stripeAdjustmentState, error) {
	state := &stripeAdjustmentState{
		Refunds:  make(map[string]stripeAdjustmentRecord),
		Disputes: make(map[string]stripeAdjustmentRecord),
	}
	if topUp == nil || strings.TrimSpace(topUp.StripeAdjustmentState) == "" {
		return state, nil
	}
	if err := common.UnmarshalJsonStr(topUp.StripeAdjustmentState, state); err != nil {
		return nil, err
	}
	if state.Refunds == nil {
		state.Refunds = make(map[string]stripeAdjustmentRecord)
	}
	if state.Disputes == nil {
		state.Disputes = make(map[string]stripeAdjustmentRecord)
	}
	return state, nil
}

func saveStripeAdjustmentState(topUp *TopUp, state *stripeAdjustmentState) error {
	if topUp == nil || state == nil {
		return errors.New("invalid Stripe adjustment state")
	}
	data, err := common.Marshal(state)
	if err != nil {
		return err
	}
	if len(data) > 60000 {
		return errors.New("Stripe adjustment audit state is too large")
	}
	topUp.StripeAdjustmentState = string(data)
	return nil
}

func stripeRefundActive(status string) (bool, error) {
	switch status {
	case "succeeded":
		return true, nil
	case "pending", "requires_action", "failed", "canceled":
		return false, nil
	default:
		return false, fmt.Errorf("unknown Stripe refund status %q", status)
	}
}

func stripeDisputeActive(eventType string, status string) (bool, error) {
	switch eventType {
	case "charge.dispute.created":
		switch status {
		case "warning_needs_response", "warning_under_review", "needs_response", "under_review", "lost":
			return true, nil
		default:
			return false, fmt.Errorf("invalid created dispute status %q", status)
		}
	case "charge.dispute.closed":
		switch status {
		case "lost":
			return true, nil
		case "won", "warning_closed", "prevented":
			return false, nil
		default:
			return false, fmt.Errorf("invalid closed dispute status %q", status)
		}
	default:
		return false, fmt.Errorf("invalid Stripe dispute event type %q", eventType)
	}
}

func stripeDesiredReversedQuota(topUp *TopUp, state *stripeAdjustmentState) (int64, error) {
	if topUp == nil ||
		state == nil ||
		topUp.ExpectedAmountMinor <= 0 ||
		topUp.CreditedQuota <= 0 {
		return 0, errors.New("invalid Stripe adjustment accounting state")
	}
	var totalMinor int64
	addActive := func(record stripeAdjustmentRecord) error {
		if !record.Active {
			return nil
		}
		if record.AmountMinor <= 0 ||
			record.AmountMinor > topUp.ExpectedAmountMinor ||
			totalMinor > topUp.ExpectedAmountMinor-record.AmountMinor {
			return errors.New("Stripe adjustments exceed the paid amount")
		}
		totalMinor += record.AmountMinor
		return nil
	}
	for _, refund := range state.Refunds {
		if err := addActive(refund); err != nil {
			return 0, err
		}
	}
	for _, dispute := range state.Disputes {
		if err := addActive(dispute); err != nil {
			return 0, err
		}
	}
	if totalMinor == 0 {
		return 0, nil
	}
	quota := decimal.NewFromInt(topUp.CreditedQuota).
		Mul(decimal.NewFromInt(totalMinor)).
		Div(decimal.NewFromInt(topUp.ExpectedAmountMinor)).
		Round(0)
	if quota.IsNegative() ||
		quota.GreaterThan(decimal.NewFromInt(topUp.CreditedQuota)) ||
		quota.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, errors.New("Stripe adjustment quota is out of range")
	}
	return quota.IntPart(), nil
}

func disableStripeFundingTargetTx(tx *gorm.DB, topUp *TopUp) error {
	if topUp == nil {
		return ErrPrepaidTargetInvalid
	}
	switch topUp.FundingTarget {
	case StripeFundingTargetUser:
		var user User
		if err := lockForUpdate(tx).
			Select("id", "status").
			Where("id = ?", topUp.FundingTargetId).
			First(&user).Error; err != nil {
			return err
		}
		// Bump even when the account was already disabled. A shortfall is a
		// fresh security boundary and must invalidate any stale auth/session
		// state that may still exist outside the database. Once this order has
		// already frozen an already-disabled account, webhook retries do not
		// needlessly advance the version again.
		if user.Status == common.UserStatusEnabled || !topUp.DisabledByStripeAdjustment {
			if _, err := IncrementUserAuthVersionWithTx(tx, user.Id); err != nil {
				return err
			}
		}
		if user.Status == common.UserStatusEnabled {
			if err := tx.Model(&User{}).
				Where("id = ?", user.Id).
				Update("status", common.UserStatusDisabled).Error; err != nil {
				return err
			}
		}
		topUp.DisabledByStripeAdjustment = true
	case StripeFundingTargetTeam:
		var team Team
		if err := lockForUpdate(tx).
			Select("id", "status").
			Where("id = ?", topUp.FundingTargetId).
			First(&team).Error; err != nil {
			return err
		}
		if team.Status == TeamStatusEnabled {
			if err := tx.Model(&Team{}).
				Where("id = ?", team.Id).
				Update("status", TeamStatusDisabled).Error; err != nil {
				return err
			}
		}
		topUp.DisabledByStripeAdjustment = true
	default:
		return ErrPrepaidTargetInvalid
	}
	return nil
}

func failClosedStripeAdjustmentTx(
	tx *gorm.DB,
	topUp *TopUp,
	eventId string,
	note string,
) error {
	if topUp == nil {
		return ErrStripeReconciliationRequired
	}
	alreadyQuarantined := topUp.StripeAdjustmentQuarantined
	topUp.StripeAdjustmentQuarantined = true
	if topUp.CreditedQuota <= 0 ||
		topUp.AppliedReversedQuota < 0 ||
		topUp.AppliedReversedQuota > topUp.CreditedQuota {
		if err := disableStripeFundingTargetTx(tx, topUp); err != nil {
			return err
		}
		setStripeReconciliation(
			topUp,
			fmt.Sprintf("%s; event=%s; invalid credited/reversed quota snapshot", note, eventId),
		)
		return tx.Save(topUp).Error
	}
	// Once a permanent anomaly has already quarantined this order and capped
	// its liability at the full credited amount, a replay is fully handled.
	// In particular, do not reuse the original reversal idempotency key with a
	// smaller outstanding amount after a shortfall.
	if alreadyQuarantined &&
		topUp.DisabledByStripeAdjustment &&
		topUp.ReconciliationRequired &&
		topUp.ReversedQuota == topUp.CreditedQuota {
		return nil
	}
	// A valid full refund/dispute can already have established the maximum
	// liability while disabling the target because some credit was consumed.
	// The first later permanent anomaly must persist quarantine without trying
	// to reverse the same liability again.
	if topUp.DisabledByStripeAdjustment &&
		topUp.ReconciliationRequired &&
		topUp.ReversedQuota == topUp.CreditedQuota {
		outstanding := topUp.ReversedQuota - topUp.AppliedReversedQuota
		setStripeReconciliation(
			topUp,
			fmt.Sprintf("%s; event=%s; outstanding_quota=%d", note, eventId, outstanding),
		)
		return tx.Save(topUp).Error
	}
	if topUp.AppliedReversedQuota < topUp.CreditedQuota {
		requested := topUp.CreditedQuota - topUp.AppliedReversedQuota
		applied, shortfall, err := ApplyPrepaidReversalTx(
			tx,
			topUp.FundingTarget,
			topUp.FundingTargetId,
			topUp.UserId,
			requested,
			"stripe-fail-closed:"+common.Sha1([]byte(topUp.TradeNo+":"+eventId)),
			"Stripe financial anomaly fail-closed reversal",
		)
		if err != nil {
			return err
		}
		if applied < 0 ||
			shortfall < 0 ||
			applied > requested ||
			shortfall != requested-applied ||
			topUp.AppliedReversedQuota > math.MaxInt64-applied {
			return ErrPrepaidBalanceOutOfRange
		}
		topUp.AppliedReversedQuota += applied
	}
	// A permanent, signed adjustment anomaly cannot safely leave the
	// purchased credit spendable. Liability is capped at the order's immutable
	// credited snapshot, so neither this path nor retries can over-reverse.
	topUp.ReversedQuota = topUp.CreditedQuota
	if err := disableStripeFundingTargetTx(tx, topUp); err != nil {
		return err
	}
	outstanding := topUp.ReversedQuota - topUp.AppliedReversedQuota
	setStripeReconciliation(
		topUp,
		fmt.Sprintf("%s; event=%s; outstanding_quota=%d", note, eventId, outstanding),
	)
	return tx.Save(topUp).Error
}

func applyStripeAdjustmentStateTx(
	tx *gorm.DB,
	topUp *TopUp,
	state *stripeAdjustmentState,
	eventId string,
) error {
	desiredReversedQuota, err := stripeDesiredReversedQuota(topUp, state)
	if err != nil {
		return err
	}
	if desiredReversedQuota > topUp.AppliedReversedQuota {
		requested := desiredReversedQuota - topUp.AppliedReversedQuota
		applied, shortfall, err := ApplyPrepaidReversalTx(
			tx,
			topUp.FundingTarget,
			topUp.FundingTargetId,
			topUp.UserId,
			requested,
			"stripe-adjust:"+topUp.TradeNo+":"+eventId+":reverse",
			"Stripe refund or dispute reversal",
		)
		if err != nil {
			return err
		}
		if applied < 0 ||
			shortfall < 0 ||
			applied > requested ||
			shortfall != requested-applied ||
			topUp.AppliedReversedQuota > math.MaxInt64-applied {
			return ErrPrepaidBalanceOutOfRange
		}
		topUp.AppliedReversedQuota += applied
		if shortfall > 0 {
			if err := disableStripeFundingTargetTx(tx, topUp); err != nil {
				return err
			}
			setStripeReconciliation(
				topUp,
				fmt.Sprintf("Stripe reversal shortfall requires manual reconciliation: %d quota units", shortfall),
			)
		}
	} else if desiredReversedQuota < topUp.AppliedReversedQuota {
		restoreQuota := topUp.AppliedReversedQuota - desiredReversedQuota
		if err := ApplyPrepaidRestorationTx(
			tx,
			topUp.FundingTarget,
			topUp.FundingTargetId,
			topUp.UserId,
			restoreQuota,
			"stripe-adjust:"+topUp.TradeNo+":"+eventId+":restore",
			"Stripe dispute or refund reversal restored",
		); err != nil {
			return err
		}
		topUp.AppliedReversedQuota -= restoreQuota
	}
	topUp.ReversedQuota = desiredReversedQuota
	if topUp.AppliedReversedQuota < topUp.ReversedQuota {
		setStripeReconciliation(
			topUp,
			fmt.Sprintf(
				"Stripe reversal has an outstanding shortfall of %d quota units",
				topUp.ReversedQuota-topUp.AppliedReversedQuota,
			),
		)
	} else if topUp.DisabledByStripeAdjustment {
		// Do not automatically re-enable: an administrator may have separately
		// decided to keep the target disabled after the financial event.
		setStripeReconciliation(topUp, "Stripe balance is reconciled; target status requires manual review")
	} else {
		topUp.ReconciliationRequired = false
		topUp.ReconciliationNote = ""
	}
	return saveStripeAdjustmentState(topUp, state)
}

func stripeAdjustmentRecordChanged(
	existing stripeAdjustmentRecord,
	found bool,
	kind string,
	eventId string,
	eventCreated int64,
	amountMinor int64,
	status string,
	active bool,
	terminal bool,
) (stripeAdjustmentRecord, bool, error) {
	if kind != stripeAdjustmentKindRefund && kind != stripeAdjustmentKindDispute {
		return existing, false, errors.New("invalid Stripe adjustment kind")
	}
	if found {
		if existing.AmountMinor != amountMinor {
			return existing, false, errors.New("Stripe adjustment amount changed for an existing object")
		}
		if existing.EventId == eventId {
			return existing, false, nil
		}
		if eventCreated > 0 && existing.EventCreated > eventCreated {
			return existing, false, nil
		}
		if existing.Terminal && terminal {
			if kind == stripeAdjustmentKindDispute {
				if existing.Status == "lost" &&
					(status == "won" || status == "prevented") &&
					eventCreated > existing.EventCreated {
					// Stripe can rarely overturn a lost dispute after closure.
					// A strictly newer inactive terminal state restores only
					// the amount that was actually reversed.
				} else if !existing.Active && active {
					// A won/prevented dispute cannot be reopened by a late or
					// out-of-order lost event.
					return existing, false, nil
				} else if existing.Active == active {
					return existing, false, nil
				} else {
					return existing, false, errors.New("Stripe dispute terminal state changed")
				}
			} else {
				if existing.Status != status || existing.Active != active {
					return existing, false, errors.New("Stripe adjustment terminal state changed")
				}
				return existing, false, nil
			}
		}
		if existing.Terminal && !terminal {
			return existing, false, nil
		}
	}
	return stripeAdjustmentRecord{
		EventId:      eventId,
		EventCreated: eventCreated,
		AmountMinor:  amountMinor,
		Status:       status,
		Active:       active,
		Terminal:     terminal,
	}, true, nil
}

func applyStripeAdjustmentToTopUpTx(
	tx *gorm.DB,
	topUp *TopUp,
	event *StripeAdjustmentInbox,
) (bool, error) {
	if tx == nil || topUp == nil || event == nil {
		return false, ErrStripeTopUpVerification
	}
	if topUp.PaymentProvider != PaymentProviderStripe ||
		topUp.Status != common.TopUpStatusSuccess ||
		topUp.StripePaymentIntent == nil ||
		*topUp.StripePaymentIntent != event.PaymentIntentId {
		return false, ErrStripeTopUpVerification
	}
	// A previous permanent anomaly has already quarantined the entire credited
	// amount. Later events remain auditable in the inbox but must never restore
	// any part of that fail-closed liability.
	if topUp.StripeAdjustmentQuarantined &&
		topUp.DisabledByStripeAdjustment &&
		topUp.ReconciliationRequired &&
		topUp.CreditedQuota > 0 &&
		topUp.ReversedQuota == topUp.CreditedQuota {
		return false, nil
	}

	kindLabel := "refund"
	if event.Kind == stripeAdjustmentKindDispute {
		kindLabel = "dispute"
	} else if event.Kind != stripeAdjustmentKindRefund {
		return false, ErrStripeTopUpVerification
	}
	failClosed := func(note string) (bool, error) {
		if err := failClosedStripeAdjustmentTx(tx, topUp, event.EventId, note); err != nil {
			return false, err
		}
		return true, nil
	}

	if event.Currency != topUp.ExpectedCurrency ||
		event.Livemode != topUp.ExpectedLivemode {
		return failClosed(fmt.Sprintf(
			"Stripe %s currency or mode mismatch; target frozen for review",
			kindLabel,
		))
	}
	if event.AmountMinor <= 0 || event.AmountMinor > topUp.ExpectedAmountMinor {
		return failClosed(fmt.Sprintf(
			"Stripe %s exceeds the verified paid amount; target frozen for review",
			kindLabel,
		))
	}

	var active bool
	var terminal bool
	var err error
	switch event.Kind {
	case stripeAdjustmentKindRefund:
		active, err = stripeRefundActive(event.Status)
		terminal = event.Status == "succeeded" ||
			event.Status == "failed" ||
			event.Status == "canceled"
	case stripeAdjustmentKindDispute:
		active, err = stripeDisputeActive(event.EventType, event.Status)
		terminal = event.EventType == "charge.dispute.closed"
	}
	if err != nil {
		return failClosed(fmt.Sprintf(
			"Unrecognized signed Stripe %s transition; target frozen for review",
			kindLabel,
		))
	}

	state, err := loadStripeAdjustmentState(topUp)
	if err != nil {
		return failClosed(fmt.Sprintf(
			"Stripe %s audit state is invalid; target frozen for review",
			kindLabel,
		))
	}
	var existing stripeAdjustmentRecord
	var found bool
	if event.Kind == stripeAdjustmentKindRefund {
		existing, found = state.Refunds[event.ObjectId]
	} else {
		existing, found = state.Disputes[event.ObjectId]
	}
	if !found && len(state.Refunds)+len(state.Disputes) >= stripeAdjustmentRecordLimit {
		return failClosed("Stripe adjustment audit record limit reached; target frozen for review")
	}
	record, changed, err := stripeAdjustmentRecordChanged(
		existing,
		found,
		event.Kind,
		event.EventId,
		event.EventCreated,
		event.AmountMinor,
		event.Status,
		active,
		terminal,
	)
	if err != nil {
		return failClosed(fmt.Sprintf(
			"Stripe %s audit transition is inconsistent; target frozen for review",
			kindLabel,
		))
	}
	if !changed {
		return false, nil
	}
	if event.Kind == stripeAdjustmentKindRefund {
		state.Refunds[event.ObjectId] = record
	} else {
		state.Disputes[event.ObjectId] = record
	}
	if _, err := stripeDesiredReversedQuota(topUp, state); err != nil {
		return failClosed("Stripe refund/dispute liability cannot be safely reconciled; target frozen for review")
	}
	// Validate the portable TEXT audit bound before mutating balances. If it
	// cannot be persisted, freeze without applying an unauditable adjustment.
	if err := saveStripeAdjustmentState(topUp, state); err != nil {
		return failClosed("Stripe adjustment audit state is too large; target frozen for review")
	}
	if err := applyStripeAdjustmentStateTx(tx, topUp, state, event.EventId); err != nil {
		return false, err
	}
	if err := tx.Save(topUp).Error; err != nil {
		return false, err
	}
	return true, nil
}

func applyPendingStripeAdjustmentsTx(
	tx *gorm.DB,
	topUp *TopUp,
) (bool, error) {
	if tx == nil ||
		topUp == nil ||
		topUp.PaymentProvider != PaymentProviderStripe ||
		topUp.Status != common.TopUpStatusSuccess ||
		topUp.StripePaymentIntent == nil ||
		*topUp.StripePaymentIntent == "" {
		return false, ErrStripeTopUpVerification
	}
	var events []StripeAdjustmentInbox
	if err := lockForUpdate(tx).
		Where(
			"payment_intent_id = ? AND state = ?",
			*topUp.StripePaymentIntent,
			stripeAdjustmentInboxPending,
		).
		Order("event_created ASC").
		Order("event_id ASC").
		Find(&events).Error; err != nil {
		return false, err
	}
	anyChanged := false
	for index := range events {
		changed, err := applyStripeAdjustmentToTopUpTx(tx, topUp, &events[index])
		if err != nil {
			return false, err
		}
		if err := markStripeAdjustmentInboxAppliedTx(tx, &events[index], topUp.TradeNo); err != nil {
			return false, err
		}
		anyChanged = anyChanged || changed
	}
	return anyChanged, nil
}

func processStripeAdjustment(
	candidate *StripeAdjustmentInbox,
) (*StripeTopUpResult, bool, error) {
	if candidate == nil {
		return nil, false, ErrStripeTopUpVerification
	}
	var result *StripeTopUpResult
	var stateChanged bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		link, err := lockStripePaymentIntentLinkTx(tx, candidate.PaymentIntentId)
		if err != nil {
			return err
		}
		event, err := storeStripeAdjustmentInboxTx(tx, candidate)
		if err != nil {
			return err
		}

		var topUp TopUp
		if event.State == stripeAdjustmentInboxApplied {
			if event.TradeNo == "" {
				return ErrStripeTopUpVerification
			}
			if err := lockForUpdate(tx).
				Where("trade_no = ?", event.TradeNo).
				First(&topUp).Error; err != nil {
				return err
			}
			result = stripeTopUpResult(&topUp)
			return nil
		}

		if err := lockForUpdate(tx).
			Where("stripe_payment_intent = ?", event.PaymentIntentId).
			First(&topUp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Checkout completion may not have bound the PaymentIntent yet.
				// The signed event is durably queued and the link-row lock makes
				// this decision atomic with any concurrent completion.
				return nil
			}
			return err
		}
		if err := bindStripePaymentIntentLinkTx(tx, link, topUp.TradeNo); err != nil {
			if topUp.PaymentProvider == PaymentProviderStripe &&
				topUp.Status == common.TopUpStatusSuccess {
				if freezeErr := failClosedStripeAdjustmentTx(
					tx,
					&topUp,
					event.EventId,
					"Stripe PaymentIntent link conflict; target frozen for review",
				); freezeErr != nil {
					return freezeErr
				}
			} else {
				setStripeReconciliation(&topUp, "Stripe PaymentIntent link conflict requires manual review")
				if saveErr := tx.Save(&topUp).Error; saveErr != nil {
					return saveErr
				}
			}
			if err := markStripeAdjustmentInboxAppliedTx(tx, event, topUp.TradeNo); err != nil {
				return err
			}
			result = stripeTopUpResult(&topUp)
			stateChanged = true
			return nil
		}
		if topUp.PaymentProvider != PaymentProviderStripe {
			setStripeReconciliation(&topUp, "Stripe adjustment matched a non-Stripe order")
			if err := tx.Save(&topUp).Error; err != nil {
				return err
			}
			if err := markStripeAdjustmentInboxAppliedTx(tx, event, topUp.TradeNo); err != nil {
				return err
			}
			result = stripeTopUpResult(&topUp)
			return nil
		}
		if topUp.Status == common.TopUpStatusPending {
			// Leave the inbox row pending. Checkout completion will credit and
			// apply every queued adjustment in the same transaction.
			return nil
		}
		if topUp.Status != common.TopUpStatusSuccess {
			setStripeReconciliation(&topUp, "Stripe adjustment reached a terminal order that was not credited")
			if err := tx.Save(&topUp).Error; err != nil {
				return err
			}
			if err := markStripeAdjustmentInboxAppliedTx(tx, event, topUp.TradeNo); err != nil {
				return err
			}
			result = stripeTopUpResult(&topUp)
			return nil
		}
		changed, err := applyStripeAdjustmentToTopUpTx(tx, &topUp, event)
		if err != nil {
			return err
		}
		if err := markStripeAdjustmentInboxAppliedTx(tx, event, topUp.TradeNo); err != nil {
			return err
		}
		stateChanged = changed
		result = stripeTopUpResult(&topUp)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if result != nil {
		if err := finalizeStripeTopUpTarget(result); err != nil {
			return result, stateChanged, err
		}
	}
	return result, stateChanged, nil
}

func ProcessStripeRefund(adjustment StripeRefundAdjustment) (*StripeTopUpResult, error) {
	eventType := strings.TrimSpace(adjustment.EventType)
	if eventType == "" {
		eventType = "refund"
	}
	result, stateChanged, err := processStripeAdjustment(&StripeAdjustmentInbox{
		EventId:         adjustment.EventId,
		EventCreated:    adjustment.EventCreated,
		Kind:            stripeAdjustmentKindRefund,
		EventType:       eventType,
		ObjectId:        adjustment.RefundId,
		PaymentIntentId: adjustment.PaymentIntentId,
		AmountMinor:     adjustment.AmountMinor,
		Currency:        adjustment.Currency,
		Status:          adjustment.Status,
		Livemode:        adjustment.Livemode,
	})
	if err != nil {
		return nil, err
	}
	if stateChanged && result != nil {
		RecordLog(
			result.UserId,
			LogTypeTopup,
			fmt.Sprintf("Stripe退款状态更新，累计冲正额度单位: %d", result.ReversedQuota),
		)
	}
	return result, nil
}

func ProcessStripeDispute(adjustment StripeDisputeAdjustment) (*StripeTopUpResult, error) {
	result, stateChanged, err := processStripeAdjustment(&StripeAdjustmentInbox{
		EventId:         adjustment.EventId,
		EventCreated:    adjustment.EventCreated,
		Kind:            stripeAdjustmentKindDispute,
		EventType:       adjustment.EventType,
		ObjectId:        adjustment.DisputeId,
		PaymentIntentId: adjustment.PaymentIntentId,
		AmountMinor:     adjustment.AmountMinor,
		Currency:        adjustment.Currency,
		Status:          adjustment.Status,
		Livemode:        adjustment.Livemode,
	})
	if err != nil {
		return nil, err
	}
	if stateChanged && result != nil {
		RecordLog(
			result.UserId,
			LogTypeTopup,
			fmt.Sprintf("Stripe拒付状态更新，累计冲正额度单位: %d", result.ReversedQuota),
		)
	}
	return result, nil
}

func UpdatePendingTopUpStatus(tradeNo string, expectedPaymentProvider string, targetStatus string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		topUp.Status = targetStatus
		return tx.Save(topUp).Error
	})
}

func Recharge(referenceId string, customerId string, callerIp string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota float64
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}
		if topUp.ExpectedQuota > 0 {
			return errors.New("Stripe prepaid-credit orders require a verified webhook")
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota = topUp.Money * common.QuotaPerUnit
		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(map[string]interface{}{"stripe_customer": customerId, "quota": gorm.Expr("quota + ?", quota)}).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%d", logger.FormatQuota(int(quota)), topUp.Amount), callerIp, topUp.PaymentMethod, PaymentMethodStripe)

	return nil
}

// topUpQueryWindowSeconds 限制充值记录查询的时间窗口（秒）。
const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

// topUpQueryCutoff 返回允许查询的最早 create_time（秒级 Unix 时间戳）。
func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，不限制时间窗口）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&TopUp{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// SearchAllTopUps 按订单号搜索全平台充值记录（管理员使用，不限制时间窗口）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{})
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string, callerIp string) error {
	if tradeNo == "" {
		return errors.New("未提供订单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	var userId int
	var quotaToAdd int
	var payMoney float64
	var paymentMethod string

	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		// 行级锁，避免并发补单
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		// 幂等处理：已成功直接返回
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("订单状态不是待支付，无法补单")
		}
		if topUp.PaymentProvider == PaymentProviderStripe && topUp.ExpectedQuota > 0 {
			return errors.New("Stripe 订单必须通过验签并核对金额的 webhook 完成")
		}

		// 计算应充值额度：
		// - Stripe 订单：Money 代表经分组倍率换算后的美元数量，直接 * QuotaPerUnit
		// - 其他订单（如易支付）：Amount 为美元数量，* QuotaPerUnit
		if topUp.PaymentProvider == PaymentProviderStripe {
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd = int(decimal.NewFromFloat(topUp.Money).Mul(dQuotaPerUnit).IntPart())
		} else {
			dAmount := decimal.NewFromInt(topUp.Amount)
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		}
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		// 标记完成
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		// 增加用户额度（立即写库，保持一致性）
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		userId = topUp.UserId
		payMoney = topUp.Money
		paymentMethod = topUp.PaymentMethod
		return nil
	})

	if err != nil {
		return err
	}

	// 事务外记录日志，避免阻塞
	RecordTopupLog(userId, fmt.Sprintf("管理员补单成功，充值金额: %v，支付金额：%f", logger.FormatQuota(quotaToAdd), payMoney), callerIp, paymentMethod, "admin")
	return nil
}
func RechargeCreem(referenceId string, customerEmail string, customerName string, callerIp string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota int64
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderCreem {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		// Creem 直接使用 Amount 作为充值额度（整数）
		quota = topUp.Amount

		// 构建更新字段，优先使用邮箱，如果邮箱为空则使用用户名
		updateFields := map[string]interface{}{
			"quota": gorm.Expr("quota + ?", quota),
		}

		// 如果有客户邮箱，尝试更新用户邮箱（仅当用户邮箱为空时）
		if customerEmail != "" {
			// 先检查用户当前邮箱是否为空
			var user User
			err = tx.Where("id = ?", topUp.UserId).First(&user).Error
			if err != nil {
				return err
			}

			// 如果用户邮箱为空，则更新为支付时使用的邮箱
			if user.Email == "" {
				updateFields["email"] = customerEmail
			}
		}

		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(updateFields).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用Creem充值成功，充值额度: %v，支付金额：%.2f", quota, topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodCreem)

	return nil
}

func RechargeWaffo(tradeNo string, callerIp string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderWaffo {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil // 幂等：已成功直接返回
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		dAmount := decimal.NewFromInt(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	if quotaToAdd > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("Waffo充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodWaffo)
	}

	return nil
}

func RechargeWaffoPancake(tradeNo string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderWaffoPancake {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		quotaToAdd = int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("waffo pancake topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	if quotaToAdd > 0 {
		RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("Waffo Pancake充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money))
	}

	return nil
}
