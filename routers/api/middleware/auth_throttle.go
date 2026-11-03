package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/ratelimit"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// AuthThrottle limits repeated authentication requests by IP and credential.
// Controllers invoke it after parsing the relevant email, phone or token.
type AuthThrottle struct {
	client *redis.Client
}

func NewAuthThrottle(client *redis.Client) *AuthThrottle {
	return &AuthThrottle{client: client}
}

func (l *AuthThrottle) Allow(ctx *gin.Context, operation, credential string, perIP, perCredential int64, window time.Duration) bool {
	if l == nil || l.client == nil {
		response.RateLimitUnavailable(ctx)
		return false
	}
	namespace := authThrottleKeyPrefix + operation + ":"
	ipKey := namespace + "ip:" + ratelimit.Hash(ctx.ClientIP())
	credentialKey := namespace + "identity:" + ratelimit.Hash(strings.ToLower(strings.TrimSpace(credential)))
	allowed, err := ratelimit.AllowPair(ctx.Request.Context(), l.client,
		ipKey, credentialKey, int64(window.Seconds()), perIP, perCredential)
	if err != nil {
		log.Ctx(ctx.Request.Context()).Error().
			Str("operation", operation).
			Str("dependency", "redis").
			Str("error_type", fmt.Sprintf("%T", err)).
			Msg("auth.rate_limit_unavailable")
		response.RateLimitUnavailable(ctx)
		return false
	}
	if allowed != 1 {
		response.RateLimited(ctx, int64(window.Seconds()))
		return false
	}
	return true
}
