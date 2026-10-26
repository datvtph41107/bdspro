package routes

import (
    "time"

    "github.com/gin-gonic/gin"
)

type GoogleHandlers struct {
    StartLink, StartLogin, Callback gin.HandlerFunc
    Authenticate gin.HandlerFunc
    Source func(string, int64, time.Duration) gin.HandlerFunc
}

func MountGoogle(route *gin.Engine, h GoogleHandlers) {
    route.POST("/users/me/google/start", h.Authenticate,
        h.Source("google_link_start", 10, time.Minute), h.StartLink)
    route.GET("/users/google/login/start",
        h.Source("google_login_start", 20, time.Minute), h.StartLogin)
    route.GET("/users/google/callback", h.Callback)
}
