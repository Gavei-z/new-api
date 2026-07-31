package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTeamRouterTest(t *testing.T) (*gin.Engine, *model.User, *model.User) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousDatabaseType := common.MainDatabaseType()
	previousRedisEnabled := common.RedisEnabled
	previousBatchUpdateEnabled := common.BatchUpdateEnabled
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Team{},
		&model.TeamMember{},
		&model.TeamQuotaTransaction{},
		&model.EnterpriseInquiry{},
		&model.Log{},
	))

	managerToken := "team-manager-router-pat"
	manager := &model.User{
		Username: "router-team-manager", Password: "password",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		Group: "default", AccessToken: &managerToken, AuthVersion: 1,
		AffCode: "router-team-manager",
	}
	rootToken := "team-root-router-pat"
	root := &model.User{
		Username: "router-team-root", Password: "password",
		Role: common.RoleRootUser, Status: common.UserStatusEnabled,
		Group: "default", AccessToken: &rootToken, AuthVersion: 1,
		AffCode: "router-team-root",
	}
	require.NoError(t, db.Create(manager).Error)
	require.NoError(t, db.Create(root).Error)
	team := &model.Team{
		Name: "Router Tenant", Slug: "router-tenant",
		Status: model.TeamStatusEnabled, CreatedBy: root.Id,
	}
	require.NoError(t, db.Create(team).Error)
	require.NoError(t, db.Create(&model.TeamMember{
		TeamId: team.Id, UserId: manager.Id,
		Role: model.TeamMemberRoleOwner, Status: model.TeamMemberStatusEnabled,
		CreatedBy: root.Id,
	}).Error)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetMainDatabaseType(previousDatabaseType)
		common.RedisEnabled = previousRedisEnabled
		common.BatchUpdateEnabled = previousBatchUpdateEnabled
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return engine, manager, root
}

func performTeamRouterRequest(engine *gin.Engine, method string, path string, token string) *httptest.ResponseRecorder {
	return performTeamRouterRequestWithBody(engine, method, path, token, "")
}

func performTeamRouterRequestWithBody(engine *gin.Engine, method string, path string, token string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func TestTeamManagerRouteCannotCrossIntoGlobalAdministration(t *testing.T) {
	engine, manager, _ := setupTeamRouterTest(t)
	managerToken := manager.GetAccessToken()

	currentTeam := performTeamRouterRequest(engine, http.MethodGet, "/api/team", managerToken)
	assert.Equal(t, http.StatusOK, currentTeam.Code)

	rootTeams := performTeamRouterRequest(engine, http.MethodGet, "/api/team/admin", managerToken)
	assert.Equal(t, http.StatusForbidden, rootTeams.Code)

	channels := performTeamRouterRequest(engine, http.MethodGet, "/api/channel/", managerToken)
	assert.Equal(t, http.StatusForbidden, channels.Code)

	settings := performTeamRouterRequest(engine, http.MethodGet, "/api/option/", managerToken)
	assert.Equal(t, http.StatusForbidden, settings.Code)
}

func TestRootCanAccessGlobalTeamAdministration(t *testing.T) {
	engine, _, root := setupTeamRouterTest(t)
	response := performTeamRouterRequest(
		engine,
		http.MethodGet,
		"/api/team/admin",
		root.GetAccessToken(),
	)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "router-tenant")
}

func TestRootCannotTurnTeamMemberIntoGlobalAdminOrFundPersonalWallet(t *testing.T) {
	engine, manager, root := setupTeamRouterTest(t)
	rootToken := root.GetAccessToken()

	promote := performTeamRouterRequestWithBody(
		engine,
		http.MethodPost,
		"/api/user/manage",
		rootToken,
		fmt.Sprintf(`{"id":%d,"action":"promote"}`, manager.Id),
	)
	assert.Contains(t, promote.Body.String(), "team members cannot be promoted")

	fundPersonal := performTeamRouterRequestWithBody(
		engine,
		http.MethodPost,
		"/api/user/manage",
		rootToken,
		fmt.Sprintf(`{"id":%d,"action":"add_quota","mode":"add","value":100}`, manager.Id),
	)
	assert.Contains(t, fundPersonal.Body.String(), "team balance")

	var stored model.User
	require.NoError(t, model.DB.First(&stored, manager.Id).Error)
	assert.Equal(t, common.RoleCommonUser, stored.Role)
	assert.Zero(t, stored.Quota)
}
