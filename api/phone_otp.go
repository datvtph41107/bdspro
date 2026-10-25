package api

import (
    "errors"
    "net/http"
    "time"

    "github.com/datvtph41107/bdspro/auth"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgconn"
)

type phoneBindRequest struct {
    Phone string `json:"phone" binding:"required"`
    Password string `json:"password" binding:"required"`
}
type phoneVerifyRequest struct {
    Phone string `json:"phone" binding:"required"`
    Code string `json:"code" binding:"required"`
}
type phoneStartRequest struct {
    Phone string `json:"phone" binding:"required"`
}
const phoneInputContext="phone.input"

func PrePhoneBind() gin.HandlerFunc {return func(ctx *gin.Context){
    var input phoneBindRequest
    if err:=ctx.ShouldBindJSON(&input);err!=nil{invalidJSON(ctx,err);ctx.Abort();return}
    ctx.Set(phoneInputContext,input);ctx.Next()
}}
func PrePhoneVerify() gin.HandlerFunc {return func(ctx *gin.Context){
    var input phoneVerifyRequest
    if err:=ctx.ShouldBindJSON(&input);err!=nil{invalidJSON(ctx,err);ctx.Abort();return}
    ctx.Set(phoneInputContext,input);ctx.Next()
}}
func PrePhoneLoginStart() gin.HandlerFunc {return func(ctx *gin.Context){
    var input phoneStartRequest
    if err:=ctx.ShouldBindJSON(&input);err!=nil{invalidJSON(ctx,err);ctx.Abort();return}
    ctx.Set(phoneInputContext,input);ctx.Next()
}}
func phoneBindIdentity(ctx *gin.Context)string{return ctx.MustGet(phoneInputContext).(phoneBindRequest).Phone}
func phoneVerifyIdentity(ctx *gin.Context)string{return ctx.MustGet(phoneInputContext).(phoneVerifyRequest).Phone}
func phoneStartIdentity(ctx *gin.Context)string{return ctx.MustGet(phoneInputContext).(phoneStartRequest).Phone}

func (server *Server) EnablePhoneOTP(repository db.PhoneOTPRepository) error {
    if server.config.Environment!="development" {return nil}
    if repository==nil {return errors.New("development phone OTP requires repository")}
    server.phoneOTP = repository

    server.router.POST("/users/phone/login/start",
        server.IPActiveSource("phone_login_start",10,time.Hour),
        PrePhoneLoginStart(),
        server.IPActive("phone_login_start",10,3,time.Hour,phoneStartIdentity),
        server.startPhoneLogin,
    )
    server.router.POST("/users/phone/login/verify",
        SufLogin(),
        server.IPActiveSource("phone_login_verify",40,10*time.Minute),
        PrePhoneVerify(),
        server.IPActive("phone_login_verify",40,5,10*time.Minute,phoneVerifyIdentity),
        server.verifyPhoneLogin,
    )
    protected := server.router.Group("/users/me/phone").Use(Authenticate(server.tokenMaker))
    protected.POST("/start",
        server.IPActiveSource("phone_bind_start",10,time.Hour),
        PrePhoneBind(),
        server.IPActive("phone_bind_start",10,3,time.Hour,phoneBindIdentity),
        server.startPhoneBind,
    )
    protected.POST("/verify",
        server.IPActiveSource("phone_bind_verify",40,10*time.Minute),
        PrePhoneVerify(),
        server.IPActive("phone_bind_verify",40,5,10*time.Minute,phoneVerifyIdentity),
        server.verifyPhoneBind,
    )
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
        var pgErr *pgconn.PgError
        if errors.As(err,&pgErr)&&pgErr.Code=="23505" {ctx.JSON(http.StatusConflict,gin.H{"error":"phone unavailable"});return}
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
