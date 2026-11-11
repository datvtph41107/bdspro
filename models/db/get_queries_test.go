package db

import (
	"context"
	"errors"
	"testing"
)

func TestGetQueriesRejectsIncompleteTransaction(t *testing.T) {
	ctx := context.WithValue(context.Background(), txStateContextKey{}, &transactionState{})
	if _, err := GetQueries(ctx); !errors.Is(err, ErrTransactionRequired) {
		t.Fatalf("incomplete transaction must not fall back to pool: %v", err)
	}
}

func TestGetQueriesPrefersTransactionAndRejectsClosedScope(t *testing.T) {
	state := &transactionState{}
	queries := new(Queries)
	ctx := context.WithValue(context.Background(), txStateContextKey{}, state)
	ctx = context.WithValue(ctx, queriesContextKey{}, queries)

	got, err := GetQueries(ctx)
	if err != nil || got != queries {
		t.Fatalf("GetQueries must select transaction executor: got=%p err=%v", got, err)
	}

	state.close()
	if _, err := GetQueries(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("closed transaction must not fall back to pool: %v", err)
	}
}

func TestGetQueriesRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := GetQueries(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context must not execute SQL: %v", err)
	}
}

func TestIncompleteTransactionCannotReachPoolOrStartAnotherTransaction(t *testing.T) {
	state := &transactionState{}
	ctx := context.WithValue(context.Background(), txStateContextKey{}, state)

	executor := contextDBTX{}
	if _, err := executor.executor(ctx); !errors.Is(err, ErrTxStoreUnavailable) {
		t.Fatalf("pool executor accepted incomplete transaction: %v", err)
	}
	if err := state.failure(); !errors.Is(err, ErrTxStoreUnavailable) {
		t.Fatalf("incomplete transaction did not mark rollback-only: %v", err)
	}
	if err := WithTx(ctx, func(context.Context) error {
		t.Fatal("nested callback must not run with incomplete state")
		return nil
	}); !errors.Is(err, ErrNestedTransaction) {
		t.Fatalf("WithTx accepted incomplete transaction: %v", err)
	}
	if _, err := StoreFromContext(ctx); !errors.Is(err, ErrTxStoreUnavailable) {
		t.Fatalf("StoreFromContext escaped incomplete transaction: %v", err)
	}
}
