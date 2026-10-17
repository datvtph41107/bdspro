package api

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Authentication defines only the auth behaviors consumed by HTTP.
// It allows handler tests to simulate failures without a PostgreSQL connection.
type Authentication interface {
	Register(context.Context, string, string) (auth.Registration, error)
	VerifyEmail(context.Context, string) error
	ResendVerification(context.Context, string) error
	Login(context.Context, string, string) (auth.LoginResult, error)
	AccountFromToken(context.Context, string) (int64, error)
	Logout(context.Context, string) error
}

// Server serves HTTP requests for BDSPro, following SimpleBank's API layout.
type Server struct {
	pool           *pgxpool.Pool
	authentication Authentication
	organizations  *organization.Service
	sources        *source.Service
	development    bool
	allowedOrigin  string
	router         *gin.Engine
}

// NewServer creates a Gin server and sets up all HTTP routes.
func NewServer(pool *pgxpool.Pool, authentication Authentication, organizations *organization.Service, sources *source.Service, development bool, allowedOrigin string) *Server {
	server := &Server{
		pool: pool, authentication: authentication,
		organizations: organizations, sources: sources,
		development: development, allowedOrigin: allowedOrigin,
	}
	router := gin.Default()
	binding.EnableDecoderDisallowUnknownFields = true
	router.Use(server.checkOrigin)

	router.POST("/auth/register", server.createAccount)
	router.POST("/auth/verify-email", server.verifyEmail)
	router.POST("/auth/resend-verification", server.resendVerification)
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
	if server.pool == nil {
		ctx.String(http.StatusServiceUnavailable, "not ready")
		return
	}
	ping, cancel := context.WithTimeout(ctx.Request.Context(), time.Second)
	defer cancel()
	if err := server.pool.Ping(ping); err != nil {
		ctx.String(http.StatusServiceUnavailable, "not ready")
		return
	}
	ctx.String(http.StatusOK, "ready")
}

// Start serves HTTP and shuts down gracefully on SIGINT or SIGTERM.
func (server *Server) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	httpServer := &http.Server{
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan error, 1)
	go func() { done <- httpServer.Serve(listener) }()
	log.Printf("BDSPro listening on %s", listener.Addr())

	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(ctx)
	}
}

// Handler supports the current graceful-shutdown HTTP server.
func (server *Server) Handler() http.Handler {
	return server.router
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
