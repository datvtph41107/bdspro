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
