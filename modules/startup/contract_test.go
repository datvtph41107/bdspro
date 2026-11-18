package startup

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRunOrdersRequiredSteps(t *testing.T) {
	var order []string
	err := Run(context.Background(),
		Step{Name: "settings", Init: func(context.Context) error {
			order = append(order, "settings")
			return nil
		}},
		Step{Name: "database", Init: func(context.Context) error {
			order = append(order, "database")
			return nil
		}},
		Step{Name: "redis", Init: func(context.Context) error {
			order = append(order, "redis")
			return nil
		}},
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := []string{"settings", "database", "redis"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("initialization order = %v, want %v", order, want)
	}
}

func TestRunFailedStepRollsBackCompletedInReverseOrder(t *testing.T) {
	failure := errors.New("redis unavailable")
	var order []string
	err := Run(context.Background(),
		Step{Name: "database", Init: func(context.Context) error {
			order = append(order, "init database")
			return nil
		}, Rollback: func() { order = append(order, "close database") }},
		Step{Name: "cache", Init: func(context.Context) error {
			order = append(order, "init cache")
			return nil
		}, Rollback: func() { order = append(order, "close cache") }},
		Step{Name: "redis", Init: func(context.Context) error {
			order = append(order, "init redis")
			return failure
		}, Rollback: func() { order = append(order, "close redis") }},
		Step{Name: "http", Init: func(context.Context) error {
			order = append(order, "init http")
			return nil
		}},
	)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "redis") {
		t.Fatalf("Run error = %v; want wrapped redis failure", err)
	}
	want := []string{"init database", "init cache", "init redis", "close cache", "close database"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("event order = %v, want %v", order, want)
	}
}

func TestRunCanceledAfterSuccessfulStepRollsBack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var order []string
	err := Run(ctx,
		Step{Name: "database", Init: func(context.Context) error {
			order = append(order, "init database")
			cancel()
			return nil
		}, Rollback: func() { order = append(order, "close database") }},
		Step{Name: "redis", Init: func(context.Context) error {
			order = append(order, "init redis")
			return nil
		}},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
	if want := []string{"init database", "close database"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("event order = %v, want %v", order, want)
	}
}

func TestRunRejectsInvalidStepsBeforeExecuting(t *testing.T) {
	called := false
	err := Run(context.Background(),
		Step{Name: "database", Init: func(context.Context) error {
			called = true
			return nil
		}},
		Step{Name: "redis"},
	)
	if err == nil || called {
		t.Fatalf("Run error = %v; called = %v, want validation before execution", err, called)
	}
}

func TestRunRejectsNilContextAndEmptyPipeline(t *testing.T) {
	if err := Run(nil, Step{Name: "ok", Init: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected nil-context error")
	}
	if err := Run(context.Background()); err == nil {
		t.Fatal("expected no-steps error")
	}
}
