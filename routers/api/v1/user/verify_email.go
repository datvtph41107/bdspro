package user

import (
	"net/http"
	"strconv"
	"time"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/templates"
	"github.com/gin-gonic/gin"
)

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

// VerifyEmailPage is safe for mail scanners: GET never consumes a token.
func (h *AuthHandler) VerifyEmailPage(c *gin.Context) {
	var req structs.VerifyEmailQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Reject(c, http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
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
