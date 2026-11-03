package user

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) StartEmailLink(c *gin.Context) {
	var req structs.EmailLinkStartRequest
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
	var req structs.EmailLinkConfirmRequest
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
