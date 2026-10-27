package routes

import (
	"time"

	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/binding"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

// RegisterGoogle mounts OAuth routes only when bootstrap configures Google.
func RegisterGoogle(router *gin.Engine, google *authcontroller.GoogleController, maker token.Maker, guard RateGuard) {
	router.POST("/users/me/google/start",
		middleware.Authenticate(maker),
		guard.IPActiveSource("google_link_start", 10, time.Minute),
		binding.JSON[authrequest.GoogleLinkBody]("google.link"),
		google.StartLink,
	)
	router.GET("/users/google/login/start",
		guard.IPActiveSource("google_login_start", 20, time.Minute),
		google.StartLogin,
	)
	router.GET("/users/google/callback", google.Callback)
}
