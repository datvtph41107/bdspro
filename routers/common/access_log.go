package common

import (
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

// AccessLogger writes safe route-template outcomes without raw identifiers,
// token-bearing query strings or duplicate HTTP diagnostics.
// AccessLogger records one safe request outcome, distinct from a failure
// diagnostic. Route templates avoid token-bearing query strings and raw IDs.
func AccessLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		path := ctx.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		event := log.Ctx(ctx.Request.Context()).Info()
		if code := ctx.GetString(appctx.ErrorCodeKey); code != "" {
			event = event.Str("error_code", code)
		}
		event.Str("method", ctx.Request.Method).
			Str("path", path).
			Int("status", ctx.Writer.Status()).
			Dur("duration", time.Since(startedAt)).
			Msg("http.request")
	}
}
