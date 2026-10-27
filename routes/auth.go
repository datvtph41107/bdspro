package routes

import (
	"net/http"
	"time"

	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterAuth declares the public endpoints and their source rate limits.
// Each controller owns parsing its own request, as in SimpleBank.
func RegisterAuth(router *gin.Engine, controller *authcontroller.Controller, guard RateGuard) {
	router.GET("/health", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	router.POST("/users",
		guard.IPActiveSource("register", 10, time.Hour),
		controller.Register,
	)
	router.POST("/users/login",
		middleware.LoginAudit(),
		guard.IPActiveSource("login", 30, time.Minute),
		controller.Login,
	)
	router.GET("/users/verify_email",
		guard.IPActiveSource("verify_email", 60, time.Minute),
		controller.VerifyEmail,
	)
	router.POST("/tokens/renew_access",
		guard.IPActiveSource("refresh", 60, time.Minute),
		controller.Renew,
	)
	router.POST("/tokens/revoke",
		guard.IPActiveSource("revoke", 60, time.Minute),
		controller.Revoke,
	)
}
