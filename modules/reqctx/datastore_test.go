package reqctx

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestRequestContextPreservesCancellationAndFinish(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ctx, finish := NewRequestContext(parent)
	store, ok := FromContext(ctx)
	if !ok {
		t.Fatal("missing store")
	}
	store.Set("account", int64(7))
	if got, ok := store.Get("account"); !ok || got != int64(7) {
		t.Fatalf("got %v %v", got, ok)
	}
	var order []int
	store.AddCleanup(func() { order = append(order, 1) })
	store.AddCleanup(func() { order = append(order, 2) })
	finish()
	finish()
	if !reflect.DeepEqual(order, []int{2, 1}) {
		t.Fatalf("cleanup order %v", order)
	}
	if _, ok := store.Get("account"); ok {
		t.Fatal("request data survived cleanup")
	}
	cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("parent cancellation not propagated")
	}
}

func TestSeparateAndConcurrentRequests(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, finish := NewRequestContext(context.Background())
			defer finish()
			s := MustFromContext(ctx)
			s.Set("id", i)
			if got, ok := s.Get("id"); !ok || got != i {
				t.Errorf("id %d != %v", i, got)
			}
		}(i)
	}
	wg.Wait()
}

func TestLateCleanupPanics(t *testing.T) {
	ctx, finish := NewRequestContext(context.Background())
	finish()
	defer func() {
		if recover() == nil {
			t.Error("expected panic for cleanup after finish")
		}
	}()
	MustFromContext(ctx).AddCleanup(func() {})
}

