package model

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestPrepaidReserveRealDatabases is intentionally opt-in. Each configured DSN
// must point to a disposable test database; the test migrates the real tables
// and leaves its uniquely named rows in place for post-failure inspection.
func TestPrepaidReserveRealDatabases(t *testing.T) {
	configurations := []struct {
		name         string
		environment  string
		databaseType common.DatabaseType
		open         func(string) gorm.Dialector
	}{
		{
			name:         "PostgreSQL",
			environment:  "PREPAID_TEST_POSTGRES_DSN",
			databaseType: common.DatabaseTypePostgreSQL,
			open:         func(dsn string) gorm.Dialector { return postgres.Open(dsn) },
		},
		{
			name:         "MySQL",
			environment:  "PREPAID_TEST_MYSQL_DSN",
			databaseType: common.DatabaseTypeMySQL,
			open:         func(dsn string) gorm.Dialector { return mysql.Open(dsn) },
		},
	}

	for _, configuration := range configurations {
		configuration := configuration
		t.Run(configuration.name, func(t *testing.T) {
			dsn := strings.TrimSpace(os.Getenv(configuration.environment))
			if dsn == "" {
				t.Skipf("%s is not configured", configuration.environment)
			}

			db, err := gorm.Open(configuration.open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(16)
			sqlDB.SetMaxIdleConns(4)
			require.NoError(t, sqlDB.Ping())

			previousDB, previousLogDB := DB, LOG_DB
			previousRedisEnabled := common.RedisEnabled
			previousBatchUpdateEnabled := common.BatchUpdateEnabled
			previousMainDatabaseType := common.MainDatabaseType()
			previousLogDatabaseType := common.LogDatabaseType()
			DB, LOG_DB = db, db
			common.RedisEnabled = false
			common.BatchUpdateEnabled = false
			common.SetDatabaseTypes(configuration.databaseType, configuration.databaseType)
			initCol()
			t.Cleanup(func() {
				DB, LOG_DB = previousDB, previousLogDB
				common.RedisEnabled = previousRedisEnabled
				common.BatchUpdateEnabled = previousBatchUpdateEnabled
				common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
				initCol()
				_ = sqlDB.Close()
			})

			require.NoError(t, db.AutoMigrate(
				&User{},
				&Team{},
				&TeamMember{},
				&TeamQuotaTransaction{},
				&PrepaidReserve{},
				&PrepaidReserveTransaction{},
				&TopUp{},
				&StripePaymentIntentLink{},
				&StripeAdjustmentInbox{},
				&Log{},
			))
			assert.True(t, db.Migrator().HasColumn(&PrepaidReserve{}, "quota"))
			assert.True(t, db.Migrator().HasColumn(&PrepaidReserveTransaction{}, "requested_quota"))
			assert.True(t, db.Migrator().HasColumn(&PrepaidReserveTransaction{}, "reserve_balance_after"))
			assert.True(t, db.Migrator().HasIndex(&PrepaidReserve{}, "idx_prepaid_reserve_target"))
			assertBigintColumn := func(modelValue any, columnName string) {
				columnTypes, columnErr := db.Migrator().ColumnTypes(modelValue)
				require.NoError(t, columnErr)
				for _, columnType := range columnTypes {
					if strings.EqualFold(columnType.Name(), columnName) {
						databaseType := strings.ToLower(columnType.DatabaseTypeName())
						assert.True(
							t,
							databaseType == "bigint" || databaseType == "int8",
							"expected bigint-compatible %s, got %s",
							columnName,
							databaseType,
						)
						return
					}
				}
				t.Fatalf("column %s was not found", columnName)
			}
			assertBigintColumn(&User{}, "used_quota")
			assertBigintColumn(&Team{}, "used_quota")
			assertBigintColumn(&TeamQuotaTransaction{}, "used_quota_delta")
			assertBigintColumn(&TeamQuotaTransaction{}, "used_quota_after")

			runTimestamp := time.Now().UnixNano()
			runID := fmt.Sprintf("%s-%d", strings.ToLower(configuration.name), runTimestamp)
			shortID := fmt.Sprintf("%c%d", configuration.name[0], runTimestamp)
			user := User{
				Username:    "prepaid-" + runID,
				Password:    "integration-test-password",
				Status:      common.UserStatusEnabled,
				AuthVersion: 1,
				AffCode:     "a-" + shortID,
			}
			require.NoError(t, db.Create(&user).Error)

			const largeCredit = int64(25_000_000_000)
			largeCreditKey := "prepaid-it:" + runID + ":large"
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return ApplyPrepaidCreditTx(
					tx,
					PrepaidTargetUser,
					user.Id,
					user.Id,
					largeCredit,
					largeCreditKey,
					"real database large credit",
				)
			}))

			largeBalance, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
			require.NoError(t, err)
			assert.Equal(t, largeCredit, largeBalance.TotalQuota)
			assert.Equal(t, int64(prepaidActiveRefillTarget), largeBalance.ActiveQuota)
			assert.Equal(t, largeCredit-int64(prepaidActiveRefillTarget), largeBalance.ReserveQuota)

			var reserve PrepaidReserve
			require.NoError(t, db.Where(
				"target_type = ? AND target_id = ?",
				PrepaidTargetUser,
				user.Id,
			).First(&reserve).Error)
			assert.Greater(t, reserve.Quota, int64(common.MaxQuota))

			duplicateReserve := PrepaidReserve{
				TargetType: PrepaidTargetUser,
				TargetId:   user.Id,
			}
			assert.Error(t, db.Create(&duplicateReserve).Error)

			const firstConcurrentCredit = int64(3_000_000_000)
			const secondConcurrentCredit = int64(4_000_000_000)
			concurrentKeys := []string{
				"prepaid-it:" + runID + ":different:1",
				"prepaid-it:" + runID + ":different:2",
			}
			concurrentAmounts := []int64{firstConcurrentCredit, secondConcurrentCredit}
			start := make(chan struct{})
			errs := make(chan error, len(concurrentAmounts))
			var wait sync.WaitGroup
			for index := range concurrentAmounts {
				wait.Add(1)
				go func(index int) {
					defer wait.Done()
					<-start
					errs <- db.Transaction(func(tx *gorm.DB) error {
						return ApplyPrepaidCreditTx(
							tx,
							PrepaidTargetUser,
							user.Id,
							user.Id,
							concurrentAmounts[index],
							concurrentKeys[index],
							"real database concurrent credit",
						)
					})
				}(index)
			}
			close(start)
			wait.Wait()
			close(errs)
			for creditErr := range errs {
				require.NoError(t, creditErr)
			}

			afterDifferentCredits, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
			require.NoError(t, err)
			assert.Equal(
				t,
				largeCredit+firstConcurrentCredit+secondConcurrentCredit,
				afterDifferentCredits.TotalQuota,
			)

			const repeatedCredit = int64(2_000_000_000)
			repeatedKey := "prepaid-it:" + runID + ":same"
			repeatedStart := make(chan struct{})
			repeatedErrors := make(chan error, 2)
			for range 2 {
				wait.Add(1)
				go func() {
					defer wait.Done()
					<-repeatedStart
					repeatedErrors <- db.Transaction(func(tx *gorm.DB) error {
						return ApplyPrepaidCreditTx(
							tx,
							PrepaidTargetUser,
							user.Id,
							user.Id,
							repeatedCredit,
							repeatedKey,
							"real database repeated credit",
						)
					})
				}()
			}
			close(repeatedStart)
			wait.Wait()
			close(repeatedErrors)
			for creditErr := range repeatedErrors {
				require.NoError(t, creditErr)
			}

			var repeatedTransactionCount int64
			require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
				Where("idempotency_key = ?", repeatedKey).
				Count(&repeatedTransactionCount).Error)
			assert.EqualValues(t, 1, repeatedTransactionCount)
			afterRepeatedCredit, err := GetPrepaidBalance(PrepaidTargetUser, user.Id)
			require.NoError(t, err)
			assert.Equal(
				t,
				afterDifferentCredits.TotalQuota+repeatedCredit,
				afterRepeatedCredit.TotalQuota,
			)

			conflictErr := db.Transaction(func(tx *gorm.DB) error {
				return ApplyPrepaidCreditTx(
					tx,
					PrepaidTargetUser,
					user.Id,
					user.Id,
					repeatedCredit+1,
					repeatedKey,
					"real database conflicting credit",
				)
			})
			require.ErrorIs(t, conflictErr, ErrPrepaidIdempotencyConflict)

			duplicateTransaction := PrepaidReserveTransaction{
				TargetType:          PrepaidTargetUser,
				TargetId:            user.Id,
				UserId:              user.Id,
				Type:                PrepaidTransactionCredit,
				RequestedQuota:      1,
				QuotaDelta:          1,
				ActiveBalanceAfter:  1,
				ReserveBalanceAfter: 0,
				TotalBalanceAfter:   1,
				IdempotencyKey:      repeatedKey,
			}
			assert.Error(t, db.Create(&duplicateTransaction).Error)

			team := Team{
				Name:   "Prepaid Integration " + runID,
				Slug:   "prepaid-integration-" + runID,
				Status: TeamStatusEnabled,
			}
			require.NoError(t, db.Create(&team).Error)
			const teamCredit = int64(6_000_000_000)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return ApplyPrepaidCreditTx(
					tx,
					PrepaidTargetTeam,
					team.Id,
					user.Id,
					teamCredit,
					"prepaid-it:"+runID+":team",
					"real database team credit",
				)
			}))
			require.NoError(t, db.Model(&Team{}).
				Where("id = ?", team.Id).
				Update("quota", 20).Error)
			teamBefore, err := GetPrepaidBalance(PrepaidTargetTeam, team.Id)
			require.NoError(t, err)
			require.Greater(t, teamBefore.ReserveQuota, int64(100))

			require.NoError(t, ApplyTeamQuotaChange(TeamQuotaChange{
				TeamId:         team.Id,
				UserId:         user.Id,
				ActorUserId:    user.Id,
				Type:           TeamQuotaTypeConsume,
				QuotaDelta:     -100,
				UsedQuotaDelta: 100,
				IdempotencyKey: "prepaid-it:" + runID + ":team-debit",
				RequireEnabled: true,
			}))
			teamAfter, err := GetPrepaidBalance(PrepaidTargetTeam, team.Id)
			require.NoError(t, err)
			assert.Equal(t, teamBefore.TotalQuota-100, teamAfter.TotalQuota)
			assert.GreaterOrEqual(t, teamAfter.ActiveQuota, int64(0))
			assert.GreaterOrEqual(t, teamAfter.ReserveQuota, int64(0))

			var storedTeam Team
			require.NoError(t, db.First(&storedTeam, team.Id).Error)
			assert.GreaterOrEqual(t, storedTeam.Quota, 0)
			assert.EqualValues(t, 100, storedTeam.UsedQuota)
			require.NoError(t, db.Model(&Team{}).
				Where("id = ?", team.Id).
				Update("used_quota", int64(3_000_000_000)).Error)
			require.NoError(t, db.First(&storedTeam, team.Id).Error)
			assert.Equal(t, int64(3_000_000_000), storedTeam.UsedQuota)
			require.NoError(t, db.Model(&User{}).
				Where("id = ?", user.Id).
				Update("used_quota", int64(3_000_000_000)).Error)
			var storedUser User
			require.NoError(t, db.First(&storedUser, user.Id).Error)
			assert.Equal(t, int64(3_000_000_000), storedUser.UsedQuota)
			var releaseCount int64
			require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
				Where(
					"target_type = ? AND target_id = ? AND type = ?",
					PrepaidTargetTeam,
					team.Id,
					PrepaidTransactionRelease,
				).
				Count(&releaseCount).Error)
			assert.EqualValues(t, 1, releaseCount)

			orderingUser := User{
				Username:    "ordering-" + runID,
				Password:    "integration-test-password",
				Status:      common.UserStatusEnabled,
				AuthVersion: 1,
				AffCode:     "o-" + shortID,
			}
			require.NoError(t, db.Create(&orderingUser).Error)
			orderingQuota, err := stripeExpectedQuotaForAmount(2)
			require.NoError(t, err)
			orderingTopUp := TopUp{
				UserId:              orderingUser.Id,
				Amount:              2,
				Money:               2,
				TradeNo:             "stripe-ordering-" + runID,
				PaymentMethod:       PaymentMethodStripe,
				PaymentProvider:     PaymentProviderStripe,
				CreateTime:          common.GetTimestamp(),
				Status:              common.TopUpStatusPending,
				FundingTarget:       StripeFundingTargetUser,
				FundingTargetId:     orderingUser.Id,
				ExpectedAmountMinor: 200,
				ExpectedCurrency:    "usd",
				ExpectedQuota:       orderingQuota,
			}
			require.NoError(t, InsertStripeTopUpOrder(&orderingTopUp))
			orderingPaymentIntent := "pi_ordering_" + shortID
			orderingStart := make(chan struct{})
			orderingErrors := make(chan error, 2)
			wait.Add(2)
			go func() {
				defer wait.Done()
				<-orderingStart
				_, adjustmentErr := ProcessStripeRefund(StripeRefundAdjustment{
					EventId:         "evt_ordering_refund_" + shortID,
					EventCreated:    time.Now().Unix(),
					EventType:       "refund.created",
					RefundId:        "re_ordering_" + shortID,
					PaymentIntentId: orderingPaymentIntent,
					AmountMinor:     200,
					Currency:        "usd",
					Status:          "succeeded",
				})
				orderingErrors <- adjustmentErr
			}()
			go func() {
				defer wait.Done()
				<-orderingStart
				_, completionErr := CompleteStripeTopUp(StripeCheckoutCompletion{
					EventId:          "evt_ordering_complete_" + shortID,
					TradeNo:          orderingTopUp.TradeNo,
					SessionId:        "cs_ordering_" + shortID,
					PaymentIntentId:  orderingPaymentIntent,
					Status:           "complete",
					PaymentStatus:    "paid",
					AmountTotalMinor: 200,
					Currency:         "usd",
				})
				orderingErrors <- completionErr
			}()
			close(orderingStart)
			wait.Wait()
			close(orderingErrors)
			for orderingErr := range orderingErrors {
				require.NoError(t, orderingErr)
			}

			var storedOrderingTopUp TopUp
			require.NoError(t, db.First(&storedOrderingTopUp, orderingTopUp.Id).Error)
			assert.Equal(t, common.TopUpStatusSuccess, storedOrderingTopUp.Status)
			assert.Equal(t, orderingQuota, storedOrderingTopUp.CreditedQuota)
			assert.Equal(t, orderingQuota, storedOrderingTopUp.ReversedQuota)
			orderingBalance, err := GetPrepaidBalance(PrepaidTargetUser, orderingUser.Id)
			require.NoError(t, err)
			assert.Zero(t, orderingBalance.TotalQuota)
			var orderingInbox StripeAdjustmentInbox
			require.NoError(t, db.Where(
				"payment_intent_id = ?",
				orderingPaymentIntent,
			).First(&orderingInbox).Error)
			assert.Equal(t, stripeAdjustmentInboxApplied, orderingInbox.State)
			assert.Equal(t, orderingTopUp.TradeNo, orderingInbox.TradeNo)

			rollbackUser := User{
				Username:    "rollback-" + runID,
				Password:    "integration-test-password",
				Status:      common.UserStatusEnabled,
				AuthVersion: 1,
				AffCode:     "r-" + shortID,
			}
			require.NoError(t, db.Create(&rollbackUser).Error)
			rollbackKey := "prepaid-it:" + runID + ":rollback"
			sentinel := errors.New("force real database parent rollback")
			rollbackErr := db.Transaction(func(tx *gorm.DB) error {
				require.NoError(t, ApplyPrepaidCreditTx(
					tx,
					PrepaidTargetUser,
					rollbackUser.Id,
					rollbackUser.Id,
					largeCredit,
					rollbackKey,
					"real database rollback",
				))
				return sentinel
			})
			require.ErrorIs(t, rollbackErr, sentinel)

			rollbackBalance, err := GetPrepaidBalance(PrepaidTargetUser, rollbackUser.Id)
			require.NoError(t, err)
			assert.Zero(t, rollbackBalance.TotalQuota)
			var rollbackTransactionCount int64
			require.NoError(t, db.Model(&PrepaidReserveTransaction{}).
				Where("idempotency_key = ?", rollbackKey).
				Count(&rollbackTransactionCount).Error)
			assert.Zero(t, rollbackTransactionCount)
		})
	}
}
