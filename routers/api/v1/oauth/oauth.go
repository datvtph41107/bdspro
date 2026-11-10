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
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *oauth_service.Service
	store   db.Store
	proofs  *oauth_service.ConsentProofs
}

func NewHandler(service *oauth_service.Service, store db.Store, proofs ...*oauth_service.ConsentProofs) *Handler {
	h := &Handler{service: service, store: store}
	if len(proofs) > 0 {
		h.proofs = proofs[0]
	}
	return h
}

type clientRequest struct {
	Name        string `json:"name" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

func failure(c *gin.Context, status int, code string) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	appctx.GetAPIContext(c).JSON(status, gin.H{"error": code})
}

func serviceFailure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, oauth_service.ErrInvalidClient):
		failure(c, http.StatusUnauthorized, "invalid_client")
	case errors.Is(err, oauth_service.ErrInvalidGrant):
		failure(c, http.StatusBadRequest, "invalid_grant")
	case errors.Is(err, oauth_service.ErrInvalidRequest):
		failure(c, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, oauth_service.ErrAccessDenied):
		failure(c, http.StatusForbidden, "access_denied")
	default:
		failure(c, http.StatusInternalServerError, "server_error")
	}
}

func (h *Handler) CreateClient(c *gin.Context) {
	var request clientRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		failure(c, http.StatusBadRequest, "invalid_request")
		return
	}
	client, err := h.service.RegisterClient(c.Request.Context(),
		appctx.AccountID(c), request.Name, request.RedirectURI)
	if err != nil {
		serviceFailure(c, err)
		return
	}
	appctx.GetAPIContext(c).JSON(http.StatusCreated, convert.ToOAuthClient(client))
}

func (h *Handler) ListClients(c *gin.Context) {
	clients, err := h.service.ListClients(c.Request.Context(), appctx.AccountID(c))
	if err != nil {
		serviceFailure(c, err)
		return
	}
	out := convert.ToOAuthClients(clients)
	appctx.GetAPIContext(c).JSON(http.StatusOK, gin.H{"clients": out})
}

func (h *Handler) DeleteClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		failure(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.service.DeleteClient(c.Request.Context(), appctx.AccountID(c), id); err != nil {
		serviceFailure(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Authenticate is the high-trust OAuth authorization boundary. Unlike
// ordinary API authentication, it resolves the access JWT's issuing session
// from PostgreSQL on every request and fails closed if the session is gone.
func (h *Handler) Authenticate(maker authcrypto.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			appctx.BearerRequired(c)
			return
		}
		payload, err := maker.VerifyToken(parts[1], authcrypto.TokenTypeAccess)
		if err != nil {
			appctx.InvalidAccessToken(c)
			return
		}
		if err := auth_model.RequireActiveSession(c.Request.Context(), h.store,
			payload.AccountID, payload.SessionID, time.Now()); err != nil {
			if errors.Is(err, auth_model.ErrSessionUnavailable) {
				appctx.InvalidAccessToken(c)
			} else {
				failure(c, http.StatusServiceUnavailable, "temporarily_unavailable")
				c.Abort()
			}
			return
		}
		c.Set(appctx.AuthorizationPayloadKey, payload)
		c.Next()
	}
}

func authorizationRequest(values url.Values) (oauth_service.AuthorizationRequest, error) {
	return oauth_service.ParseAuthorizationRequest(values)
}

// consentProofFields binds a preview to the authenticated Account and every
// OAuth request attribute. Never take AccountID from user-submitted data.
func consentProofFields(accountID int64, req oauth_service.AuthorizationRequest) oauth_service.ConsentProofFields {
	return oauth_service.ConsentProofFields{
		AccountID: accountID, ClientID: req.ClientID.String(),
		RedirectURI: req.RedirectURI, Scope: req.Scope, ResponseType: req.ResponseType,
		CodeChallenge: req.CodeChallenge, CodeChallengeMethod: req.CodeChallengeMethod,
		State: req.State, ConsentVersion: req.ConsentVersion,
	}
}

// AuthorizePreview supplies consent text to an already-authenticated browser
// frontend; it never issues a code as a side effect of GET.
func (h *Handler) AuthorizePreview(c *gin.Context) {
	request, err := authorizationRequest(c.Request.URL.Query())
	if err != nil {
		serviceFailure(c, err)
		return
	}
	client, version, err := h.service.ReviewConsent(c.Request.Context(), appctx.AccountID(c), request)
	if err != nil {
		serviceFailure(c, err)
		return
	}
	if h.proofs == nil {
		failure(c, http.StatusServiceUnavailable, "temporarily_unavailable")
		return
	}
	request.ConsentVersion = version
	proof, err := h.proofs.Issue(consentProofFields(appctx.AccountID(c), request), time.Now())
	if err != nil {
		serviceFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Frame-Options", "DENY")
	appctx.GetAPIContext(c).JSON(http.StatusOK, gin.H{
		"client_id": client.ID.String(), "name": client.Name,
		"redirect_uri": client.RedirectURI, "scope": request.Scope,
		"consent_required": true, "consent_version": version, "consent_proof": proof,
	})
}

// Authorize requires an explicit approval in an authenticated POST.
// Untrusted redirect URIs are never used in Location responses.
func (h *Handler) Authorize(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
	if err := c.Request.ParseForm(); err != nil {
		failure(c, http.StatusBadRequest, "invalid_request")
		return
	}
	request, err := authorizationRequest(c.Request.PostForm)
	if err != nil {
		serviceFailure(c, err)
		return
	}
	if c.Request.PostForm.Get("approve") == "true" {
		version, err := strconv.ParseInt(c.Request.PostForm.Get("consent_version"), 10, 64)
		if err != nil || version < 0 {
			failure(c, http.StatusBadRequest, "invalid_request")
			return
		}
		request.ConsentVersion = version
		if h.proofs == nil || h.proofs.Verify(c.Request.PostForm.Get("consent_proof"),
			consentProofFields(appctx.AccountID(c), request), time.Now()) != nil {
			failure(c, http.StatusBadRequest, "invalid_request")
			return
		}
	}
	if _, err := h.service.ReviewAuthorization(c.Request.Context(), request); err != nil {
		serviceFailure(c, err)
		return
	}
	if c.Request.PostForm.Get("approve") != "true" {
		target, _ := url.Parse(request.RedirectURI)
		q := target.Query()
		q.Set("error", "access_denied")
		q.Set("state", request.State)
		target.RawQuery = q.Encode()
		c.Header("Cache-Control", "no-store")
		c.Header("Referrer-Policy", "no-referrer")
		c.Redirect(http.StatusFound, target.String())
		return
	}
	// The DB transaction locks this exact issuing session against revoke.
	payload := c.MustGet(appctx.AuthorizationPayloadKey).(*authcrypto.Payload)
	redirect, err := h.service.Authorize(c.Request.Context(), appctx.AccountID(c), payload.SessionID, request)
	if err != nil {
		serviceFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusFound, redirect)
}

func (h *Handler) Token(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/x-www-form-urlencoded") ||
		c.Request.ParseForm() != nil {
		failure(c, http.StatusBadRequest, "invalid_request")
		return
	}
	form := c.Request.PostForm
	id, err := uuid.Parse(form.Get("client_id"))
	if err != nil {
		failure(c, http.StatusUnauthorized, "invalid_client")
		return
	}
	var result oauth_service.TokenResponse
	switch form.Get("grant_type") {
	case "authorization_code":
		result, err = h.service.ExchangeCode(c.Request.Context(), id,
			form.Get("code"), form.Get("redirect_uri"), form.Get("code_verifier"))
	case "refresh_token":
		result, err = h.service.RefreshToken(c.Request.Context(), id, form.Get("refresh_token"))
	default:
		failure(c, http.StatusBadRequest, "unsupported_grant_type")
		return
	}
	if err != nil {
		serviceFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	appctx.GetAPIContext(c).JSON(http.StatusOK, result)
}

func (h *Handler) Revoke(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/x-www-form-urlencoded") ||
		c.Request.ParseForm() != nil {
		failure(c, http.StatusBadRequest, "invalid_request")
		return
	}
	id, err := uuid.Parse(c.Request.PostForm.Get("client_id"))
	if err != nil {
		failure(c, http.StatusUnauthorized, "invalid_client")
		return
	}
	if err := h.service.RevokeToken(c.Request.Context(), id, c.Request.PostForm.Get("token")); err != nil {
		serviceFailure(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) UserInfo(c *gin.Context) {
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.Header("WWW-Authenticate", "Bearer")
		failure(c, http.StatusUnauthorized, "invalid_token")
		return
	}
	access, err := h.service.UserAccess(c.Request.Context(), parts[1])
	if errors.Is(err, oauth_service.ErrInvalidGrant) {
		c.Header("WWW-Authenticate", "Bearer")
		failure(c, http.StatusUnauthorized, "invalid_token")
		return
	}
	if err != nil {
		userinfoStoreUnavailable(c, err)
		return
	}
	if access.Scope != oauth_service.ScopeProfile {
		c.Header("WWW-Authenticate", `Bearer error="insufficient_scope", scope="profile:read"`)
		failure(c, http.StatusForbidden, "insufficient_scope")
		return
	}
	account, err := account_model.GetByID(c.Request.Context(), h.store, access.AccountID)
	if errors.Is(err, account_model.ErrAccountUnavailable) {
		c.Header("WWW-Authenticate", "Bearer")
		failure(c, http.StatusUnauthorized, "invalid_token")
		return
	}
	if err != nil {
		userinfoStoreUnavailable(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	// The sole current OAuth scope, profile:read, grants a subject identifier;
	// contact data requires a separate explicit scope and consent policy.
	appctx.GetAPIContext(c).JSON(http.StatusOK, gin.H{"sub": account.ID})
}

// Do not disguise a failed PostgreSQL lookup as a rejected OAuth credential.
func userinfoStoreUnavailable(c *gin.Context, err error) {
	log.Ctx(c.Request.Context()).Error().
		Str("dependency", "postgres").
		Str("error_type", fmt.Sprintf("%T", err)).
		Msg("oauth.userinfo_store_unavailable")
	failure(c, http.StatusServiceUnavailable, "temporarily_unavailable")
}
