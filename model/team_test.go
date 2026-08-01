package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTeamTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousBatchUpdateEnabled := common.BatchUpdateEnabled
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()

	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&User{},
		&UserSession{},
		&Token{},
		&Team{},
		&TeamMember{},
		&TeamQuotaTransaction{},
		&Log{},
	))

	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.BatchUpdateEnabled = previousBatchUpdateEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
	})
	return db
}

func TestTeamQuotaConcurrentDeductionNeverOverdraws(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{Name: "Concurrency", Slug: "concurrency", Status: TeamStatusEnabled, Quota: 100}
	require.NoError(t, db.Create(&team).Error)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
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
				IdempotencyKey: fmt.Sprintf("concurrent-%d", index),
				RequireEnabled: true,
			})
		}(index)
	}
	close(start)
	wait.Wait()
	close(errs)

	successes := 0
	insufficient := 0
	for err := range errs {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrTeamQuotaInsufficient) {
			insufficient++
		} else {
			require.NoError(t, err)
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, insufficient)

	var stored Team
	require.NoError(t, db.First(&stored, team.Id).Error)
	assert.Equal(t, 20, stored.Quota)
	assert.Equal(t, 80, stored.UsedQuota)

	var transactionCount int64
	require.NoError(t, db.Model(&TeamQuotaTransaction{}).Where("team_id = ?", team.Id).Count(&transactionCount).Error)
	assert.EqualValues(t, 1, transactionCount)
}

func TestTeamQuotaIdempotencyDoesNotDoubleCharge(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{Name: "Idempotency", Slug: "idempotency", Status: TeamStatusEnabled, Quota: 100}
	require.NoError(t, db.Create(&team).Error)
	change := TeamQuotaChange{
		TeamId:         team.Id,
		UserId:         7,
		Type:           TeamQuotaTypeConsume,
		QuotaDelta:     -30,
		UsedQuotaDelta: 30,
		IdempotencyKey: "same-request",
		RequireEnabled: true,
	}
	require.NoError(t, ApplyTeamQuotaChange(change))
	require.NoError(t, ApplyTeamQuotaChange(change))

	var stored Team
	require.NoError(t, db.First(&stored, team.Id).Error)
	assert.Equal(t, 70, stored.Quota)
	assert.Equal(t, 30, stored.UsedQuota)
	var transactionCount int64
	require.NoError(t, db.Model(&TeamQuotaTransaction{}).Count(&transactionCount).Error)
	assert.EqualValues(t, 1, transactionCount)

	conflicting := change
	conflicting.QuotaDelta = -31
	conflicting.UsedQuotaDelta = 31
	require.ErrorIs(t, ApplyTeamQuotaChange(conflicting), ErrTeamIdempotencyConflict)
	require.NoError(t, db.First(&stored, team.Id).Error)
	assert.Equal(t, 70, stored.Quota)
	assert.Equal(t, 30, stored.UsedQuota)
}

func TestDisabledTeamRejectsNewConsumptionButAcceptsRefund(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{Name: "Disabled", Slug: "disabled", Status: TeamStatusDisabled, Quota: 20, UsedQuota: 80}
	require.NoError(t, db.Create(&team).Error)

	err := ApplyTeamQuotaChange(TeamQuotaChange{
		TeamId:         team.Id,
		Type:           TeamQuotaTypeConsume,
		QuotaDelta:     -10,
		UsedQuotaDelta: 10,
		IdempotencyKey: "disabled-consume",
		RequireEnabled: true,
	})
	require.ErrorIs(t, err, ErrTeamDisabled)

	require.NoError(t, ApplyTeamQuotaChange(TeamQuotaChange{
		TeamId:         team.Id,
		Type:           TeamQuotaTypeRefund,
		QuotaDelta:     10,
		UsedQuotaDelta: -10,
		IdempotencyKey: "disabled-refund",
	}))
	var stored Team
	require.NoError(t, db.First(&stored, team.Id).Error)
	assert.Equal(t, 30, stored.Quota)
	assert.Equal(t, 70, stored.UsedQuota)
}

func TestTeamWalletTransferIsAtomicAndIdempotent(t *testing.T) {
	db := setupTeamTestDB(t)
	useUserCacheMiniRedis(t)
	manager := User{
		Username: "team-owner", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Quota: 100, AuthVersion: 1,
	}
	require.NoError(t, db.Create(&manager).Error)
	require.NoError(t, populateUserCache(manager))
	team := Team{Name: "Funded", Slug: "funded", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&team).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: team.Id, UserId: manager.Id, Role: TeamMemberRoleOwner, Status: TeamMemberStatusEnabled,
	}).Error)

	require.NoError(t, TransferUserQuotaToTeam(team.Id, manager.Id, 60, "fund-once"))
	require.NoError(t, TransferUserQuotaToTeam(team.Id, manager.Id, 60, "fund-once"))
	err := TransferUserQuotaToTeam(team.Id, manager.Id, 50, "fund-too-much")
	require.Error(t, err)

	var storedUser User
	require.NoError(t, db.First(&storedUser, manager.Id).Error)
	assert.Equal(t, 40, storedUser.Quota)
	var storedTeam Team
	require.NoError(t, db.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 60, storedTeam.Quota)
	var transactionCount int64
	require.NoError(t, db.Model(&TeamQuotaTransaction{}).Count(&transactionCount).Error)
	assert.EqualValues(t, 1, transactionCount)
	cachedQuota, err := common.RDB.HGet(t.Context(), getUserCacheKey(manager.Id), "Quota").Int()
	require.NoError(t, err)
	assert.Equal(t, 40, cachedQuota, "an idempotent retry must not decrement the Redis cache twice")
}

func TestTeamManagerCannotManageAnotherTenant(t *testing.T) {
	db := setupTeamTestDB(t)
	actor := User{Username: "owner-a", Password: "password", Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "owner-a"}
	target := User{Username: "member-b", Password: "password", Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "member-b"}
	require.NoError(t, db.Create(&actor).Error)
	require.NoError(t, db.Create(&target).Error)
	teamA := Team{Name: "A", Slug: "team-a", Status: TeamStatusEnabled}
	teamB := Team{Name: "B", Slug: "team-b", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&teamA).Error)
	require.NoError(t, db.Create(&teamB).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: teamA.Id, UserId: actor.Id, Role: TeamMemberRoleOwner, Status: TeamMemberStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: teamB.Id, UserId: target.Id, Role: TeamMemberRoleMember, Status: TeamMemberStatusEnabled,
	}).Error)

	err := SetTeamMemberStatus(teamB.Id, actor.Id, target.Id, TeamMemberStatusDisabled)
	require.ErrorIs(t, err, ErrTeamAccessDenied)

	var stored User
	require.NoError(t, db.First(&stored, target.Id).Error)
	assert.Equal(t, common.UserStatusEnabled, stored.Status)
}

func TestUserCannotBelongToMultipleTeams(t *testing.T) {
	db := setupTeamTestDB(t)
	teamA := Team{Name: "Membership A", Slug: "membership-a", Status: TeamStatusEnabled}
	teamB := Team{Name: "Membership B", Slug: "membership-b", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&teamA).Error)
	require.NoError(t, db.Create(&teamB).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: teamA.Id,
		UserId: 4242,
		Role:   TeamMemberRoleMember,
		Status: TeamMemberStatusEnabled,
	}).Error)
	require.Error(t, db.Create(&TeamMember{
		TeamId: teamB.Id,
		UserId: 4242,
		Role:   TeamMemberRoleMember,
		Status: TeamMemberStatusEnabled,
	}).Error)
}

func TestTeamUsageDoesNotExposeAnotherTenant(t *testing.T) {
	db := setupTeamTestDB(t)
	teamA := Team{Name: "Usage A", Slug: "usage-a", Status: TeamStatusEnabled}
	teamB := Team{Name: "Usage B", Slug: "usage-b", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&teamA).Error)
	require.NoError(t, db.Create(&teamB).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: teamA.Id,
		UserId: 101,
		Role:   TeamMemberRoleMember,
		Status: TeamMemberStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&TeamMember{
		TeamId: teamB.Id,
		UserId: 202,
		Role:   TeamMemberRoleMember,
		Status: TeamMemberStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&Log{
		UserId: 101, Username: "tenant-a-user", ModelName: "gpt-a",
		Type: LogTypeConsume, Quota: 10, PromptTokens: 5, CompletionTokens: 2,
	}).Error)
	require.NoError(t, db.Create(&Log{
		UserId: 202, Username: "tenant-b-user", ModelName: "gpt-b",
		Type: LogTypeConsume, Quota: 99, PromptTokens: 50, CompletionTokens: 20,
	}).Error)

	rows, err := GetTeamUsage(teamA.Id, 0, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 101, rows[0].UserId)
	assert.Equal(t, "tenant-a-user", rows[0].Username)
	assert.EqualValues(t, 10, rows[0].Quota)
}

func TestRootOverrideCreatesAndManagesMemberInAnyTeam(t *testing.T) {
	db := setupTeamTestDB(t)
	team := Team{Name: "Root Managed", Slug: "root-managed", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&team).Error)
	member := &User{
		Username: "root-created-member",
		Password: "password123",
		Email:    "member@example.com",
		AffCode:  "root-created-member",
	}

	require.NoError(t, CreateTeamMemberUserByRoot(team.Id, 999, member))
	var membership TeamMember
	require.NoError(t, db.Where("user_id = ?", member.Id).First(&membership).Error)
	assert.Equal(t, team.Id, membership.TeamId)
	assert.Equal(t, TeamMemberRoleMember, membership.Role)
	var stored User
	require.NoError(t, db.First(&stored, member.Id).Error)
	assert.Zero(t, stored.Quota)
	assert.Equal(t, common.RoleCommonUser, stored.Role)

	require.NoError(t, SetTeamMemberStatusByRoot(
		team.Id,
		999,
		member.Id,
		TeamMemberStatusDisabled,
	))
	require.NoError(t, db.First(&stored, member.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, stored.Status)
	assert.Greater(t, stored.AuthVersion, int64(0))
	context, err := GetTeamContextByUserId(member.Id)
	require.NoError(t, err)
	require.NotNil(t, context)
	assert.Equal(t, TeamMemberStatusDisabled, context.MemberStatus)
	assert.False(t, context.IsManager)
	require.ErrorIs(t, DeleteUserById(member.Id), ErrTeamMemberCannotDelete)
	require.ErrorIs(t, HardDeleteUserById(member.Id), ErrTeamMemberCannotDelete)
	require.ErrorIs(t, member.Delete(), ErrTeamMemberCannotDelete)
	require.ErrorIs(t, member.HardDelete(), ErrTeamMemberCannotDelete)
}

func TestCreateTeamOwnerAndMembersStayInsideTenant(t *testing.T) {
	db := setupTeamTestDB(t)
	previousNewUserQuota := common.QuotaForNewUser
	common.QuotaForNewUser = 12345
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUserQuota
	})

	team, owner, err := CreateTeamWithOwner(
		"Acme Enterprise",
		"acme-enterprise",
		TeamCreateOwner{
			Username:    "acme-owner",
			Password:    "password123",
			DisplayName: "Acme Owner",
			Email:       "owner@acme.example",
		},
		999,
	)
	require.NoError(t, err)
	assert.Equal(t, common.RoleCommonUser, owner.Role)
	assert.Zero(t, owner.Quota, "team owners must not receive personal signup quota")
	var ownerMembership TeamMember
	require.NoError(t, db.Where("user_id = ?", owner.Id).First(&ownerMembership).Error)
	assert.Equal(t, team.Id, ownerMembership.TeamId)
	assert.Equal(t, TeamMemberRoleOwner, ownerMembership.Role)

	member := &User{
		Username:    "acme-member",
		Password:    "password123",
		DisplayName: "Acme Member",
		Email:       "member@acme.example",
	}
	require.NoError(t, CreateTeamMemberUser(team.Id, owner.Id, member))
	assert.Equal(t, common.RoleCommonUser, member.Role)
	assert.Zero(t, member.Quota, "team members must not receive personal signup quota")
	var memberMembership TeamMember
	require.NoError(t, db.Where("user_id = ?", member.Id).First(&memberMembership).Error)
	assert.Equal(t, team.Id, memberMembership.TeamId)
	assert.Equal(t, TeamMemberRoleMember, memberMembership.Role)

	otherTeam := Team{Name: "Other Enterprise", Slug: "other-enterprise", Status: TeamStatusEnabled}
	require.NoError(t, db.Create(&otherTeam).Error)
	crossTenantMember := &User{
		Username:    "cross-tenant-member",
		Password:    "password123",
		DisplayName: "Cross Tenant",
	}
	require.ErrorIs(
		t,
		CreateTeamMemberUser(otherTeam.Id, owner.Id, crossTenantMember),
		ErrTeamAccessDenied,
	)
	var crossTenantUserCount int64
	require.NoError(t, db.Model(&User{}).Where("username = ?", crossTenantMember.Username).Count(&crossTenantUserCount).Error)
	assert.Zero(t, crossTenantUserCount, "a denied cross-tenant creation must not leave an orphan user")

	require.NoError(t, SetTeamMemberStatusByRoot(team.Id, 999, owner.Id, TeamMemberStatusDisabled))
	disabledOwnerContext, err := GetTeamContextByUserId(owner.Id)
	require.NoError(t, err)
	require.NotNil(t, disabledOwnerContext)
	assert.Equal(t, TeamMemberStatusDisabled, disabledOwnerContext.MemberStatus)
	assert.False(t, disabledOwnerContext.IsManager, "an inactive owner must not be advertised as a team manager")
}

func TestCreateTeamRollsBackOwnerWhenTeamInsertFails(t *testing.T) {
	db := setupTeamTestDB(t)
	_, _, err := CreateTeamWithOwner(
		"Existing Enterprise",
		"duplicate-enterprise",
		TeamCreateOwner{
			Username:    "existing-owner",
			Password:    "password123",
			DisplayName: "Existing Owner",
		},
		999,
	)
	require.NoError(t, err)

	_, _, err = CreateTeamWithOwner(
		"Duplicate Enterprise",
		"duplicate-enterprise",
		TeamCreateOwner{
			Username:    "rolled-back-owner",
			Password:    "password123",
			DisplayName: "Rolled Back Owner",
		},
		999,
	)
	require.Error(t, err)

	var userCount int64
	require.NoError(t, db.Model(&User{}).Where("username = ?", "rolled-back-owner").Count(&userCount).Error)
	assert.Zero(t, userCount, "owner creation and team creation must be one transaction")
	var membershipCount int64
	require.NoError(t, db.Model(&TeamMember{}).Count(&membershipCount).Error)
	assert.EqualValues(t, 1, membershipCount)
}
