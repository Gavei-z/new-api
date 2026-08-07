package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupManageUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.UserSession{}, &model.Log{}, &model.CasbinRule{}, &model.AuthzRole{},
		&model.Team{}, &model.TeamMember{}, &model.PrepaidReserve{}, &model.PrepaidReserveTransaction{},
	))

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func performManageUserRequest(t *testing.T, body string) *httptest.ResponseRecorder {
	return performManageUserRequestAs(t, body, 9999, common.RoleRootUser)
}

func performManageUserRequestAs(
	t *testing.T,
	body string,
	actorId int,
	actorRole int,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/manage", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", actorId)
	c.Set("role", actorRole)
	c.Set("username", "root-operator")
	ManageUser(c)
	return recorder
}

func TestManageUserDisableAdvancesAuthVersionOnceAndRevokesSession(t *testing.T) {
	db := setupManageUserTestDB(t)
	now := time.Now().Unix()
	user := model.User{
		Username: "managed-disable-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.UserSession{
		SID: "managed-disable-session", UserID: user.Id, Version: 1, UserAuthVersion: 1,
		Status: model.UserSessionStatusActive, RefreshHash: "refresh-hash", LoginMethod: "password",
		LastActiveAt: now, ExpiresAt: now + 3600,
	}).Error)

	recorder := performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"disable"}`, user.Id))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	var updated model.User
	require.NoError(t, db.First(&updated, user.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, updated.Status)
	assert.EqualValues(t, 2, updated.AuthVersion)
	var session model.UserSession
	require.NoError(t, db.First(&session, "sid = ?", "managed-disable-session").Error)
	assert.Equal(t, model.UserSessionStatusRevoked, session.Status)
}

func TestManageUserDemoteAdvancesAuthVersionAndRevokesSessionsOnce(t *testing.T) {
	db := setupManageUserTestDB(t)
	previousMaster := common.IsMasterNode
	common.IsMasterNode = false
	t.Cleanup(func() { common.IsMasterNode = previousMaster })
	require.NoError(t, authz.Init(db))

	now := time.Now().Unix()
	user := model.User{
		Username: "managed-demote-user", Password: "password", Role: common.RoleAdminUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)
	for _, sid := range []string{"managed-demote-session-one", "managed-demote-session-two"} {
		require.NoError(t, db.Create(&model.UserSession{
			SID: sid, UserID: user.Id, Version: 1, UserAuthVersion: 1,
			Status: model.UserSessionStatusActive, RefreshHash: "refresh-" + sid, LoginMethod: "password",
			LastActiveAt: now, ExpiresAt: now + 3600,
		}).Error)
	}

	sessionUpdateCount := 0
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:count_demote_session_updates", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "user_sessions" {
			sessionUpdateCount++
		}
	}))

	recorder := performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"demote"}`, user.Id))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	var updated model.User
	require.NoError(t, db.First(&updated, user.Id).Error)
	assert.Equal(t, common.RoleCommonUser, updated.Role)
	assert.EqualValues(t, 2, updated.AuthVersion)
	var sessions []model.UserSession
	require.NoError(t, db.Where("user_id = ?", user.Id).Order("sid asc").Find(&sessions).Error)
	require.Len(t, sessions, 2)
	for _, session := range sessions {
		assert.Equal(t, model.UserSessionStatusRevoked, session.Status)
		assert.Equal(t, "admin_demote", session.RevokedReason)
	}
	assert.Equal(t, 1, sessionUpdateCount)
}

func TestManageUserDeleteReturnsImmediatelyAndUnknownActionFails(t *testing.T) {
	db := setupManageUserTestDB(t)
	deleted := model.User{
		Username: "managed-delete-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "delete-aff",
	}
	require.NoError(t, db.Create(&deleted).Error)

	recorder := performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"delete"}`, deleted.Id))
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	var deletedCount int64
	require.NoError(t, db.Unscoped().Model(&model.User{}).Where("id = ? AND deleted_at IS NOT NULL", deleted.Id).Count(&deletedCount).Error)
	assert.EqualValues(t, 1, deletedCount)

	unchanged := model.User{
		Username: "managed-unknown-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "unknown-aff",
	}
	require.NoError(t, db.Create(&unchanged).Error)
	recorder = performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"unknown"}`, unchanged.Id))
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	require.NoError(t, db.First(&unchanged, unchanged.Id).Error)
	assert.EqualValues(t, 1, unchanged.AuthVersion)
	assert.Equal(t, common.UserStatusEnabled, unchanged.Status)
}

func TestManageUserQuotaAdjustsTotalWalletAndIsIdempotent(t *testing.T) {
	db := setupManageUserTestDB(t)
	user := model.User{
		Username: "managed-quota-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", Quota: 10, AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.PrepaidReserve{
		TargetType: model.PrepaidTargetUser,
		TargetId:   user.Id,
		Quota:      20,
	}).Error)

	body := fmt.Sprintf(
		`{"id":%d,"action":"add_quota","mode":"add","value":5,"idempotency_key":"quota-request-0001"}`,
		user.Id,
	)
	recorder := performManageUserRequest(t, body)
	assert.Equal(t, http.StatusOK, recorder.Code)
	var firstResponse struct {
		Success bool `json:"success"`
		Data    struct {
			Applied      bool  `json:"applied"`
			Quota        int64 `json:"quota"`
			ReserveQuota int64 `json:"reserve_quota"`
			TotalQuota   int64 `json:"total_quota"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &firstResponse))
	assert.True(t, firstResponse.Success)
	assert.True(t, firstResponse.Data.Applied)
	assert.Equal(t, int64(15), firstResponse.Data.Quota)
	assert.Equal(t, int64(20), firstResponse.Data.ReserveQuota)
	assert.Equal(t, int64(35), firstResponse.Data.TotalQuota)

	// An uncertain client retry reuses the same key and must not credit twice.
	recorder = performManageUserRequest(t, body)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &firstResponse))
	assert.True(t, firstResponse.Success)
	assert.False(t, firstResponse.Data.Applied)
	assert.Equal(t, int64(35), firstResponse.Data.TotalQuota)

	var auditLogs []model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeManage).Find(&auditLogs).Error)
	require.Len(t, auditLogs, 1)
	assert.Contains(t, auditLogs[0].Other, `"action":"user.quota_add"`)

	var ledgerCount int64
	require.NoError(t, db.Model(&model.PrepaidReserveTransaction{}).
		Where("idempotency_key = ?", fmt.Sprintf(
			"user-admin:%d:%d:%s",
			user.Id,
			9999,
			"quota-request-0001",
		)).
		Count(&ledgerCount).Error)
	assert.EqualValues(t, 1, ledgerCount)

	recorder = performManageUserRequest(t, fmt.Sprintf(
		`{"id":%d,"action":"add_quota","mode":"subtract","value":25,"idempotency_key":"quota-request-0002"}`,
		user.Id,
	))
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &firstResponse))
	assert.True(t, firstResponse.Success)
	assert.Equal(t, int64(10), firstResponse.Data.Quota)
	assert.Zero(t, firstResponse.Data.ReserveQuota)
	assert.Equal(t, int64(10), firstResponse.Data.TotalQuota)

	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, 10, stored.Quota)
	var reserve model.PrepaidReserve
	require.NoError(t, db.Where(
		"target_type = ? AND target_id = ?",
		model.PrepaidTargetUser,
		user.Id,
	).First(&reserve).Error)
	assert.Zero(t, reserve.Quota)
}

func TestManageUserQuotaRejectsNegativeOverrideAndTeamMembers(t *testing.T) {
	db := setupManageUserTestDB(t)
	user := model.User{
		Username: "managed-negative-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", Quota: 10, AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := performManageUserRequest(t, fmt.Sprintf(
		`{"id":%d,"action":"add_quota","mode":"override","value":-1,"idempotency_key":"quota-request-negative"}`,
		user.Id,
	))
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	recorder = performManageUserRequest(t, fmt.Sprintf(
		`{"id":%d,"action":"add_quota","mode":"add","value":1}`,
		user.Id,
	))
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	recorder = performManageUserRequest(t, fmt.Sprintf(
		`{"id":%d,"action":"add_quota","mode":"override","idempotency_key":"quota-request-missing"}`,
		user.Id,
	))
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	recorder = performManageUserRequest(t, fmt.Sprintf(
		`{"id":%d,"action":"add_quota","mode":"override","value":null,"idempotency_key":"quota-request-null"}`,
		user.Id,
	))
	assert.Contains(t, recorder.Body.String(), `"success":false`)

	team := model.Team{Name: "Managed Team", Slug: "managed-team", Status: model.TeamStatusEnabled}
	require.NoError(t, db.Create(&team).Error)
	require.NoError(t, db.Create(&model.TeamMember{
		TeamId: team.Id,
		UserId: user.Id,
		Role:   model.TeamMemberRoleMember,
		Status: model.TeamMemberStatusEnabled,
	}).Error)
	recorder = performManageUserRequest(t, fmt.Sprintf(
		`{"id":%d,"action":"add_quota","mode":"add","value":5,"idempotency_key":"quota-request-team"}`,
		user.Id,
	))
	assert.Contains(t, recorder.Body.String(), `"success":false`)

	balance, err := model.GetPrepaidBalance(model.PrepaidTargetUser, user.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(10), balance.TotalQuota)
	var ledgerCount int64
	require.NoError(t, db.Model(&model.PrepaidReserveTransaction{}).Count(&ledgerCount).Error)
	assert.Zero(t, ledgerCount)
}

func TestManageUserQuotaRejectsSameLevelAdministrator(t *testing.T) {
	db := setupManageUserTestDB(t)
	user := model.User{
		Username: "managed-admin-user", Password: "password", Role: common.RoleAdminUser,
		Status: common.UserStatusEnabled, Group: "default", Quota: 10, AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := performManageUserRequestAs(
		t,
		fmt.Sprintf(
			`{"id":%d,"action":"add_quota","mode":"add","value":5,"idempotency_key":"quota-request-admin"}`,
			user.Id,
		),
		7001,
		common.RoleAdminUser,
	)
	assert.Contains(t, recorder.Body.String(), `"success":false`)

	balance, err := model.GetPrepaidBalance(model.PrepaidTargetUser, user.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(10), balance.TotalQuota)
	var ledgerCount int64
	require.NoError(t, db.Model(&model.PrepaidReserveTransaction{}).Count(&ledgerCount).Error)
	assert.Zero(t, ledgerCount)
}
