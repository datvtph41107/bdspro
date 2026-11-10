// Package types defines HTTP-handler capabilities without depending on Gin.
package types

import "net/http"

// PreMiddlewareProvider runs before ordinary middleware within the same
// matched-route compilation scope (AfterRouting, Group, and Methods).
// It is not a BeforeRouting middleware: the route is already selected.
type PreMiddlewareProvider func(next http.Handler) http.Handler
