package response

import (
	"errors"
	"fmt"
	"net/http"

	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const ErrorCodeKey = "public_error_code"

// reject emits one public, stable error contract; underlying errors are never serialized.
func Reject(c *gin.Context, status int, code, message string) {
	c.Set(ErrorCodeKey, code)
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

func Invalid(c *gin.Context, err error) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		Reject(c, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body too large")
		return
	}
	Reject(c, http.StatusBadRequest, "invalid_request", "invalid request")
}

func Error(c *gin.Context, err error) {
	if public, ok := classifyApplicationError(err); ok {
		Reject(c, public.Status, public.Code, public.Message)
		return
	}
	// Do not serialize or log arbitrary err.Error(): driver and provider errors
	// may contain credentials, tokens, URLs or other sensitive values.
	event := log.Ctx(c.Request.Context()).Error().
		Str("error_type", fmt.Sprintf("%T", err))
	if pgErr := apperrors.Postgres(err); pgErr != nil {
		event = event.Str("postgres_code", pgErr.Code)
	}
	event.Msg("http.application_error")
	InternalError(c)
}


// BearerRequired responds to requests without a valid Bearer credential.
func BearerRequired(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	Reject(c, http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
}

// InvalidAccessToken rejects an invalid access JWT using the Bearer challenge.
func InvalidAccessToken(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
	Reject(c, http.StatusUnauthorized, "invalid_access_token", "invalid access token")
}

// AuthenticationRequired responds when the verified principal is no longer available.
func AuthenticationRequired(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	Reject(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
}

// RateLimitUnavailable fails closed when admission cannot be enforced.
func RateLimitUnavailable(c *gin.Context) {
	Reject(c, http.StatusServiceUnavailable, "rate_limit_unavailable", "authentication temporarily unavailable")
}

// RateLimited emits a 429 with the policy's Retry-After window in seconds.
func RateLimited(c *gin.Context, retryAfterSeconds int64) {
	c.Header("Retry-After", fmt.Sprint(retryAfterSeconds))
	Reject(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
}

// InternalError is the safe HTTP response to a failure already diagnosed
// at the operation or recovery boundary.
func InternalError(c *gin.Context) {
	Reject(c, http.StatusInternalServerError, "internal_error", "internal server error")
}
