package api

import (
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
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
	StartPhoneVerification(context.Context, int64, string) error
	VerifyPhoneCode(context.Context, int64, string, string) error
	AccountFromToken(context.Context, string) (int64, error)
	AccountFromSessionHash(context.Context, string) (int64, error)
	Logout(context.Context, string) error
	LogoutSessionHash(context.Context, string) error
	LogoutAll(context.Context, string) error
	LogoutAllSessionHash(context.Context, string) error
}

// Server serves HTTP requests for BDSPro, following SimpleBank's API layout.
type Server struct {
	pool           *pgxpool.Pool
	authentication Authentication
	organizations  *organization.Service
	sources        *source.Service
	development    bool
	allowedOrigin  string
	logger         *slog.Logger
	authLimiter    *authLimiter
	accessTokens   *auth.AccessTokens
	draining       atomic.Bool
	router         *gin.Engine
}

// NewServer creates a Gin server and sets up all HTTP routes.
func NewServer(pool *pgxpool.Pool, authentication Authentication, organizations *organization.Service, sources *source.Service, development bool, allowedOrigin string) *Server {
	server := &Server{
		pool: pool, authentication: authentication,
		organizations: organizations, sources: sources,
		development: development, allowedOrigin: allowedOrigin,
		logger:      slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		authLimiter: newAuthLimiter(4096),
	}
	// Default Gin logging includes raw URLs; use a strict, safe field allowlist.
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(server.logRequest)
	router.Use(gin.CustomRecoveryWithWriter(io.Discard, func(ctx *gin.Context, _ any) {
		ctx.Set("panic_recovered", true)
		writeAPIError(ctx, http.StatusInternalServerError, "internal server error")
		ctx.Abort()
	}))
	binding.EnableDecoderDisallowUnknownFields = true
	router.Use(server.rejectDuringShutdown)
	router.Use(server.browserSecurity)

	router.POST("/auth/register", server.limitAuthClient("register", ratePolicy{burst: 12, window: time.Hour}), server.createAccount)
	router.POST("/auth/verify-email", server.limitAuthClient("verify", ratePolicy{burst: 30, window: 15 * time.Minute}), server.verifyEmail)
	router.POST("/auth/resend-verification", server.limitAuthClient("resend", ratePolicy{burst: 12, window: 15 * time.Minute}), server.resendVerification)
	router.POST("/auth/login", server.limitAuthClient("login", ratePolicy{burst: 30, window: time.Minute}), server.login)
	router.POST("/auth/token/renew", server.limitAuthClient("renew", ratePolicy{burst: 30, window: time.Minute}), server.renewAccessToken)
	router.GET("/auth/me", server.requireSession, server.me)
	router.POST("/auth/logout", server.logout)
	router.POST("/auth/logout-all", server.requireSession, server.logoutAll)
	router.POST("/auth/phone/start", server.limitAuthClient("phone-start", ratePolicy{burst: 6, window: 15 * time.Minute}), server.requireSession, server.startPhoneVerification)
	router.POST("/auth/phone/verify", server.limitAuthClient("phone-verify", ratePolicy{burst: 20, window: 15 * time.Minute}), server.requireSession, server.verifyPhoneCode)

	router.POST("/organizations/registrations", server.requireSession, server.submitOrganizationRegistration)
	router.POST("/organizations/:id/claims", server.requireSession, server.submitOrganizationClaim)
	router.POST("/platform/organizations/registrations/:id/approve", server.requireSession, server.approveOrganizationRegistration)
	router.POST("/platform/organizations/claims/:id/approve", server.requireSession, server.approveOrganizationClaim)
	router.POST("/organizations/:id/invitations", server.requireSession, server.inviteOrganizationMember)
	router.POST("/organizations/invitations/:id/accept", server.requireSession, server.acceptOrganizationInvitation)
	router.GET("/organizations/:id/members", server.requireSession, server.listOrganizationMembers)
	router.DELETE("/organizations/:id/members/:membershipID", server.requireSession, server.removeOrganizationMember)

	router.POST("/organizations/:id/sources", server.requireSession, server.captureSource)
	router.GET("/organizations/:id/sources/:sourceID", server.requireSession, server.readSource)
	router.POST("/organizations/:id/sources/:sourceID/price-reports", server.requireSession, server.reportSourcePrice)
	router.GET("/organizations/:id/sources/:sourceID/price-reports", server.requireSession, server.listSourcePrices)

	router.GET("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})
	router.GET("/ready", server.ready)
	router.NoRoute(func(ctx *gin.Context) {
		writeAPIError(ctx, http.StatusNotFound, "resource not found")
	})
	router.NoMethod(func(ctx *gin.Context) {
		writeAPIError(ctx, http.StatusMethodNotAllowed, "method not allowed")
	})

	server.router = router
	return server
}

// requireSession authenticates once per request. Tenant authorization stays in SQL.
func (server *Server) requireSession(ctx *gin.Context) {
	if _, ok := server.currentAccount(ctx); !ok {
		ctx.Abort()
		return
	}
	ctx.Next()
}

// rejectDuringShutdown closes the admission gate before draining requests.
// Health still reports that the process is alive; readiness and business routes
// stop admitting work. In-flight handlers continue with their request contexts.
func (server *Server) rejectDuringShutdown(ctx *gin.Context) {
	if server.draining.Load() && ctx.FullPath() != "/health" {
		writeAPIError(ctx, http.StatusServiceUnavailable, "service unavailable")
		ctx.Abort()
		return
	}
	ctx.Next()
}

func (server *Server) ready(ctx *gin.Context) {
	if server.draining.Load() || server.pool == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "not ready")
		return
	}
	ping, cancel := context.WithTimeout(ctx.Request.Context(), time.Second)
	defer cancel()
	if err := server.pool.Ping(ping); err != nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "not ready")
		return
	}
	ctx.String(http.StatusOK, "ready")
}

const shutdownGracePeriod = 5 * time.Second

// Start owns signal subscription and the listener. The database pool is owned
// by main.go and is closed only after Start has returned.
func (server *Server) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.serve(signals, listener, shutdownGracePeriod)
}

// serve accepts a cancellation context for deterministic lifecycle tests.
// A separate, bounded background context is essential: the signal context is
// already cancelled and cannot be passed to http.Server.Shutdown.
func (server *Server) serve(ctx context.Context, listener net.Listener, gracePeriod time.Duration) error {
	httpServer := &http.Server{
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	done := make(chan error, 1)
	go func() { done <- httpServer.Serve(listener) }()
	log.Printf("BDSPro listening on %s", listener.Addr())

	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// Fail readiness and reject new application work, but do not cancel
		// contexts of requests that were already admitted.
		server.draining.Store(true)

		grace, cancel := context.WithTimeout(context.Background(), gracePeriod)
		shutdownErr := httpServer.Shutdown(grace)
		cancel()
		if shutdownErr != nil {
			// After the grace period, stop active connections too. Request
			// contexts are then cancelled so pgx can abandon pending queries.
			if closeErr := httpServer.Close(); closeErr != nil {
				shutdownErr = errors.Join(shutdownErr, closeErr)
			}
		}

		// Wait for the HTTP accept loop to finish before main closes pgxpool.
		serveErr := <-done
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return errors.Join(shutdownErr, serveErr)
		}
		return shutdownErr
	}
}

// Handler supports the current graceful-shutdown HTTP server.
func (server *Server) Handler() http.Handler {
	return server.router
}

// EnableAccessTokens configures signed, short-lived JWT access tokens.
// It is called during startup, before the router serves requests.
func (server *Server) EnableAccessTokens(issuer *auth.AccessTokens) {
	server.accessTokens = issuer
}
