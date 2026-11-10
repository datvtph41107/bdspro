package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type contextTestStore struct {
	Store
	afterCallback error
	transactions  int
	committed     bool
}

func (s *contextTestStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	s.transactions++
	if err := fn(&Queries{}); err != nil {
		return err
	}
	if s.afterCallback != nil {
		return s.afterCallback
	}
	s.committed = true
	return nil
}

func TestWithTx2DoesNotPublishFailedCommit(t *testing.T) {
	store := &contextTestStore{afterCallback: errors.New("failed commit")}
	ctx := WithStore(context.Background(), store)
	value, err := WithTx2(ctx, func(txCtx context.Context) (int, error) {
		if _, err := TxQueries(txCtx); err != nil {
			t.Fatal(err)
		}
		if err := WithTx(txCtx, func(nested context.Context) error {
			_, err := TxQueries(nested)
			return err
		}); err != nil {
			t.Fatalf("nested transaction did not reuse executor: %v", err)
		}
		return 42, nil
	})
	if err == nil || value != 0 || store.transactions != 1 || store.committed {
		t.Fatalf("value=%d err=%v transactions=%d committed=%v", value, err, store.transactions, store.committed)
	}
}

func TestWithTxReusesTransactionWithoutOpeningAnother(t *testing.T) {
	store := &contextTestStore{}
	ctx := WithStore(context.Background(), store)
	err := WithTx(ctx, func(txCtx context.Context) error {
		outer, err := TxQueries(txCtx)
		if err != nil {
			return err
		}
		return WithTx(txCtx, func(nested context.Context) error {
			inner, err := TxQueries(nested)
			if err != nil {
				return err
			}
			if inner != outer {
				t.Fatal("nested operation did not use the same SQLC transaction")
			}
			return nil
		})
	})
	if err != nil || !store.committed || store.transactions != 1 {
		t.Fatalf("err=%v committed=%v transactions=%d", err, store.committed, store.transactions)
	}
}

func TestWithTxInnerFailureCannotBeIgnoredAndCommitted(t *testing.T) {
	store := &contextTestStore{}
	ctx := WithStore(context.Background(), store)
	sentinel := errors.New("inner mutation rejected")
	err := WithTx(ctx, func(txCtx context.Context) error {
		inner := WithTx(txCtx, func(context.Context) error { return sentinel })
		if !errors.Is(inner, sentinel) {
			t.Fatalf("inner error=%v", inner)
		}
		// Simulate a buggy caller that swallows the inner error and tries to
		// run more work and commit the outer transaction anyway.
		if nested := WithTx(txCtx, func(context.Context) error {
			t.Fatal("poisoned transaction allowed additional work")
			return nil
		}); !errors.Is(nested, sentinel) {
			t.Fatalf("poisoned nested transaction returned %v", nested)
		}
		return nil
	})
	if !errors.Is(err, sentinel) || store.committed || store.transactions != 1 {
		t.Fatalf("err=%v committed=%v transactions=%d", err, store.committed, store.transactions)
	}
}

func TestWithTx2DiscardsResultOnNestedFailure(t *testing.T) {
	store := &contextTestStore{}
	ctx := WithStore(context.Background(), store)
	sentinel := errors.New("nested failure")
	value, err := WithTx2(ctx, func(txCtx context.Context) (string, error) {
		_ = WithTx(txCtx, func(context.Context) error { return sentinel })
		return "never committed", nil
	})
	if value != "" || !errors.Is(err, sentinel) || store.committed {
		t.Fatalf("result=%q err=%v committed=%v", value, err, store.committed)
	}
}

func TestWithTxRequiresStore(t *testing.T) {
	err := WithTx(context.Background(), func(context.Context) error { return nil })
	if !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("got %v", err)
	}
}

func TestSQLStoreExecTxReusesContext(t *testing.T) {
	owner := &contextTestStore{}
	ctx := WithStore(context.Background(), owner)
	// An uninitialized SQLStore must not touch its nil pool when a
	// transaction-scoped executor is already available.
	nested := &SQLStore{}
	err := WithTx(ctx, func(txCtx context.Context) error {
		expected, err := TxQueries(txCtx)
		if err != nil {
			return err
		}
		return nested.ExecTx(txCtx, func(got *Queries) error {
			if got != expected {
				t.Fatal("SQLStore.ExecTx opened another executor")
			}
			return nil
		})
	})
	if err != nil || !owner.committed || owner.transactions != 1 {
		t.Fatalf("err=%v owner=%+v", err, owner)
	}
}

func TestSQLStoreExecTxInnerErrorPoisonsOwner(t *testing.T) {
	owner := &contextTestStore{}
	ctx := WithStore(context.Background(), owner)
	sentinel := errors.New("reject inner SQLStore write")
	nested := &SQLStore{}
	err := WithTx(ctx, func(txCtx context.Context) error {
		inner := nested.ExecTx(txCtx, func(*Queries) error { return sentinel })
		if !errors.Is(inner, sentinel) {
			t.Fatalf("inner error=%v", inner)
		}
		return nil // swallowed, but the owner must roll back
	})
	if !errors.Is(err, sentinel) || owner.committed || owner.transactions != 1 {
		t.Fatalf("err=%v owner=%+v", err, owner)
	}
}

func TestSQLStoreRawTransactionCannotNestInSQLC(t *testing.T) {
	owner := &contextTestStore{}
	ctx := WithStore(context.Background(), owner)
	nested := &SQLStore{}
	err := WithTx(ctx, func(txCtx context.Context) error {
		inner := nested.ExecTxRaw(txCtx, func(pgx.Tx) error {
			t.Fatal("nested raw transaction acquired a separate connection")
			return nil
		})
		if !errors.Is(inner, ErrRawTransactionInSQLC) {
			t.Fatalf("raw transaction error=%v", inner)
		}
		return nil // refusing the raw tx must poison the outer transaction
	})
	if !errors.Is(err, ErrRawTransactionInSQLC) || owner.committed || owner.transactions != 1 {
		t.Fatalf("err=%v owner=%+v", err, owner)
	}
}

func TestWithTxCancellationPoisonsCommit(t *testing.T) {
	owner := &contextTestStore{}
	ctx, cancel := context.WithCancel(WithStore(context.Background(), owner))
	defer cancel()
	err := WithTx(ctx, func(context.Context) error {
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || owner.committed || owner.transactions != 1 {
		t.Fatalf("err=%v owner=%+v", err, owner)
	}
}

func TestWithTxAlreadyCanceledDoesNotBegin(t *testing.T) {
	owner := &contextTestStore{}
	ctx, cancel := context.WithCancel(WithStore(context.Background(), owner))
	cancel()
	err := WithTx(ctx, func(context.Context) error {
		t.Fatal("canceled transaction should not execute callback")
		return nil
	})
	if !errors.Is(err, context.Canceled) || owner.transactions != 0 {
		t.Fatalf("err=%v owner=%+v", err, owner)
	}
}

func TestWithTxRecoveredNestedPanicCannotCommit(t *testing.T) {
	owner := &contextTestStore{}
	ctx := WithStore(context.Background(), owner)
	err := WithTx(ctx, func(txCtx context.Context) error {
		func() {
			defer func() {
				if got := recover(); got != "nested panic" {
					t.Fatalf("recovered=%v, want nested panic", got)
				}
			}()
			_ = WithTx(txCtx, func(context.Context) error {
				panic("nested panic")
			})
		}()
		// A broken caller catches and ignores the nested panic.
		return nil
	})
	if !errors.Is(err, errTransactionPanicked) || owner.committed || owner.transactions != 1 {
		t.Fatalf("err=%v committed=%v transactions=%d", err, owner.committed, owner.transactions)
	}
}

func TestWithTxRecoveredNestedResultPanicCannotCommit(t *testing.T) {
	owner := &contextTestStore{}
	ctx := WithStore(context.Background(), owner)
	err := WithTx(ctx, func(txCtx context.Context) error {
		func() {
			defer func() {
				if got := recover(); got != "result panic" {
					t.Fatalf("recovered=%v, want result panic", got)
				}
			}()
			_, _ = WithTx2(txCtx, func(context.Context) (string, error) {
				panic("result panic")
			})
		}()
		return nil
	})
	if !errors.Is(err, errTransactionPanicked) || owner.committed || owner.transactions != 1 {
		t.Fatalf("err=%v committed=%v transactions=%d", err, owner.committed, owner.transactions)
	}
}

func TestTxQueriesRejectsPoisonedTransaction(t *testing.T) {
	store := &contextTestStore{}
	ctx := WithStore(context.Background(), store)
	sentinel := errors.New("nested write failed")
	err := WithTx(ctx, func(txCtx context.Context) error {
		if _, err := TxQueries(txCtx); err != nil {
			return err
		}
		inner := WithTx(txCtx, func(context.Context) error { return sentinel })
		if !errors.Is(inner, sentinel) {
			t.Fatalf("inner err=%v", inner)
		}
		if q, err := TxQueries(txCtx); q != nil || !errors.Is(err, sentinel) {
			t.Fatalf("poisoned transaction returned q=%p err=%v", q, err)
		}
		return nil
	})
	if !errors.Is(err, sentinel) || store.committed || store.transactions != 1 {
		t.Fatalf("err=%v committed=%v transactions=%d", err, store.committed, store.transactions)
	}
}

func TestTxQueriesRejectsUnownedExecutor(t *testing.T) {
	ctx := context.WithValue(context.Background(), queriesContextKey{}, &Queries{})
	if q, err := TxQueries(ctx); q != nil || !errors.Is(err, ErrNestedTransaction) {
		t.Fatalf("unowned q=%p err=%v", q, err)
	}
}

// Transaction-bound contexts must not provide a live executor after their
// owning callback returns. This must hold after both COMMIT and ROLLBACK.
func TestTransactionContextExpiresAfterOwner(t *testing.T) {
	cases := []struct {
		name     string
		rollback bool
	}{
		{name: "commit"},
		{name: "rollback", rollback: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner := &contextTestStore{}
			ctx := WithStore(context.Background(), owner)
			var escaped context.Context
			sentinel := errors.New("abort")
			err := WithTx(ctx, func(txCtx context.Context) error {
				escaped = txCtx
				if _, err := TxQueries(txCtx); err != nil {
					return err
				}
				if tc.rollback {
					return sentinel
				}
				return nil
			})
			if tc.rollback && !errors.Is(err, sentinel) {
				t.Fatalf("rollback error=%v", err)
			}
			if !tc.rollback && err != nil {
				t.Fatalf("commit error=%v", err)
			}
			if q, err := TxQueries(escaped); q != nil || !errors.Is(err, ErrTransactionClosed) {
				t.Fatalf("escaped executor q=%p err=%v", q, err)
			}
			nested := WithTx(escaped, func(context.Context) error {
				t.Fatal("expired context executed transaction callback")
				return nil
			})
			if !errors.Is(nested, ErrTransactionClosed) {
				t.Fatalf("expired nested transaction err=%v", nested)
			}
			if owner.transactions != 1 || owner.committed == tc.rollback {
				t.Fatalf("owner=%+v rollback=%v", owner, tc.rollback)
			}
		})
	}
}

func TestTransactionContextExpiresAfterPanic(t *testing.T) {
	owner := &contextTestStore{}
	ctx := WithStore(context.Background(), owner)
	var escaped context.Context
	func() {
		defer func() {
			if got := recover(); got != "owner panic" {
				t.Fatalf("recovered=%v", got)
			}
		}()
		_ = WithTx(ctx, func(txCtx context.Context) error {
			escaped = txCtx
			panic("owner panic")
		})
	}()
	if escaped == nil {
		t.Fatal("callback never provided context")
	}
	if q, err := TxQueries(escaped); q != nil || !errors.Is(err, ErrTransactionClosed) {
		t.Fatalf("panic leaked transaction executor q=%p err=%v", q, err)
	}
	if owner.committed {
		t.Fatal("panicked transaction committed")
	}
}

// contextTxOnly deliberately does not embed Store or implement Querier.
// Context-first transaction callbacks must work with narrow transaction
// capabilities without inventing another transaction entrypoint.
type contextTxOnly struct {
	owner *contextTestStore
}

func (s contextTxOnly) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	return s.owner.ExecTx(ctx, fn)
}

func TestWithTxNarrowTransactionerUsesOneOwner(t *testing.T) {
	owner := &contextTestStore{}
	ctx := WithStore(context.Background(), contextTxOnly{owner: owner})

	if _, err := StoreFromContext(ctx); !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("transaction-only capability passed as full query Store: %v", err)
	}
	result, err := WithTx2(ctx, func(txCtx context.Context) (int, error) {
		first, err := TxQueries(txCtx)
		if err != nil {
			return 0, err
		}
		inner, err := WithTx2(txCtx, func(nested context.Context) (int, error) {
			q, err := TxQueries(nested)
			if err != nil {
				return 0, err
			}
			if q != first {
				t.Fatal("narrow nested transaction acquired another executor")
			}
			return 42, nil
		})
		return inner, err
	})
	if err != nil || result != 42 || !owner.committed || owner.transactions != 1 {
		t.Fatalf("nested result=%d err=%v owner=%+v", result, err, owner)
	}
}
