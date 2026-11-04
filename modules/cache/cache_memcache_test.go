package cache

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

func TestMemcacheTTLAndKeyValidation(t *testing.T) {
	if _, err := memcacheExpiration(0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("zero TTL accepted: %v", err)
	}
	if exp, err := memcacheExpiration(24 * time.Hour); err != nil || exp != 86400 {
		t.Fatalf("relative expiration: %d %v", exp, err)
	}
	if exp, err := memcacheExpiration(31 * 24 * time.Hour); err != nil || int64(exp) <= time.Now().Unix() {
		t.Fatalf("absolute expiration: %d %v", exp, err)
	}
	if validMemcacheKey("bad key") || validMemcacheKey(strings.Repeat("a", 251)) || !validMemcacheKey("cache:item") {
		t.Fatal("Memcached key validation incorrect")
	}
	if _, err := NewStringCache(setting.CacheSettings{Adapter: "memcache", Prefix: "c:"}, nil); err == nil {
		t.Fatal("Memcached without host was accepted")
	}
}

func TestMemcacheIntegration(t *testing.T) {
	host := os.Getenv("BDS_TEST_MEMCACHE_ADDR")
	if host == "" {
		t.Skip("set BDS_TEST_MEMCACHE_ADDR to test Memcached backend")
	}
	cache, err := NewMemcacheStringCache(host, "bds-test:")
	if err != nil { t.Fatal(err) }
	ctx := context.Background()
	key := "test:" + time.Now().Format("150405.000000000")
	if _, hit, err := cache.Get(ctx, key); err != nil || hit {
		t.Fatalf("expected miss: hit=%v err=%v", hit, err)
	}
	if err := cache.Put(ctx, key, "abc", time.Second); err != nil { t.Fatal(err) }
	if value, hit, err := cache.Get(ctx, key); err != nil || !hit || value != "abc" {
		t.Fatalf("expected hit: %q hit=%v err=%v", value, hit, err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, hit, err := cache.Get(ctx, key); err != nil || hit {
		t.Fatalf("expired entry should miss: hit=%v err=%v", hit, err)
	}
	if err := cache.Delete(ctx, key); err != nil { t.Fatal(err) }
}
