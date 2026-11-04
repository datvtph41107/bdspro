package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/connection"
)

func TestRedisCacheIntegration(t *testing.T) {
	address := os.Getenv("BDS_TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("set BDS_TEST_REDIS_ADDR to test the Redis backend")
	}
	ctx := context.Background()
	manager := connection.NewRedisManager()
	defer manager.Close()
	client, err := manager.Client(ctx, address)
	if err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("cache-test:%d:", time.Now().UnixNano())
	cache, err := NewRedisStringCache(client, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if _, hit, err := cache.Get(ctx, "missing"); err != nil || hit {
		t.Fatalf("missing key must be an ordinary miss: hit=%v err=%v", hit, err)
	}
	if err := cache.Put(ctx, "entry", "value", time.Minute); err != nil {
		t.Fatal(err)
	}
	if value, hit, err := cache.Get(ctx, "entry"); err != nil || !hit || value != "value" {
		t.Fatalf("get: value=%q hit=%v err=%v", value, hit, err)
	}
	if err := cache.Delete(ctx, "entry"); err != nil {
		t.Fatal(err)
	}
	if _, hit, err := cache.Get(ctx, "entry"); err != nil || hit {
		t.Fatalf("delete: hit=%v err=%v", hit, err)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cache.Get(ctx, "entry"); err == nil {
		t.Fatal("closed Redis backend was silently treated as a miss")
	}
}
