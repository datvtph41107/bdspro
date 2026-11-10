package security

import (
	"context"
	"errors"
	"net/http"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/timeutil"
	appctx "github.com/datvtph41107/bdspro/services/context"
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

var errThrottleUnavailable = errors.New("auth: throttle unavailable")

var authThrottleScript = redis.NewScript(`
local a = redis.call('INCR', KEYS[1])
if a == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local b = redis.call('INCR', KEYS[2])
if b == 1 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
if a > tonumber(ARGV[2]) or b > tonumber(ARGV[3]) then return 0 end
return 1
`)

// Allow uses the same Redis admission algorithm for legacy Gin actions.
// Transport-specific response writing is confined to this boundary.
func (l *AuthThrottle) Allow(ctx *gin.Context, operation, credential string, perIP, perCredential int64, window time.Duration) bool {
	allowed, err := l.check(ctx.Request.Context(), ctx.ClientIP(), operation, credential, perIP, perCredential, window)
	if err != nil {
		appctx.RateLimitUnavailable(ctx)
		return false
	}
	if !allowed {
		appctx.RateLimited(ctx, timeutil.SecondsCeil(window))
	}
	return allowed
}

// AllowAPI is the typed API action admission boundary. It consumes the
// trusted ClientIP from APIContext and uses the exact same Redis keys, limits,
// and fail-closed policy as existing Gin endpoints.
func (l *AuthThrottle) AllowAPI(ctx *appctx.APIContext, operation, credential string, perIP, perCredential int64, window time.Duration) bool {
	allowed, err := l.check(ctx, ctx.ClientIP(), operation, credential, perIP, perCredential, window)
	if err != nil {
		ctx.APIError(http.StatusServiceUnavailable, "rate_limit_unavailable", "authentication temporarily unavailable")
		return false
	}
	if !allowed {
		ctx.Header("Retry-After", fmt.Sprint(timeutil.SecondsCeil(window)))
		ctx.APIError(http.StatusTooManyRequests, "rate_limited", "too many requests")
	}
	return allowed
}

// check is the single security mechanism. Its inputs are request context and
// a client IP already validated by the transport, never a forwarded header.
func (l *AuthThrottle) check(ctx context.Context, clientIP, operation, credential string, perIP, perCredential int64, window time.Duration) (bool, error) {
	if l == nil || l.client == nil || perIP <= 0 || perCredential <= 0 || timeutil.SecondsCeil(window) == 0 {
		return false, errThrottleUnavailable
	}
	namespace := authThrottleKeyPrefix + operation + ":"
	ipKey := namespace + "ip:" + rateHash(clientIP)
	credentialKey := namespace + "identity:" + rateHash(strings.ToLower(strings.TrimSpace(credential)))
	allowed, err := authThrottleScript.Run(ctx, l.client,
		[]string{ipKey, credentialKey}, timeutil.SecondsCeil(window), perIP, perCredential).Int64()
	if err != nil {
		log.Ctx(ctx).Error().
			Str("operation", operation).
			Str("dependency", "redis").
			Str("error_type", fmt.Sprintf("%T", err)).
			Msg("auth.rate_limit_unavailable")
		return false, err
	}
	return allowed == 1, nil
}


func rateHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
