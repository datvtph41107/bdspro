package routes

import (
    apicontroller "github.com/datvtph41107/bdspro/app/http/controllers/api"
    authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
    "github.com/datvtph41107/bdspro/app/http/middleware"
    "github.com/datvtph41107/bdspro/token"
    "github.com/gin-gonic/gin"
)

// Mount registers the actual controller methods once. There is no handler map,
// registry, dynamic resolver or request-context binding.
func Mount(router *gin.Engine, auth *authcontroller.Controller, api *apicontroller.OrganizationController,
    google *authcontroller.GoogleController, phone *authcontroller.PhoneController,
    maker token.Maker, limiter *middleware.RateLimiter,
) {
    mountAuth(router,auth,limiter)
    mountAPI(router,api,maker)
    if google != nil { mountGoogle(router,google,maker,limiter) }
    if phone != nil { mountPhone(router,phone,maker,limiter) }
}
