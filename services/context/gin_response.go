package context

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Legacy Gin entrypoints delegate to APIContext's one response/error system.
// They remain for middleware and public Auth/OAuth endpoints that still
// operate on *gin.Context; no alternate writer or error mapping exists.
func Reject(c *gin.Context, status int, code, message string) {
	GetAPIContext(c).APIError(status, code, message)
}

func Invalid(c *gin.Context, err error) { GetAPIContext(c).APIInvalid(err) }
func Error(c *gin.Context, err error)   { GetAPIContext(c).APIErrorAuto(err) }
func InternalError(c *gin.Context) {
	GetAPIContext(c).APIError(http.StatusInternalServerError, "internal_error", "internal server error")
}

func BearerRequired(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	Reject(c, http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
}

func InvalidAccessToken(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
	Reject(c, http.StatusUnauthorized, "invalid_access_token", "invalid access token")
}

func AuthenticationRequired(c *gin.Context) { GetAPIContext(c).AuthenticationRequired() }

func RateLimitUnavailable(c *gin.Context) {
	Reject(c, http.StatusServiceUnavailable, "rate_limit_unavailable", "authentication temporarily unavailable")
}

func RateLimited(c *gin.Context, retryAfterSeconds int64) {
	c.Header("Retry-After", fmt.Sprint(retryAfterSeconds))
	Reject(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
}
