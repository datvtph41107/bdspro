// Package context owns request-scoped API state and HTTP output.
package context

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/gin-gonic/gin"
)

// Base is the transition HTTP boundary. Its Gin embedding is retained for
// legacy consumers; typed API actions should use Base and APIContext methods.
type Base struct {
	*gin.Context
	Data map[string]any
}

type APIContext struct {
	*Base
	// Doer is the verified first-party session principal; conversion to an
	// Account model is a separate authentication migration.
	Doer *authcrypto.Payload
	IsSigned bool
	Org *APIOrganization
}

type APIHandler func(*APIContext)

const apiContextKey = "bdspro.api.context"

// GetAPIContext returns the single APIContext owned by the current Gin request.
func GetAPIContext(c *gin.Context) *APIContext {
	if value, ok := c.Get(apiContextKey); ok {
		if api, ok := value.(*APIContext); ok && api != nil {
			return api
		}
		panic("context: APIContext type mismatch")
	}
	api := &APIContext{
		Base: &Base{Context: c, Data: make(map[string]any)},
		Org: &APIOrganization{},
	}
	c.Set(apiContextKey, api)
	return api
}

// APIContexter prepares request-local state once at HTTP ingress.
func APIContexter() gin.HandlerFunc {
	return func(c *gin.Context) {
		GetAPIContext(c)
		c.Next()
	}
}

func (c *APIContext) GetData() map[string]any {
	return c.Data
}

// Forward context.Context to the current HTTP request, including its
// transaction and verified-actor values installed after APIContexter runs.
func (c *APIContext) Deadline() (time.Time, bool) {
	return c.Request.Context().Deadline()
}

func (c *APIContext) Done() <-chan struct{} {
	return c.Request.Context().Done()
}

func (c *APIContext) Err() error {
	return c.Request.Context().Err()
}

func (c *APIContext) Value(key any) any {
	return c.Request.Context().Value(key)
}

func Wrap(handler APIHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(GetAPIContext(c))
	}
}

func (c *APIContext) APIErrorAuto(err error) {
	Error(c.Context, err)
}

func (c *APIContext) APIInvalid(err error) {
	Invalid(c.Context, err)
}

func (c *APIContext) APIError(status int, code, message string) {
	Reject(c.Context, status, code, message)
}

func (c *APIContext) APIErrorNotFound() {
	c.APIError(http.StatusNotFound, "not_found", "not found")
}

func (c *APIContext) APIErrorInternal(err error) {
	if err == nil {
		err = errors.New("internal API failure")
	}
	log.Ctx(c.Request.Context()).Error().
		Str("error_type", fmt.Sprintf("%T", err)).
		Msg("http.api_internal_error")
	InternalError(c.Context)
}
