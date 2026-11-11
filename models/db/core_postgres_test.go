package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"os"
	"testing"
	"time"
)

// TestCorePostgresTransactions exercises the canonical engine and transaction
// behavior against PostgreSQL without importing legacy domain implementations.
func TestCorePostgresTransactions(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("BDS_TEST_DB_URL is required for PostgreSQL transaction integration")
	}
	ctx := context.Background()
	pool, err := Connect(ctx, source)
	if err != nil {
		t.Fatalf("connect disposable PostgreSQL: %v", err)
	}
	defer pool.Close()

	name := fmt.Sprintf("db_core_tx_%d", time.Now().UnixNano())
	table := pgx.Identifier{name}.Sanitize()
	if _, err := pool.Exec(ctx, "CREATE TABLE "+table+" (value integer NOT NULL)"); err != nil {
		t.Fatalf("create isolated transaction table: %v", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "DROP TABLE "+table); err != nil {
			t.Errorf("drop isolated transaction table: %v", err)
		}
	}()

	defaultEngine.mu.Lock()
	if defaultEngine.value != nil || defaultEngine.closed {
		defaultEngine.mu.Unlock()
		t.Fatal("test requires a fresh database engine; do not share process state")
	}
	defaultEngine.value = &engine{pool: pool, queries: New(pool)}
	defaultEngine.mu.Unlock()
	defer func() {
		defaultEngine.mu.Lock()
		defaultEngine.value = nil
		defaultEngine.mu.Unlock()
	}()

	count := func() int {
		t.Helper()
		var total int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&total); err != nil {
			t.Fatalf("count committed rows: %v", err)
		}
		return total
	}

	err = WithTx(ctx, func(txCtx context.Context) error {
		q, err := GetQueries(txCtx)
		if err != nil {
			return err
		}
		scope := txCtx.Value(transactionContextKey{}).(*transactionScope)
		if q != scope.queries {
			t.Fatal("GetQueries did not resolve transaction-scoped SQLC executor")
		}
		var missing int
		scanErr := scope.tx.QueryRow(
			txCtx, "SELECT 1 WHERE FALSE",
		).Scan(&missing)
		if !errors.Is(scanErr, pgx.ErrNoRows) {
			t.Fatalf("expected normal no-rows lookup: %v", scanErr)
		}
		if err := scope.failure(); err != nil {
			t.Fatalf("handled pgx.ErrNoRows poisoned transaction: %v", err)
		}
		if _, err := scope.tx.Exec(txCtx, "INSERT INTO "+table+" VALUES (1)"); err != nil {
			return err
		}
		return WithTx(txCtx, func(innerCtx context.Context) error {
			if innerCtx.Value(transactionContextKey{}) != scope {
				t.Fatal("nested WithTx opened a different transaction")
			}
			_, err := scope.tx.Exec(innerCtx, "INSERT INTO "+table+" VALUES (2)")
			return err
		})
	})
	if err != nil || count() != 2 {
		t.Fatalf("commit/nested reuse failed: err=%v committed=%d", err, count())
	}

	cause := errors.New("abort mutation")
	err = WithTx(ctx, func(txCtx context.Context) error {
		scope := txCtx.Value(transactionContextKey{}).(*transactionScope)
		if _, err := scope.tx.Exec(txCtx, "INSERT INTO "+table+" VALUES (3)"); err != nil {
			return err
		}
		return cause
	})
	if !errors.Is(err, cause) || count() != 2 {
		t.Fatalf("failed callback did not rollback: err=%v committed=%d", err, count())
	}

	err = WithTx(ctx, func(txCtx context.Context) error {
		scope := txCtx.Value(transactionContextKey{}).(*transactionScope)
		if _, err := scope.tx.Exec(txCtx, "INSERT INTO "+table+" VALUES (4)"); err != nil {
			return err
		}
		_ = WithTx(txCtx, func(context.Context) error { return cause })
		return nil // Outer caller ignores nested failure.
	})
	if !errors.Is(err, cause) || count() != 2 {
		t.Fatalf("ignored nested failure committed writes: err=%v committed=%d", err, count())
	}
}
