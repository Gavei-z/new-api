package model

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPrepaidCreditStoresLargeBalanceWithoutOverflow(t *testing.T) {
	db := setupTeamTestDB(t)
	user := User{
		Username:    "large-prepaid-user",
		Password:    "password",
		Status:      common.UserStatusEnabled,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)

	const quota = int64(25_000_000_000)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetUser,
			user.Id,
			user.Id,
			quota,
			"stripe:checkout:large-credit",
			"Stripe checkout",
		)
	}))
	// A webhook retry must be a no-op.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetUser,
			user.Id,
			user.Id,
			quota,
			"stripe:checkout:large-credit",
			"Stripe checkout retry",
		)
	}))

	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, common.MaxQuota/2, stored.Quota)

	balance, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
	require.NoError(t, err)
	assert.Equal(t, quota, balance.TotalQuota)
	assert.Equal(t, int64(common.MaxQuota/2), balance.ActiveQuota)
	assert.Equal(t, quota-int64(common.MaxQuota/2), balance.ReserveQuota)

	var transactionCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).Count(&transactionCount).Error)
	assert.EqualValues(t, 1, transactionCount)

	err = db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetUser,
			user.Id,
			user.Id,
			quota+1,
			"stripe:checkout:large-credit",
			"conflicting retry",
		)
	})
	require.ErrorIs(t, err, ErrPrepaidIdempotencyConflict)
}

func TestPrepaidReversalIsIdempotentAndReportsSpentShortfall(t *testing.T) {
	db := setupTeamTestDB(t)
	user := User{
		Username:    "refund-prepaid-user",
		Password:    "password",
		Status:      common.UserStatusEnabled,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)

	const credited = int64(5_000_000_000)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetUser,
			user.Id,
			user.Id,
			credited,
			"stripe:checkout:refund-source",
			"Stripe checkout",
		)
	}))

	var applied, shortfall int64
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		applied, shortfall, err = ApplyPrepaidReversalTx(
			tx,
			PrepaidTargetUser,
			user.Id,
			user.Id,
			credited+2_000_000_000,
			"stripe:refund:full",
			"Stripe refund",
		)
		return err
	}))
	assert.Equal(t, credited, applied)
	assert.Equal(t, int64(2_000_000_000), shortfall)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		retryApplied, retryShortfall, err := ApplyPrepaidReversalTx(
			tx,
			PrepaidTargetUser,
			user.Id,
			user.Id,
			credited+2_000_000_000,
			"stripe:refund:full",
			"Stripe refund retry",
		)
		assert.Equal(t, applied, retryApplied)
		assert.Equal(t, shortfall, retryShortfall)
		return err
	}))

	balance, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
	require.NoError(t, err)
	assert.Zero(t, balance.TotalQuota)
	var transactionCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).Count(&transactionCount).Error)
	assert.EqualValues(t, 2, transactionCount)
}

func TestPrepaidRestorationUsesSeparateIdempotencyDomain(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{
		Name:   "Dispute Team",
		Slug:   "dispute-team",
		Status: TeamStatusEnabled,
	}
	require.NoError(t, db.Create(&team).Error)

	const quota = int64(3_000_000_000)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetTeam,
			team.Id,
			11,
			quota,
			"stripe:checkout:dispute",
			"Stripe checkout",
		)
	}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, err := ApplyPrepaidReversalTx(
			tx,
			PrepaidTargetTeam,
			team.Id,
			11,
			quota,
			"stripe:dispute:opened",
			"Stripe dispute opened",
		)
		return err
	}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidRestorationTx(
			tx,
			PrepaidTargetTeam,
			team.Id,
			11,
			quota,
			"stripe:dispute:won",
			"Stripe dispute won",
		)
	}))

	balance, err := GetPrepaidBalance(PrepaidTargetTeam, team.Id)
	require.NoError(t, err)
	assert.Equal(t, quota, balance.TotalQuota)

	var transactions []PrepaidReserveTransaction
	require.NoError(t, db.Order("id").Find(&transactions).Error)
	require.Len(t, transactions, 3)
	assert.Equal(t, PrepaidTransactionCredit, transactions[0].Type)
	assert.Equal(t, PrepaidTransactionReversal, transactions[1].Type)
	assert.Equal(t, PrepaidTransactionRestoration, transactions[2].Type)
}

func TestPreparePrepaidSpendAndTeamDebitAreAtomic(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{
		Name:   "Reserve Team",
		Slug:   "reserve-team",
		Status: TeamStatusEnabled,
	}
	require.NoError(t, db.Create(&team).Error)

	const credited = int64(4_000_000_000)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetTeam,
			team.Id,
			22,
			credited,
			"stripe:checkout:team-reserve",
			"Stripe checkout",
		)
	}))

	// Simulate the active bucket being nearly exhausted.
	require.NoError(t, db.Model(&Team{}).Where("id = ?", team.Id).Update("quota", 20).Error)
	before, err := GetPrepaidBalance(PrepaidTargetTeam, team.Id)
	require.NoError(t, err)

	require.NoError(t, ApplyTeamQuotaChange(TeamQuotaChange{
		TeamId:         team.Id,
		UserId:         22,
		ActorUserId:    22,
		Type:           TeamQuotaTypeConsume,
		QuotaDelta:     -100,
		UsedQuotaDelta: 100,
		IdempotencyKey: "team:reserve:consume",
		RequireEnabled: true,
	}))

	after, err := GetPrepaidBalance(PrepaidTargetTeam, team.Id)
	require.NoError(t, err)
	assert.Equal(t, before.TotalQuota-100, after.TotalQuota)
	assert.GreaterOrEqual(t, after.ActiveQuota, int64(0))

	var stored Team
	require.NoError(t, db.First(&stored, team.Id).Error)
	assert.EqualValues(t, 100, stored.UsedQuota)

	var releaseCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where("target_type = ? AND target_id = ? AND type = ?", PrepaidTargetTeam, team.Id, PrepaidTransactionRelease).
		Count(&releaseCount).Error)
	assert.EqualValues(t, 1, releaseCount)
}

func TestTransferUserQuotaToTeamCanUsePersonalPrepaidReserve(t *testing.T) {
	db := setupTeamTestDB(t)
	manager := User{
		Username:    "reserve-transfer-owner",
		Password:    "password",
		Status:      common.UserStatusEnabled,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(&manager).Error)
	team := Team{
		Name:   "Reserve Transfer Team",
		Slug:   "reserve-transfer-team",
		Status: TeamStatusEnabled,
	}
	require.NoError(t, db.Create(&team).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: team.Id,
		UserId: manager.Id,
		Role:   TeamMemberRoleOwner,
		Status: TeamMemberStatusEnabled,
	}).Error)

	const credited = int64(3_000_000_000)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetUser,
			manager.Id,
			manager.Id,
			credited,
			"stripe:checkout:transfer-reserve",
			"Stripe checkout",
		)
	}))
	require.NoError(t, db.Model(&User{}).Where("id = ?", manager.Id).Update("quota", 10).Error)

	const transfer = 100
	require.NoError(t, TransferUserQuotaToTeam(
		team.Id,
		manager.Id,
		transfer,
		"team-fund:reserve-transfer",
	))

	personalBalance, err := GetPrepaidBalance(PrepaidTargetUser, manager.Id)
	require.NoError(t, err)
	assert.Equal(t, credited-int64(prepaidActiveRefillTarget)+10-transfer, personalBalance.TotalQuota)

	teamBalance, err := GetPrepaidBalance(PrepaidTargetTeam, team.Id)
	require.NoError(t, err)
	assert.EqualValues(t, transfer, teamBalance.TotalQuota)

	var releaseCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
		Where(
			"target_type = ? AND target_id = ? AND type = ?",
			PrepaidTargetUser,
			manager.Id,
			PrepaidTransactionRelease,
		).
		Count(&releaseCount).Error)
	assert.EqualValues(t, 1, releaseCount)
}

func TestPrepaidCreditRollsBackWithParentTransaction(t *testing.T) {
	db := setupTeamTestDB(t)
	user := User{
		Username:    "rollback-prepaid-user",
		Password:    "password",
		Status:      common.UserStatusEnabled,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)

	sentinel := errors.New("force parent rollback")
	err := db.Transaction(func(tx *gorm.DB) error {
		require.NoError(t, ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetUser,
			user.Id,
			user.Id,
			1_000_000,
			"stripe:checkout:rollback",
			"Stripe checkout",
		))
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	balance, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
	require.NoError(t, err)
	assert.Zero(t, balance.TotalQuota)
	var transactionCount int64
	require.NoError(t, db.Model(&PrepaidReserveTransaction{}).Count(&transactionCount).Error)
	assert.Zero(t, transactionCount)
}

func TestConcurrentPersonalQuotaDeductionNeverOverdraws(t *testing.T) {
	db := setupTeamTestDB(t)
	user := User{
		Username:    "concurrent-wallet-user",
		Password:    "password",
		Status:      common.UserStatusEnabled,
		Quota:       100,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			errs <- DecreaseUserQuota(user.Id, 80, true)
		}()
	}
	close(start)
	wait.Wait()
	close(errs)

	successes := 0
	insufficient := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrUserQuotaInsufficient):
			insufficient++
		default:
			require.NoError(t, fmt.Errorf("unexpected deduction error: %w", err))
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, insufficient)

	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, 20, stored.Quota)
}

func TestTeamBalanceTransactionsIncludeStripeButHideInternalRelease(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{
		Name:   "Ledger Team",
		Slug:   "ledger-team",
		Status: TeamStatusEnabled,
	}
	require.NoError(t, db.Create(&team).Error)

	const credited = int64(2_000_000_000)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ApplyPrepaidCreditTx(
			tx,
			PrepaidTargetTeam,
			team.Id,
			91,
			credited,
			"stripe:checkout:ledger-team",
			"Stripe checkout",
		)
	}))
	consume := int(prepaidActiveRefillTarget + 100)
	require.NoError(t, ApplyTeamQuotaChange(TeamQuotaChange{
		TeamId:         team.Id,
		UserId:         91,
		ActorUserId:    91,
		Type:           TeamQuotaTypeConsume,
		QuotaDelta:     -consume,
		UsedQuotaDelta: consume,
		IdempotencyKey: "team:ledger:consume",
		RequireEnabled: true,
	}))

	rows, total, err := ListTeamBalanceTransactions(team.Id, 0, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, rows, 2)

	types := make(map[string]TeamBalanceTransactionView, len(rows))
	for _, row := range rows {
		types[row.Type] = row
		assert.NotEqual(t, PrepaidTransactionRelease, row.Type)
	}
	stripeRow, found := types["stripe_recharge"]
	require.True(t, found)
	assert.Equal(t, "stripe", stripeRow.Source)
	assert.Equal(t, credited, stripeRow.QuotaDelta)
	assert.Equal(t, credited, stripeRow.BalanceAfter)
	consumeRow, found := types[TeamQuotaTypeConsume]
	require.True(t, found)
	assert.Equal(t, "team", consumeRow.Source)
	assert.EqualValues(t, -consume, consumeRow.QuotaDelta)
	assert.Zero(t, consumeRow.ActiveBalanceAfter)
	assert.Equal(t, credited-int64(consume), consumeRow.ReserveBalanceAfter)
	assert.Equal(t, credited-int64(consume), consumeRow.BalanceAfter)

	var storedConsume TeamQuotaTransaction
	require.NoError(t, db.Where("idempotency_key = ?", "team:ledger:consume").First(&storedConsume).Error)
	assert.Zero(t, storedConsume.BalanceAfter)
	assert.Equal(t, credited-int64(consume), storedConsume.ReserveBalanceAfter)
	assert.Equal(t, credited-int64(consume), storedConsume.TotalBalanceAfter)
}

func TestTeamUsedQuotaCanExceedLegacyInt32Limit(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{
		Name:      "Bigint Usage Team",
		Slug:      "bigint-usage-team",
		Status:    TeamStatusEnabled,
		Quota:     10,
		UsedQuota: int64(common.MaxQuota) + 100,
	}
	require.NoError(t, db.Create(&team).Error)

	require.NoError(t, ApplyTeamQuotaChange(TeamQuotaChange{
		TeamId:         team.Id,
		Type:           TeamQuotaTypeConsume,
		QuotaDelta:     -1,
		UsedQuotaDelta: 1,
		IdempotencyKey: "team:bigint-used-quota",
		RequireEnabled: true,
	}))

	var stored Team
	require.NoError(t, db.First(&stored, team.Id).Error)
	assert.Equal(t, int64(common.MaxQuota)+101, stored.UsedQuota)

	var ledger TeamQuotaTransaction
	require.NoError(t, db.Where("idempotency_key = ?", "team:bigint-used-quota").First(&ledger).Error)
	assert.Equal(t, int64(common.MaxQuota)+101, ledger.UsedQuotaAfter)
}
