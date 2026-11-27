package context

import (
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/log"
)

// Admit translates one limiter decision at the HTTP boundary.
// Neither the destination nor its digest belongs in operational logs.
func (c *APIContext) Admit(action ipactive.Action, scope string, event ipactive.Event, err error) bool {
	if err == nil && event.Allowed {
		return true
	}
	logger := log.For(c, "security")
	entry := logger.Warn().Str("action", string(action)).Str("scope", scope)
	if err != nil {
		entry.Str("reason", "limiter_unavailable").Msg("admission.denied")
		c.Data[ErrorReasonKey] = "limiter_unavailable"
		c.APIError(http.StatusServiceUnavailable, "rate_limit_unavailable", "request admission unavailable")
		return false
	}
	entry.Int64("attempts", event.Attempts).Int64("limit", event.Limit).
		Int64("retry_after", event.RetryAfter).Msg("admission.denied")
	c.Data[ErrorReasonKey] = "rate_limit:" + scope + ":" + string(action)
	if event.RetryAfter > 0 {
		c.Header("Retry-After", strconv.FormatInt(event.RetryAfter, 10))
	}
	c.APIError(http.StatusTooManyRequests, "rate_limited", "too many requests")
	return false
}
