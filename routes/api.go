package routes

import (
	apicontroller "github.com/datvtph41107/bdspro/app/http/controllers/api"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

func RegisterAPI(router *gin.Engine, c *apicontroller.OrganizationController, maker token.Maker) {
	authenticated := router.Group("/").Use(middleware.Authenticate(maker))
	authenticated.GET("/users/me", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"account_id": middleware.AccountID(ctx)})
	})
	authenticated.POST("/organizations", c.Create)
	authenticated.GET("/organizations", c.Index)
	authenticated.POST("/organizations/:id/members", c.AddMember)
	authenticated.PATCH("/organizations/:id", c.Update)
}
