package user

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/structs"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/datvtph41107/bdspro/services/mail"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/datvtph41107/bdspro/templates"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles HTTP input and output; authentication operations belong to AuthService.
type AuthHandler struct {
	auth        *auth_service.AuthService
	limiter     *security.AuthThrottle
	resetSender mail.EmailSender
}

func NewAuthHandler(auth *auth_service.AuthService, limiter *security.AuthThrottle, sender mail.EmailSender) *AuthHandler {
	return &AuthHandler{auth: auth, limiter: limiter, resetSender: sender}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req structs.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "register", req.Email, 10, 3, time.Hour) {
		return
	}
	account, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusCreated, convert.ToUser(account))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req structs.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "login", req.Email, 30, 10, time.Minute) {
		return
	}
	result, err := h.auth.Login(c.Request.Context(),
		req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusOK, convert.ToLogin(result))
}

func (h *AuthHandler) StartPhoneBind(c *gin.Context) {
	var req structs.PhoneBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_bind_start", req.Phone, 10, 3, time.Hour) {
		return
	}
	if err := h.auth.StartPhoneBind(c.Request.Context(), appctx.AccountID(c), req.Password, req.Phone); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *AuthHandler) VerifyPhoneBind(c *gin.Context) {
	var req structs.PhoneVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_bind_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	if err := h.auth.VerifyPhoneBind(c.Request.Context(), appctx.AccountID(c), req.Phone, req.Code); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Phone signup and login are public; no AccountID is available until OTP
// succeeds. The same IP/credential admission contract applies to each action.
func (h *AuthHandler) StartPhoneRegister(c *gin.Context) {
	var req structs.PhoneStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_register_start", req.Phone, 10, 3, time.Hour) {
		return
	}
	if err := h.auth.StartPhoneRegister(c.Request.Context(), req.Phone); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}
func (h *AuthHandler) VerifyPhoneRegister(c *gin.Context) {
	var req structs.PhoneVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_register_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	result, err := h.auth.VerifyPhoneRegister(c.Request.Context(), req.Phone, req.Code, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusCreated, convert.ToLogin(result))
}
func (h *AuthHandler) StartPhoneLogin(c *gin.Context) {
	var req structs.PhoneStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_login_start", req.Phone, 30, 10, 10*time.Minute) {
		return
	}
	if err := h.auth.StartPhoneLogin(c.Request.Context(), req.Phone); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}
func (h *AuthHandler) VerifyPhoneLogin(c *gin.Context) {
	var req structs.PhoneVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_login_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	result, err := h.auth.VerifyPhoneLogin(c.Request.Context(), req.Phone, req.Code, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusOK, convert.ToLogin(result))
}

func (h *AuthHandler) StartEmailLink(c *gin.Context) {
	var req structs.EmailLinkStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "email_link_start", req.Email, 10, 3, time.Hour) {
		return
	}
	if err := h.auth.StartEmailLink(c.Request.Context(), appctx.AccountID(c), req.Email, req.PhoneCode, h.resetSender); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *AuthHandler) ConfirmEmailLink(c *gin.Context) {
	var req structs.EmailLinkConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "email_link_verify", req.Token, 20, 5, time.Hour) {
		return
	}
	if err := h.auth.ConfirmEmailLink(c.Request.Context(), appctx.AccountID(c), req.Token); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req structs.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "password_reset", req.Email, 10, 3, time.Hour) {
		return
	}
	if err := h.auth.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

// GET is a non-consuming preview; passwords are submitted only with POST.
func (h *AuthHandler) PasswordResetPage(c *gin.Context) {
	token := c.Query("token")
	if len(token) != 43 {
		appctx.Reject(c, http.StatusBadRequest, "invalid_reset_token", "invalid or expired reset token")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = templates.PasswordResetPage.Execute(c.Writer, token)
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req structs.PasswordResetConfirmRequest
	var err error
	if strings.HasPrefix(c.ContentType(), "application/json") {
		err = c.ShouldBindJSON(&req)
	} else {
		err = c.ShouldBind(&req)
	}
	if err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "password_reset_confirm", req.Token, 20, 6, time.Hour) {
		return
	}
	if err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) ResendVerificationEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "verify_email_resend", req.Email, 10, 2, time.Hour) {
		return
	}
	if err := h.auth.ResendVerificationEmail(c.Request.Context(), req.Email); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

// VerifyEmailPage is safe for mail scanners: GET never consumes a token.
func (h *AuthHandler) VerifyEmailPage(c *gin.Context) {
	var req structs.VerifyEmailQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		appctx.Reject(c, http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = templates.VerifyEmailPage.Execute(c.Writer, req)
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req structs.VerifyEmailQuery
	if err := c.ShouldBind(&req); err != nil {
		appctx.Reject(c, http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
		return
	}
	if !h.limiter.Allow(c, "verify_email", strconv.FormatInt(req.EmailID, 10), 60, 10, time.Minute) {
		return
	}
	if err := h.auth.VerifyEmail(c.Request.Context(), req.EmailID, req.SecretCode); err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusOK, gin.H{"verified": true})
}

func (h *AuthHandler) Renew(c *gin.Context) {
	var req structs.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "refresh", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	result, err := h.auth.Renew(c.Request.Context(), req.RefreshToken)
	if err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusOK, structs.RenewAccessTokenResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})
}

func (h *AuthHandler) Revoke(c *gin.Context) {
	var req structs.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "revoke", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	if err := h.auth.Revoke(c.Request.Context(), req.RefreshToken); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	sessions, err := h.auth.ListSessions(c.Request.Context(), appctx.AccountID(c))
	if err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusOK, gin.H{"sessions": convert.ToSessions(sessions)})
}

func (h *AuthHandler) RevokeOwnSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		appctx.Reject(c, http.StatusBadRequest, "invalid_session_id", "invalid session id")
		return
	}
	payload, ok := appctx.AuthorizationPayload(c)
	if !ok {
		appctx.AuthenticationRequired(c)
		return
	}
	if err := h.auth.RevokeOwnSession(c.Request.Context(), payload.AccountID, payload.SessionID, id); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) RevokeAllSessions(c *gin.Context) {
	payload, ok := appctx.AuthorizationPayload(c)
	if !ok {
		appctx.AuthenticationRequired(c)
		return
	}
	if err := h.auth.RevokeAllSessions(c.Request.Context(), payload.AccountID, payload.SessionID); err != nil {
		appctx.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetConnections returns a bounded history of this Account's session issuance.
// The account identifier is always taken from the authenticated request context.
func (h *AuthHandler) GetConnections(c *gin.Context) {
	connections, err := h.auth.GetConnections(c.Request.Context(), appctx.AccountID(c))
	if err != nil {
		appctx.Error(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusOK, gin.H{"connections": convert.ToConnections(connections)})
}
