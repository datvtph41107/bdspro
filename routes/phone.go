package routes

import (
    "time"

    "github.com/gin-gonic/gin"
)

type PhoneHandlers struct {
    StartLogin, VerifyLogin, StartBind, VerifyBind gin.HandlerFunc
    PreStart, PreVerify, PreBind, SufLogin, Authenticate gin.HandlerFunc
    Source func(string, int64, time.Duration) gin.HandlerFunc
    Identity func(string, int64, int64, time.Duration, func(*gin.Context) string) gin.HandlerFunc
    StartIdentity, VerifyIdentity, BindIdentity func(*gin.Context) string
}

func MountPhone(route *gin.Engine, h PhoneHandlers) {
    route.POST("/users/phone/login/start",
        h.Source("phone_login_start", 10, time.Hour),
        h.PreStart,
        h.Identity("phone_login_start", 10, 3, time.Hour, h.StartIdentity),
        h.StartLogin,
    )
    route.POST("/users/phone/login/verify",
        h.SufLogin,
        h.Source("phone_login_verify", 40, 10*time.Minute),
        h.PreVerify,
        h.Identity("phone_login_verify", 40, 5, 10*time.Minute, h.VerifyIdentity),
        h.VerifyLogin,
    )
    protected := route.Group("/users/me/phone").Use(h.Authenticate)
    protected.POST("/start",
        h.Source("phone_bind_start", 10, time.Hour),
        h.PreBind,
        h.Identity("phone_bind_start", 10, 3, time.Hour, h.BindIdentity),
        h.StartBind,
    )
    protected.POST("/verify",
        h.Source("phone_bind_verify", 40, 10*time.Minute),
        h.PreVerify,
        h.Identity("phone_bind_verify", 40, 5, 10*time.Minute, h.VerifyIdentity),
        h.VerifyBind,
    )
}
