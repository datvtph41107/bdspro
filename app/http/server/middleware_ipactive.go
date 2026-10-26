package server

import (
    "strconv"
    "fmt"
    "net/http"

    "github.com/go-redis/redis/v8"
    "github.com/rs/zerolog/log"
    "time"

    "github.com/gin-gonic/gin"
)

// IPActive checks action-specific admission before entering a handler.
// It is not an authentication or authorization decision.
func (server *Server) IPActive(operation string, perIP, perIdentity int64, window time.Duration, identity func(*gin.Context) string) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        if !server.allowAuthRequest(ctx, operation, identity(ctx), perIP, perIdentity, window) {
            ctx.Abort()
            return
        }
        ctx.Next()
    }
}

func loginIdentity(ctx *gin.Context) string { return loginInput(ctx).Email }
func registerIdentity(ctx *gin.Context) string { return registerInput(ctx).Email }
func tokenIdentity(ctx *gin.Context) string { return tokenInput(ctx).RefreshToken }
func emailVerificationIdentity(ctx *gin.Context) string {
    return strconv.FormatInt(verifyEmailInput(ctx).EmailID, 10)
}

var ipActiveSourceScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return n
`)

// IPActiveSource is the earliest route guard. It counts every attempt from the
// direct peer before parsing credentials, including malformed JSON.
func (server *Server) IPActiveSource(operation string, maxAttempts int64, window time.Duration) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        if server.authRateRedis == nil {
            ctx.Next()
            return
        }
        key := "bdspro:source_limit:v1:" + operation + ":" + hashedRateKey(ctx.ClientIP())
        count, err := ipActiveSourceScript.Run(ctx.Request.Context(), server.authRateRedis,
            []string{key}, int64(window.Seconds())).Int64()
        if err != nil {
            log.Ctx(ctx.Request.Context()).Error().Str("operation", operation).
                Msg("auth.rate_limit_unavailable")
            ctx.AbortWithStatusJSON(http.StatusServiceUnavailable,
                gin.H{"error":"authentication temporarily unavailable"})
            return
        }
        if count > maxAttempts {
            ctx.Header("Retry-After", fmt.Sprint(int64(window.Seconds())))
            ctx.AbortWithStatusJSON(http.StatusTooManyRequests,
                gin.H{"error":"too many requests"})
            return
        }
        ctx.Next()
    }
}
