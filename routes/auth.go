package routes

import (
	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func RegisterAuth(router *gin.Engine, c *authcontroller.Controller, limit *middleware.RateLimiter) {
	router.GET("/health", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
	router.POST("/users", limit.Source("register", 10, time.Hour), c.Register)
	router.POST("/users/login", middleware.SufLogin(), limit.Source("login", 30, time.Minute), c.Login)
	router.GET("/users/verify_email", limit.Source("verify_email", 60, time.Minute), c.VerifyEmail)
	router.POST("/tokens/renew_access", limit.Source("refresh", 60, time.Minute), c.Renew)
	router.POST("/tokens/revoke", limit.Source("revoke", 60, time.Minute), c.Revoke)
}
