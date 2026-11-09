package db

import (
	"context"
	"errors"
	"testing"
)

type contextTestStore struct {
	Store
	afterCallback error
}

func (s *contextTestStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	if err := fn(&Queries{}); err != nil {
		return err
	}
	return s.afterCallback
}

func TestWithTxValueDoesNotPublishFailedCommit(t *testing.T) {
	ctx := WithStore(context.Background(), &contextTestStore{afterCallback: errors.New("failed commit")})
	value, err := WithTxValue(ctx, func(txCtx context.Context) (int, error) {
		if _, err := TxQueries(txCtx); err != nil {
			t.Fatal(err)
		}
		if err := WithTx(txCtx, func(context.Context) error { return nil }); !errors.Is(err, ErrNestedTransaction) {
			t.Fatalf("nested transaction error=%v", err)
		}
		return 42, nil
	})
	if err == nil || value != 0 {
		t.Fatalf("value=%d err=%v", value, err)
	}
}

func TestWithTxRequiresStore(t *testing.T) {
	err := WithTx(context.Background(), func(context.Context) error { return nil })
	if !errors.Is(err, ErrStoreMissing) {
		t.Fatalf("got %v", err)
	}
}
