package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// SufLogin records the final result of the login operation.
// No credentials are verified and no sessions are issued here.
func SufLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
		status := ctx.Writer.Status()
		event := log.Ctx(ctx.Request.Context()).Info()
		outcome := "succeeded"
		switch {
		case status == http.StatusUnauthorized:
			outcome = "rejected"
			event = log.Ctx(ctx.Request.Context()).Warn()
		case status == http.StatusTooManyRequests:
			outcome = "throttled"
			event = log.Ctx(ctx.Request.Context()).Warn()
		case status >= 500:
			outcome = "failed"
			event = log.Ctx(ctx.Request.Context()).Error()
		case status < 200 || status >= 300:
			outcome = "rejected"
			event = log.Ctx(ctx.Request.Context()).Warn()
		}
		event.Str("outcome", outcome).Int("status", status).Msg("auth.login_completed")
	}
}
