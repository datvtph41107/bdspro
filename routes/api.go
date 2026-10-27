package routes

import (
	apicontroller "github.com/datvtph41107/bdspro/app/http/controllers/api"
	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

// RegisterAPI declares all authenticated HTTP routes with concrete actions.
func RegisterAPI(router *gin.Engine, auth *authcontroller.Controller, organizations *apicontroller.OrganizationController, maker token.Maker) {
	authorized := router.Group("/")
	authorized.Use(middleware.Authenticate(maker))

	authorized.GET("/users/me", auth.Me)
	authorized.POST("/organizations", organizations.Create)
	authorized.GET("/organizations", organizations.Index)
	authorized.POST("/organizations/:id/members", organizations.AddMember)
	authorized.PATCH("/organizations/:id", organizations.Update)
}
