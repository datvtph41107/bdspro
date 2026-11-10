// Package oauth owns OAuth2 authorization-server HTTP actions.
// Browser and API frontends share these typed actions; grant validation,
// revocation and transactional issuance remain in services/oauth and models.
package oauth

import (
	"errors"
	"fmt"
	"net/http"
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
	Name        string `json:"name" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

// OAuth protocol errors use the OAuth envelope rather than first-party API errors.
func failure(ctx *appctx.APIContext, status int, code string) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.JSON(status, map[string]any{"error": code})
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
	case errors.Is(err, oauth_service.ErrConsentProofUnavailable):
		failure(ctx, http.StatusServiceUnavailable, "temporarily_unavailable")
	default:
		failure(ctx, http.StatusInternalServerError, "server_error")
	}
}

func service(ctx *appctx.APIContext) (*oauth_service.Service, bool) {
	s := oauth_service.ServiceFromContext(ctx)
	if s == nil {
		ctx.APIErrorNotFound()
		return nil, false
	}
	return s, true
}

func actorID(ctx *appctx.APIContext) (int64, bool) {
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return 0, false
	}
	return actor.AccountID, true
}

// RequireEnabled is evaluated before OAuth protocol operations.
func RequireEnabled(ctx *appctx.APIContext) {
	if oauth_service.ServiceFromContext(ctx) == nil {
		ctx.APIErrorNotFound()
	}
}

func CreateClient(ctx *appctx.APIContext) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	accountID, ok := actorID(ctx)
	if !ok {
		return
	}
	var request clientRequest
	if err := ctx.BindJSON(&request); err != nil {
		failure(ctx, http.StatusBadRequest, "invalid_request")
		return
	}
	client, err := s.RegisterClient(ctx, accountID, request.Name, request.RedirectURI)
	if err != nil {
		serviceFailure(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToOAuthClient(client))
}

func ListClients(ctx *appctx.APIContext) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	accountID, ok := actorID(ctx)
	if !ok {
		return
	}
	clients, err := s.ListClients(ctx, accountID)
	if err != nil {
		serviceFailure(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{"clients": convert.ToOAuthClients(clients)})
}

func DeleteClient(ctx *appctx.APIContext) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	accountID, ok := actorID(ctx)
	if !ok {
		return
	}
	id, err := uuid.Parse(ctx.PathParam("id"))
	if err != nil {
		failure(ctx, http.StatusBadRequest, "invalid_request")
		return
	}
	if err = s.DeleteClient(ctx, accountID, id); err != nil {
		serviceFailure(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// AuthorizePreview is an HTTP representation of the canonical OAuth consent
// decision owned by services/oauth. GET does not consume or issue grants.
func AuthorizePreview(ctx *appctx.APIContext) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	accountID, ok := actorID(ctx)
	if !ok {
		return
	}
	req, err := oauth_service.ParseAuthorizationRequest(ctx.CurrentRequest().URL.Query())
	if err != nil {
		serviceFailure(ctx, err)
		return
	}
	preview, err := s.PreviewConsent(ctx, accountID, req, oauth_service.ProofsFromContext(ctx), time.Now())
	if err != nil {
		serviceFailure(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("X-Frame-Options", "DENY")
	ctx.JSON(http.StatusOK, map[string]any{
		"client_id": preview.Client.ID.String(), "name": preview.Client.Name,
		"redirect_uri": preview.Client.RedirectURI, "scope": preview.Request.Scope,
		"consent_required": true, "consent_version": preview.Request.ConsentVersion,
		"consent_proof": preview.Proof,
	})
}

// Authorize delegates signed consent approval and registered redirect validation
// to services/oauth; HTTP retains body limits, response and redirect policy.
func Authorize(ctx *appctx.APIContext, w http.ResponseWriter) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	accountID, ok := actorID(ctx)
	if !ok {
		return
	}
	req := ctx.CurrentRequest()
	req.Body = http.MaxBytesReader(w, req.Body, 8<<10)
	if err := req.ParseForm(); err != nil {
		failure(ctx, http.StatusBadRequest, "invalid_request")
		return
	}
	decision, approved, proof, err := oauth_service.ParseConsentDecision(req.PostForm)
	if err != nil {
		serviceFailure(ctx, err)
		return
	}
	payload, ok := ctx.SignedPrincipal()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	redirect, err := s.DecideConsent(ctx, accountID, payload.SessionID,
		decision, approved, proof, oauth_service.ProofsFromContext(ctx), time.Now())
	if err != nil {
		serviceFailure(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	http.Redirect(w, req, redirect, http.StatusFound)
}

// Authorization-server tokens are intentionally independent of first-party
// login sessions. Preserve the exact OAuth error codes and form-size bound.
func Token(ctx *appctx.APIContext, w http.ResponseWriter) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	req := ctx.CurrentRequest()
	req.Body = http.MaxBytesReader(w, req.Body, 8<<10)
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") || req.ParseForm() != nil {
		failure(ctx, http.StatusBadRequest, "invalid_request")
		return
	}
	form := req.PostForm
	id, err := uuid.Parse(form.Get("client_id"))
	if err != nil {
		failure(ctx, http.StatusUnauthorized, "invalid_client")
		return
	}
	var result oauth_service.TokenResponse
	switch form.Get("grant_type") {
	case "authorization_code":
		result, err = s.ExchangeCode(ctx, id, form.Get("code"), form.Get("redirect_uri"), form.Get("code_verifier"))
	case "refresh_token":
		result, err = s.RefreshToken(ctx, id, form.Get("refresh_token"))
	default:
		failure(ctx, http.StatusBadRequest, "unsupported_grant_type")
		return
	}
	if err != nil {
		serviceFailure(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.JSON(http.StatusOK, result)
}

func Revoke(ctx *appctx.APIContext, w http.ResponseWriter) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	req := ctx.CurrentRequest()
	req.Body = http.MaxBytesReader(w, req.Body, 8<<10)
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") || req.ParseForm() != nil {
		failure(ctx, http.StatusBadRequest, "invalid_request")
		return
	}
	id, err := uuid.Parse(req.PostForm.Get("client_id"))
	if err != nil {
		failure(ctx, http.StatusUnauthorized, "invalid_client")
		return
	}
	if err := s.RevokeToken(ctx, id, req.PostForm.Get("token")); err != nil {
		serviceFailure(ctx, err)
		return
	}
	ctx.Status(http.StatusOK)
}

func UserInfo(ctx *appctx.APIContext) {
	s, ok := service(ctx)
	if !ok {
		return
	}
	parts := strings.Fields(ctx.CurrentRequest().Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		ctx.Header("WWW-Authenticate", "Bearer")
		failure(ctx, http.StatusUnauthorized, "invalid_token")
		return
	}
	access, err := s.UserAccess(ctx, parts[1])
	if errors.Is(err, oauth_service.ErrInvalidGrant) {
		ctx.Header("WWW-Authenticate", "Bearer")
		failure(ctx, http.StatusUnauthorized, "invalid_token")
		return
	}
	if err != nil {
		userinfoStoreUnavailable(ctx, err)
		return
	}
	if access.Scope != oauth_service.ScopeProfile {
		failure(ctx, http.StatusServiceUnavailable, "temporarily_unavailable")
		failure(ctx, http.StatusForbidden, "insufficient_scope")
		return
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		userinfoStoreUnavailable(ctx, err)
		return
	}
	account, err := account_model.GetByID(ctx, store, access.AccountID)
	if errors.Is(err, account_model.ErrAccountUnavailable) {
		ctx.Header("WWW-Authenticate", "Bearer")
		failure(ctx, http.StatusUnauthorized, "invalid_token")
		return
	}
	if err != nil {
		userinfoStoreUnavailable(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, map[string]any{"sub": account.ID})
}

func userinfoStoreUnavailable(ctx *appctx.APIContext, err error) {
	log.Ctx(ctx).Error().Str("dependency", "postgres").
		Str("error_type", fmt.Sprintf("%T", err)).Msg("oauth.userinfo_store_unavailable")
	failure(ctx, http.StatusServiceUnavailable, "temporarily_unavailable")
}
