package middleware

import (
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/ratelimit"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// IPActiveSource is the source admission check that runs before controller binding.
func IPActiveSource(redisClient *redis.Client, operation string, maxAttempts int64, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if redisClient == nil {
			response.RateLimitUnavailable(ctx)
			return
		}
		key := sourceLimitKeyPrefix + operation + ":" + ratelimit.Hash(ctx.ClientIP())
		count, err := ratelimit.Increment(ctx.Request.Context(), redisClient, key, int64(window.Seconds()))
		if err != nil {
			log.Ctx(ctx.Request.Context()).Error().
				Str("operation", operation).
				Str("dependency", "redis").
				Str("error_type", fmt.Sprintf("%T", err)).
				Msg("auth.rate_limit_unavailable")
			response.RateLimitUnavailable(ctx)
			return
		}
		if count > maxAttempts {
			response.RateLimited(ctx, int64(window.Seconds()))
			return
		}
		ctx.Next()
	}
}
