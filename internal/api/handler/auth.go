package handler

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/datvtph41107/bdspro/internal/service"
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
	var req RegisterRequest
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
	c.JSON(http.StatusCreated, newUserResponse(account))
}

func (h *AuthHandler) ResendVerificationEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "verify_email_resend", req.Email, 10, 2, time.Hour) {
		return
	}
	if err := h.auth.ResendVerificationEmail(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
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
	c.JSON(http.StatusOK, newLoginResponse(result))
}

// VerifyEmailPage is safe for mail scanners: GET never consumes a token.
func (h *AuthHandler) VerifyEmailPage(c *gin.Context) {
	var req VerifyEmailQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Reject(c, http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	c.Header("Content-Type", "text/html; charset=utf-8")
	const confirmation = `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Verify email</title></head><body><h1>Verify your email</h1><form method="post" action="/users/verify_email"><input type="hidden" name="email_id" value="{{.EmailID}}"><input type="hidden" name="secret_code" value="{{.SecretCode}}"><button type="submit">Confirm email</button></form></body></html>`
	page := template.Must(template.New("verify").Parse(confirmation))
	_ = page.Execute(c.Writer, req)
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailQuery
	if err := c.ShouldBind(&req); err != nil {
		response.Reject(c, http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
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
	var req TokenRequest
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
	c.JSON(http.StatusOK, RenewAccessTokenResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})
}

func (h *AuthHandler) Revoke(c *gin.Context) {
	var req TokenRequest
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
