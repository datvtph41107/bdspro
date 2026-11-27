package ipactive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	redis "github.com/go-redis/redis/v8"
)

func TestRetryAfterCeiling(t *testing.T) {
	addr := os.Getenv("BDS_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("requires disposable Redis")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	for _, mode := range []Mode{ModeCount, ModeCooldown} {
		t.Run(string(mode), func(t *testing.T) {
			action := Action("test_retry_" + string(mode))
			limiter, err := NewRedis(client, map[Action]Policy{
				action: {Mode: mode, Limit: 1, Window: 1500 * time.Millisecond},
			})
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256([]byte(uuid.NewString()))
			subject := hex.EncodeToString(digest[:])
			key := "bdspro:subject_limit:v1:" + string(action) + ":" + subject
			defer client.Del(context.Background(), key)
			first, err := limiter.CheckSubject(ctx, action, subject)
			if err != nil || !first.Allowed {
				t.Fatal("first admission failed", err)
			}
			ttl, err := client.PTTL(ctx, key).Result()
			if err != nil || ttl <= 1500*time.Millisecond || ttl > 2*time.Second {
				t.Fatal("input window was not rounded up", ttl, err)
			}
			// Reset only this random test key to a fractional remaining window.
			if err := client.PExpire(ctx, key, 1200*time.Millisecond).Err(); err != nil {
				t.Fatal(err)
			}
			event, err := limiter.CheckSubject(ctx, action, subject)
			if err != nil || event.Allowed || event.RetryAfter != 2 {
				t.Fatalf("expected denied with ceil(1.2s)=2, got %+v, %v", event, err)
			}
		})
	}
}
