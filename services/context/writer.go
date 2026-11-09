package context

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
)

const ErrorCodeKey = "public_error_code"

// APIError is the sole public error envelope for API actions. The existing
// Gin helpers in gin_response.go also call this implementation.
func (c *APIContext) APIError(status int, code, message string) {
	c.writeError(status, code, map[string]any{
		"error": message,
		"code":  code,
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
