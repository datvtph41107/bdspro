package auth

import (
    "context"
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "errors"
    "fmt"
    "math/big"
    "regexp"

    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/token"
    "github.com/datvtph41107/bdspro/util"
    "github.com/jackc/pgx/v5"
)

var (
    ErrPhoneInvalid=errors.New("invalid E.164 phone number")
    ErrPhoneUnavailable=errors.New("phone number unavailable")
)
var phoneE164=regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var phoneCode=regexp.MustCompile(`^[0-9]{6}$`)

// PhoneOTP is a development-only delivery mode. Codes are saved only in a
// disposable dev PostgreSQL database, never in responses or application logs.
type PhoneOTP struct {
    Store db.Store
    Challenges db.PhoneOTPRepository
    Maker token.Maker
    Config util.Config
}
func(s PhoneOTP) allowed() error {
    if s.Config.Environment!="development" { return errors.New("phone OTP delivery unavailable") }
    return nil
}
func (s PhoneOTP) generateCode() (string,error) {
    n,err:=rand.Int(rand.Reader,big.NewInt(1000000))
    if err!=nil{return "",err}
    return fmt.Sprintf("%06d",n.Int64()),nil
}
func(s PhoneOTP) mac(phone,purpose string,accountID int64,code string) []byte {
    h:=hmac.New(sha256.New,[]byte(s.Config.TokenSymmetricKey))
    _,_=fmt.Fprintf(h,"bdspro:phone-otp:v1|%s|%s|%d|%s",purpose,phone,accountID,code)
    return h.Sum(nil)
}
func(s PhoneOTP) start(ctx context.Context,phone,purpose string,accountID int64) error {
    code,err:=s.generateCode()
    if err!=nil{return err}
    return s.Challenges.StartChallenge(ctx,phone,purpose,accountID,s.mac(phone,purpose,accountID,code),code)
}
// StartBind requires an already-authenticated BDSPro account plus its current
// password before adding a new login method.
func(s PhoneOTP) StartBind(ctx context.Context,accountID int64,password,phone string) error {
    if err:=s.allowed();err!=nil{return err}
    if !phoneE164.MatchString(phone){return ErrPhoneInvalid}
    account,err:=s.Store.GetAccountByID(ctx,accountID)
    if err!=nil{return err}
    if util.CheckPassword(password,account.PasswordHash)!=nil{return ErrInvalidCredentials}
    existing,err:=s.Challenges.AccountIDForPhone(ctx,phone)
    if err!=nil && !errors.Is(err,pgx.ErrNoRows){return err}
    if err==nil && existing!=accountID{return ErrPhoneUnavailable}
    return s.start(ctx,phone,"bind",accountID)
}
func(s PhoneOTP) VerifyBind(ctx context.Context,accountID int64,phone,code string) error {
    if err:=s.allowed();err!=nil{return err}
    if !phoneE164.MatchString(phone)||!phoneCode.MatchString(code){return db.ErrOTPInvalid}
    return s.Challenges.ConsumeChallenge(ctx,phone,"bind",accountID,s.mac(phone,"bind",accountID,code))
}
// Unknown phone and resend cooldown return the same success response.
func(s PhoneOTP) StartLogin(ctx context.Context,phone string) error {
    if err:=s.allowed();err!=nil{return err}
    if !phoneE164.MatchString(phone){return ErrPhoneInvalid}
    accountID,err:=s.Challenges.AccountIDForPhone(ctx,phone)
    if errors.Is(err,pgx.ErrNoRows){return nil}
    if err!=nil{return err}
    err=s.start(ctx,phone,"login",accountID)
    if errors.Is(err,db.ErrOTPCooldown){return nil}
    return err
}
func(s PhoneOTP) VerifyLogin(ctx context.Context,phone,code,userAgent,clientIP string) (LoginResult,error) {
    var empty LoginResult
    if err:=s.allowed();err!=nil{return empty,err}
    if !phoneE164.MatchString(phone)||!phoneCode.MatchString(code){return empty,db.ErrOTPInvalid}
    accountID,err:=s.Challenges.AccountIDForPhone(ctx,phone)
    if errors.Is(err,pgx.ErrNoRows){return empty,db.ErrOTPInvalid}
    if err!=nil{return empty,err}
    if err=s.Challenges.ConsumeChallenge(ctx,phone,"login",accountID,s.mac(phone,"login",accountID,code));err!=nil{return empty,err}
    account,err:=s.Store.GetAccountByID(ctx,accountID)
    if err!=nil{return empty,err}
    return IssueSession(ctx,s.Store,s.Maker,s.Config,account,userAgent,clientIP)
}
