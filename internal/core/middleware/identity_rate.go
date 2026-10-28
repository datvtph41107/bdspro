package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// IdentityLimiter is called explicitly after a controller parses its input.
// This avoids pre-binding middleware and hidden Gin context request keys.
type IdentityLimiter struct {
	client *redis.Client
}

func NewIdentityLimiter(client *redis.Client) *IdentityLimiter {
	return &IdentityLimiter{client: client}
}

var identityRateScript = redis.NewScript(`
local a = redis.call('INCR', KEYS[1])
if a == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local b = redis.call('INCR', KEYS[2])
if b == 1 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
if a > tonumber(ARGV[2]) or b > tonumber(ARGV[3]) then return 0 end
return 1
`)

func (l *IdentityLimiter) Allow(ctx *gin.Context, operation, identity string, perIP, perIdentity int64, window time.Duration) bool {
	namespace := identityLimitKeyPrefix + operation + ":"
	ipKey := namespace + "ip:" + rateHash(ctx.ClientIP())
	identityKey := namespace + "identity:" + rateHash(strings.ToLower(strings.TrimSpace(identity)))
	allowed, err := identityRateScript.Run(ctx.Request.Context(), l.client,
		[]string{ipKey, identityKey}, int64(window.Seconds()), perIP, perIdentity).Int64()
	if err != nil {
		log.Ctx(ctx.Request.Context()).Error().Str("operation", operation).
			Msg("auth.rate_limit_unavailable")
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "authentication temporarily unavailable"})
		return false
	}
	if allowed != 1 {
		ctx.Header("Retry-After", fmt.Sprint(int64(window.Seconds())))
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		return false
	}
	return true
}

func rateHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
