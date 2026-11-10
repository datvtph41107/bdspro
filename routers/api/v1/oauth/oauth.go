// Package oauth owns OAuth2 authorization-server HTTP actions.
// Browser and API frontends share these typed actions; grant validation,
// revocation and transactional issuance remain in services/oauth and models.
package oauth

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/google/uuid"
)

type clientRequest struct {
	Name string `json:"name" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

// OAuth protocol errors use the OAuth envelope rather than first-party API errors.
func failure(ctx *appctx.APIContext, status int, code string) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.JSON(status, map[string]any{"error":code})
}

func serviceFailure(ctx *appctx.APIContext, err error) {
	switch {
	case errors.Is(err, oauth_service.ErrInvalidClient):
		failure(ctx, http.StatusUnauthorized, "invalid_client")
	case errors.Is(err, oauth_service.ErrInvalidGrant):
		failure(ctx, http.StatusBadRequest, "invalid_grant")
	case errors.Is(err, oauth_service.ErrInvalidRequest):
		failure(ctx, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, oauth_service.ErrAccessDenied):
		failure(ctx, http.StatusForbidden, "access_denied")
	default:
		failure(ctx, http.StatusInternalServerError, "server_error")
	}
}

func service(ctx *appctx.APIContext) (*oauth_service.Service, bool) {
	s := oauth_service.ServiceFromContext(ctx)
	if s == nil {
		ctx.APIErrorNotFound()
		return nil,false
	}
	return s,true
}

func actorID(ctx *appctx.APIContext) (int64,bool) {
	actor,ok:=ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return 0,false
	}
	return actor.AccountID,true
}

// RequireEnabled is evaluated before OAuth protocol operations.
func RequireEnabled(ctx *appctx.APIContext) {
	if oauth_service.ServiceFromContext(ctx)==nil {ctx.APIErrorNotFound()}
}

func CreateClient(ctx *appctx.APIContext) {
	s,ok:=service(ctx);if !ok{return}
	accountID,ok:=actorID(ctx);if !ok{return}
	var request clientRequest
	if err:=ctx.BindJSON(&request);err!=nil {
		failure(ctx,http.StatusBadRequest,"invalid_request")
		return
	}
	client,err:=s.RegisterClient(ctx,accountID,request.Name,request.RedirectURI)
	if err!=nil {serviceFailure(ctx,err);return}
	ctx.JSON(http.StatusCreated,convert.ToOAuthClient(client))
}

func ListClients(ctx *appctx.APIContext) {
	s,ok:=service(ctx);if !ok{return}
	accountID,ok:=actorID(ctx);if !ok{return}
	clients,err:=s.ListClients(ctx,accountID)
	if err!=nil {serviceFailure(ctx,err);return}
	ctx.JSON(http.StatusOK,map[string]any{"clients":convert.ToOAuthClients(clients)})
}

func DeleteClient(ctx *appctx.APIContext) {
	s,ok:=service(ctx);if !ok{return}
	accountID,ok:=actorID(ctx);if !ok{return}
	id,err:=uuid.Parse(ctx.PathParam("id"))
	if err!=nil {failure(ctx,http.StatusBadRequest,"invalid_request");return}
	if err=s.DeleteClient(ctx,accountID,id);err!=nil {serviceFailure(ctx,err);return}
	ctx.Status(http.StatusNoContent)
}

func authorizationRequest(values url.Values) (oauth_service.AuthorizationRequest,error) {
	return oauth_service.ParseAuthorizationRequest(values)
}

func consentProofFields(accountID int64, req oauth_service.AuthorizationRequest) oauth_service.ConsentProofFields {
	return oauth_service.ConsentProofFields{
		AccountID: accountID, ClientID: req.ClientID.String(),
		RedirectURI: req.RedirectURI, Scope: req.Scope, ResponseType: req.ResponseType,
		CodeChallenge: req.CodeChallenge, CodeChallengeMethod: req.CodeChallengeMethod,
		State: req.State, ConsentVersion: req.ConsentVersion,
	}
}

// GET is a non-consuming consent preview, not an authorization grant.
func AuthorizePreview(ctx *appctx.APIContext) {
	s,ok:=service(ctx);if !ok{return}
	accountID,ok:=actorID(ctx);if !ok{return}
	request,err:=authorizationRequest(ctx.CurrentRequest().URL.Query())
	if err!=nil {serviceFailure(ctx,err);return}
	client,version,err:=s.ReviewConsent(ctx,accountID,request)
	if err!=nil {serviceFailure(ctx,err);return}
	proofs:=oauth_service.ProofsFromContext(ctx)
	if proofs==nil {failure(ctx,http.StatusServiceUnavailable,"temporarily_unavailable");return}
	request.ConsentVersion=version
	proof,err:=proofs.Issue(consentProofFields(accountID,request),time.Now())
	if err!=nil {serviceFailure(ctx,err);return}
	ctx.Header("Cache-Control","no-store")
	ctx.Header("Referrer-Policy","no-referrer")
	ctx.Header("X-Frame-Options","DENY")
	ctx.JSON(http.StatusOK,map[string]any{
		"client_id":client.ID.String(),"name":client.Name,
		"redirect_uri":client.RedirectURI,"scope":request.Scope,
		"consent_required":true,"consent_version":version,"consent_proof":proof,
	})
}

// Authorize requires a persisted first-party session and a matching consent
// proof. RedirectURI is validated by services/oauth before http.Redirect.
func Authorize(ctx *appctx.APIContext,w http.ResponseWriter) {
	s,ok:=service(ctx);if !ok{return}
	accountID,ok:=actorID(ctx);if !ok{return}
	req:=ctx.CurrentRequest()
	req.Body=http.MaxBytesReader(w,req.Body,8<<10)
	if err:=req.ParseForm();err!=nil {failure(ctx,http.StatusBadRequest,"invalid_request");return}
	request,err:=authorizationRequest(req.PostForm)
	if err!=nil {serviceFailure(ctx,err);return}
	if req.PostForm.Get("approve")=="true" {
		version,err:=strconv.ParseInt(req.PostForm.Get("consent_version"),10,64)
		if err!=nil||version<0 {failure(ctx,http.StatusBadRequest,"invalid_request");return}
		request.ConsentVersion=version
		proofs:=oauth_service.ProofsFromContext(ctx)
		if proofs==nil||proofs.Verify(req.PostForm.Get("consent_proof"),
			consentProofFields(accountID,request),time.Now())!=nil {
			failure(ctx,http.StatusBadRequest,"invalid_request")
			return
		}
	}
	if _,err:=s.ReviewAuthorization(ctx,request);err!=nil {serviceFailure(ctx,err);return}
	if req.PostForm.Get("approve")!="true" {
		target,_:=url.Parse(request.RedirectURI)
		q:=target.Query();q.Set("error","access_denied");q.Set("state",request.State)
		target.RawQuery=q.Encode()
		ctx.Header("Cache-Control","no-store")
		ctx.Header("Referrer-Policy","no-referrer")
		http.Redirect(w,req,target.String(),http.StatusFound)
		return
	}
	payload,ok:=ctx.SignedPrincipal()
	if !ok {ctx.AuthenticationRequired();return}
	redirect,err:=s.Authorize(ctx,accountID,payload.SessionID,request)
	if err!=nil {serviceFailure(ctx,err);return}
	ctx.Header("Cache-Control","no-store")
	ctx.Header("Referrer-Policy","no-referrer")
	http.Redirect(w,req,redirect,http.StatusFound)
}

// Authorization-server tokens are intentionally independent of first-party
// login sessions. Preserve the exact OAuth error codes and form-size bound.
func Token(ctx *appctx.APIContext,w http.ResponseWriter) {
	s,ok:=service(ctx);if !ok{return}
	req:=ctx.CurrentRequest()
	req.Body=http.MaxBytesReader(w,req.Body,8<<10)
	if !strings.HasPrefix(req.Header.Get("Content-Type"),"application/x-www-form-urlencoded")||req.ParseForm()!=nil {
		failure(ctx,http.StatusBadRequest,"invalid_request")
		return
	}
	form:=req.PostForm
	id,err:=uuid.Parse(form.Get("client_id"))
	if err!=nil {failure(ctx,http.StatusUnauthorized,"invalid_client");return}
	var result oauth_service.TokenResponse
	switch form.Get("grant_type") {
	case "authorization_code":
		result,err=s.ExchangeCode(ctx,id,form.Get("code"),form.Get("redirect_uri"),form.Get("code_verifier"))
	case "refresh_token":
		result,err=s.RefreshToken(ctx,id,form.Get("refresh_token"))
	default:
		failure(ctx,http.StatusBadRequest,"unsupported_grant_type")
		return
	}
	if err!=nil {serviceFailure(ctx,err);return}
	ctx.Header("Cache-Control","no-store")
	ctx.Header("Pragma","no-cache")
	ctx.JSON(http.StatusOK,result)
}

func Revoke(ctx *appctx.APIContext,w http.ResponseWriter) {
	s,ok:=service(ctx);if !ok{return}
	req:=ctx.CurrentRequest()
	req.Body=http.MaxBytesReader(w,req.Body,8<<10)
	if !strings.HasPrefix(req.Header.Get("Content-Type"),"application/x-www-form-urlencoded")||req.ParseForm()!=nil {
		failure(ctx,http.StatusBadRequest,"invalid_request")
		return
	}
	id,err:=uuid.Parse(req.PostForm.Get("client_id"))
	if err!=nil {failure(ctx,http.StatusUnauthorized,"invalid_client");return}
	if err:=s.RevokeToken(ctx,id,req.PostForm.Get("token"));err!=nil {serviceFailure(ctx,err);return}
	ctx.Status(http.StatusOK)
}

func UserInfo(ctx *appctx.APIContext) {
	s,ok:=service(ctx);if !ok{return}
	parts:=strings.Fields(ctx.CurrentRequest().Header.Get("Authorization"))
	if len(parts)!=2||!strings.EqualFold(parts[0],"Bearer") {
		ctx.Header("WWW-Authenticate","Bearer")
		failure(ctx,http.StatusUnauthorized,"invalid_token")
		return
	}
	access,err:=s.UserAccess(ctx,parts[1])
	if errors.Is(err,oauth_service.ErrInvalidGrant) {
		ctx.Header("WWW-Authenticate","Bearer")
		failure(ctx,http.StatusUnauthorized,"invalid_token")
		return
	}
	if err!=nil {userinfoStoreUnavailable(ctx,err);return}
	if access.Scope!=oauth_service.ScopeProfile {
		ctx.Header("WWW-Authenticate",`Bearer error="insufficient_scope", scope="profile:read"`)
		failure(ctx,http.StatusForbidden,"insufficient_scope")
		return
	}
	store,err:=db.StoreFromContext(ctx)
	if err!=nil {userinfoStoreUnavailable(ctx,err);return}
	account,err:=account_model.GetByID(ctx,store,access.AccountID)
	if errors.Is(err,account_model.ErrAccountUnavailable) {
		ctx.Header("WWW-Authenticate","Bearer")
		failure(ctx,http.StatusUnauthorized,"invalid_token")
		return
	}
	if err!=nil {userinfoStoreUnavailable(ctx,err);return}
	ctx.Header("Cache-Control","no-store")
	ctx.JSON(http.StatusOK,map[string]any{"sub":account.ID})
}

func userinfoStoreUnavailable(ctx *appctx.APIContext,err error) {
	log.Ctx(ctx).Error().Str("dependency","postgres").
		Str("error_type",fmt.Sprintf("%T",err)).Msg("oauth.userinfo_store_unavailable")
	failure(ctx,http.StatusServiceUnavailable,"temporarily_unavailable")
}
