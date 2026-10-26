package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

const maxRequestBodyBytes int64 = 1 << 20 // 1 MiB: current APIs only accept small JSON requests.

// limitRequestBody rejects declared oversized bodies early and enforces the same
// limit on chunked/streamed requests when handlers decode them.
// RequestIngress applies the transport-admission body limit before route handlers.
func RequestIngress() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.ContentLength > maxRequestBodyBytes {
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxRequestBodyBytes)
		ctx.Next()
	}
}

func invalidJSON(ctx *gin.Context, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
		return
	}
	ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}

// Redis uses one atomic Lua execution to increment the client IP and the
// identity/token counters. A denial does not extend the TTL.
var authRateScript = redis.NewScript(`
local a = redis.call('INCR', KEYS[1])
if a == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local b = redis.call('INCR', KEYS[2])
if b == 1 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
if a > tonumber(ARGV[2]) or b > tonumber(ARGV[3]) then return 0 end
return 1
`)

func hashedRateKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// allowAuthRequest counts *attempts*, not only failed credentials. This avoids
// race conditions across API replicas and does not reveal whether an account exists.
func (server *Server) allowAuthRequest(ctx *gin.Context, operation, identity string, perIP, perIdentity int64, window time.Duration) bool {
	// Unit tests and unused legacy server construction omit the Redis limiter.
	// The active Gin runtime explicitly sets the limiter before serving.
	if server.authRateRedis == nil {
		return true
	}
	namespace := "bdspro:auth_limit:v1:" + operation + ":"
	ipKey := namespace + "ip:" + hashedRateKey(ctx.ClientIP())
	identityKey := namespace + "identity:" + hashedRateKey(strings.ToLower(strings.TrimSpace(identity)))
	allowed, err := authRateScript.Run(ctx.Request.Context(), server.authRateRedis,
		[]string{ipKey, identityKey}, int64(window.Seconds()), perIP, perIdentity).Int64()
	if err != nil {
		log.Ctx(ctx.Request.Context()).Error().Err(err).Str("operation", operation).Msg("auth.rate_limit_unavailable")
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
