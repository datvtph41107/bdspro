// Package ratelimit provides Gin-independent Redis atomic counters and hashing.
package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/go-redis/redis/v8"
)

var incrementScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return n
`)

var pairScript = redis.NewScript(`
local a = redis.call('INCR', KEYS[1])
if a == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local b = redis.call('INCR', KEYS[2])
if b == 1 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
if a > tonumber(ARGV[2]) or b > tonumber(ARGV[3]) then return 0 end
return 1
`)

// Hash retains the SHA-256 lowercase hex representation of Redis source keys.
func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Increment increments a key and sets expiry only on its first increment.
func Increment(ctx context.Context, client *redis.Client, key string, ttlSeconds int64) (int64, error) {
	return incrementScript.Run(ctx, client, []string{key}, ttlSeconds).Int64()
}

// AllowPair increments both keys even when either quota is exhausted.
// It returns 1 if allowed, 0 if denied; the HTTP caller decides 429/503.
func AllowPair(ctx context.Context, client *redis.Client, ipKey, identityKey string, ttlSeconds, perIP, perIdentity int64) (int64, error) {
	return pairScript.Run(ctx, client, []string{ipKey, identityKey}, ttlSeconds, perIP, perIdentity).Int64()
}
