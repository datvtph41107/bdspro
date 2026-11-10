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
