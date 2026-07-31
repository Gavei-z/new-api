package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	TeamContextIdKey   = "team_id"
	TeamContextRoleKey = "team_role"
)

// TeamManagerAuth enforces tenant scope independently of the global Admin
// role. Team managers remain ordinary users, so they cannot inherit global
// channel, system-setting, user, or log permissions.
func TeamManagerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		context, _, err := model.GetActiveTeamFundingContext(userId)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		if context == nil || !context.IsManager {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "team manager access required",
			})
			return
		}
		c.Set(TeamContextIdKey, context.TeamId)
		c.Set(TeamContextRoleKey, context.Role)
		c.Next()
	}
}
