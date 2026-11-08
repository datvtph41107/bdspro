// Package context owns the Gin-specific API response contract.
package context

import (
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/gin-gonic/gin"
)

// APIContext is a request-local view over Gin. It adds no middleware executor
// or global dependency registry; the existing principal policy is unchanged.
type APIContext struct {
	*gin.Context

	// Doer is the signed JWT principal, not a live account record.
	// Protected routes resolve session authority before invoking the handler;
	// account status and domain permissions are independent decisions.
	Doer *authcrypto.Payload
}

// APIHandler is the terminal endpoint signature.
type APIHandler func(*APIContext)

// Wrap constructs a request-local context, preserving Gin's normal chain.
func Wrap(handler APIHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		doer, _ := AuthorizationPayload(c)
		handler(&APIContext{Context: c, Doer: doer})
	}
}

// APIErrorAuto maps only explicitly public domain errors to HTTP.
func (c *APIContext) APIErrorAuto(err error) {
	Error(c.Context, err)
}

// APIInvalid preserves binding error status codes and response fields.
func (c *APIContext) APIInvalid(err error) {
	Invalid(c.Context, err)
}
