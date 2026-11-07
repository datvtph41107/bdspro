// Package context owns the Gin-specific API response contract.
package context

import "github.com/gin-gonic/gin"

// APIContext is a request-local view over Gin. It adds no middleware executor
// or global dependency registry; the existing principal policy is unchanged.
type APIContext struct {
	*gin.Context
}

// APIHandler is the terminal endpoint signature.
type APIHandler func(*APIContext)

// Wrap constructs a request-local context, preserving Gin's normal chain.
func Wrap(handler APIHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(&APIContext{Context: c})
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
