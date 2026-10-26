package routes

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// Authentication route declarations; middleware order is part of the security contract.
func mountAuth(route *gin.Engine, h Handlers) {
    route.GET("/health", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
    route.POST("/users", h.Source("register", 10, time.Hour), h.PreRegister,
        h.Identity("register", 10, 3, time.Hour, h.RegisterIdentity), h.Register)
    route.POST("/users/login", h.SufLogin, h.Source("login", 30, time.Minute), h.PreLogin,
        h.Identity("login", 30, 10, time.Minute, h.LoginIdentity), h.Login)
    route.GET("/users/verify_email", h.Source("verify_email", 60, time.Minute), h.PreVerifyEmail,
        h.Identity("verify_email", 60, 10, time.Minute, h.VerifyEmailIdentity), h.VerifyEmail)
    route.POST("/tokens/renew_access", h.Source("refresh", 60, time.Minute), h.PreToken,
        h.Identity("refresh", 60, 20, time.Minute, h.TokenIdentity), h.Renew)
    route.POST("/tokens/revoke", h.Source("revoke", 60, time.Minute), h.PreToken,
        h.Identity("revoke", 60, 20, time.Minute, h.TokenIdentity), h.Revoke)
}
