package handler

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/notification"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
)

// PhoneHandler handles HTTP concerns. PhoneOTP owns validation and database errors.
type PhoneHandler struct {
	phone   service.PhoneOTP
	limiter *middleware.IdentityLimiter
}

func NewPhoneHandler(
	store db.Store,
	cfg config.Config,
	sender notification.PhoneCodeSender,
	limiter *middleware.IdentityLimiter,
) *PhoneHandler {
	return &PhoneHandler{
		phone: service.PhoneOTP{
			Store: store, Config: cfg, Sender: sender,
		},
		limiter: limiter,
	}
}

func (h *PhoneHandler) StartBind(c *gin.Context) {
	var req PhoneBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_bind_start", req.Phone, 10, 3, time.Hour) {
		return
	}
	if err := h.phone.StartBind(c.Request.Context(), middleware.AccountID(c), req.Password, req.Phone); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *PhoneHandler) VerifyBind(c *gin.Context) {
	var req PhoneVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_bind_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	if err := h.phone.VerifyBind(c.Request.Context(), middleware.AccountID(c), req.Phone, req.Code); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PhoneHandler) SetTokenMaker(maker token.Maker) { h.phone.Maker = maker }

// Phone signup and login are public; no AccountID is available until OTP
// succeeds. The same IP/identity admission contract applies to each action.
func (h *PhoneHandler) StartRegister(c *gin.Context) {
	var req PhoneStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_register_start", req.Phone, 10, 3, time.Hour) {
		return
	}
	if err := h.phone.StartRegister(c.Request.Context(), req.Phone); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}
func (h *PhoneHandler) VerifyRegister(c *gin.Context) {
	var req PhoneVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_register_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	result, err := h.phone.VerifyRegister(c.Request.Context(), req.Phone, req.Code, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, newLoginResponse(result))
}
func (h *PhoneHandler) StartLogin(c *gin.Context) {
	var req PhoneStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_login_start", req.Phone, 30, 10, 10*time.Minute) {
		return
	}
	if err := h.phone.StartLogin(c.Request.Context(), req.Phone); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}
func (h *PhoneHandler) VerifyLogin(c *gin.Context) {
	var req PhoneVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "phone_login_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	result, err := h.phone.VerifyLogin(c.Request.Context(), req.Phone, req.Code, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, newLoginResponse(result))
}
