package startup

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRunRollbackReverseOrder(t *testing.T) {
	var calls []string
	fail := errors.New("JWT unavailable")
	appendCall := func(name string) { calls = append(calls, name) }
	err := Run(context.Background(),
		Step{
			Name:     "PostgreSQL",
			Init:     func(context.Context) error { appendCall("db.open"); return nil },
			Rollback: func() { appendCall("db.close") },
		},
		Step{
			Name:     "Redis",
			Init:     func(context.Context) error { appendCall("redis.open"); return nil },
			Rollback: func() { appendCall("redis.close") },
		},
		Step{
			Name: "JWT",
			Init: func(context.Context) error { appendCall("jwt.init"); return fail },
		},
	)
	if !errors.Is(err, fail) || !strings.Contains(err.Error(), "JWT") {
		t.Fatalf("missing wrapped startup failure: %v", err)
	}
	want := []string{"db.open", "redis.open", "jwt.init", "redis.close", "db.close"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("call order = %v, want %v", calls, want)
	}
}

func TestRunFailedStepMustCleanUpOwnPartialState(t *testing.T) {
	called := false
	err := Run(context.Background(), Step{
		Name:     "PostgreSQL",
		Init:     func(context.Context) error { return errors.New("ping failed") },
		Rollback: func() { called = true },
	})
	if err == nil {
		t.Fatal("want failure")
	}
	if called {
		t.Fatal("Run must not rollback a failed Init; that Init owns partial cleanup")
	}
}

func TestRunRollbackOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var events []string
	err := Run(ctx,
		Step{
			Name: "PostgreSQL",
			Init: func(context.Context) error {
				events = append(events, "db.ready")
				cancel()
				return nil
			},
			Rollback: func() { events = append(events, "db.closed") },
		},
		Step{
			Name: "Redis",
			Init: func(context.Context) error {
				t.Fatal("redis startup must not run after cancellation")
				return nil
			},
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"db.ready", "db.closed"}) {
		t.Fatalf("wrong cancellation cleanup: %v", events)
	}
}

func TestIdempotentRollbackAndTerminationCloser(t *testing.T) {
	calls := 0
	var once sync.Once
	closeResource := func() {
		once.Do(func() { calls++ })
	}

	err := Run(context.Background(),
		Step{Name: "DB", Init: func(context.Context) error { return nil }, Rollback: closeResource},
		Step{Name: "Redis", Init: func(context.Context) error { return errors.New("offline") }},
	)
	if err == nil {
		t.Fatal("missing failure")
	}
	closeResource()
	if calls != 1 {
		t.Fatalf("resource cleanup was not idempotent: %d", calls)
	}
}
