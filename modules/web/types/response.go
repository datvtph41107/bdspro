// Package types holds interfaces shared by router machinery and typed contexts.
// It must not import routers, services or models.
package types

import "net/http"

// ResponseStatusProvider reports whether a handler has already produced a response.
// Zero means the response has not been written.
type ResponseStatusProvider interface {
	WrittenStatus() int
}

// PreMiddlewareProvider wraps an HTTP handler before normal route middleware.
// Prefer it only when early processing is required by the HTTP contract.
type PreMiddlewareProvider func(next http.Handler) http.Handler

