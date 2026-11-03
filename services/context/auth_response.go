package context

import (
	"fmt"
	"net/http"
)

func (ctx *APIContext) BearerRequired() {
	ctx.Resp.Header().Set("WWW-Authenticate", "Bearer")
	ctx.APIError(http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
}
func (ctx *APIContext) InvalidAccessToken() {
	ctx.Resp.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	ctx.APIError(http.StatusUnauthorized, "invalid_access_token", "invalid access token")
}
func (ctx *APIContext) AuthenticationRequired() {
	ctx.Resp.Header().Set("WWW-Authenticate", "Bearer")
	ctx.APIError(http.StatusUnauthorized, "unauthenticated", "authentication required")
}
func (ctx *APIContext) RateLimitUnavailable() {
	ctx.APIError(http.StatusServiceUnavailable, "rate_limit_unavailable", "authentication temporarily unavailable")
}
func (ctx *APIContext) RateLimited(seconds int64) {
	ctx.Resp.Header().Set("Retry-After", fmt.Sprint(seconds))
	ctx.APIError(http.StatusTooManyRequests, "rate_limited", "too many requests")
}
func (ctx *APIContext) IdentityStoreUnavailable() {
	ctx.APIError(http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
}
