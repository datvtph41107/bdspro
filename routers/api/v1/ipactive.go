package apiv1

import (
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// reqIPAdmission executes after verified actor/organization owner guards,
// before JSON binding or a state-changing service. A missing Redis admission
// dependency fails closed; no client-supplied forwarding header is trusted.
func reqIPAdmission(action ipactive.Action) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		limiter, ok := ipactive.LookupMiddleware(ctx)
		if !ok {
			ctx.APIError(http.StatusServiceUnavailable, "rate_limit_unavailable", "request admission unavailable")
			return
		}
		event, err := limiter.Check(ctx, action, ctx.ClientIP())
		if err != nil {
			ctx.APIError(http.StatusServiceUnavailable, "rate_limit_unavailable", "request admission unavailable")
			return
		}
		if !event.Allowed {
			if event.RetryAfter > 0 {
				ctx.Header("Retry-After", strconv.FormatInt(event.RetryAfter, 10))
			}
			ctx.APIError(http.StatusTooManyRequests, "rate_limited", "too many requests")
		}
	}
}
