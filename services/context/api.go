// Package context owns the HTTP request context and response contract.
package context

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/web"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type apiContextRequestKey struct{}

// GetAPIContextFromRequest is the canonical net/http context accessor used by
// modules/web's typed response-status provider. Gin remains the route matcher;
// the request context is the only source of this typed capability.
func GetAPIContextFromRequest(req *http.Request) *APIContext {
	api, ok := req.Context().Value(apiContextRequestKey{}).(*APIContext)
	if !ok || api == nil {
		panic("context: APIContext missing from HTTP request")
	}
	return api
}

func init() {
	web.RegisterResponseStatusProvider[*APIContext](func(req *http.Request) web_types.ResponseStatusProvider {
		return GetAPIContextFromRequest(req)
	})
}

// APIContext is the API-specific request state. It does not embed or import
// Gin. The only Gin boundary is gin.go, which builds and reuses this context.
type APIContext struct {
	*Base

	// Doer is signed JWT claims, not an Account model and not write authority.
	// IsSigned becomes true only after PostgreSQL session verification.
	Doer     *authcrypto.Payload
	IsSigned bool
	Org      *APIOrganization
}

// SignedPrincipal returns signed JWT claims for compatibility-only read paths.
// Session authorization remains exclusively available through VerifiedActor.
func (c *APIContext) SignedPrincipal() (*authcrypto.Payload, bool) {
	if c == nil || c.Doer == nil || c.Doer.AccountID <= 0 {
		return nil, false
	}
	return c.Doer, true
}

var _ context.Context = (*APIContext)(nil)
var _ web_types.ResponseStatusProvider = (*APIContext)(nil)

// VerifiedActor is the sole HTTP-to-domain verified principal accessor.
func (c *APIContext) VerifiedActor() (auth_model.Actor, bool) {
	if c == nil || c.Base == nil || !c.IsSigned || c.Doer == nil {
		return auth_model.Actor{}, false
	}
	actor, ok := auth_model.ActorFromContext(c)
	if !ok || actor.AccountID != c.Doer.AccountID || actor.SessionID != c.Doer.SessionID {
		return auth_model.Actor{}, false
	}
	return actor, true
}

const ErrorCodeKey = "public_error_code"

// APIErrorResponse is the canonical public API error envelope shared by typed
// actions and the Gin protocol compatibility entrypoints.
type APIErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// APIError selects the public contract. Base.writeError owns HTTP emission,
// monitoring metadata and termination of the remaining Gin action chain.
func (c *APIContext) APIError(status int, code, message string) {
	c.writeError(status, code, APIErrorResponse{
		Error: message,
		Code:  code,
	})
}

func (c *APIContext) APIErrorAuto(err error) {
	if status, code, message, ok := PublicError(err); ok {
		c.APIError(status, code, message)
		return
	}
	// Never serialize or log raw driver/provider errors: they can contain
	// credentials, SQL parameters and secret callback URLs.
	log.Ctx(c).Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("http.application_error")
	c.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
}

func (c *APIContext) APIInvalid(err error) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		c.APIError(http.StatusRequestEntityTooLarge, "request_body_too_large", "request body too large")
		return
	}
	c.APIError(http.StatusBadRequest, "invalid_request", "invalid request")
}

func (c *APIContext) APIErrorNotFound() {
	c.APIError(http.StatusNotFound, "not_found", "not found")
}

func (c *APIContext) APIErrorInternal(err error) {
	if err == nil {
		err = errors.New("internal API failure")
	}
	log.Ctx(c).Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("http.api_internal_error")
	c.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
}

func (c *APIContext) AuthenticationRequired() {
	c.Header("WWW-Authenticate", "Bearer")
	c.APIError(http.StatusUnauthorized, "unauthenticated", "authentication required")
}
