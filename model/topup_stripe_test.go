package model

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupStripeTopUpTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTeamTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&TopUp{},
		&StripePaymentIntentLink{},
		&StripeAdjustmentInbox{},
	))
	return db
}

func createStripeTopUpTestUser(t *testing.T, db *gorm.DB, username string) *User {
	t.Helper()
	user := &User{
		Username:    username,
		Password:    "password",
		DisplayName: username,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     username,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func insertStripeTopUpTestOrder(
	t *testing.T,
	userId int,
	targetType string,
	targetId int,
	amount int64,
	suffix string,
) *TopUp {
	t.Helper()
	expectedQuota, err := stripeExpectedQuotaForAmount(amount)
	require.NoError(t, err)
	topUp := &TopUp{
		UserId:              userId,
		Amount:              amount,
		Money:               float64(amount),
		TradeNo:             "stripe-test-" + suffix,
		PaymentMethod:       PaymentMethodStripe,
		PaymentProvider:     PaymentProviderStripe,
		CreateTime:          common.GetTimestamp(),
		Status:              common.TopUpStatusPending,
		FundingTarget:       targetType,
		FundingTargetId:     targetId,
		ExpectedAmountMinor: amount * 100,
		ExpectedCurrency:    "usd",
		ExpectedLivemode:    false,
		ExpectedQuota:       expectedQuota,
	}
	require.NoError(t, InsertStripeTopUpOrder(topUp))
	return topUp
}

func completeStripeTopUpTestOrder(t *testing.T, topUp *TopUp, suffix string) *StripeTopUpResult {
	t.Helper()
	result, err := CompleteStripeTopUp(StripeCheckoutCompletion{
		EventId:          "evt_" + suffix,
		TradeNo:          topUp.TradeNo,
		SessionId:        "cs_test_" + suffix,
		PaymentIntentId:  "pi_" + suffix,
		CustomerId:       "cus_" + suffix,
		Status:           "complete",
		PaymentStatus:    "paid",
		AmountTotalMinor: topUp.ExpectedAmountMinor,
		Currency:         "usd",
		Livemode:         false,
		ObjectLivemode:   false,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	return result
}

func TestStripeTopUpRejectsPersonalFundingForEveryTeamMember(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	manager := createStripeTopUpTestUser(t, db, "stripe-team-manager")
	team := Team{Name: "Stripe Team", Slug: "stripe-team", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&team).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: team.Id,
		UserId: manager.Id,
		Role:   TeamMemberRoleOwner,
		Status: TeamMemberStatusEnabled,
	}).Error)

	err := ValidateStripeTopUpTarget(manager.Id, StripeFundingTargetUser, manager.Id)
	require.ErrorIs(t, err, ErrTeamMemberPersonalQuota)

	expectedQuota, err := stripeExpectedQuotaForAmount(2)
	require.NoError(t, err)
	personalOrder := &TopUp{
		UserId:              manager.Id,
		Amount:              2,
		Money:               2,
		TradeNo:             "stripe-personal-bypass",
		PaymentMethod:       PaymentMethodStripe,
		PaymentProvider:     PaymentProviderStripe,
		Status:              common.TopUpStatusPending,
		FundingTarget:       StripeFundingTargetUser,
		FundingTargetId:     manager.Id,
		ExpectedAmountMinor: 200,
		ExpectedCurrency:    "usd",
		ExpectedQuota:       expectedQuota,
	}
	require.ErrorIs(t, InsertStripeTopUpOrder(personalOrder), ErrTeamMemberPersonalQuota)

	var orderCount int64
	require.NoError(t, db.Model(&TopUp{}).Where("trade_no = ?", personalOrder.TradeNo).Count(&orderCount).Error)
	assert.Zero(t, orderCount)
}

func TestStripeTopUpVerifiesCheckoutAndCreditsExactlyOnce(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-idempotent-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"idempotent",
	)

	mismatch := StripeCheckoutCompletion{
		EventId:          "evt_mismatch",
		TradeNo:          topUp.TradeNo,
		SessionId:        "cs_test_idempotent",
		PaymentIntentId:  "pi_idempotent",
		Status:           "complete",
		PaymentStatus:    "paid",
		AmountTotalMinor: 199,
		Currency:         "usd",
		Livemode:         false,
		ObjectLivemode:   false,
	}
	mismatchResult, err := CompleteStripeTopUp(mismatch)
	require.NoError(t, err)
	require.NotNil(t, mismatchResult)
	assert.True(t, mismatchResult.ReconciliationRequired)

	var stored TopUp
	require.NoError(t, db.Where("id = ?", topUp.Id).First(&stored).Error)
	assert.Equal(t, common.TopUpStatusPending, stored.Status)
	assert.True(t, stored.ReconciliationRequired)
	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Zero(t, storedUser.Quota)

	completion := mismatch
	completion.EventId = "evt_correct"
	completion.AmountTotalMinor = 200
	result, err := CompleteStripeTopUp(completion)
	require.NoError(t, err)
	assert.Equal(t, topUp.ExpectedQuota, result.CreditedQuota)

	duplicateResult, err := CompleteStripeTopUp(completion)
	require.NoError(t, err)
	assert.Equal(t, result.CreditedQuota, duplicateResult.CreditedQuota)

	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.EqualValues(t, topUp.ExpectedQuota, storedUser.Quota)
	var creditCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where("target_type = ? AND target_id = ? AND type = ?", PrepaidTargetUser, user.Id, PrepaidTransactionCredit).
		Count(&creditCount).Error)
	assert.EqualValues(t, 1, creditCount)
}

func TestStripeSuccessfulCheckoutMismatchFailsClosedAndIsHandled(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-completion-fail-closed")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"completion-fail-closed",
	)
	completeStripeTopUpTestOrder(t, topUp, "completion_fail_closed")

	mismatch := StripeCheckoutCompletion{
		EventId:          "evt_completion_mismatch",
		TradeNo:          topUp.TradeNo,
		SessionId:        "cs_test_completion_fail_closed",
		PaymentIntentId:  "pi_completion_fail_closed",
		Status:           "complete",
		PaymentStatus:    "paid",
		AmountTotalMinor: topUp.ExpectedAmountMinor - 1,
		Currency:         "usd",
		Livemode:         false,
		ObjectLivemode:   false,
	}
	result, err := CompleteStripeTopUp(mismatch)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.DisabledByStripeAdjustment)
	assert.True(t, result.ReconciliationRequired)
	assert.Equal(t, topUp.ExpectedQuota, result.ReversedQuota)

	replayed, err := CompleteStripeTopUp(mismatch)
	require.NoError(t, err)
	require.NotNil(t, replayed)
	assert.Equal(t, result.ReversedQuota, replayed.ReversedQuota)

	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Zero(t, storedUser.Quota)
	assert.Equal(t, common.UserStatusDisabled, storedUser.Status)
	assert.EqualValues(t, 2, storedUser.AuthVersion)

	var reversalCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where(
			"target_type = ? AND target_id = ? AND type = ?",
			PrepaidTargetUser,
			user.Id,
			PrepaidTransactionReversal,
		).
		Count(&reversalCount).Error)
	assert.EqualValues(t, 1, reversalCount)
}

func TestStripeMaximumTopUpUsesBigintReserveWithoutOverflow(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-maximum-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		StripeMaximumTopUpUSD,
		"maximum",
	)
	require.Greater(t, topUp.ExpectedQuota, int64(common.MaxQuota))

	result := completeStripeTopUpTestOrder(t, topUp, "maximum")
	assert.Equal(t, topUp.ExpectedQuota, result.CreditedQuota)
	balance, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
	require.NoError(t, err)
	assert.Equal(t, topUp.ExpectedQuota, balance.TotalQuota)
	assert.LessOrEqual(t, balance.ActiveQuota, int64(common.MaxQuota))
	assert.Positive(t, balance.ReserveQuota)
}

func TestStripeTeamTopUpCreditsTeamLedgerWithoutPersonalBalance(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	manager := createStripeTopUpTestUser(t, db, "stripe-team-credit-manager")
	team := Team{Name: "Stripe Credits", Slug: "stripe-credits", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&team).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: team.Id,
		UserId: manager.Id,
		Role:   TeamMemberRoleAdmin,
		Status: TeamMemberStatusEnabled,
	}).Error)

	topUp := insertStripeTopUpTestOrder(
		t,
		manager.Id,
		StripeFundingTargetTeam,
		team.Id,
		2,
		"team-credit",
	)
	result := completeStripeTopUpTestOrder(t, topUp, "team_credit")
	assert.Equal(t, StripeFundingTargetTeam, result.FundingTarget)
	assert.Equal(t, team.Id, result.FundingTargetId)

	var storedTeam Team
	require.NoError(t, db.First(&storedTeam, team.Id).Error)
	assert.EqualValues(t, topUp.ExpectedQuota, storedTeam.Quota)
	var storedManager User
	require.NoError(t, db.First(&storedManager, manager.Id).Error)
	assert.Zero(t, storedManager.Quota)

	var ledger PrepaidReserveTransaction
	require.NoError(t, db.Where(
		"target_type = ? AND target_id = ? AND type = ?",
		PrepaidTargetTeam,
		team.Id,
		PrepaidTransactionCredit,
	).First(&ledger).Error)
	assert.Equal(t, topUp.ExpectedQuota, ledger.RequestedQuota)
	assert.Equal(t, topUp.ExpectedQuota, ledger.QuotaDelta)
}

func TestStripeDisputeShortfallNeverGoesNegativeAndWonRestoresOnlyApplied(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-dispute-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"dispute",
	)
	completeStripeTopUpTestOrder(t, topUp, "dispute")
	session := UserSession{
		SID:             "stripe-shortfall-session",
		UserID:          user.Id,
		Version:         1,
		UserAuthVersion: 1,
		Status:          UserSessionStatusActive,
		RefreshHash:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		LoginMethod:     "password",
		LastActiveAt:    time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, db.Create(&session).Error)

	// Simulate nearly all credit having been consumed before the dispute.
	require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).Update("quota", 100).Error)
	created := StripeDisputeAdjustment{
		EventId:         "evt_dispute_created",
		EventCreated:    100,
		EventType:       "charge.dispute.created",
		DisputeId:       "dp_shortfall",
		PaymentIntentId: "pi_dispute",
		AmountMinor:     200,
		Currency:        "usd",
		Status:          "needs_response",
		Livemode:        false,
	}
	result, err := ProcessStripeDispute(created)
	require.NoError(t, err)
	assert.Equal(t, topUp.ExpectedQuota, result.ReversedQuota)
	assert.True(t, result.ReconciliationRequired)

	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Zero(t, storedUser.Quota)
	assert.Equal(t, common.UserStatusDisabled, storedUser.Status)
	assert.EqualValues(t, 2, storedUser.AuthVersion)
	var revokedSession UserSession
	require.NoError(t, db.Where("sid = ?", session.SID).First(&revokedSession).Error)
	assert.Equal(t, UserSessionStatusRevoked, revokedSession.Status)
	assert.Equal(t, "stripe_adjustment_shortfall", revokedSession.RevokedReason)

	var stored TopUp
	require.NoError(t, db.First(&stored, topUp.Id).Error)
	assert.EqualValues(t, 100, stored.AppliedReversedQuota)
	assert.True(t, stored.DisabledByStripeAdjustment)

	// A duplicate created event must not deduct again.
	duplicate, err := ProcessStripeDispute(created)
	require.NoError(t, err)
	assert.Equal(t, result.ReversedQuota, duplicate.ReversedQuota)

	closed := created
	closed.EventId = "evt_dispute_won"
	closed.EventCreated = 200
	closed.EventType = "charge.dispute.closed"
	closed.Status = "won"
	wonResult, err := ProcessStripeDispute(closed)
	require.NoError(t, err)
	assert.Zero(t, wonResult.ReversedQuota)
	assert.True(t, wonResult.ReconciliationRequired, "manual review remains required before re-enabling")

	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.EqualValues(t, 100, storedUser.Quota, "only the 100 actually reversed quota may be restored")
	assert.Equal(t, common.UserStatusDisabled, storedUser.Status)
	require.NoError(t, db.First(&stored, topUp.Id).Error)
	assert.Zero(t, stored.AppliedReversedQuota)

	staleCreated := created
	staleCreated.EventId = "evt_dispute_created_late"
	staleCreated.EventCreated = closed.EventCreated
	staleResult, err := ProcessStripeDispute(staleCreated)
	require.NoError(t, err)
	assert.Zero(t, staleResult.ReversedQuota, "a terminal dispute state cannot be reopened by an out-of-order created event")
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.EqualValues(t, 100, storedUser.Quota)

	var reversal PrepaidReserveTransaction
	require.NoError(t, db.Where(
		"target_type = ? AND target_id = ? AND type = ?",
		PrepaidTargetUser,
		user.Id,
		PrepaidTransactionReversal,
	).First(&reversal).Error)
	assert.EqualValues(t, 100, -reversal.QuotaDelta)
	assert.Equal(t, topUp.ExpectedQuota-100, reversal.ShortfallQuota)

	var restoration PrepaidReserveTransaction
	require.NoError(t, db.Where(
		"target_type = ? AND target_id = ? AND type = ?",
		PrepaidTargetUser,
		user.Id,
		PrepaidTransactionRestoration,
	).First(&restoration).Error)
	assert.EqualValues(t, 100, restoration.RequestedQuota)
	assert.EqualValues(t, 100, restoration.QuotaDelta)
}

func TestStripePermanentAnomalyAfterValidShortfallPreventsRestoration(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-shortfall-quarantine-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"shortfall-quarantine",
	)
	completeStripeTopUpTestOrder(t, topUp, "shortfall_quarantine")
	require.NoError(t, db.Model(&User{}).
		Where("id = ?", user.Id).
		Update("quota", 100).Error)

	opened := StripeDisputeAdjustment{
		EventId:         "evt_shortfall_quarantine_opened",
		EventCreated:    100,
		EventType:       "charge.dispute.created",
		DisputeId:       "dp_shortfall_quarantine",
		PaymentIntentId: "pi_shortfall_quarantine",
		AmountMinor:     200,
		Currency:        "usd",
		Status:          "needs_response",
	}
	openedResult, err := ProcessStripeDispute(opened)
	require.NoError(t, err)
	require.NotNil(t, openedResult)
	assert.True(t, openedResult.DisabledByStripeAdjustment)
	assert.False(t, openedResult.StripeAdjustmentQuarantined)

	invalidResult, err := ProcessStripeRefund(StripeRefundAdjustment{
		EventId:         "evt_shortfall_quarantine_invalid",
		EventCreated:    200,
		EventType:       "refund.created",
		RefundId:        "re_shortfall_quarantine_invalid",
		PaymentIntentId: "pi_shortfall_quarantine",
		AmountMinor:     200,
		Currency:        "eur",
		Status:          "succeeded",
	})
	require.NoError(t, err)
	require.NotNil(t, invalidResult)
	assert.True(t, invalidResult.StripeAdjustmentQuarantined)

	var stored TopUp
	require.NoError(t, db.First(&stored, topUp.Id).Error)
	assert.True(t, stored.StripeAdjustmentQuarantined)
	assert.Equal(t, topUp.ExpectedQuota, stored.ReversedQuota)
	assert.EqualValues(t, 100, stored.AppliedReversedQuota)

	won := opened
	won.EventId = "evt_shortfall_quarantine_won"
	won.EventCreated = 300
	won.EventType = "charge.dispute.closed"
	won.Status = "won"
	wonResult, err := ProcessStripeDispute(won)
	require.NoError(t, err)
	require.NotNil(t, wonResult)
	assert.Equal(t, topUp.ExpectedQuota, wonResult.ReversedQuota)
	assert.True(t, wonResult.StripeAdjustmentQuarantined)

	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Zero(t, storedUser.Quota)
	var restorationCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where(
			"target_type = ? AND target_id = ? AND type = ?",
			PrepaidTargetUser,
			user.Id,
			PrepaidTransactionRestoration,
		).
		Count(&restorationCount).Error)
	assert.Zero(t, restorationCount)
}

func TestStripeRefundReplayAndPartialRefundsUseCumulativeExactQuota(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-refund-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"refund",
	)
	completeStripeTopUpTestOrder(t, topUp, "refund")

	first := StripeRefundAdjustment{
		EventId:         "evt_refund_one",
		EventCreated:    100,
		RefundId:        "re_one",
		PaymentIntentId: "pi_refund",
		AmountMinor:     100,
		Currency:        "usd",
		Status:          "succeeded",
		Livemode:        false,
	}
	result, err := ProcessStripeRefund(first)
	require.NoError(t, err)
	assert.Equal(t, topUp.ExpectedQuota/2, result.ReversedQuota)
	duplicate, err := ProcessStripeRefund(first)
	require.NoError(t, err)
	assert.Equal(t, result.ReversedQuota, duplicate.ReversedQuota)

	second := first
	second.EventId = "evt_refund_two"
	second.EventCreated = 200
	second.RefundId = "re_two"
	result, err = ProcessStripeRefund(second)
	require.NoError(t, err)
	assert.Equal(t, topUp.ExpectedQuota, result.ReversedQuota)

	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Zero(t, storedUser.Quota)
	var reversalCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where("target_type = ? AND target_id = ? AND type = ?", PrepaidTargetUser, user.Id, PrepaidTransactionReversal).
		Count(&reversalCount).Error)
	assert.EqualValues(t, 2, reversalCount)

	var totalReversed int64
	var reversals []PrepaidReserveTransaction
	require.NoError(t, db.Where(
		"target_type = ? AND target_id = ? AND type = ?",
		PrepaidTargetUser,
		user.Id,
		PrepaidTransactionReversal,
	).Find(&reversals).Error)
	for _, reversal := range reversals {
		totalReversed += -reversal.QuotaDelta
	}
	assert.Equal(t, topUp.ExpectedQuota, totalReversed)
}

func TestStripeAdjustmentsQueuedBeforeCheckoutCompletionAreApplied(t *testing.T) {
	tests := []struct {
		name    string
		process func(paymentIntentId string) (*StripeTopUpResult, error)
	}{
		{
			name: "refund",
			process: func(paymentIntentId string) (*StripeTopUpResult, error) {
				return ProcessStripeRefund(StripeRefundAdjustment{
					EventId:         "evt_out_of_order_refund",
					EventCreated:    100,
					EventType:       "refund.created",
					RefundId:        "re_out_of_order",
					PaymentIntentId: paymentIntentId,
					AmountMinor:     200,
					Currency:        "usd",
					Status:          "succeeded",
					Livemode:        false,
				})
			},
		},
		{
			name: "dispute",
			process: func(paymentIntentId string) (*StripeTopUpResult, error) {
				return ProcessStripeDispute(StripeDisputeAdjustment{
					EventId:         "evt_out_of_order_dispute",
					EventCreated:    100,
					EventType:       "charge.dispute.created",
					DisputeId:       "dp_out_of_order",
					PaymentIntentId: paymentIntentId,
					AmountMinor:     200,
					Currency:        "usd",
					Status:          "needs_response",
					Livemode:        false,
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupStripeTopUpTestDB(t)
			user := createStripeTopUpTestUser(t, db, "stripe-out-of-order-"+test.name)
			topUp := insertStripeTopUpTestOrder(
				t,
				user.Id,
				StripeFundingTargetUser,
				user.Id,
				2,
				"out-of-order-"+test.name,
			)
			paymentIntentId := "pi_out_of_order_" + test.name

			queued, err := test.process(paymentIntentId)
			require.NoError(t, err)
			assert.Nil(t, queued)

			var pending StripeAdjustmentInbox
			require.NoError(t, db.Where(
				"payment_intent_id = ?",
				paymentIntentId,
			).First(&pending).Error)
			assert.Equal(t, stripeAdjustmentInboxPending, pending.State)
			assert.Empty(t, pending.TradeNo)

			result, err := CompleteStripeTopUp(StripeCheckoutCompletion{
				EventId:          "evt_complete_out_of_order_" + test.name,
				TradeNo:          topUp.TradeNo,
				SessionId:        "cs_out_of_order_" + test.name,
				PaymentIntentId:  paymentIntentId,
				Status:           "complete",
				PaymentStatus:    "paid",
				AmountTotalMinor: topUp.ExpectedAmountMinor,
				Currency:         "usd",
				Livemode:         false,
				ObjectLivemode:   false,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, topUp.ExpectedQuota, result.CreditedQuota)
			assert.Equal(t, topUp.ExpectedQuota, result.ReversedQuota)

			balance, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
			require.NoError(t, err)
			assert.Zero(t, balance.TotalQuota)

			require.NoError(t, db.Where(
				"event_id = ?",
				pending.EventId,
			).First(&pending).Error)
			assert.Equal(t, stripeAdjustmentInboxApplied, pending.State)
			assert.Equal(t, topUp.TradeNo, pending.TradeNo)

			var link StripePaymentIntentLink
			require.NoError(t, db.First(&link, "payment_intent_id = ?", paymentIntentId).Error)
			assert.Equal(t, topUp.TradeNo, link.TradeNo)
		})
	}
}

func TestStripeAdjustmentStateIsBoundedForPortableTextColumns(t *testing.T) {
	topUp := &TopUp{}
	state := &stripeAdjustmentState{
		Refunds: map[string]stripeAdjustmentRecord{
			"re_oversized": {
				EventId: strings.Repeat("x", 60000),
			},
		},
		Disputes: map[string]stripeAdjustmentRecord{},
	}
	require.ErrorContains(t, saveStripeAdjustmentState(topUp, state), "too large")
	assert.Empty(t, topUp.StripeAdjustmentState)
}

func TestStripeAdjustmentAggregateOverflowFailsClosedAndIsHandledOnce(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-fail-closed-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"fail-closed",
	)
	completeStripeTopUpTestOrder(t, topUp, "fail_closed")
	session := UserSession{
		SID:             "stripe-fail-closed-session",
		UserID:          user.Id,
		Version:         1,
		UserAuthVersion: 1,
		Status:          UserSessionStatusActive,
		RefreshHash:     strings.Repeat("b", 64),
		LoginMethod:     "password",
		LastActiveAt:    time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, db.Create(&session).Error)

	_, err := ProcessStripeRefund(StripeRefundAdjustment{
		EventId:         "evt_fail_closed_refund",
		EventCreated:    100,
		RefundId:        "re_fail_closed",
		PaymentIntentId: "pi_fail_closed",
		AmountMinor:     150,
		Currency:        "usd",
		Status:          "succeeded",
	})
	require.NoError(t, err)

	overflow := StripeDisputeAdjustment{
		EventId:         "evt_fail_closed_dispute",
		EventCreated:    200,
		EventType:       "charge.dispute.created",
		DisputeId:       "dp_fail_closed",
		PaymentIntentId: "pi_fail_closed",
		AmountMinor:     100,
		Currency:        "usd",
		Status:          "needs_response",
	}
	result, err := ProcessStripeDispute(overflow)
	require.NoError(t, err, "a committed fail-closed anomaly is webhook-handled")
	require.NotNil(t, result)
	assert.Equal(t, topUp.ExpectedQuota, result.ReversedQuota)
	assert.True(t, result.ReconciliationRequired)
	assert.True(t, result.DisabledByStripeAdjustment)

	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Zero(t, storedUser.Quota)
	assert.Equal(t, common.UserStatusDisabled, storedUser.Status)
	assert.EqualValues(t, 2, storedUser.AuthVersion)
	var storedSession UserSession
	require.NoError(t, db.Where("sid = ?", session.SID).First(&storedSession).Error)
	assert.Equal(t, UserSessionStatusRevoked, storedSession.Status)

	var reversalCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where("target_type = ? AND target_id = ? AND type = ?", PrepaidTargetUser, user.Id, PrepaidTransactionReversal).
		Count(&reversalCount).Error)
	assert.EqualValues(t, 2, reversalCount)

	duplicate, err := ProcessStripeDispute(overflow)
	require.NoError(t, err)
	assert.Equal(t, topUp.ExpectedQuota, duplicate.ReversedQuota)
	var duplicateCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where("target_type = ? AND target_id = ? AND type = ?", PrepaidTargetUser, user.Id, PrepaidTransactionReversal).
		Count(&duplicateCount).Error)
	assert.Equal(t, reversalCount, duplicateCount)
}

func TestStripeFailClosedShortfallReplayIsHandled(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-fail-closed-shortfall")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"fail-closed-shortfall",
	)
	completeStripeTopUpTestOrder(t, topUp, "fail_closed_shortfall")

	const remainingQuota = 10
	require.NoError(t, db.Model(&User{}).
		Where("id = ?", user.Id).
		Update("quota", remainingQuota).Error)

	adjustment := StripeRefundAdjustment{
		EventId:         "evt_fail_closed_shortfall",
		EventCreated:    100,
		RefundId:        "re_fail_closed_shortfall",
		PaymentIntentId: "pi_fail_closed_shortfall",
		AmountMinor:     topUp.ExpectedAmountMinor,
		Currency:        "eur",
		Status:          "succeeded",
	}
	first, err := ProcessStripeRefund(adjustment)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.True(t, first.DisabledByStripeAdjustment)
	assert.True(t, first.ReconciliationRequired)
	assert.Equal(t, topUp.ExpectedQuota, first.ReversedQuota)

	replayed, err := ProcessStripeRefund(adjustment)
	require.NoError(t, err)
	require.NotNil(t, replayed)
	assert.Equal(t, first.ReversedQuota, replayed.ReversedQuota)

	var reversalCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where(
			"target_type = ? AND target_id = ? AND type = ?",
			PrepaidTargetUser,
			user.Id,
			PrepaidTransactionReversal,
		).
		Count(&reversalCount).Error)
	assert.EqualValues(t, 1, reversalCount)

	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, storedUser.Status)
	assert.EqualValues(t, 2, storedUser.AuthVersion)
}

func TestStripePermanentAdjustmentMismatchFreezesOnlyMatchedPaymentIntent(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-fail-closed-match-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"fail-closed-match",
	)
	completeStripeTopUpTestOrder(t, topUp, "fail_closed_match")

	adjustment := StripeRefundAdjustment{
		EventId:         "evt_fail_closed_match",
		EventCreated:    100,
		RefundId:        "re_fail_closed_match",
		PaymentIntentId: "pi_unrelated",
		AmountMinor:     200,
		Currency:        "eur",
		Status:          "succeeded",
	}
	unmatched, err := ProcessStripeRefund(adjustment)
	require.NoError(t, err)
	assert.Nil(t, unmatched)
	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Equal(t, common.UserStatusEnabled, storedUser.Status)

	adjustment.EventId = "evt_fail_closed_matched"
	adjustment.RefundId = "re_fail_closed_matched"
	adjustment.PaymentIntentId = "pi_fail_closed_match"
	result, err := ProcessStripeRefund(adjustment)
	require.NoError(t, err)
	assert.True(t, result.DisabledByStripeAdjustment)
	assert.True(t, result.ReconciliationRequired)
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, storedUser.Status)
	assert.Zero(t, storedUser.Quota)
}

func TestStripeDisputeLateWinRestoresAndPreventedIsInactiveTerminal(t *testing.T) {
	db := setupStripeTopUpTestDB(t)
	user := createStripeTopUpTestUser(t, db, "stripe-late-win-user")
	topUp := insertStripeTopUpTestOrder(
		t,
		user.Id,
		StripeFundingTargetUser,
		user.Id,
		2,
		"late-win",
	)
	completeStripeTopUpTestOrder(t, topUp, "late_win")

	lost := StripeDisputeAdjustment{
		EventId:         "evt_lost",
		EventCreated:    100,
		EventType:       "charge.dispute.closed",
		DisputeId:       "dp_late_win",
		PaymentIntentId: "pi_late_win",
		AmountMinor:     200,
		Currency:        "usd",
		Status:          "lost",
	}
	lostResult, err := ProcessStripeDispute(lost)
	require.NoError(t, err)
	assert.Equal(t, topUp.ExpectedQuota, lostResult.ReversedQuota)

	won := lost
	won.EventId = "evt_late_won"
	won.EventCreated = 200
	won.Status = "won"
	wonResult, err := ProcessStripeDispute(won)
	require.NoError(t, err)
	assert.Zero(t, wonResult.ReversedQuota)
	var storedUser User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.EqualValues(t, topUp.ExpectedQuota, storedUser.Quota)
	assert.Equal(t, common.UserStatusEnabled, storedUser.Status)

	staleLost := lost
	staleLost.EventId = "evt_stale_lost"
	staleLost.EventCreated = 150
	staleResult, err := ProcessStripeDispute(staleLost)
	require.NoError(t, err)
	assert.Zero(t, staleResult.ReversedQuota)
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.EqualValues(t, topUp.ExpectedQuota, storedUser.Quota)

	preventedUser := createStripeTopUpTestUser(t, db, "stripe-prevented-user")
	preventedTopUp := insertStripeTopUpTestOrder(
		t,
		preventedUser.Id,
		StripeFundingTargetUser,
		preventedUser.Id,
		2,
		"prevented",
	)
	completeStripeTopUpTestOrder(t, preventedTopUp, "prevented")
	preventedResult, err := ProcessStripeDispute(StripeDisputeAdjustment{
		EventId:         "evt_prevented",
		EventCreated:    100,
		EventType:       "charge.dispute.closed",
		DisputeId:       "dp_prevented",
		PaymentIntentId: "pi_prevented",
		AmountMinor:     200,
		Currency:        "usd",
		Status:          "prevented",
	})
	require.NoError(t, err)
	assert.Zero(t, preventedResult.ReversedQuota)
	var preventedStored User
	require.NoError(t, db.First(&preventedStored, preventedUser.Id).Error)
	assert.EqualValues(t, preventedTopUp.ExpectedQuota, preventedStored.Quota)
	assert.Equal(t, common.UserStatusEnabled, preventedStored.Status)
}
