package middleware

import "testing"

func TestRedisKeyPrefixesRemainStable(t *testing.T) {
	if securityDeniedKeyPrefix != "bdspro:security:denied:v1:" ||
		sourceLimitKeyPrefix != "bdspro:source_limit:v1:" ||
		authThrottleKeyPrefix != "bdspro:auth_limit:v1:" {
		t.Fatal("Redis keys changed; a compatibility migration is required")
	}
}
