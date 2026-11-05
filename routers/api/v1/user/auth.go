package user

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles HTTP input and output; authentication operations belong to AuthService.
type AuthHandler struct {
	auth        *auth_service.AuthService
	limiter     *middleware.AuthThrottle
	resetSender mail.EmailSender
}

func NewAuthHandler(auth *auth_service.AuthService, limiter *middleware.AuthThrottle) *AuthHandler {
	return &AuthHandler{auth: auth, limiter: limiter}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req structs.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "register", req.Email, 10, 3, time.Hour) {
		return
	}
	account, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToUser(account))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req structs.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "login", req.Email, 30, 10, time.Minute) {
		return
	}
	result, err := h.auth.Login(c.Request.Context(),
		req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToLogin(result))
}

