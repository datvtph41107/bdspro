package handler

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/datvtph41107/bdspro/internal/api/response"
	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/mail"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strings"
)

// AuthHandler handles HTTP input and output; authentication operations belong to AuthService.
type AuthHandler struct {
	auth        *service.AuthService
	limiter     *middleware.IdentityLimiter
	resetSender mail.EmailSender
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

func (h *AuthHandler) SetPasswordResetSender(sender mail.EmailSender) { h.resetSender = sender }

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "password_reset", req.Email, 10, 3, time.Hour) {
		return
	}
	if err := h.auth.RequestPasswordReset(c.Request.Context(), req.Email, h.resetSender); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

// GET is a non-consuming preview; passwords are submitted only with POST.
func (h *AuthHandler) PasswordResetPage(c *gin.Context) {
	token := c.Query("token")
	if len(token) != 43 {
		response.Reject(c, http.StatusBadRequest, "invalid_reset_token", "invalid or expired reset token")
		return
	}
	const page = `<!doctype html><html><head><meta charset="utf-8"><title>Reset password</title></head><body><form method="post" action="/users/password/reset"><input type="hidden" name="token" value="{{.}}"><input type="password" name="password" minlength="8" required><button type="submit">Set password</button></form></body></html>`
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("reset").Parse(page))
	_ = tmpl.Execute(c.Writer, token)
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req PasswordResetConfirmRequest
	var err error
	if strings.HasPrefix(c.ContentType(), "application/json") {
		err = c.ShouldBindJSON(&req)
	} else {
		err = c.ShouldBind(&req)
	}
	if err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "password_reset_confirm", req.Token, 20, 6, time.Hour) {
		return
	}
	if err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) StartEmailLink(c *gin.Context) {
	var req EmailLinkStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "email_link_start", req.Email, 10, 3, time.Hour) {
		return
	}
	if err := h.auth.StartEmailLink(c.Request.Context(), middleware.AccountID(c), req.Email, req.PhoneCode, h.resetSender); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *AuthHandler) ConfirmEmailLink(c *gin.Context) {
	var req EmailLinkConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "email_link_verify", req.Token, 20, 5, time.Hour) {
		return
	}
	if err := h.auth.ConfirmEmailLink(c.Request.Context(), middleware.AccountID(c), req.Token); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
    sessions, err := h.auth.ListSessions(c.Request.Context(), middleware.AccountID(c))
    if err != nil { response.Error(c, err); return }
    if sessions == nil { sessions = []db.ListActiveSessionsRow{} }
    c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}
func (h *AuthHandler) RevokeOwnSession(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil { response.Reject(c, http.StatusBadRequest, "invalid_session_id", "invalid session id"); return }
    if err := h.auth.RevokeOwnSession(c.Request.Context(), middleware.AccountID(c), id); err != nil {
        response.Error(c, err); return
    }
    c.Status(http.StatusNoContent)
}
func (h *AuthHandler) RevokeAllSessions(c *gin.Context) {
    if err := h.auth.RevokeAllSessions(c.Request.Context(), middleware.AccountID(c)); err != nil {
        response.Error(c, err); return
    }
    c.Status(http.StatusNoContent)
}
