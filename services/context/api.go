// Package context owns request-scoped API state and HTTP output.
package context

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/gin-gonic/gin"
)

type APIContext struct {
	*Base
	// Gin remains only at the API adapter boundary for legacy HTTP actions.
	// Base does not depend on Gin and domain services only see context.Context.
	*gin.Context
	// Doer is the signed first-party JWT principal. IsSigned is true only
	// after RequireSession verifies the live PostgreSQL session.
	Doer     *authcrypto.Payload
	IsSigned bool
	Org      *APIOrganization
}

// VerifiedActor is the sole HTTP-to-domain authority boundary. Doer retains
// signed JWT claims for old HTTP actions; only a live session can establish
// an actor permitted to access protected resources.
func (c *APIContext) VerifiedActor() (auth_model.Actor, bool) {
	if c == nil || !c.IsSigned || c.Doer == nil {
		return auth_model.Actor{}, false
	}
	actor, ok := auth_model.ActorFromContext(c)
	if !ok || actor.AccountID != c.Doer.AccountID || actor.SessionID != c.Doer.SessionID {
		return auth_model.Actor{}, false
	}
	return actor, true
}

// Explicit delegation prevents ambiguity with Gin's own context methods.
var _ context.Context = (*APIContext)(nil)

func (c *APIContext) Deadline() (time.Time, bool) { return c.Base.Deadline() }
func (c *APIContext) Done() <-chan struct{}       { return c.Base.Done() }
func (c *APIContext) Err() error                  { return c.Base.Err() }
func (c *APIContext) Value(key any) any           { return c.Base.Value(key) }

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
		Base:    NewBaseContext(ginRequestSource{ctx: c}),
		Context: c,
		Org:     &APIOrganization{},
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
