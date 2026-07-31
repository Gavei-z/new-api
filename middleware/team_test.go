package middleware

import (
	"fmt"
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

func setupTeamMiddlewareTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType := common.MainDatabaseType()
	common.RedisEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Team{}, &model.TeamMember{}))

	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedisEnabled
		common.SetMainDatabaseType(previousMainDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func runTeamManagerAuth(userId int) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", userId)
	TeamManagerAuth()(c)
	return recorder, c
}

func TestTeamManagerAuthAllowsOnlyActiveTenantManagers(t *testing.T) {
	db := setupTeamMiddlewareTestDB(t)
	team := model.Team{Name: "Tenant A", Slug: "tenant-a", Status: model.TeamStatusEnabled}
	require.NoError(t, db.Create(&team).Error)
	manager := model.User{Username: "team-manager", Password: "password", AffCode: "team-manager"}
	member := model.User{Username: "team-member", Password: "password", AffCode: "team-member"}
	globalRoot := model.User{Username: "global-root", Password: "password", AffCode: "global-root", Role: common.RoleRootUser}
	require.NoError(t, db.Create(&manager).Error)
	require.NoError(t, db.Create(&member).Error)
	require.NoError(t, db.Create(&globalRoot).Error)
	require.NoError(t, db.Create(&model.TeamMember{
		TeamId: team.Id, UserId: manager.Id, Role: model.TeamMemberRoleOwner, Status: model.TeamMemberStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.TeamMember{
		TeamId: team.Id, UserId: member.Id, Role: model.TeamMemberRoleMember, Status: model.TeamMemberStatusEnabled,
	}).Error)

	_, managerContext := runTeamManagerAuth(manager.Id)
	assert.False(t, managerContext.IsAborted())
	assert.Equal(t, team.Id, managerContext.GetInt(TeamContextIdKey))

	memberRecorder, memberContext := runTeamManagerAuth(member.Id)
	assert.True(t, memberContext.IsAborted())
	assert.Equal(t, 403, memberRecorder.Code)

	rootRecorder, rootContext := runTeamManagerAuth(globalRoot.Id)
	assert.True(t, rootContext.IsAborted(), "global root uses root-only team routes, not tenant-scoped routes")
	assert.Equal(t, 403, rootRecorder.Code)

	require.NoError(t, db.Model(&model.Team{}).Where("id = ?", team.Id).Update("status", model.TeamStatusDisabled).Error)
	disabledRecorder, disabledContext := runTeamManagerAuth(manager.Id)
	assert.True(t, disabledContext.IsAborted())
	assert.Equal(t, 403, disabledRecorder.Code)
}
