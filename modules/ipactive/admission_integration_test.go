package ipactive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

// Transport header trust and HTTP response policy belong to services/security.
// The limiter counts only the source address supplied by its trusted caller.
func TestRedisAdmissionIsScopedToTrustedSource(t *testing.T) {
	client := redisForTest(t)
	action := Action("ipactive_admission_" + strconv.FormatInt(time.Now().UnixNano(), 10))
	limiter, err := NewRedis(client, map[Action]Policy{
		action: {Mode: ModeCount, Limit: 1, Window: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}
	middleware := NewMiddleware(limiter)
	const sourceIP = "192.0.2.78"
	for attempt := 0; attempt < 2; attempt++ {
		event, err := middleware.Check(context.Background(), action, sourceIP)
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if attempt == 0 && !event.Allowed {
			t.Fatalf("first admission unexpectedly rejected: %+v", event)
		}
		if attempt == 1 && (event.Allowed || event.RetryAfter <= 0) {
			t.Fatalf("repeated trusted source bypassed admission: %+v", event)
		}
	}
	hash := sha256.Sum256([]byte(sourceIP))
	key := SourceKeyPrefix + string(action) + ":" + hex.EncodeToString(hash[:])
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })
}
