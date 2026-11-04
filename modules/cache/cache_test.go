package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

func TestMemoryCacheTTLAndLRU(t *testing.T) {
	ctx := context.Background()
	raw, err := NewMemoryStringCache(2)
	if err != nil {
		t.Fatal(err)
	}
	c := raw.(*memoryCache)
	now := time.Unix(100, 0)
	c.now = func() time.Time { return now }

	if err := c.Put(ctx, "one", "1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Put(ctx, "two", "2", time.Minute); err != nil {
		t.Fatal(err)
	}
	if value, hit, err := c.Get(ctx, "one"); err != nil || !hit || value != "1" {
		t.Fatalf("expected cache hit: %q %v %v", value, hit, err)
	}
	if err := c.Put(ctx, "three", "3", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := c.Get(ctx, "two"); hit {
		t.Fatal("least recently used entry not evicted")
	}
	now = now.Add(2 * time.Minute)
	if _, hit, _ := c.Get(ctx, "one"); hit {
		t.Fatal("expired entry returned")
	}
}

func TestCacheAsideAndInvalidTTL(t *testing.T) {
	ctx := context.Background()
	c, err := NewStringCache(setting.CacheSettings{Adapter: "memory", MaxEntries: 8}, nil)
	if err != nil {
		t.Fatal(err)
	}
	loads := 0
	load := func(context.Context) (string, error) { loads++; return "value", nil }
	for range 2 {
		value, err := GetString(ctx, c, "key", time.Minute, load)
		if err != nil || value != "value" {
			t.Fatalf("cache aside: %q %v", value, err)
		}
	}
	if loads != 1 {
		t.Fatalf("loader called %d times", loads)
	}
	if err := c.Put(ctx, "key", "x", 0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("zero TTL accepted: %v", err)
	}
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := c.Get(ctx, "key"); hit {
		t.Fatal("deleted entry was returned")
	}
}

func TestCacheFactoryRequiresRedisClient(t *testing.T) {
	if _, err := NewStringCache(setting.CacheSettings{Adapter: "redis", Prefix: "cache:"}, nil); err == nil {
		t.Fatal("Redis cache accepted nil client")
	}
	if _, err := NewStringCache(setting.CacheSettings{Adapter: "invalid", MaxEntries: 2}, nil); err == nil {
		t.Fatal("unknown cache backend accepted")
	}
}
