package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func registerTeamRoutes(apiRouter *gin.RouterGroup, anonymousRequestBodyLimit gin.HandlerFunc) {
	enterpriseRoute := apiRouter.Group("/enterprise")
	{
		enterpriseRoute.POST(
			"/inquiries",
			middleware.CriticalRateLimit(),
			anonymousRequestBodyLimit,
			controller.CreateEnterpriseInquiry,
		)
		enterpriseAdminRoute := enterpriseRoute.Group("/inquiries")
		enterpriseAdminRoute.Use(middleware.RootAuth())
		{
			enterpriseAdminRoute.GET("", controller.AdminListEnterpriseInquiries)
			enterpriseAdminRoute.PATCH("/:id/status", controller.AdminUpdateEnterpriseInquiryStatus)
		}
	}

	teamRoute := apiRouter.Group("/team")
	teamRoute.Use(middleware.UserAuth(), middleware.TeamManagerAuth())
	{
		teamRoute.GET("", controller.GetCurrentTeam)
		teamRoute.GET("/members", controller.GetCurrentTeamMembers)
		teamRoute.POST("/members", middleware.CriticalRateLimit(), controller.CreateCurrentTeamMember)
		teamRoute.PATCH("/members/:user_id/status", controller.UpdateCurrentTeamMemberStatus)
		teamRoute.POST("/members/:user_id/reset-password", middleware.CriticalRateLimit(), controller.ResetCurrentTeamMemberPassword)
		teamRoute.GET("/usage", controller.GetCurrentTeamUsage)
		teamRoute.GET("/transactions", controller.GetCurrentTeamTransactions)
		teamRoute.POST("/fund", middleware.CriticalRateLimit(), controller.FundCurrentTeam)
	}

	teamAdminRoute := apiRouter.Group("/team/admin")
	teamAdminRoute.Use(middleware.RootAuth())
	{
		teamAdminRoute.GET("", controller.AdminListTeams)
		teamAdminRoute.POST("", middleware.CriticalRateLimit(), controller.AdminCreateTeam)
		teamAdminRoute.GET("/:id", controller.AdminGetTeam)
		teamAdminRoute.GET("/:id/members", controller.AdminGetTeamMembers)
		teamAdminRoute.POST("/:id/members", middleware.CriticalRateLimit(), controller.AdminCreateTeamMember)
		teamAdminRoute.PATCH("/:id/members/:user_id/status", controller.AdminUpdateTeamMemberStatus)
		teamAdminRoute.POST("/:id/members/:user_id/reset-password", middleware.CriticalRateLimit(), controller.AdminResetTeamMemberPassword)
		teamAdminRoute.GET("/:id/usage", controller.AdminGetTeamUsage)
		teamAdminRoute.GET("/:id/transactions", controller.AdminGetTeamTransactions)
		teamAdminRoute.POST("/:id/quota", middleware.CriticalRateLimit(), controller.AdminAdjustTeamQuota)
		teamAdminRoute.PATCH("/:id/status", controller.AdminUpdateTeamStatus)
	}
}
