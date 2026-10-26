package routes

import (
	apicontroller "github.com/datvtph41107/bdspro/app/http/controllers/api"
    authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

func RegisterAPI(router *gin.Engine, user *authcontroller.Controller, c *apicontroller.OrganizationController, maker token.Maker) {
	authenticated := router.Group("/").Use(middleware.Authenticate(maker))
	authenticated.GET("/users/me", user.Me)
	authenticated.POST("/organizations", c.Create)
	authenticated.GET("/organizations", c.Index)
	authenticated.POST("/organizations/:id/members", c.AddMember)
	authenticated.PATCH("/organizations/:id", c.Update)
}
