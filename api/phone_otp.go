package api

import (
    "github.com/datvtph41107/bdspro/routes"
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
    "errors"
    "net/http"

    "github.com/datvtph41107/bdspro/auth"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/gin-gonic/gin"
)

type phoneBindRequest = authrequest.PhoneBindBody
type phoneVerifyRequest = authrequest.PhoneVerifyBody
type phoneStartRequest = authrequest.PhoneStartBody
const phoneInputContext="phone.input"

func PrePhoneBind() gin.HandlerFunc {
    return preJSON[phoneBindRequest](phoneInputContext)
}
func PrePhoneVerify() gin.HandlerFunc {
    return preJSON[phoneVerifyRequest](phoneInputContext)
}
func PrePhoneLoginStart() gin.HandlerFunc {
    return preJSON[phoneStartRequest](phoneInputContext)
}
func phoneBindIdentity(ctx *gin.Context)string{return ctx.MustGet(phoneInputContext).(phoneBindRequest).Phone}
func phoneVerifyIdentity(ctx *gin.Context)string{return ctx.MustGet(phoneInputContext).(phoneVerifyRequest).Phone}
func phoneStartIdentity(ctx *gin.Context)string{return ctx.MustGet(phoneInputContext).(phoneStartRequest).Phone}

func (server *Server) EnablePhoneOTP(repository db.PhoneOTPRepository) error {
    if server.config.Environment!="development" {return nil}
    if repository==nil {return errors.New("development phone OTP requires repository")}
    server.phoneOTP = repository

    routes.MountPhone(server.router, routes.PhoneHandlers{
        StartLogin:server.startPhoneLogin, VerifyLogin:server.verifyPhoneLogin,
        StartBind:server.startPhoneBind, VerifyBind:server.verifyPhoneBind,
        PreStart:PrePhoneLoginStart(), PreVerify:PrePhoneVerify(), PreBind:PrePhoneBind(),
        SufLogin:SufLogin(), Authenticate:Authenticate(server.tokenMaker),
        Source:server.IPActiveSource, Identity:server.IPActive,
        StartIdentity:phoneStartIdentity, VerifyIdentity:phoneVerifyIdentity,
        BindIdentity:phoneBindIdentity,
    })
    return nil
}

func (server *Server) phoneOTPService() auth.PhoneOTP {
    return auth.PhoneOTP{Store:server.store,Challenges:server.phoneOTP,Maker:server.tokenMaker,Config:server.config}
}
func (server *Server) startPhoneBind(ctx *gin.Context) {
    req:=ctx.MustGet(phoneInputContext).(phoneBindRequest)
    accountID:=authenticatedAccountID(ctx)
    err:=server.phoneOTPService().StartBind(ctx.Request.Context(),accountID,req.Password,req.Phone)
    switch {
    case err==nil:ctx.Status(http.StatusAccepted)
    case errors.Is(err,auth.ErrPhoneInvalid):ctx.JSON(http.StatusBadRequest,gin.H{"error":"invalid phone"})
    case errors.Is(err,auth.ErrInvalidCredentials):ctx.JSON(http.StatusUnauthorized,gin.H{"error":"invalid credentials"})
    case errors.Is(err,auth.ErrPhoneUnavailable):ctx.JSON(http.StatusConflict,gin.H{"error":"phone unavailable"})
    case errors.Is(err,db.ErrOTPCooldown):ctx.JSON(http.StatusTooManyRequests,gin.H{"error":"code temporarily unavailable"})
    default:ctx.JSON(http.StatusInternalServerError,gin.H{"error":"phone bind failed"})
    }
}
func (server *Server) verifyPhoneBind(ctx *gin.Context) {
    req:=ctx.MustGet(phoneInputContext).(phoneVerifyRequest)
    accountID:=authenticatedAccountID(ctx)
    err:=server.phoneOTPService().VerifyBind(ctx.Request.Context(),accountID,req.Phone,req.Code)
    switch {
    case err==nil:ctx.Status(http.StatusNoContent)
    case errors.Is(err,db.ErrOTPInvalid):ctx.JSON(http.StatusUnauthorized,gin.H{"error":"invalid or expired code"})
    default:
        if pgErr := postgresError(err); pgErr != nil && pgErr.Code=="23505" {ctx.JSON(http.StatusConflict,gin.H{"error":"phone unavailable"});return}
        ctx.JSON(http.StatusInternalServerError,gin.H{"error":"phone verification failed"})
    }
}
func (server *Server) startPhoneLogin(ctx *gin.Context) {
    req:=ctx.MustGet(phoneInputContext).(phoneStartRequest)
    err:=server.phoneOTPService().StartLogin(ctx.Request.Context(),req.Phone)
    if errors.Is(err,auth.ErrPhoneInvalid){ctx.JSON(http.StatusBadRequest,gin.H{"error":"invalid phone"});return}
    if err!=nil{ctx.JSON(http.StatusInternalServerError,gin.H{"error":"phone login unavailable"});return}
    ctx.Status(http.StatusAccepted) // neutral to registered/unregistered phone.
}
func (server *Server) verifyPhoneLogin(ctx *gin.Context) {
    req:=ctx.MustGet(phoneInputContext).(phoneVerifyRequest)
    result,err:=server.phoneOTPService().VerifyLogin(ctx.Request.Context(),req.Phone,req.Code,ctx.Request.UserAgent(),ctx.ClientIP())
    if errors.Is(err,db.ErrOTPInvalid){ctx.JSON(http.StatusUnauthorized,gin.H{"error":"invalid or expired code"});return}
    if err!=nil{ctx.JSON(http.StatusInternalServerError,gin.H{"error":"phone login failed"});return}
    ctx.JSON(http.StatusOK,newLoginUserResponse(result))
}
