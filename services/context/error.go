package context

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
)

const (
	ErrorCodeKey    = "public_error_code"
	ErrorMessageKey = "public_error_message"
	ErrorReasonKey  = "error_reason"
)

type APIErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
	URL   string `json:"url,omitempty"`
}

func (c *APIContext) APIError(status int, code, message string) {
	if !c.Written() {
		c.Data[ErrorMessageKey] = message
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			logger := log.For(c, "security")
			event := logger.Warn().Str("code", code).Int("status", status)
			if reason, ok := c.Data[ErrorReasonKey].(string); ok {
				event.Str("reason", reason)
			}
			event.Msg("auth.denied")
		}
	}
	c.writeError(status, code, APIErrorResponse{Error: message, Code: code, URL: setting.API.SwaggerURL})
}

func (c *APIContext) APIErrorAuto(err error) {
	category, code, message, ok := util.ErrorForClient(err)
	if ok {
		var status int
		switch {
		case errors.Is(category, util.ErrInvalidArgument):
			status = http.StatusBadRequest
		case errors.Is(category, util.ErrUnauthenticated):
			status = http.StatusUnauthorized
		case errors.Is(category, util.ErrPermissionDenied):
			status = http.StatusForbidden
		case errors.Is(category, util.ErrNotExist):
			status = http.StatusNotFound
		case errors.Is(category, util.ErrAlreadyExist):
			status = http.StatusConflict
		case errors.Is(category, util.ErrRateLimited):
			status = http.StatusTooManyRequests
		}
		if status != 0 {
			c.APIError(status, code, message)
			return
		}
	}
	c.apiErrorInternal(err, "http.application_error")
}

func (c *APIContext) APIInvalid(err error) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		c.APIError(http.StatusRequestEntityTooLarge, "request_body_too_large", "request body too large")
		return
	}
	c.APIError(http.StatusBadRequest, "invalid_request", "invalid request")
}

// APIErrorInternal forces a 500 even when err has a public classification.
func (c *APIContext) APIErrorInternal(err error) {
	if err == nil {
		err = errors.New("internal API failure")
	}
	c.apiErrorInternal(err, "http.api_internal_error")
}

// Both internal-error entry points use the same redacted diagnostics and response.
func (c *APIContext) apiErrorInternal(err error, event string) {
	logger := log.Ctx(c)
	log.ErrorDetails(logger.Error(), err, logger.GetLevel() <= log.DEBUG).Msg(event)
	c.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
}

func (c *APIContext) AuthenticationRequired() {
	c.Header("WWW-Authenticate", "Bearer")
	c.APIError(http.StatusUnauthorized, "unauthenticated", "authentication required")
}
