package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// LoginAudit logs the completed response; it never issues credentials.
func LoginAudit() gin.HandlerFunc {
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
			// The operation boundary owns the ERROR event. This records only the outcome.
		case status < 200 || status >= 300:
			outcome = "rejected"
			event = log.Ctx(ctx.Request.Context()).Warn()
		}
		event.Str("outcome", outcome).Int("status", status).Msg("auth.login_completed")
	}
}
