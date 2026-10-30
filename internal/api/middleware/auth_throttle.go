package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// AuthThrottle limits repeated authentication requests by IP and credential.
// Controllers invoke it after parsing the relevant email, phone or token.
type AuthThrottle struct {
	client *redis.Client
}

func NewAuthThrottle(client *redis.Client) *AuthThrottle {
	return &AuthThrottle{client: client}
}

var authThrottleScript = redis.NewScript(`
local a = redis.call('INCR', KEYS[1])
if a == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local b = redis.call('INCR', KEYS[2])
if b == 1 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
if a > tonumber(ARGV[2]) or b > tonumber(ARGV[3]) then return 0 end
return 1
`)

func (l *AuthThrottle) Allow(ctx *gin.Context, operation, credential string, perIP, perCredential int64, window time.Duration) bool {
	if l == nil || l.client == nil {
		response.RateLimitUnavailable(ctx)
		return false
	}
	namespace := authThrottleKeyPrefix + operation + ":"
	ipKey := namespace + "ip:" + rateHash(ctx.ClientIP())
	credentialKey := namespace + "identity:" + rateHash(strings.ToLower(strings.TrimSpace(credential)))
	allowed, err := authThrottleScript.Run(ctx.Request.Context(), l.client,
		[]string{ipKey, credentialKey}, int64(window.Seconds()), perIP, perCredential).Int64()
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

func rateHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
