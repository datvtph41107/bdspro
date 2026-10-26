package routes

import (
	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
	"time"
)

func mountGoogle(router *gin.Engine, c *authcontroller.GoogleController, maker token.Maker, limit *middleware.RateLimiter) {
	router.POST("/users/me/google/start", middleware.Authenticate(maker),
		limit.Source("google_link_start", 10, time.Minute), c.StartLink)
	router.GET("/users/google/login/start", limit.Source("google_login_start", 20, time.Minute), c.StartLogin)
	router.GET("/users/google/callback", c.Callback)
}
