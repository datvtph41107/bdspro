package ipactive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	redis "github.com/go-redis/redis/v8"
)

func redisForTest(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("BDS_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set BDS_TEST_REDIS_ADDR for Redis integration tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("Redis test dependency unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRedisCountAtomicAndLegacyKey(t *testing.T) {
	client := redisForTest(t)
	ctx := context.Background()
	action := Action(fmt.Sprintf("ipactive_test_%d", time.Now().UnixNano()))
	const source = "192.0.2.47"
	limiter, err := NewRedis(client, map[Action]Policy{
		action: {Mode: ModeCount, Limit: 5, Window: 30 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	var admitted atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event, checkErr := limiter.Check(ctx, action, source)
			if checkErr != nil {
				t.Errorf("Redis check: %v", checkErr)
				return
			}
			if event.Allowed {
				admitted.Add(1)
			} else if event.RetryAfter < 1 || event.RetryAfter > 30 {
				t.Errorf("invalid retry-after: %+v", event)
			}
		}()
	}
	wg.Wait()
	if admitted.Load() != 5 {
		t.Fatalf("expected 5 admitted out of 24, got %d", admitted.Load())
	}
	event, err := limiter.Check(ctx, action, source)
	if err != nil || event.Allowed || event.Attempts != 25 {
		t.Fatalf("incorrect window counter: %+v %v", event, err)
	}
	key := SourceKeyPrefix + string(action) + ":" + event.SourceID
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })
	if n, err := client.Get(ctx, key).Int64(); err != nil || n != 25 {
		t.Fatalf("legacy namespace or counter changed: %d %v", n, err)
	}
	if event.SourceID == source {
		t.Fatal("event exposed source IP")
	}
	if event.OccurredAt.IsZero() || event.OccurredAt.AsTime().After(time.Now().Add(time.Second)) {
		t.Fatalf("admission event timestamp invalid: %v", event.OccurredAt)
	}
}

func TestRedisCooldownOnlyRecordsAdmittedAttempts(t *testing.T) {
	client := redisForTest(t)
	ctx := context.Background()
	action := Action(fmt.Sprintf("ipactive_cooldown_%d", time.Now().UnixNano()))
	limiter, err := NewRedis(client, map[Action]Policy{
		action: {Mode: ModeCooldown, Limit: 1, Window: 5 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := limiter.Check(ctx, action, "192.0.2.49")
	if err != nil || !first.Allowed {
		t.Fatalf("first attempt rejected: %+v %v", first, err)
	}
	key := SourceKeyPrefix + string(action) + ":" + first.SourceID
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })
	second, err := limiter.Check(ctx, action, "192.0.2.49")
	if err != nil || second.Allowed || second.RetryAfter < 1 {
		t.Fatalf("cooldown bypassed: %+v %v", second, err)
	}
	other, err := limiter.Check(ctx, action, "192.0.2.50")
	if err != nil || !other.Allowed {
		t.Fatalf("unrelated IP throttled: %+v %v", other, err)
	}
	t.Cleanup(func() { _ = client.Del(context.Background(), SourceKeyPrefix+string(action)+":"+other.SourceID).Err() })
}

func TestRedisRejectedInvalidSourceAndCancelledContext(t *testing.T) {
	client := redisForTest(t)
	action := Action(fmt.Sprintf("ipactive_invalid_%d", time.Now().UnixNano()))
	policies := map[Action]Policy{
		action: {Mode: ModeCount, Limit: 1, Window: time.Second},
	}
	limiter, err := NewRedis(client, policies)
	if err != nil {
		t.Fatal(err)
	}
	policies[action] = Policy{}
	if _, err := limiter.Check(context.Background(), Action("unknown"), "192.0.2.51"); !errors.Is(err, ErrUnknownAction) {
		t.Fatalf("unknown operation admitted: %v", err)
	}
	if _, err := limiter.Check(context.Background(), action, "not-an-ip"); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("untrusted source admitted: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := limiter.Check(ctx, action, "192.0.2.51"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("canceled Redis operation did not fail closed: %v", err)
	}
}
