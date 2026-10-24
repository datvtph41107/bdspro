package api

import (
    "net/http"
    "time"

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
func (server *Server) RequestMonitor() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        ctx.Next()
        status := ctx.Writer.Status()
        var reason string
        switch status {
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
        route := ctx.FullPath()
        if route == "" { route = "<unmatched>" }
        ipHash := hashedRateKey(ctx.ClientIP())
        log.Ctx(ctx.Request.Context()).Warn().
            Str("reason", reason).
            Str("route", route).
            Str("method", ctx.Request.Method).
            Str("source_id", ipHash[:16]).
            Int("status", status).
            Msg("security.request_denied")

        if server.authRateRedis == nil { return }
        key := "bdspro:security:denied:v1:" + ipHash
        count, err := securityCountScript.Run(ctx.Request.Context(), server.authRateRedis,
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
