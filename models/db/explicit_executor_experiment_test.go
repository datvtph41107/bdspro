package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// trialExplicitScope is an R-SQL01 experiment, not a production DB Core API.
// All operations are deliberately single-goroutine; pgx.Tx isn't safe for
// concurrent use. A caller receives a transaction-bound object, never a
// context-dependent resolver that falls back to the process pool.
type trialExplicitScope struct {
	tx      pgx.Tx
	queries *Queries
	failure error
	closed  bool
}

func (s *trialExplicitScope) check() error {
	if s.closed {
		return ErrTransactionClosed
	}
	return s.failure
}

func (s *trialExplicitScope) exec(ctx context.Context, sql string) error {
	if err := s.check(); err != nil {
		return err
	}
	_, err := s.tx.Exec(ctx, sql)
	return err
}

func (s *trialExplicitScope) nested(ctx context.Context, fn func(*trialExplicitScope) error) (err error) {
	if err := s.check(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		s.failure = err
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			s.failure = errTransactionPanicked
			panic(recovered)
		}
		if err != nil && s.failure == nil {
			s.failure = err
		}
		if ctxErr := ctx.Err(); ctxErr != nil && s.failure == nil {
			s.failure = ctxErr
		}
		if err == nil {
			err = s.failure
		}
	}()
	return fn(s)
}

func runTrialExplicitTx(ctx context.Context, pool *pgxpool.Pool, fn func(*trialExplicitScope) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	s := &trialExplicitScope{tx: tx, queries: New(tx)}
	defer func() {
		s.closed = true
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	if err := fn(s); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.check(); err != nil {
		return err
	}
	s.closed = true
	return tx.Commit(ctx)
}

// TestTrialExplicitExecutorPostgres is a differential feasibility test for
// explicit transaction ownership. It does NOT establish that this architecture
// is faster/better than BDSPro's production context-bound WithTx.
func TestTrialExplicitExecutorPostgres(t *testing.T) {
	if os.Getenv("BDS_TEST_MODE") != "disposable" || os.Getenv("BDS_TEST_DB_URL") == "" {
		t.Skip("only run on an explicitly disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := Connect(ctx, os.Getenv("BDS_TEST_DB_URL"))
	if err != nil {
		t.Fatalf("connect disposable Postgres: %v", err)
	}
	defer pool.Close()

	table := pgx.Identifier{fmt.Sprintf("rsql01_explicit_%d", time.Now().UnixNano())}.Sanitize()
	if _, err := pool.Exec(ctx, "CREATE TABLE "+table+" (id bigint PRIMARY KEY)"); err != nil {
		t.Fatalf("create scratch table: %v", err)
	}
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if _, err := pool.Exec(cleanupCtx, "DROP TABLE "+table); err != nil {
			t.Errorf("drop scratch table: %v", err)
		}
	}()

	t.Run("commit", func(t *testing.T) {
		var leaked *trialExplicitScope
		err := runTrialExplicitTx(ctx, pool, func(s *trialExplicitScope) error {
			leaked = s
			if s.queries == nil {
				return errors.New("SQLC Queries must bind explicitly to tx")
			}
			return s.exec(ctx, "INSERT INTO "+table+" (id) VALUES (10)")
		})
		if err != nil {
			t.Fatalf("explicit transaction commit: %v", err)
		}
		if err := leaked.exec(ctx, "SELECT 1"); !errors.Is(err, ErrTransactionClosed) {
			t.Fatalf("scope reused after commit: %v; want ErrTransactionClosed", err)
		}
	})

	t.Run("rollback", func(t *testing.T) {
		wanted := errors.New("abort explicit transaction")
		err := runTrialExplicitTx(ctx, pool, func(s *trialExplicitScope) error {
			if err := s.exec(ctx, "INSERT INTO "+table+" (id) VALUES (11)"); err != nil {
				return err
			}
			return wanted
		})
		if !errors.Is(err, wanted) {
			t.Fatalf("rollback error = %v; want %v", err, wanted)
		}
	})

	t.Run("ignored_nested_error", func(t *testing.T) {
		wanted := errors.New("nested operation failed")
		err := runTrialExplicitTx(ctx, pool, func(s *trialExplicitScope) error {
			if err := s.exec(ctx, "INSERT INTO "+table+" (id) VALUES (12)"); err != nil {
				return err
			}
			_ = s.nested(ctx, func(*trialExplicitScope) error { return wanted })
			return nil
		})
		if !errors.Is(err, wanted) {
			t.Fatalf("ignored nested error = %v; want sticky %v", err, wanted)
		}
	})

	t.Run("canceled_context", func(t *testing.T) {
		innerCtx, stop := context.WithCancel(ctx)
		defer stop()
		err := runTrialExplicitTx(innerCtx, pool, func(s *trialExplicitScope) error {
			if err := s.exec(innerCtx, "INSERT INTO "+table+" (id) VALUES (13)"); err != nil {
				return err
			}
			stop()
			return nil
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled transaction = %v; want context.Canceled", err)
		}
	})

	rows, err := pool.Query(ctx, "SELECT id FROM "+table+" ORDER BY id")
	if err != nil {
		t.Fatalf("query visible rows: %v", err)
	}
	defer rows.Close()
	var got []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		got = append(got, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate visible rows: %v", err)
	}
	if len(got) != 1 || got[0] != 10 {
		t.Fatalf("visible ids = %v; want only committed id 10", got)
	}
}
