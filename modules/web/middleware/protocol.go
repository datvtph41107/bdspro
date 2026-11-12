// Package middleware owns framework-independent HTTP capabilities.
package middleware

import "net/http"

// ProtocolMiddlewares run before Gin route matching.
func ProtocolMiddlewares() []any {
	return []any{SecurityHeaders}
}

// SecurityHeaders applies to successful responses, errors and unmatched routes.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, req)
	})
}
