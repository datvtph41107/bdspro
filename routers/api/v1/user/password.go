package user

import (
	"net/http"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/templates"
	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) SetPasswordResetSender(sender mail.EmailSender) { h.resetSender = sender }

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req structs.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "password_reset", req.Email, 10, 3, time.Hour) {
		return
	}
	if err := h.auth.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
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
