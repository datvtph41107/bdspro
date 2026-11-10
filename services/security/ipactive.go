package security

import (
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/log"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// IPActiveFor is the typed API boundary for source admission. The mechanism
// lives in modules/ipactive; this HTTP policy maps denial to the public API
// response without importing routers or Gin into the limiter.
func IPActiveFor(m *ipactive.Middleware, action ipactive.Action) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		event, err := m.Check(ctx, action, ctx.ClientIP())
		if err != nil {
			log.Ctx(ctx).Error().
				Str("action", string(action)).
				Str("error_type", fmt.Sprintf("%T", err)).
				Msg("ipactive.admission_unavailable")
			ctx.APIError(http.StatusServiceUnavailable, "rate_limit_unavailable", "authentication temporarily unavailable")
			return
		}
		if !event.Allowed {
			log.Ctx(ctx).Warn().
				Str("action", string(event.Action)).
				Int64("occurred_at_unix", int64(event.OccurredAt)).
				Int64("attempts", event.Attempts).
				Msg("ipactive.rate_limited")
			ctx.Header("Retry-After", fmt.Sprint(event.RetryAfter))
			ctx.APIError(http.StatusTooManyRequests, "rate_limited", "too many requests")
		}
	}
}
