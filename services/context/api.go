// Package context owns the HTTP API request and response contract.
package context

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/web"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type apiContextRequestKey struct{}

func GetAPIContextFromRequest(req *http.Request) *APIContext {
	api, ok := req.Context().Value(apiContextRequestKey{}).(*APIContext)
	if !ok || api == nil {
		panic("context: APIContext missing from HTTP request")
	}
	return api
}

// Register the existing request-scoped APIContext as a typed web handler argument.
func init() {
	web.RegisterResponseStatusProvider[*APIContext](func(req *http.Request) web_types.ResponseStatusProvider {
		return GetAPIContextFromRequest(req)
	})
}

type APIContext struct {
	*Base
}

var _ context.Context = (*APIContext)(nil)
var _ web_types.ResponseStatusProvider = (*APIContext)(nil)

const ErrorCodeKey = "public_error_code"

type APIErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func (c *APIContext) APIError(status int, code, message string) {
	c.writeError(status, code, APIErrorResponse{Error: message, Code: code})
}

func (c *APIContext) APIErrorAuto(err error) {
	if status, code, message, ok := PublicError(err); ok {
		c.APIError(status, code, message)
		return
	}
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
