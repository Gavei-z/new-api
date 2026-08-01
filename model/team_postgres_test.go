package model

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This test is opt-in so the normal suite remains self-contained. CI or a
// release check can point it at an empty, disposable PostgreSQL database.
func TestTeamQuotaConcurrentDeductionPostgres(t *testing.T) {
	dsn := os.Getenv("TEAM_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEAM_TEST_POSTGRES_DSN is not configured")
	}

	previousDB := DB
	previousDatabaseType := common.MainDatabaseType()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
	require.NoError(t, db.AutoMigrate(&Team{}, &TeamMember{}, &TeamQuotaTransaction{}))
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousDatabaseType)
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})

	team := Team{
		Name:   "PostgreSQL Concurrency",
		Slug:   "postgresql-concurrency",
		Status: TeamStatusEnabled,
		Quota:  100,
	}
	require.NoError(t, db.Create(&team).Error)
	secondTeam := Team{
		Name:   "PostgreSQL Membership",
		Slug:   "postgresql-membership",
		Status: TeamStatusEnabled,
	}
	require.NoError(t, db.Create(&secondTeam).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: team.Id,
		UserId: 4242,
		Role:   TeamMemberRoleMember,
		Status: TeamMemberStatusEnabled,
	}).Error)
	require.Error(t, db.Create(&TeamMember{
		TeamId: secondTeam.Id,
		UserId: 4242,
		Role:   TeamMemberRoleMember,
		Status: TeamMemberStatusEnabled,
	}).Error)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for index := range 2 {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			errs <- ApplyTeamQuotaChange(TeamQuotaChange{
				TeamId:         team.Id,
				UserId:         index + 1,
				Type:           TeamQuotaTypeConsume,
				QuotaDelta:     -80,
				UsedQuotaDelta: 80,
				IdempotencyKey: fmt.Sprintf("postgres-concurrent-%d", index),
				RequireEnabled: true,
			})
		}(index)
	}
	close(start)
	wait.Wait()
	close(errs)

	successes := 0
	insufficient := 0
	for quotaErr := range errs {
		switch {
		case quotaErr == nil:
			successes++
		case errors.Is(quotaErr, ErrTeamQuotaInsufficient):
			insufficient++
		default:
			require.NoError(t, quotaErr)
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, insufficient)

	var stored Team
	require.NoError(t, db.First(&stored, team.Id).Error)
	assert.Equal(t, 20, stored.Quota)
	assert.Equal(t, 80, stored.UsedQuota)
	var transactionCount int64
	require.NoError(t, db.Model(&TeamQuotaTransaction{}).Count(&transactionCount).Error)
	assert.EqualValues(t, 1, transactionCount)
}
