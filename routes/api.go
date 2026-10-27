package routes

import (
	"github.com/datvtph41107/bdspro/app/http/binding"
	apicontroller "github.com/datvtph41107/bdspro/app/http/controllers/api"
	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	apirequest "github.com/datvtph41107/bdspro/app/http/requests/api"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

// RegisterAPI declares all authenticated HTTP routes with concrete actions.
func RegisterAPI(router *gin.Engine, auth *authcontroller.Controller, organizations *apicontroller.OrganizationController, maker token.Maker) {
	authorized := router.Group("/")
	authorized.Use(middleware.Authenticate(maker))

	authorized.GET("/users/me", auth.Me)
	authorized.POST("/organizations", binding.JSON[apirequest.CreateOrganizationBody]("organization.create"), organizations.Create)
	authorized.GET("/organizations", organizations.Index)
	authorized.POST("/organizations/:id/members", binding.JSON[apirequest.AddOrganizationMemberBody]("organization.member"), organizations.AddMember)
	authorized.PATCH("/organizations/:id", binding.JSON[apirequest.UpdateOrganizationBody]("organization.update"), organizations.Update)
}
