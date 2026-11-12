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

type APIError struct {
	Message string `json:"message"`
}

func (c *APIContext) APIError(status int, message string) {
	c.JSON(status, APIError{Message: message})
}

func (c *APIContext) APIErrorNotFound(messages ...string) {
	message := "not found"
	if len(messages) > 0 {
		message = messages[0]
	}
	c.APIError(http.StatusNotFound, message)
}

func (c *APIContext) APIErrorAuto(err error) {
	category, _, message, ok := util.ErrorForClient(err)
	if ok {
		switch {
		case errors.Is(category, util.ErrInvalidArgument):
			c.APIError(http.StatusBadRequest, message)
			return
		case errors.Is(category, util.ErrUnauthenticated):
			c.APIError(http.StatusUnauthorized, message)
			return
		case errors.Is(category, util.ErrPermissionDenied):
			c.APIError(http.StatusForbidden, message)
			return
		case errors.Is(category, util.ErrNotExist):
			c.APIError(http.StatusNotFound, message)
			return
		case errors.Is(category, util.ErrAlreadyExist):
			c.APIError(http.StatusConflict, message)
			return
		case errors.Is(category, util.ErrRateLimited):
			c.APIError(http.StatusTooManyRequests, message)
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
