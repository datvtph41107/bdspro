package db

import (
	"context"
	"errors"
	"testing"
)

// These tests exercise the existing in-memory transaction policy, not a
// PostgreSQL transaction. Database-level atomicity and wrong-context pool
// escape require separate disposable-Postgres integration tests.
func TestTransactionScopeIgnoredNestedFailureIsSticky(t *testing.T) {
	wanted := errors.New("nested write failed")
	scope := &transactionScope{}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)

	err := WithTx(ctx, func(txCtx context.Context) error {
		// Simulate a caller that incorrectly ignores the nested failure.
		_ = WithTx(txCtx, func(context.Context) error { return wanted })
		return nil
	})
	if !errors.Is(err, wanted) {
		t.Fatalf("outer transaction returned %v; want sticky %v", err, wanted)
	}
	if !errors.Is(scope.failure(), wanted) {
		t.Fatalf("scope failure = %v; want %v", scope.failure(), wanted)
	}
	if InTransaction(ctx) {
		t.Fatal("failed scope must not report an active usable transaction")
	}

	called := false
	err = WithTx(ctx, func(context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, wanted) || called {
		t.Fatalf("failed scope reused: err=%v called=%v", err, called)
	}
}

func TestTransactionScopeCanceledNestedContextIsSticky(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	scope := &transactionScope{}
	ctx := context.WithValue(parent, transactionContextKey{}, scope)
	cancel()

	called := false
	err := WithTx(ctx, func(context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("canceled scope: err=%v called=%v", err, called)
	}
	if !errors.Is(scope.failure(), context.Canceled) {
		t.Fatalf("scope failure = %v; want context.Canceled", scope.failure())
	}
}

func TestTransactionScopeMissingQueriesPoisonsScope(t *testing.T) {
	scope := &transactionScope{} // Deliberately invalid transaction scope.
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)

	queries, err := GetQueries(ctx)
	if queries != nil || !errors.Is(err, ErrTransactionContext) {
		t.Fatalf("GetQueries = (%v, %v); want nil, ErrTransactionContext", queries, err)
	}
	if !errors.Is(scope.failure(), ErrTransactionContext) {
		t.Fatalf("scope failure = %v; want ErrTransactionContext", scope.failure())
	}
	if err := WithTx(ctx, func(context.Context) error { return nil }); !errors.Is(err, ErrTransactionContext) {
		t.Fatalf("WithTx after poisoned scope = %v; want ErrTransactionContext", err)
	}
}

func TestTransactionScopeClosedRejectsAccess(t *testing.T) {
	scope := &transactionScope{}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)
	scope.close()

	if _, err := GetQueries(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("GetQueries after close = %v; want ErrTransactionClosed", err)
	}
	if _, err := GetEngine(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("GetEngine after close = %v; want ErrTransactionClosed", err)
	}
	called := false
	if err := WithTx(ctx, func(context.Context) error {
		called = true
		return nil
	}); !errors.Is(err, ErrTransactionClosed) || called {
		t.Fatalf("WithTx after close: err=%v called=%v", err, called)
	}
}

func TestTransactionScopePanicPoisonsScope(t *testing.T) {
	scope := &transactionScope{}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = WithTx(ctx, func(context.Context) error {
			panic("nested transaction panic")
		})
	}()

	if recovered != "nested transaction panic" {
		t.Fatalf("recovered = %v; want original panic value", recovered)
	}
	if !errors.Is(scope.failure(), errTransactionPanicked) {
		t.Fatalf("scope failure = %v; want errTransactionPanicked", scope.failure())
	}
}

func TestTransactionScopeWithTx2ZeroValueOnNestedError(t *testing.T) {
	wanted := errors.New("write rejected")
	scope := &transactionScope{}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, scope)

	value, err := WithTx2(ctx, func(context.Context) (int, error) {
		return 42, wanted
	})
	if value != 0 || !errors.Is(err, wanted) {
		t.Fatalf("WithTx2 = (%d, %v); want zero and %v", value, err, wanted)
	}
}
