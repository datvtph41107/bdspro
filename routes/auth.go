package routes

import (
	"net/http"
	"time"

	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterAuth declares the public authentication endpoints directly.
// The source limit must run before request parsing; identity limit runs after.
func RegisterAuth(router *gin.Engine, controller *authcontroller.Controller, guard RateGuard) {
	router.GET("/health", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	router.POST("/users",
		guard.IPActiveSource("register", 10, time.Hour),
		middleware.PreRegister(),
		guard.IPActive("register", 10, 3, time.Hour, middleware.RegisterIdentity),
		controller.Register,
	)
	router.POST("/users/login",
		middleware.LoginAudit(),
		guard.IPActiveSource("login", 30, time.Minute),
		middleware.PreLogin(),
		guard.IPActive("login", 30, 10, time.Minute, middleware.LoginIdentity),
		controller.Login,
	)
	router.GET("/users/verify_email",
		guard.IPActiveSource("verify_email", 60, time.Minute),
		middleware.PreVerifyEmail(),
		guard.IPActive("verify_email", 60, 10, time.Minute, middleware.VerifyEmailIdentity),
		controller.VerifyEmail,
	)
	router.POST("/tokens/renew_access",
		guard.IPActiveSource("refresh", 60, time.Minute),
		middleware.PreToken(),
		guard.IPActive("refresh", 60, 20, time.Minute, middleware.TokenIdentity),
		controller.Renew,
	)
	router.POST("/tokens/revoke",
		guard.IPActiveSource("revoke", 60, time.Minute),
		middleware.PreToken(),
		guard.IPActive("revoke", 60, 20, time.Minute, middleware.TokenIdentity),
		controller.Revoke,
	)
}
