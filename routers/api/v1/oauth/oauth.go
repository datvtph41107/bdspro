package oauth

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *oauth_service.Service
	store   db.Store
}

func NewHandler(service *oauth_service.Service, store db.Store) *Handler {
	return &Handler{service: service, store: store}
}

type clientRequest struct {
	Name        string `json:"name" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

func clientResponse(client db.OauthClient) gin.H {
	return gin.H{
		"client_id":    client.ID.String(),
		"name":         client.Name,
		"redirect_uri": client.RedirectUri,
		"created_at":   client.CreatedAt,
	}
}

func failure(c *gin.Context, status int, code string) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(status, gin.H{"error": code})
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
		middleware.AccountID(c), request.Name, request.RedirectURI)
	if err != nil {
		serviceFailure(c, err)
		return
	}
	c.JSON(http.StatusCreated, clientResponse(client))
}

func (h *Handler) ListClients(c *gin.Context) {
	clients, err := h.service.ListClients(c.Request.Context(), middleware.AccountID(c))
	if err != nil {
		serviceFailure(c, err)
		return
	}
	out := make([]gin.H, 0, len(clients))
	for _, client := range clients {
		out = append(out, clientResponse(client))
	}
	c.JSON(http.StatusOK, gin.H{"clients": out})
}

func (h *Handler) DeleteClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		failure(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.service.DeleteClient(c.Request.Context(), middleware.AccountID(c), id); err != nil {
		serviceFailure(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func authorizationRequest(values url.Values) (oauth_service.AuthorizationRequest, error) {
	id, err := uuid.Parse(values.Get("client_id"))
	if err != nil {
		return oauth_service.AuthorizationRequest{}, oauth_service.ErrInvalidRequest
	}
	return oauth_service.AuthorizationRequest{
		ClientID:            id,
		RedirectURI:         values.Get("redirect_uri"),
		ResponseType:        values.Get("response_type"),
		CodeChallenge:       values.Get("code_challenge"),
		CodeChallengeMethod: values.Get("code_challenge_method"),
		Scope:               values.Get("scope"),
		State:               values.Get("state"),
	}, nil
}

// AuthorizePreview supplies consent text to an already-authenticated browser
// frontend; it never issues a code as a side effect of GET.
func (h *Handler) AuthorizePreview(c *gin.Context) {
	request, err := authorizationRequest(c.Request.URL.Query())
	if err != nil {
		serviceFailure(c, err)
		return
	}
	client, version, err := h.service.ReviewConsent(c.Request.Context(), middleware.AccountID(c), request)
	if err != nil {
		serviceFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"client_id": client.ID.String(), "name": client.Name,
		"redirect_uri": client.RedirectUri, "scope": request.Scope,
		"consent_required": true, "consent_version": version,
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
		c.Redirect(http.StatusFound, target.String())
		return
	}
	redirect, err := h.service.Authorize(c.Request.Context(), middleware.AccountID(c), request)
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
	c.JSON(http.StatusOK, result)
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
	c.JSON(http.StatusOK, gin.H{"sub": account.ID})
}

// Do not disguise a failed PostgreSQL lookup as a rejected OAuth credential.
func userinfoStoreUnavailable(c *gin.Context, err error) {
	log.Ctx(c.Request.Context()).Error().
		Str("dependency", "postgres").
		Str("error_type", fmt.Sprintf("%T", err)).
		Msg("oauth.userinfo_store_unavailable")
	failure(c, http.StatusServiceUnavailable, "temporarily_unavailable")
}
