package db

import (
	"context"
	"errors"
	"testing"
)

type contextTestStore struct {
	Store
	afterCallback error
	transactions int
	committed bool
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
