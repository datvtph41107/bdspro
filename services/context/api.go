package context

import (
	"context"
	"errors"
	"net/http"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type apiContextRequestKey struct{}

func GetAPIContextFromRequest(req *http.Request) *APIContext {
	api, ok := req.Context().Value(apiContextRequestKey{}).(*APIContext)
	if !ok || api == nil {
		panic("context: APIContext missing from HTTP request")
	}

	if source, ok := api.Base.source.(*httpRequestSource); ok {
		source.request = req
	}
	return api
}

func init() {
	web.RegisterResponseStatusProvider[*APIContext](func(req *http.Request) web_types.ResponseStatusProvider {
		return GetAPIContextFromRequest(req)
	})
}

type APIContext struct {
	*Base

	Doer     *auth_model.Actor
	IsSigned bool
	Org      *APIOrganization
}

var _ context.Context = (*APIContext)(nil)
var _ web_types.ResponseStatusProvider = (*APIContext)(nil)

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
	if status, code, message, ok := PublicError(err); ok {
		c.APIError(status, code, message)
		return
	}
	logger := log.Ctx(c)
	log.ErrorDetails(logger.Error(), err, logger.GetLevel() <= log.DEBUG).Msg("http.application_error")
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

func (c *APIContext) APIErrorNotFound(msg ...string) {
	message := "not found"
	if len(msg) > 0 {
		message = msg[0]
	}
	c.APIError(http.StatusNotFound, "not_found", message)
}

func (c *APIContext) APIErrorInternal(err error) {
	if err == nil {
		err = errors.New("internal API failure")
	}
	logger := log.Ctx(c)
	log.ErrorDetails(logger.Error(), err, logger.GetLevel() <= log.DEBUG).Msg("http.api_internal_error")
	c.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
}

func (c *APIContext) AuthenticationRequired() {
	c.Header("WWW-Authenticate", "Bearer")
	c.APIError(http.StatusUnauthorized, "unauthenticated", "authentication required")
}
