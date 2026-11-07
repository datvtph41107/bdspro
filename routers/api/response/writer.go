// Package response provides temporary compatibility shims during API migration.
package response

import (
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

const ErrorCodeKey = appctx.ErrorCodeKey

func Reject(c *gin.Context, status int, code, message string) {
	appctx.Reject(c, status, code, message)
}

func Invalid(c *gin.Context, err error) {
	appctx.Invalid(c, err)
}

func Error(c *gin.Context, err error) {
	appctx.Error(c, err)
}

func BearerRequired(c *gin.Context) {
	appctx.BearerRequired(c)
}

func InvalidAccessToken(c *gin.Context) {
	appctx.InvalidAccessToken(c)
}

func AuthenticationRequired(c *gin.Context) {
	appctx.AuthenticationRequired(c)
}

func RateLimitUnavailable(c *gin.Context) {
	appctx.RateLimitUnavailable(c)
}

func RateLimited(c *gin.Context, retryAfterSeconds int64) {
	appctx.RateLimited(c, retryAfterSeconds)
}

func InternalError(c *gin.Context) {
	appctx.InternalError(c)
}
