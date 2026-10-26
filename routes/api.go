package routes

import "github.com/gin-gonic/gin"

func mountAPI(route *gin.Engine, h Handlers) {
    authorized := route.Group("/").Use(h.Authenticate)
    authorized.GET("/users/me", h.Me)
    authorized.POST("/organizations", h.CreateOrganization)
    authorized.GET("/organizations", h.ListOrganizations)
    authorized.POST("/organizations/:id/members", h.AddOrganizationMember)
    authorized.PATCH("/organizations/:id", h.UpdateOrganization)
}
