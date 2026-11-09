package reqctx

import (
	"context"
	"io"
	"sync"
	"testing"
)

type trackedCloser struct{ calls int }

func (c *trackedCloser) Close() error { c.calls++; return nil }

var _ io.Closer = (*trackedCloser)(nil)

func TestRequestDataStoreIsolationAndCleanup(t *testing.T) {
	first, doneFirst := NewRequestContext(context.Background())
	second, doneSecond := NewRequestContext(context.Background())
	a, b := FromContext(first), FromContext(second)
	if a == nil || b == nil || a == b {
		t.Fatal("request stores must be isolated")
	}
	a.GetData()["form"] = "A"
	b.GetData()["form"] = "B"
	if b.GetData()["form"] != "B" {
		t.Fatal("form data leaked between requests")
	}
	var once int
	closer := &trackedCloser{}
	a.AddCleanup(func() { once++ })
	a.AddCloser(closer)
	doneFirst()
	doneFirst()
	if once != 1 || closer.calls != 1 {
		t.Fatalf("cleanup count=%d closer=%d", once, closer.calls)
	}
	var after int
	a.AddCleanup(func() { after++ })
	if after != 1 {
		t.Fatalf("late cleanup was lost: %d", after)
	}
	doneSecond()
}

func TestRequestDataStoreCanPreserveParentValues(t *testing.T) {
	type testKey struct{}
	parent := context.WithValue(context.Background(), testKey{}, "actor")
	ctx, finish := NewRequestContext(parent)
	defer finish()
	if got := ctx.Value(testKey{}); got != "actor" {
		t.Fatalf("lost parent context: %v", got)
	}
	if FromContext(context.Background()) != nil {
		t.Fatal("unexpected request store on background context")
	}
}

func TestRequestStoreConcurrentCleanupRegistration(t *testing.T) {
	ctx, done := NewRequestContext(context.Background())
	store := FromContext(ctx)
	var wg sync.WaitGroup
	var mu sync.Mutex
	called := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.AddCleanup(func() { mu.Lock(); called++; mu.Unlock() })
		}()
	}
	wg.Wait()
	done()
	if called != 10 {
		t.Fatalf("cleanup count=%d", called)
	}
}

func TestFromContextDoesNotReturnTypedNilInterface(t *testing.T) {
	var missing *requestDataStore
	ctx := context.WithValue(context.Background(), dataStoreKey{}, missing)
	if store := FromContext(ctx); store != nil {
		t.Fatalf("typed-nil pointer escaped as a non-nil interface: %T", store)
	}
}
