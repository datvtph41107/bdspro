package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

var ipActiveSourceScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return n
`)

// IPActiveSource is the source admission check that runs before controller binding.
func IPActiveSource(redisClient *redis.Client, operation string, maxAttempts int64, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if redisClient == nil {
			ctx.Next()
			return
		}
		key := sourceLimitKeyPrefix + operation + ":" + hashedRateKey(ctx.ClientIP())
		count, err := ipActiveSourceScript.Run(
			ctx.Request.Context(), redisClient,
			[]string{key}, int64(window.Seconds()),
		).Int64()
		if err != nil {
			log.Ctx(ctx.Request.Context()).Error().Str("operation", operation).
				Msg("auth.rate_limit_unavailable")
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication temporarily unavailable"})
			return
		}
		if count > maxAttempts {
			ctx.Header("Retry-After", fmt.Sprint(int64(window.Seconds())))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		ctx.Next()
	}
}
