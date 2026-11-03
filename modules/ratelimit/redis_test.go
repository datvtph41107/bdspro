package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

func TestHashKeyStable(t *testing.T) {
	if got := Hash("hello"); got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("hash changed: %q", got)
	}
}

func TestRedisCounterAndPairAreAtomic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	prefix := "test:ratelimit:" + uuid.NewString() + ":"
	counter, ip, identity := prefix+"counter", prefix+"ip", prefix+"identity"
	defer client.Del(context.Background(), counter, ip, identity)

	first, err := Increment(ctx, client, counter, 120)
	if err != nil || first != 1 {
		t.Fatalf("first increment: %d, %v", first, err)
	}
	second, err := Increment(ctx, client, counter, 120)
	if err != nil || second != 2 {
		t.Fatalf("second increment: %d, %v", second, err)
	}
	ttl, err := client.TTL(ctx, counter).Result()
	if err != nil || ttl <= 0 || ttl > 120*time.Second {
		t.Fatalf("TTL after reuse: %s, %v", ttl, err)
	}
	allowed, err := AllowPair(ctx, client, ip, identity, 120, 1, 1)
	if err != nil || allowed != 1 {
		t.Fatalf("allowed pair: %d, %v", allowed, err)
	}
	denied, err := AllowPair(ctx, client, ip, identity, 120, 1, 1)
	if err != nil || denied != 0 {
		t.Fatalf("denied pair: %d, %v", denied, err)
	}
	for _, key := range []string{ip, identity} {
		count, err := client.Get(ctx, key).Int64()
		if err != nil || count != 2 {
			t.Fatalf("denied attempt must increment %s: %d, %v", key, count, err)
		}
	}
}
