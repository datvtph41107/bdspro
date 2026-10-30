package middleware

import (
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/response"
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
			response.Fail(ctx, response.RateLimitUnavailable)
			return
		}
		key := sourceLimitKeyPrefix + operation + ":" + hashedRateKey(ctx.ClientIP())
		count, err := ipActiveSourceScript.Run(
			ctx.Request.Context(), redisClient,
			[]string{key}, int64(window.Seconds()),
		).Int64()
		if err != nil {
			log.Ctx(ctx.Request.Context()).Error().
				Str("operation", operation).
				Str("dependency", "redis").
				Str("error_type", fmt.Sprintf("%T", err)).
				Msg("auth.rate_limit_unavailable")
			response.Fail(ctx, response.RateLimitUnavailable)
			return
		}
		if count > maxAttempts {
			ctx.Header("Retry-After", fmt.Sprint(int64(window.Seconds())))
			response.Fail(ctx, response.RateLimited)
			return
		}
		ctx.Next()
	}
}
