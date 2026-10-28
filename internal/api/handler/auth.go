package handler

import (
	"net/http"
	"strconv"
	"time"

	authrequest "github.com/datvtph41107/bdspro/internal/api/handler/request/auth"
	"github.com/datvtph41107/bdspro/internal/api/handler/resource"
	"github.com/datvtph41107/bdspro/internal/api/service"
	"github.com/datvtph41107/bdspro/internal/core/middleware"
	"github.com/datvtph41107/bdspro/internal/core/response"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles HTTP input and output; authentication operations belong to AuthService.
type AuthHandler struct {
	auth    *service.AuthService
	limiter *middleware.IdentityLimiter
}

func NewAuthHandler(auth *service.AuthService, limiter *middleware.IdentityLimiter) *AuthHandler {
	return &AuthHandler{auth: auth, limiter: limiter}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req authrequest.RegisterBody
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
	c.JSON(http.StatusCreated, resources.NewUser(account))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req authrequest.LoginBody
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
	c.JSON(http.StatusOK, resources.NewLogin(result))
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req authrequest.VerifyEmailQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification link"})
		return
	}
	if !h.limiter.Allow(c, "verify_email", strconv.FormatInt(req.EmailID, 10), 60, 10, time.Minute) {
		return
	}
	if err := h.auth.VerifyEmail(c.Request.Context(), req.EmailID, req.SecretCode); err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"verified": true})
}

func (h *AuthHandler) Renew(c *gin.Context) {
	var req authrequest.TokenBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "refresh", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	result, err := h.auth.Renew(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, resources.RenewAccessToken{
		AccessToken: result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})
}

func (h *AuthHandler) Revoke(c *gin.Context) {
	var req authrequest.TokenBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "revoke", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	if err := h.auth.Revoke(c.Request.Context(), req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Me uses the verified account identity provided by authentication middleware.
func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"account_id": middleware.AccountID(c)})
}
