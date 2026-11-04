package cache

import (
	"context"
	"errors"
	"testing"
)

func TestContextCacheReadsAndInvalidation(t *testing.T) {
	ctx := WithCacheContext(context.Background())
	if GetContextCache(ctx) == nil {
		t.Fatal("request cache not attached")
	}
	if WithCacheContext(ctx) != ctx {
		t.Fatal("duplicate context wrapper created")
	}
	calls := 0
	load := func(_ context.Context, _ int) (string, error) {
		calls++
		return "loaded", nil
	}
	for range 2 {
		value, err := GetWithContextCache(ctx, "account", 10, load)
		if err != nil || value != "loaded" {
			t.Fatalf("cache read: %q %v", value, err)
		}
	}
	if calls != 1 {
		t.Fatalf("memoization missed: %d calls", calls)
	}
	DeleteContextCache(ctx, "account", 10)
	_, err := GetWithContextCache(ctx, "account", 10, load)
	if err != nil || calls != 2 {
		t.Fatalf("invalidation failed: %v calls=%d", err, calls)
	}
	_, err = GetWithContextCache(context.Background(), "account", 10, load)
	if err != nil || calls != 3 {
		t.Fatalf("missing context cache must invoke loader: %v calls=%d", err, calls)
	}
}

func TestContextCacheDoesNotStoreErrors(t *testing.T) {
	ctx := WithCacheContext(context.Background())
	calls := 0
	load := func(context.Context, string) (int, error) {
		calls++
		if calls == 1 {
			return 0, errors.New("dependency unavailable")
		}
		return 42, nil
	}
	if _, err := GetWithContextCache(ctx, "lookup", "key", load); err == nil {
		t.Fatal("first loader error lost")
	}
	got, err := GetWithContextCache(ctx, "lookup", "key", load)
	if err != nil || got != 42 || calls != 2 {
		t.Fatalf("error cached unexpectedly: value=%d err=%v calls=%d", got, err, calls)
	}
}

func TestContextCacheExpiredFallsThrough(t *testing.T) {
	ctx := WithCacheContext(context.Background())
	c := GetContextCache(ctx)
	c.expires = c.expires.Add(-2 * contextCacheLifetime)
	calls := 0
	load := func(context.Context, int) (int, error) { calls++; return calls, nil }
	for range 2 {
		if _, err := GetWithContextCache(ctx, "x", 1, load); err != nil { t.Fatal(err) }
	}
	if calls != 2 { t.Fatal("expired context cache unexpectedly retained values") }
}
