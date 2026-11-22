package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestTransactionPostgresExecutorEscape demonstrates the executor-selection
// boundary using the real BDSPro GetEngine/GetQueries/WithTx implementations.
// It deliberately writes via the original context inside a tx callback and
// verifies that the pool write survives the outer rollback.
//
// This test only runs against an explicitly disposable PostgreSQL database;
// it never changes the application's migration ledger or production tables.
func TestTransactionPostgresExecutorEscape(t *testing.T) {
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Skip("requires BDS_TEST_MODE=disposable")
	}
	databaseURL := os.Getenv("BDS_TEST_DB_URL")
	if databaseURL == "" {
		t.Skip("requires BDS_TEST_DB_URL for disposable PostgreSQL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect disposable database: %v", err)
	}
	defer pool.Close()

	// The global engine is only replaced within this non-parallel package test.
	// Retain its original state and restore it before closing the test pool.
	defaultEngine.mu.Lock()
	previous, previouslyClosed := defaultEngine.value, defaultEngine.closed
	if previous != nil || previouslyClosed {
		defaultEngine.mu.Unlock()
		t.Fatal("test requires uninitialized DB core; refusing to replace an active engine")
	}
	defaultEngine.value = &engine{pool: pool, queries: New(pool)}
	defaultEngine.mu.Unlock()
	defer func() {
		defaultEngine.mu.Lock()
		defaultEngine.value = previous
		defaultEngine.closed = previouslyClosed
		defaultEngine.mu.Unlock()
	}()

	// Unique unqualified table name avoids interfering with parallel packages.
	// pgx.Identifier safely quotes the dynamically generated identifier.
	tableName := fmt.Sprintf("rsql01_probe_%d", time.Now().UnixNano())
	table := pgx.Identifier{tableName}.Sanitize()
	if _, err := pool.Exec(ctx, "CREATE TABLE "+table+" (id bigint PRIMARY KEY)"); err != nil {
		t.Fatalf("create disposable probe table: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, "DROP TABLE "+table); err != nil {
			t.Errorf("remove disposable probe table: %v", err)
		}
	}()

	forcedRollback := errors.New("forced rollback")
	err = WithTx(ctx, func(txCtx context.Context) error {
		inTx, err := GetEngine(txCtx)
		if err != nil {
			return err
		}
		outer, err := GetEngine(ctx)
		if err != nil {
			return err
		}
		txQueries, err := GetQueries(txCtx)
		if err != nil {
			return err
		}
		poolQueries, err := GetQueries(ctx)
		if err != nil {
			return err
		}
		if txQueries == poolQueries {
			return errors.New("transaction and process pool unexpectedly share one SQLC executor")
		}
		if _, err := inTx.Exec(txCtx, "INSERT INTO "+table+" (id) VALUES (1)"); err != nil {
			return fmt.Errorf("transaction write: %w", err)
		}
		// Deliberate misuse: the ORIGINAL context resolves the global pool.
		if _, err := outer.Exec(ctx, "INSERT INTO "+table+" (id) VALUES (2)"); err != nil {
			return fmt.Errorf("pool write: %w", err)
		}
		return forcedRollback
	})
	if !errors.Is(err, forcedRollback) {
		t.Fatalf("WithTx = %v; want forced rollback", err)
	}

	var count, survivor int64
	if err := pool.QueryRow(ctx, "SELECT count(*), max(id) FROM "+table).Scan(&count, &survivor); err != nil {
		t.Fatalf("inspect committed rows: %v", err)
	}
	if count != 1 || survivor != 2 {
		t.Fatalf("committed rows = (count=%d, max=%d), want only id=2 (pool write)", count, survivor)
	}

	// Now exercise sticky rollback-only with an actual pgx transaction.
	nestedErr := errors.New("nested operation failed")
	err = WithTx(ctx, func(txCtx context.Context) error {
		queries, err := GetEngine(txCtx)
		if err != nil {
			return err
		}
		if _, err := queries.Exec(txCtx, "INSERT INTO "+table+" (id) VALUES (3)"); err != nil {
			return err
		}
		// Caller ignores nested error, but scope must remain rollback-only.
		_ = WithTx(txCtx, func(context.Context) error { return nestedErr })
		return nil
	})
	if !errors.Is(err, nestedErr) {
		t.Fatalf("outer WithTx = %v; want sticky nested error", err)
	}

	var nestedCommitted bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM "+table+" WHERE id=3)").Scan(&nestedCommitted); err != nil {
		t.Fatalf("inspect nested rollback: %v", err)
	}
	if nestedCommitted {
		t.Fatal("ignored nested transaction error was committed")
	}
}
