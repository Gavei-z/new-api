package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedTeamBillingFixture(t *testing.T, teamQuota int, userQuota int) (*model.User, *model.Team) {
	t.Helper()
	user := &model.User{
		Username: "team-billing-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		Quota:    userQuota,
	}
	require.NoError(t, model.DB.Create(user).Error)
	team := &model.Team{
		Name:   "Billing Team",
		Slug:   "billing-team",
		Status: model.TeamStatusEnabled,
		Quota:  teamQuota,
	}
	require.NoError(t, model.DB.Create(team).Error)
	require.NoError(t, model.DB.Create(&model.TeamMember{
		TeamId: team.Id,
		UserId: user.Id,
		Role:   model.TeamMemberRoleMember,
		Status: model.TeamMemberStatusEnabled,
	}).Error)
	return user, team
}

func TestNewBillingSessionUsesTeamWalletInsteadOfPersonalWallet(t *testing.T) {
	truncate(t)
	user, team := seedTeamBillingFixture(t, 100, 900)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		UserId:       user.Id,
		RequestId:    "team-billing-request",
		IsPlayground: true,
	}

	session, apiErr := NewBillingSession(c, info, 40)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	assert.Equal(t, BillingSourceTeam, info.BillingSource)
	assert.Equal(t, team.Id, info.TeamId)

	require.NoError(t, session.Settle(55))
	var storedTeam model.Team
	require.NoError(t, model.DB.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 45, storedTeam.Quota)
	assert.Equal(t, 55, storedTeam.UsedQuota)
	var storedUser model.User
	require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 900, storedUser.Quota)
}

func TestNewBillingSessionRejectsInsufficientTeamWalletWithoutPersonalFallback(t *testing.T) {
	truncate(t)
	user, team := seedTeamBillingFixture(t, 20, 900)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		UserId:       user.Id,
		RequestId:    "team-insufficient-request",
		IsPlayground: true,
	}

	session, apiErr := NewBillingSession(c, info, 40)
	require.Nil(t, session)
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeInsufficientUserQuota, apiErr.GetErrorCode())

	var storedTeam model.Team
	require.NoError(t, model.DB.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 20, storedTeam.Quota)
	assert.Zero(t, storedTeam.UsedQuota)
	var storedUser model.User
	require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 900, storedUser.Quota)
}

func TestTeamBillingSessionReturnsUnusedPrecharge(t *testing.T) {
	truncate(t)
	user, team := seedTeamBillingFixture(t, 100, 900)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		UserId:       user.Id,
		RequestId:    "team-lower-settlement-request",
		IsPlayground: true,
	}

	session, apiErr := NewBillingSession(c, info, 60)
	require.Nil(t, apiErr)
	require.NoError(t, session.Settle(40))

	var storedTeam model.Team
	require.NoError(t, model.DB.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 60, storedTeam.Quota)
	assert.Equal(t, 40, storedTeam.UsedQuota)
	var storedUser model.User
	require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 900, storedUser.Quota)
}

func TestTeamFundingRefundIsIdempotent(t *testing.T) {
	truncate(t)
	user, team := seedTeamBillingFixture(t, 100, 0)
	funding := &TeamFunding{
		teamId:    team.Id,
		userId:    user.Id,
		requestId: "team-refund-request",
	}

	require.NoError(t, funding.PreConsume(30))
	require.NoError(t, funding.Refund())
	require.NoError(t, funding.Refund())

	var storedTeam model.Team
	require.NoError(t, model.DB.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 100, storedTeam.Quota)
	assert.Zero(t, storedTeam.UsedQuota)
	var transactionCount int64
	require.NoError(t, model.DB.Model(&model.TeamQuotaTransaction{}).Count(&transactionCount).Error)
	assert.EqualValues(t, 2, transactionCount)
}

func TestLegacyTeamBillingChargesEverySequenceAndNeverPersonalWallet(t *testing.T) {
	truncate(t)
	user, team := seedTeamBillingFixture(t, 100, 900)
	info := &relaycommon.RelayInfo{
		UserId:       user.Id,
		RequestId:    "legacy-team-request",
		IsPlayground: true,
	}

	require.NoError(t, PostConsumeQuota(info, 20, 0, false))
	require.NoError(t, PostConsumeQuota(info, 15, 0, false))

	var storedTeam model.Team
	require.NoError(t, model.DB.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 65, storedTeam.Quota)
	assert.Equal(t, 35, storedTeam.UsedQuota)
	var storedUser model.User
	require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 900, storedUser.Quota)
	assert.Equal(t, BillingSourceTeam, info.BillingSource)
	assert.Equal(t, team.Id, info.TeamId)
	var transactionCount int64
	require.NoError(t, model.DB.Model(&model.TeamQuotaTransaction{}).Count(&transactionCount).Error)
	assert.EqualValues(t, 2, transactionCount)
}

func TestTeamBillingGeneratesUniqueRequestIdsWhenMissing(t *testing.T) {
	truncate(t)
	user, team := seedTeamBillingFixture(t, 100, 900)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	first := &relaycommon.RelayInfo{UserId: user.Id, IsPlayground: true}
	second := &relaycommon.RelayInfo{UserId: user.Id, IsPlayground: true}

	firstSession, firstErr := NewBillingSession(c, first, 10)
	require.Nil(t, firstErr)
	require.NoError(t, firstSession.Settle(10))
	secondSession, secondErr := NewBillingSession(c, second, 10)
	require.Nil(t, secondErr)
	require.NoError(t, secondSession.Settle(10))

	assert.NotEmpty(t, first.RequestId)
	assert.NotEmpty(t, second.RequestId)
	assert.NotEqual(t, first.RequestId, second.RequestId)
	var storedTeam model.Team
	require.NoError(t, model.DB.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 80, storedTeam.Quota)
	assert.Equal(t, 20, storedTeam.UsedQuota)
}

func TestLegacyCapturedTeamFundingCanSettleAfterTeamDisabled(t *testing.T) {
	truncate(t)
	user, team := seedTeamBillingFixture(t, 100, 900)
	info := &relaycommon.RelayInfo{
		UserId:       user.Id,
		RequestId:    "legacy-disabled-settlement",
		IsPlayground: true,
	}

	require.NoError(t, PostConsumeQuota(info, 20, 0, false))
	require.NoError(t, model.UpdateTeamStatus(team.Id, model.TeamStatusDisabled))
	require.NoError(t, PostConsumeQuota(info, -5, 20, false))

	var storedTeam model.Team
	require.NoError(t, model.DB.First(&storedTeam, team.Id).Error)
	assert.Equal(t, 85, storedTeam.Quota)
	assert.Equal(t, 15, storedTeam.UsedQuota)
	var storedUser model.User
	require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 900, storedUser.Quota)
}
