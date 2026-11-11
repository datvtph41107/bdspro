package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func TestLegacyTransactionAPIsRejectPartialContext(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *SQLStore) error
	}{
		{
			name: "ExecTx",
			run: func(ctx context.Context, store *SQLStore) error {
				return store.ExecTx(ctx, func(*Queries) error {
					t.Fatal("partial context opened an independent SQLC transaction")
					return nil
				})
			},
		},
		{
			name: "ExecTxRaw",
			run: func(ctx context.Context, store *SQLStore) error {
				return store.ExecTxRaw(ctx, func(pgx.Tx) error {
					t.Fatal("partial context opened an independent raw transaction")
					return nil
				})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &transactionState{}
			ctx := context.WithValue(context.Background(), txStateContextKey{}, state)
			if err := test.run(ctx, &SQLStore{}); !errors.Is(err, ErrNestedTransaction) {
				t.Fatalf("partial transaction must fail closed: %v", err)
			}
			if err := state.failure(); !errors.Is(err, ErrNestedTransaction) {
				t.Fatalf("partial transaction did not poison owner: %v", err)
			}
		})
	}
}

func TestTxQueriesPoisonsOwnerWhenQueriesMetadataIsMissing(t *testing.T) {
	state := &transactionState{}
	ctx := context.WithValue(context.Background(), txStateContextKey{}, state)
	if _, err := TxQueries(ctx); !errors.Is(err, ErrTransactionRequired) {
		t.Fatalf("missing executor must fail: %v", err)
	}
	if err := state.failure(); !errors.Is(err, ErrTransactionRequired) {
		t.Fatalf("owner was not marked rollback-only: %v", err)
	}
}

func TestGetQueriesPrefersRequestBoundExecutor(t *testing.T) {
	pool := new(pgxpool.Pool) // identity only: no SQL is executed
	store := NewStore(pool)
	ctx := WithStore(context.Background(), store)
	got, err := GetQueries(ctx)
	if err != nil || got != store.Queries {
		t.Fatalf("request-bound queries ignored: got=%p err=%v", got, err)
	}
}

func TestGetQueriesRejectsMalformedRequestBoundExecutor(t *testing.T) {
	ctx := WithStore(context.Background(), &contextTestStore{})
	if _, err := GetQueries(ctx); !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("malformed scoped engine must not fall back to process global: %v", err)
	}
}

func TestGetQueriesRejectsTransactionViewWithoutScope(t *testing.T) {
	ctx := WithStore(context.Background(), &SQLStore{Queries: new(Queries)})
	if _, err := GetQueries(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("transaction-only store cannot be reused without scope: %v", err)
	}
}

// A malformed explicit binding must never silently fall through to the
// process engine. Request isolation takes priority over convenience.
func TestStoreFromContextRejectsMalformedBoundCapability(t *testing.T) {
	ctx := context.WithValue(context.Background(), storeContextKey{}, struct{}{})
	if _, err := StoreFromContext(ctx); !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("malformed bound Store should fail closed: %v", err)
	}
	if err := WithTx(ctx, func(context.Context) error {
		t.Fatal("malformed binding must not run transaction callback")
		return nil
	}); !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("malformed bound transaction owner should fail closed: %v", err)
	}
}

func TestStoreFromContextRejectsTypedNilSQLStore(t *testing.T) {
	var missing *SQLStore
	ctx := context.WithValue(context.Background(), storeContextKey{}, missing)
	if _, err := StoreFromContext(ctx); !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("typed nil Store must fail closed: %v", err)
	}
	if err := WithTx(ctx, func(context.Context) error {
		t.Fatal("typed nil transaction store must not execute callback")
		return nil
	}); !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("typed nil transaction store must fail closed: %v", err)
	}
}
