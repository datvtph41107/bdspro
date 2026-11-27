package db

import (
	"context"
	"errors"
	"testing"
)

func TestQueryContextRejection(t *testing.T) {
	ctx := context.Background()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	failed := errors.New("fixture transaction failure")
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"nil context", nil, ErrTransactionContext},
		{"canceled context", canceled, context.Canceled},
		{"wrong scope type", context.WithValue(ctx, transactionContextKey{}, "invalid"), ErrTransactionContext},
		{"typed nil scope", context.WithValue(ctx, transactionContextKey{}, (*transactionScope)(nil)), ErrTransactionContext},
		{"closed scope", context.WithValue(ctx, transactionContextKey{}, &transactionScope{closed: true}), ErrTransactionClosed},
		{"failed scope", context.WithValue(ctx, transactionContextKey{}, &transactionScope{err: failed}), failed},
		{"incomplete scope", context.WithValue(ctx, transactionContextKey{}, &transactionScope{}), ErrTransactionContext},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if InTransaction(tc.ctx) {
				t.Fatal("invalid scope reported usable")
			}
			if conn, err := GetEngine(tc.ctx); conn != nil || !errors.Is(err, tc.want) {
				t.Fatal("unexpected raw execution handle", err)
			}
			if q, err := GetQueries(tc.ctx); q != nil || !errors.Is(err, tc.want) {
				t.Fatal("unexpected SQLC query handle", err)
			}
		})
	}
}
