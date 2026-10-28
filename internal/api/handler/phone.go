package handler

import (
	"errors"
	"net/http"
	"time"

	authrequest "github.com/datvtph41107/bdspro/internal/api/handler/request/auth"
	"github.com/datvtph41107/bdspro/internal/api/handler/resource"
	authservice "github.com/datvtph41107/bdspro/internal/api/service"
	"github.com/datvtph41107/bdspro/internal/config"
	apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
	middleware "github.com/datvtph41107/bdspro/internal/core/middleware"
	"github.com/datvtph41107/bdspro/internal/core/response"
	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
)

type PhoneHandler struct {
	store      db.Store
	challenges db.PhoneOTPRepository
	maker      token.Maker
	config     config.Config
	limiter    *middleware.IdentityLimiter
}

func NewPhoneHandler(store db.Store, challenges db.PhoneOTPRepository, maker token.Maker, config config.Config, limiter *middleware.IdentityLimiter) *PhoneHandler {
	return &PhoneHandler{store: store, challenges: challenges, maker: maker, config: config, limiter: limiter}
}

func (c *PhoneHandler) phoneOTP() authservice.PhoneOTP {
	return authservice.PhoneOTP{Store: c.store, Challenges: c.challenges, Maker: c.maker, Config: c.config}
}

func (c *PhoneHandler) StartBind(ctx *gin.Context) {
	var input authrequest.PhoneBindBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Invalid(ctx, err)
		return
	}
	if !c.limiter.Allow(ctx, "phone_bind_start", input.Phone, 10, 3, time.Hour) {
		return
	}
	err := c.phoneOTP().StartBind(ctx.Request.Context(), middleware.AccountID(ctx), input.Password, input.Phone)
	if err == nil {
		ctx.Status(http.StatusAccepted)
		return
	}
	if errors.Is(err, authservice.ErrPhoneInvalid) {
		err = apperrors.InvalidPhone
	}
	if errors.Is(err, authservice.ErrPhoneUnavailable) {
		err = apperrors.PhoneUnavailable
	}
	if errors.Is(err, authservice.ErrInvalidCredentials) {
		err = apperrors.InvalidCredentials
	}
	if errors.Is(err, db.ErrOTPCooldown) {
		err = apperrors.OTPCooldown
	}
	response.Error(ctx, err)
}

func (c *PhoneHandler) VerifyBind(ctx *gin.Context) {
	var input authrequest.PhoneVerifyBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Invalid(ctx, err)
		return
	}
	if !c.limiter.Allow(ctx, "phone_bind_verify", input.Phone, 40, 5, 10*time.Minute) {
		return
	}
	err := c.phoneOTP().VerifyBind(ctx.Request.Context(), middleware.AccountID(ctx), input.Phone, input.Code)
	if err == nil {
		ctx.Status(http.StatusNoContent)
		return
	}
	if errors.Is(err, db.ErrOTPInvalid) {
		err = apperrors.OTPInvalid
	}
	if pgErr := apperrors.Postgres(err); pgErr != nil && pgErr.Code == "23505" {
		err = apperrors.PhoneUnavailable
	}
	response.Error(ctx, err)
}

func (c *PhoneHandler) StartLogin(ctx *gin.Context) {
	var input authrequest.PhoneStartBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Invalid(ctx, err)
		return
	}
	if !c.limiter.Allow(ctx, "phone_login_start", input.Phone, 10, 3, time.Hour) {
		return
	}
	err := c.phoneOTP().StartLogin(ctx.Request.Context(), input.Phone)
	if err == nil {
		ctx.Status(http.StatusAccepted)
		return
	}
	if errors.Is(err, authservice.ErrPhoneInvalid) {
		err = apperrors.InvalidPhone
	}
	response.Error(ctx, err)
}

func (c *PhoneHandler) VerifyLogin(ctx *gin.Context) {
	var input authrequest.PhoneVerifyBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Invalid(ctx, err)
		return
	}
	if !c.limiter.Allow(ctx, "phone_login_verify", input.Phone, 40, 5, 10*time.Minute) {
		return
	}
	result, err := c.phoneOTP().VerifyLogin(
		ctx.Request.Context(), input.Phone, input.Code, ctx.Request.UserAgent(), ctx.ClientIP(),
	)
	if errors.Is(err, db.ErrOTPInvalid) {
		err = apperrors.OTPInvalid
	}
	if err != nil {
		response.Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, resources.NewLogin(result))
}
