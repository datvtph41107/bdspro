package routes
import (
    "time"
    authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
    "github.com/datvtph41107/bdspro/app/http/middleware"
    "github.com/datvtph41107/bdspro/token"
    "github.com/gin-gonic/gin"
)
func mountPhone(router *gin.Engine,c *authcontroller.PhoneController,maker token.Maker,limit *middleware.RateLimiter){
    router.POST("/users/phone/login/start",limit.Source("phone_login_start",10,time.Hour),c.StartLogin)
    router.POST("/users/phone/login/verify",middleware.SufLogin(),
        limit.Source("phone_login_verify",40,10*time.Minute),c.VerifyLogin)
    protected:=router.Group("/users/me/phone").Use(middleware.Authenticate(maker))
    protected.POST("/start",limit.Source("phone_bind_start",10,time.Hour),c.StartBind)
    protected.POST("/verify",limit.Source("phone_bind_verify",40,10*time.Minute),c.VerifyBind)
}
