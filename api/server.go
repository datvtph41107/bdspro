package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Server serves HTTP requests for BDSPro, following SimpleBank's API layout.
type Server struct {
	store          *db.Store
	authentication *auth.Service
	organizations  *organization.Service
	sources        *source.Service
	development    bool
	allowedOrigin  string
	router         *gin.Engine
}

// NewServer creates a Gin server and sets up all HTTP routes.
func NewServer(store *db.Store, authentication *auth.Service, organizations *organization.Service, sources *source.Service, development bool, allowedOrigin string) *Server {
	server := &Server{
		store: store, authentication: authentication,
		organizations: organizations, sources: sources,
		development: development, allowedOrigin: allowedOrigin,
	}
	router := gin.Default()
	binding.EnableDecoderDisallowUnknownFields = true
	router.Use(server.checkOrigin)

	router.POST("/auth/register", server.createAccount)
	router.POST("/auth/verify-email", server.verifyEmail)
	router.POST("/auth/login", server.login)
	router.GET("/auth/me", server.me)
	router.POST("/auth/logout", server.logout)

	router.POST("/organizations/registrations", server.submitOrganizationRegistration)
	router.POST("/organizations/:id/claims", server.submitOrganizationClaim)
	router.POST("/platform/organizations/registrations/:id/approve", server.approveOrganizationRegistration)
	router.POST("/platform/organizations/claims/:id/approve", server.approveOrganizationClaim)
	router.POST("/organizations/:id/invitations", server.inviteOrganizationMember)
	router.POST("/organizations/invitations/:id/accept", server.acceptOrganizationInvitation)
	router.GET("/organizations/:id/members", server.listOrganizationMembers)

	router.POST("/organizations/:id/sources", server.captureSource)
	router.GET("/organizations/:id/sources/:sourceID", server.readSource)
	router.POST("/organizations/:id/sources/:sourceID/price-reports", server.reportSourcePrice)
	router.GET("/organizations/:id/sources/:sourceID/price-reports", server.listSourcePrices)

	router.GET("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})
	router.GET("/ready", server.ready)

	server.router = router
	return server
}

// checkOrigin keeps the existing browser-origin boundary for state-changing requests.
func (server *Server) checkOrigin(ctx *gin.Context) {
	method := ctx.Request.Method
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		ctx.Next()
		return
	}
	if strings.EqualFold(ctx.GetHeader("Sec-Fetch-Site"), "cross-site") {
		ctx.String(http.StatusForbidden, "cross-site request denied")
		ctx.Abort()
		return
	}
	if origin := ctx.GetHeader("Origin"); origin != "" && origin != server.allowedOrigin {
		ctx.String(http.StatusForbidden, "origin not allowed")
		ctx.Abort()
		return
	}
	ctx.Next()
}

func (server *Server) ready(ctx *gin.Context) {
	if server.authentication == nil || server.authentication.Pool == nil {
		ctx.String(http.StatusServiceUnavailable, "not ready")
		return
	}
	ping, cancel := context.WithTimeout(ctx.Request.Context(), time.Second)
	defer cancel()
	if err := server.authentication.Pool.Ping(ping); err != nil {
		ctx.String(http.StatusServiceUnavailable, "not ready")
		return
	}
	ctx.String(http.StatusOK, "ready")
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// Handler supports the current graceful-shutdown HTTP server.
func (server *Server) Handler() http.Handler {
	return server.router
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
