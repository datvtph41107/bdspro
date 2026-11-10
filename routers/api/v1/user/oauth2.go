package user

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/structs"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	browser "github.com/datvtph41107/bdspro/routers/web/oauth"
)

const oauth2StateCookie = "bdspro_oidc_state"

// stateCookie persists exactly the provider challenge state and no user token.
func stateCookie(ctx *appctx.APIContext, w http.ResponseWriter, state string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: oauth2StateCookie, Value: state, Path: "/users/oauth2/callback",
		MaxAge: maxAge, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func oidc(ctx *appctx.APIContext) (*auth_service.OAuth2Client, bool, bool) {
	client, secure := auth_service.OAuth2FromContext(ctx)
	if client == nil {
		ctx.APIErrorNotFound()
		return nil, false, false
	}
	return client, secure, true
}

// Disabled provider endpoints remain publicly unavailable.
func RequireOIDC(ctx *appctx.APIContext) {
	client,_:=auth_service.OAuth2FromContext(ctx)
	if client==nil {ctx.APIErrorNotFound()}
}

func RequireBrowserOIDC(ctx *appctx.APIContext) {
	client, _ := auth_service.OAuth2FromContext(ctx)
	b := browser.FromContext(ctx)
	if client == nil || b == nil || !b.OIDCEnabled {
		ctx.APIErrorNotFound()
	}
}

func StartLogin(ctx *appctx.APIContext, w http.ResponseWriter) {
	client, secure, ok := oidc(ctx); if !ok {return}
	start, err := client.Begin(ctx, "login", 0, "")
	if err != nil {oidcUnavailable(ctx, err);return}
	stateCookie(ctx,w,start.State,start.MaxAge,secure)
	ctx.Header("Cache-Control","no-store")
	http.Redirect(w,ctx.CurrentRequest(),start.AuthorizationURL,http.StatusFound)
}

// Only an already validated local consent return target is stored in Redis.
func StartBrowserLogin(ctx *appctx.APIContext, w http.ResponseWriter) {
	client,secure,ok:=oidc(ctx);if !ok{return}
	completer:=browser.FromContext(ctx)
	if completer==nil||!completer.OIDCEnabled {ctx.APIErrorNotFound();return}
	next,err:=authcrypto.BrowserNextPath(ctx.CurrentRequest().URL.Query().Get("next"))
	if err!=nil {ctx.APIError(http.StatusBadRequest,"oauth2.invalid_return_path","invalid consent return path");return}
	start,err:=client.BeginBrowserLogin(ctx,next)
	if err!=nil {oidcUnavailable(ctx,err);return}
	stateCookie(ctx,w,start.State,start.MaxAge,secure)
	ctx.Header("Cache-Control","no-store")
	ctx.Header("Referrer-Policy","no-referrer")
	http.Redirect(w,ctx.CurrentRequest(),start.AuthorizationURL,http.StatusFound)
}

// Linking requires reauthentication; the JWT actor was verified at ingress.
func StartLink(ctx *appctx.APIContext,w http.ResponseWriter) {
	client,secure,ok:=oidc(ctx);if !ok{return}
	actor,ok:=ctx.VerifiedActor();if !ok {ctx.AuthenticationRequired();return}
	var req structs.OAuth2LinkRequest
	if err:=ctx.BindJSON(&req);err!=nil {ctx.APIInvalid(err);return}
	if (req.Password=="")==(req.PhoneCode=="") {
		ctx.APIError(http.StatusBadRequest,"oauth2.invalid_link_proof","provide exactly one reauthentication method")
		return
	}
	auth:=auth_service.FromContext(ctx)
	if req.PhoneCode!="" {
		if !auth.HasPhone() {ctx.APIError(http.StatusForbidden,"oauth2.phone_proof_unavailable","phone reauthentication unavailable");return}
		if err:=auth.AuthorizePhoneProof(ctx,actor.AccountID,req.PhoneCode);err!=nil {ctx.APIErrorAuto(err);return}
	} else if err:=auth.CheckPassword(ctx,actor.AccountID,req.Password);err!=nil {
		ctx.APIErrorAuto(err);return
	}
	start,err:=client.Begin(ctx,"link",actor.AccountID,"")
	if err!=nil {oidcUnavailable(ctx,err);return}
	stateCookie(ctx,w,start.State,start.MaxAge,secure)
	ctx.Header("Cache-Control","no-store")
	ctx.JSON(http.StatusOK,map[string]any{"authorization_url":start.AuthorizationURL})
}

func StartOAuth2PhoneBind(ctx *appctx.APIContext,w http.ResponseWriter) {
	client,secure,ok:=oidc(ctx);if !ok{return}
	auth:=auth_service.FromContext(ctx)
	if !auth.HasPhone() {ctx.APIErrorNotFound();return}
	actor,ok:=ctx.VerifiedActor();if !ok {ctx.AuthenticationRequired();return}
	var req struct {Phone string `json:"phone" binding:"required"`}
	if err:=ctx.BindJSON(&req);err!=nil {ctx.APIInvalid(err);return}
	start,err:=client.Begin(ctx,"phone_bind",actor.AccountID,req.Phone)
	if err!=nil {oidcUnavailable(ctx,err);return}
	stateCookie(ctx,w,start.State,start.MaxAge,secure)
	ctx.Header("Cache-Control","no-store")
	ctx.JSON(http.StatusOK,map[string]any{"authorization_url":start.AuthorizationURL})
}

func Callback(ctx *appctx.APIContext,w http.ResponseWriter) {
	client,secure,ok:=oidc(ctx);if !ok{return}
	ctx.Header("Cache-Control","no-store")
	ctx.Header("Referrer-Policy","no-referrer")
	state:=ctx.CurrentRequest().URL.Query().Get("state")
	cookie,err:=ctx.CurrentRequest().Cookie(oauth2StateCookie)
	if err!=nil||len(state)!=43||subtle.ConstantTimeCompare([]byte(state),[]byte(cookie.Value))!=1 {
		ctx.APIError(http.StatusBadRequest,"oauth2.invalid_state","invalid authorization state")
		return
	}
	stateCookie(ctx,w,"",-1,secure)
	pending,err:=client.Consume(ctx,state)
	if err!=nil {
		if errors.Is(err,auth_service.ErrOAuth2StateExpired)||errors.Is(err,auth_service.ErrOAuth2StateInvalid) {
			ctx.APIError(http.StatusBadRequest,"oauth2.invalid_state","expired or invalid authorization state")
		} else {oidcUnavailable(ctx,err)}
		return
	}
	query:=ctx.CurrentRequest().URL.Query()
	if query.Get("error")!=""||query.Get("code")=="" {
		ctx.APIError(http.StatusUnauthorized,"oauth2.authorization_declined","authorization declined")
		return
	}
	idToken,err:=client.Exchange(ctx,query.Get("code"),pending.Verifier)
	if errors.Is(err,auth_service.ErrOAuth2AuthorizationFailed) {
		ctx.APIError(http.StatusUnauthorized,"oauth2.authorization_failed","authorization code rejected")
		return
	}
	if err!=nil {oidcUnavailable(ctx,err);return}
	identity,err:=client.Verify(ctx,idToken,pending.Nonce)
	if err!=nil {
		ctx.APIError(http.StatusUnauthorized,"oauth2.invalid_identity","invalid identity assertion")
		return
	}
	auth:=auth_service.FromContext(ctx)
	switch pending.Purpose {
	case "phone_bind":
		if pending.AccountID<=0||!auth.HasPhone() {
			ctx.APIError(http.StatusBadRequest,"oauth2.invalid_state","invalid authorization state")
			return
		}
		if err:=auth.StartOAuth2PhoneBind(ctx,pending.AccountID,identity,pending.Phone);err!=nil {
			ctx.APIErrorAuto(err);return
		}
		ctx.JSON(http.StatusAccepted,map[string]any{"phone_verification_started":true})
	case "link":
		if pending.AccountID<=0 {
			ctx.APIError(http.StatusBadRequest,"oauth2.invalid_state","invalid authorization state")
			return
		}
		if err:=auth.LinkOIDC(ctx,pending.AccountID,identity);err!=nil {
			ctx.APIErrorAuto(err);return
		}
		ctx.JSON(http.StatusOK,map[string]any{"linked":true})
	case "login":
		if pending.AccountID!=0 {
			ctx.APIError(http.StatusBadRequest,"oauth2.invalid_state","invalid authorization state")
			return
		}
		completer:=browser.FromContext(ctx)
		if pending.ReturnPath!="" && completer==nil {
			ctx.APIError(http.StatusServiceUnavailable,"oauth2.browser_unavailable","browser OAuth login unavailable")
			return
		}
		result,err:=auth.LoginOIDC(ctx,identity,ctx.CurrentRequest().UserAgent(),ctx.ClientIP())
		if err!=nil {ctx.APIErrorAuto(err);return}
		if pending.ReturnPath!="" {
			completer.CompleteOIDCLogin(ctx,w,result,pending.ReturnPath)
			return
		}
		ctx.JSON(http.StatusOK,convert.ToLogin(result))
	default:
		ctx.APIError(http.StatusBadRequest,"oauth2.invalid_state","unsupported authorization purpose")
	}
}

func oidcUnavailable(ctx *appctx.APIContext,err error) {
	log.Ctx(ctx).Error().Str("dependency","openid_connect").
		Str("error_type",fmt.Sprintf("%T",err)).Msg("auth.provider_unavailable")
	ctx.APIError(http.StatusServiceUnavailable,"oauth2.unavailable","identity provider unavailable")
}
