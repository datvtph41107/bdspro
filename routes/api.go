package routes

import (
    "github.com/gin-gonic/gin"
    "github.com/datvtph41107/bdspro/app/http/binding"
    apirequest "github.com/datvtph41107/bdspro/app/http/requests/api"
)

func mountAPI(route *gin.Engine, h Handlers) {
    authorized := route.Group("/").Use(h.Authenticate)
    authorized.GET("/users/me", h.Me)
    authorized.POST("/organizations", binding.JSON[apirequest.CreateOrganizationBody]("organization.create"), h.CreateOrganization)
    authorized.GET("/organizations", h.ListOrganizations)
    authorized.POST("/organizations/:id/members", binding.JSON[apirequest.AddOrganizationMemberBody]("organization.member"), h.AddOrganizationMember)
    authorized.PATCH("/organizations/:id", binding.JSON[apirequest.UpdateOrganizationBody]("organization.update"), h.UpdateOrganization)
}
