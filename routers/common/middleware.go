// Package common defines protocol-level HTTP middleware shared by routers.
package common

import "net/http"

// ProtocolMiddlewares run before Gin route matching. They cannot require
// a matched route, path parameters, or APIContext.
func ProtocolMiddlewares() []any {
	return []any{SecurityHeaders}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, req)
	})
}
