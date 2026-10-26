package auth

import (
    "errors"
    "net/http"

    apperrors "github.com/datvtph41107/bdspro/app/errors"
    "github.com/datvtph41107/bdspro/app/http/binding"
    middleware "github.com/datvtph41107/bdspro/app/http/middleware"
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
    "github.com/datvtph41107/bdspro/app/http/resources"
    "github.com/datvtph41107/bdspro/app/http/responses"
    authservice "github.com/datvtph41107/bdspro/auth"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/token"
    "github.com/datvtph41107/bdspro/util"
    "github.com/gin-gonic/gin"
)

type PhoneController struct {
    store db.Store
    challenges db.PhoneOTPRepository
    maker token.Maker
    config util.Config
}

func NewPhoneController(store db.Store, challenges db.PhoneOTPRepository, maker token.Maker, config util.Config) *PhoneController {
    return &PhoneController{store:store,challenges:challenges,maker:maker,config:config}
}

func (c *PhoneController) phoneOTP() authservice.PhoneOTP {
    return authservice.PhoneOTP{Store:c.store,Challenges:c.challenges,Maker:c.maker,Config:c.config}
}

func (c *PhoneController) StartBind(ctx *gin.Context) {
    input := binding.Input[authrequest.PhoneBindBody](ctx, authrequest.PhoneKey)
    err := c.phoneOTP().StartBind(ctx.Request.Context(), middleware.AccountID(ctx), input.Password, input.Phone)
    if err == nil { ctx.Status(http.StatusAccepted); return }
    if errors.Is(err, authservice.ErrPhoneInvalid) { err = apperrors.InvalidPhone }
    if errors.Is(err, authservice.ErrPhoneUnavailable) { err = apperrors.PhoneUnavailable }
    if errors.Is(err, authservice.ErrInvalidCredentials) { err = apperrors.InvalidCredentials }
    if errors.Is(err, db.ErrOTPCooldown) { err = apperrors.OTPCooldown }
    responses.Error(ctx, err, "phone bind failed")
}

func (c *PhoneController) VerifyBind(ctx *gin.Context) {
    input := binding.Input[authrequest.PhoneVerifyBody](ctx, authrequest.PhoneKey)
    err := c.phoneOTP().VerifyBind(ctx.Request.Context(), middleware.AccountID(ctx), input.Phone, input.Code)
    if err == nil { ctx.Status(http.StatusNoContent); return }
    if errors.Is(err, db.ErrOTPInvalid) { err = apperrors.OTPInvalid }
    if pgErr := apperrors.Postgres(err); pgErr != nil && pgErr.Code == "23505" {
        err = apperrors.PhoneUnavailable
    }
    responses.Error(ctx, err, "phone verification failed")
}

func (c *PhoneController) StartLogin(ctx *gin.Context) {
    input := binding.Input[authrequest.PhoneStartBody](ctx, authrequest.PhoneKey)
    err := c.phoneOTP().StartLogin(ctx.Request.Context(), input.Phone)
    if err == nil { ctx.Status(http.StatusAccepted); return }
    if errors.Is(err, authservice.ErrPhoneInvalid) { err = apperrors.InvalidPhone }
    responses.Error(ctx, err, "phone login unavailable")
}

func (c *PhoneController) VerifyLogin(ctx *gin.Context) {
    input := binding.Input[authrequest.PhoneVerifyBody](ctx, authrequest.PhoneKey)
    result, err := c.phoneOTP().VerifyLogin(
        ctx.Request.Context(), input.Phone, input.Code, ctx.Request.UserAgent(), ctx.ClientIP(),
    )
    if errors.Is(err, db.ErrOTPInvalid) { err = apperrors.OTPInvalid }
    responses.Send(ctx, http.StatusOK, resources.NewLogin(result), err, "phone login failed")
}
