package middleware

import (
	"testing"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

func TestRedisKeyPrefixesRemainStable(t *testing.T) {
	if securityDeniedKeyPrefix != "bdspro:security:denied:v1:" ||
		ipactive.SourceKeyPrefix != "bdspro:source_limit:v1:" ||
		authThrottleKeyPrefix != "bdspro:auth_limit:v1:" {
		t.Fatal("Redis keys changed; a compatibility migration is required")
	}
}
