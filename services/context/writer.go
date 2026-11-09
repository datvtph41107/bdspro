package context

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
)

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
