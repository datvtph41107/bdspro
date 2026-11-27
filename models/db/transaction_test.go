package db

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Called within the dedicated engine lifecycle test, while the process pool
// is running. Only disposable PostgreSQL is supplied by that test.
func verifyTransactionContract(t *testing.T, pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	table := "req053_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(ctx, "CREATE TABLE "+table+" (id integer PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := pool.Exec(cleanup, "DROP TABLE "+table); err != nil {
			t.Error("fixture cleanup failed", err)
		}
	}()
	insert := func(txCtx context.Context, id int) error {
		conn, err := GetEngine(txCtx)
		if err != nil {
			return err
		}
		_, err = conn.Exec(txCtx, "INSERT INTO "+table+" VALUES ($1)", id)
		return err
	}
	assertRows := func(want int) {
		t.Helper()
		if n := pool.Stat().AcquiredConns(); n != 0 {
			t.Fatalf("transaction leaked %d acquired connections", n)
		}
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != want {
			t.Fatalf("persisted row count=%d want=%d error=%v", count, want, err)
		}
	}
	failure := errors.New("fixture business failure")
	if err := WithTx(ctx, nil); !errors.Is(err, ErrTransactionCallback) {
		t.Fatal("nil callback accepted", err)
	}
	if value, err := WithTx2[int](ctx, nil); value != 0 || !errors.Is(err, ErrTransactionCallback) {
		t.Fatal("nil result callback accepted", err)
	}
	var escaped context.Context
	value, err := WithTx2(ctx, func(txCtx context.Context) (int, error) {
		escaped = txCtx
		if !InTransaction(txCtx) {
			t.Fatal("active transaction not recognized")
		}
		if err := insert(txCtx, 1); err != nil {
			return 0, err
		}
		return WithTx2(txCtx, func(nested context.Context) (int, error) {
			a, err := GetQueries(txCtx)
			if err != nil {
				return 0, err
			}
			b, err := GetQueries(nested)
			if err != nil || a != b {
				t.Fatal("nested call did not reuse queries", err)
			}
			return 42, insert(nested, 2)
		})
	})
	if err != nil || value != 42 {
		t.Fatal("nested success failed", err)
	}
	assertRows(2)
	if InTransaction(escaped) {
		t.Fatal("escaped scope remained active")
	}
	if err := WithTx(escaped, func(context.Context) error {
		t.Fatal("escaped callback executed")
		return nil
	}); !errors.Is(err, ErrTransactionClosed) {
		t.Fatal("escaped transaction accepted", err)
	}
	value, err = WithTx2(ctx, func(txCtx context.Context) (int, error) {
		if err := insert(txCtx, 3); err != nil {
			return 0, err
		}
		return 99, failure
	})
	if value != 0 || !errors.Is(err, failure) {
		t.Fatal("failed callback result leaked", err)
	}
	assertRows(2)
	err = WithTx(ctx, func(txCtx context.Context) error {
		if err := insert(txCtx, 4); err != nil {
			return err
		}
		nestedErr := WithTx(txCtx, func(nested context.Context) error {
			if err := insert(nested, 5); err != nil {
				return err
			}
			return failure
		})
		if !errors.Is(nestedErr, failure) || InTransaction(txCtx) {
			t.Fatal("nested failure did not poison scope", nestedErr)
		}
		return nil // intentionally swallow failure
	})
	if !errors.Is(err, failure) {
		t.Fatal("swallowed failure committed", err)
	}
	assertRows(2)
	err = WithTx(ctx, func(txCtx context.Context) error {
		if err := insert(txCtx, 6); err != nil {
			return err
		}
		func() {
			defer func() {
				if recover() != "nested fixture panic" {
					t.Fatal("nested panic not propagated")
				}
			}()
			_ = WithTx(txCtx, func(context.Context) error { panic("nested fixture panic") })
		}()
		return nil
	})
	if !errors.Is(err, errTransactionPanicked) {
		t.Fatal("recovered nested panic committed", err)
	}
	assertRows(2)
	func() {
		defer func() {
			if recover() != "outer fixture panic" {
				t.Fatal("outer panic not propagated")
			}
		}()
		_ = WithTx(ctx, func(txCtx context.Context) error {
			if err := insert(txCtx, 7); err != nil {
				t.Fatal(err)
			}
			panic("outer fixture panic")
		})
	}()
	assertRows(2)
	parent, stop := context.WithCancel(ctx)
	defer stop()
	err = WithTx(parent, func(txCtx context.Context) error {
		if err := insert(txCtx, 8); err != nil {
			return err
		}
		stop()
		if InTransaction(txCtx) {
			t.Fatal("canceled scope reported active")
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("canceled parent committed", err)
	}
	assertRows(2)
	err = WithTx(ctx, func(txCtx context.Context) error {
		if err := insert(txCtx, 9); err != nil {
			return err
		}
		child, cancel := context.WithCancel(txCtx)
		cancel()
		if err := WithTx(child, func(context.Context) error {
			t.Fatal("canceled child callback executed")
			return nil
		}); !errors.Is(err, context.Canceled) {
			t.Fatal("child cancellation lost", err)
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("swallowed child cancellation committed", err)
	}
	assertRows(2)
	err = WithTx(ctx, func(txCtx context.Context) error {
		if err := insert(txCtx, 10); err != nil {
			return err
		}
		if _, err := WithTx2[int](txCtx, nil); !errors.Is(err, ErrTransactionCallback) {
			t.Fatal("nested nil callback accepted", err)
		}
		return nil
	})
	if !errors.Is(err, ErrTransactionCallback) {
		t.Fatal("swallowed nil callback committed", err)
	}
	assertRows(2)
	err = WithTx(ctx, func(txCtx context.Context) error {
		if err := insert(txCtx, 11); err != nil {
			return err
		}
		if err := insert(txCtx, 1); err == nil {
			t.Fatal("duplicate primary key unexpectedly accepted")
		}
		return nil // PostgreSQL marks the transaction aborted
	})
	if err == nil {
		t.Fatal("aborted PostgreSQL transaction committed")
	}
	assertRows(2)
}
