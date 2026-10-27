package routes

import (
	"time"

	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

// RegisterPhone mounts development-only phone OTP routes when configured.
func RegisterPhone(router *gin.Engine, phone *authcontroller.PhoneController, maker token.Maker, guard RateGuard) {
	router.POST("/users/phone/login/start",
		guard.IPActiveSource("phone_login_start", 10, time.Hour),
		middleware.PrePhoneLoginStart(),
		guard.IPActive("phone_login_start", 10, 3, time.Hour, middleware.PhoneStartIdentity),
		phone.StartLogin,
	)
	router.POST("/users/phone/login/verify",
		middleware.LoginAudit(),
		guard.IPActiveSource("phone_login_verify", 40, 10*time.Minute),
		middleware.PrePhoneVerify(),
		guard.IPActive("phone_login_verify", 40, 5, 10*time.Minute, middleware.PhoneVerifyIdentity),
		phone.VerifyLogin,
	)
	protected := router.Group("/users/me/phone")
	protected.Use(middleware.Authenticate(maker))
	protected.POST("/start",
		guard.IPActiveSource("phone_bind_start", 10, time.Hour),
		middleware.PrePhoneBind(),
		guard.IPActive("phone_bind_start", 10, 3, time.Hour, middleware.PhoneBindIdentity),
		phone.StartBind,
	)
	protected.POST("/verify",
		guard.IPActiveSource("phone_bind_verify", 40, 10*time.Minute),
		middleware.PrePhoneVerify(),
		guard.IPActive("phone_bind_verify", 40, 5, 10*time.Minute, middleware.PhoneVerifyIdentity),
		phone.VerifyBind,
	)
}
