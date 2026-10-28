package middleware

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

const securityDetectionThreshold int64 = 12
const securityDetectionWindow = 10 * time.Minute

var securityCountScript = redis.NewScript(`
local v = redis.call('INCR', KEYS[1])
if v == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return v
`)

// RequestMonitor records policy denials and one threshold alert per fixed
// observation window. It does not automatically ban a client.
func RequestMonitor(redisClient *redis.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
		status := ctx.Writer.Status()
		code := ctx.GetString(response.ErrorCodeKey)
		var reason string
		if code == "auth.otp_invalid" || code == "auth.otp_cooldown" {
			reason = "otp_verification"
		}
		if code == "rate_limited" {
			reason = "rate_limit"
		}
		if reason == "" {
			switch status {
			case http.StatusBadRequest:
				reason = "invalid_request"
			case http.StatusUnauthorized:
				reason = "authentication"
			case http.StatusForbidden:
				reason = "authorization"
			case http.StatusTooManyRequests:
				reason = "rate_limit"
			case http.StatusRequestEntityTooLarge:
				reason = "request_size"
			default:
				return
			}
		}
		route := ctx.FullPath()
		if route == "" {
			route = "<unmatched>"
		}
		ipHash := hashedRateKey(ctx.ClientIP())
		log.Ctx(ctx.Request.Context()).Warn().
			Str("reason", reason).
			Str("route", route).
			Str("method", ctx.Request.Method).
			Str("source_id", ipHash[:16]).
			Int("status", status).
			Msg("security.request_denied")

		if redisClient == nil {
			return
		}
		key := securityDeniedKeyPrefix + ipHash
		count, err := securityCountScript.Run(ctx.Request.Context(), redisClient,
			[]string{key}, int64(securityDetectionWindow.Seconds())).Int64()
		if err != nil {
			log.Ctx(ctx.Request.Context()).Warn().
				Str("component", "request_monitor").
				Msg("security.monitor_unavailable")
			return
		}
		if count == securityDetectionThreshold {
			log.Ctx(ctx.Request.Context()).Warn().
				Str("reason", "repeated_denials").
				Str("source_id", ipHash[:16]).
				Int64("count", count).
				Dur("window", securityDetectionWindow).
				Msg("security.suspicious_activity")
		}
	}
}
