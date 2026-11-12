// Package context owns the HTTP API request and response contract.
package context

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/web"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type APIContext struct {
	*Base
}

type apiContextKeyType struct{}

var apiContextKey = apiContextKeyType{}

func init() {
	web.RegisterResponseStatusProvider[*APIContext](func(req *http.Request) web_types.ResponseStatusProvider {
		return GetAPIContext(req)
	})
}

func GetAPIContext(req *http.Request) *APIContext {
	if req == nil {
		panic("context: nil API request")
	}
	return reqctx.MustContextValue[*APIContext](req.Context(), apiContextKey)
}

// APIContexter creates one APIContext after a matched route and passes
// the same Response and Request to every subsequent typed action.
func APIContexter() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			base := NewBaseContext(w, req)
			ctx := &APIContext{Base: base}
			ctx.SetContextValue(apiContextKey, ctx)
			next.ServeHTTP(ctx.Resp, ctx.Req)
		})
	}
}

// APIError preserves BDSPro's public wire format while retaining Gitea's
// APIContext-owned HTTP error handling.
type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func apiErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request"
	case http.StatusRequestEntityTooLarge:
		return "request_too_large"
	case http.StatusUnauthorized:
		return "unauthenticated"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusInternalServerError:
		return "internal_error"
	default:
		return "http_error"
	}
}

func (c *APIContext) apiError(status int, code, message string) {
	if code == "" {
		code = apiErrorCode(status)
	}
	c.JSON(status, APIError{Error: message, Code: code})
}

func (c *APIContext) APIError(status int, message string) {
	c.apiError(status, apiErrorCode(status), message)
}

func (c *APIContext) APIErrorNotFound(messages ...string) {
	message := "not found"
	if len(messages) > 0 {
		message = messages[0]
	}
	c.APIError(http.StatusNotFound, message)
}

func (c *APIContext) APIErrorAuto(err error) {
	category, code, message, ok := util.ErrorForClient(err)
	if ok {
		switch {
		case errors.Is(category, util.ErrInvalidArgument):
			c.apiError(http.StatusBadRequest, code, message)
			return
		case errors.Is(category, util.ErrUnauthenticated):
			c.apiError(http.StatusUnauthorized, code, message)
			return
		case errors.Is(category, util.ErrPermissionDenied):
			c.apiError(http.StatusForbidden, code, message)
			return
		case errors.Is(category, util.ErrNotExist):
			c.apiError(http.StatusNotFound, code, message)
			return
		case errors.Is(category, util.ErrAlreadyExist):
			c.apiError(http.StatusConflict, code, message)
			return
		case errors.Is(category, util.ErrRateLimited):
			c.apiError(http.StatusTooManyRequests, code, message)
			return
		}
	}
	c.APIErrorInternal(err)
}

func (c *APIContext) APIErrorInternal(err error) {
	if err != nil {
		log.Error("API internal error: %v", err)
	}
	c.APIError(http.StatusInternalServerError, "internal server error")
}
