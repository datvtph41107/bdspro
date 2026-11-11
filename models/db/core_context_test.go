package db

import (
	"context"
	"errors"
	"testing"
)

// These tests exercise the database core without depending on legacy domain
// Store contracts or a running PostgreSQL server.
func TestCoreNestedTransactionReusesScope(t *testing.T) {
	scope := &transactionScope{queries: New(nil)}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)

	err := WithTx(ctx, func(txCtx context.Context) error {
		if txCtx != ctx {
			t.Fatal("nested transaction replaced the owner context")
		}
		queries, err := GetQueries(txCtx)
		if err != nil || queries != scope.queries {
			t.Fatalf("nested transaction selected different queries: got=%p err=%v", queries, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("nested transaction failed: %v", err)
	}
	if err := scope.failure(); err != nil {
		t.Fatalf("successful nested callback poisoned scope: %v", err)
	}
}

func TestCoreIgnoredNestedErrorPoisonsTransaction(t *testing.T) {
	cause := errors.New("nested write failed")
	scope := &transactionScope{queries: New(nil)}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)

	err := WithTx(ctx, func(txCtx context.Context) error {
		_ = WithTx(txCtx, func(context.Context) error { return cause })
		return nil
	})
	if !errors.Is(err, cause) {
		t.Fatalf("ignored nested failure did not poison transaction: %v", err)
	}
}

func TestCoreNestedPanicPoisonsTransaction(t *testing.T) {
	scope := &transactionScope{queries: New(nil)}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Error("nested panic did not propagate")
			}
		}()
		_ = WithTx(ctx, func(context.Context) error {
			panic("failed callback")
		})
	}()
	if err := scope.failure(); !errors.Is(err, errTransactionPanicked) {
		t.Fatalf("nested panic failed to poison transaction: %v", err)
	}
}

func TestCoreClosedScopeCannotBeReused(t *testing.T) {
	scope := &transactionScope{queries: New(nil)}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)
	scope.close()

	if _, err := GetQueries(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("GetQueries accepted closed transaction: %v", err)
	}
	if err := WithTx(ctx, func(context.Context) error {
		t.Fatal("closed transaction executed callback")
		return nil
	}); !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("WithTx accepted closed transaction: %v", err)
	}
}

// Native pgx executors do not validate per-query transaction context identity.
func TestCoreGetEngineRejectsClosedTransaction(t *testing.T) {
	scope := &transactionScope{queries: New(nil)}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)
	scope.close()
	if _, err := GetEngine(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("closed scope exposed native executor: %v", err)
	}
}

func TestCoreGetEngineRejectsMalformedTransaction(t *testing.T) {
	ctx := context.WithValue(context.Background(), transactionContextKey{}, struct{}{})
	if _, err := GetEngine(ctx); !errors.Is(err, ErrTransactionContext) {
		t.Fatalf("malformed transaction selected process pool: %v", err)
	}
}

func TestCoreWithTx2PropagatesFailureAndZerosResult(t *testing.T) {
	cause := errors.New("callback failed")
	scope := &transactionScope{queries: New(nil)}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)
	value, err := WithTx2(ctx, func(context.Context) (int, error) {
		return 7, cause
	})
	if value != 0 || !errors.Is(err, cause) {
		t.Fatalf("WithTx2 returned uncommitted result: value=%d err=%v", value, err)
	}
}
