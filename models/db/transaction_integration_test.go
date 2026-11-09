package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The value returned by WithTxResult must describe committed PostgreSQL rows,
// not merely a successful callback. Runs only against disposable migrated DB.
func TestWithTxResultPostgresAtomicity(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory for database writes")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)

	var rolledBackID int64
	expectedErr := errors.New("abort after inserted Party")
	result, err := WithTxResult(ctx, store, func(q *Queries) (int64, error) {
		id, err := q.CreateParty(ctx)
		if err != nil {
			return 0, err
		}
		rolledBackID = id
		return id, expectedErr
	})
	if result != 0 || !errors.Is(err, expectedErr) {
		t.Fatalf("failed transaction published value=%d err=%v", result, err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM parties WHERE id=$1", rolledBackID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed transaction committed an unpaired Party")
	}

	committedID, err := WithTxResult(ctx, store, func(q *Queries) (int64, error) {
		return CreatePersonWithParty(ctx, q)
	})
	if err != nil || committedID <= 0 {
		t.Fatalf("commit result=%d err=%v", committedID, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM persons WHERE party_id=$1", committedID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("committed Person missing: party=%d count=%d", committedID, count)
	}
}

// TestWithTxNestedPostgresAtomicity verifies the same SQLC transaction is
// reused across nested services and that an ignored inner error still forces
// the outer PostgreSQL transaction to roll back. It uses the disposable DB.
func TestWithTxNestedPostgresAtomicity(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory for database writes")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write outside bdspro_test")
	}
	base, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(base, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(base); err != nil {
		t.Fatal(err)
	}
	ctx := WithStore(base, NewStore(pool))

	t.Run("one transaction commits nested changes", func(t *testing.T) {
		var created int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			outer, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			created, err = WithTx2(txCtx, func(inner context.Context) (int64, error) {
				nested, err := TxQueries(inner)
				if err != nil {
					return 0, err
				}
				if nested != outer {
					t.Fatal("nested SQLC executor differed from enclosing transaction")
				}
				return CreatePersonWithParty(inner, nested)
			})
			if err != nil {
				return err
			}
			// WithTx2 succeeded inside an uncommitted parent. Its returned
			// value is provisional until the parent WithTx commits.
			var visible int
			if err := pool.QueryRow(txCtx, "SELECT count(*) FROM persons WHERE party_id=$1", created).Scan(&visible); err != nil {
				return err
			}
			if visible != 0 {
				t.Fatal("nested writes were committed before outer transaction")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		var visible int
		if err := pool.QueryRow(base, "SELECT count(*) FROM persons WHERE party_id=$1", created).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		if visible != 1 {
			t.Fatalf("outer commit did not publish nested person: %d", visible)
		}
	})

	t.Run("ignored nested error poisons outer commit", func(t *testing.T) {
		sentinel := errors.New("nested operation rejected")
		var uncommittedID int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			q, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			uncommittedID, err = q.CreateParty(txCtx)
			if err != nil {
				return err
			}
			innerErr := WithTx(txCtx, func(inner context.Context) error {
				nested, err := TxQueries(inner)
				if err != nil {
					return err
				}
				if _, err := nested.CreateParty(inner); err != nil {
					return err
				}
				return sentinel
			})
			if !errors.Is(innerErr, sentinel) {
				return innerErr
			}
			// Simulate a buggy service that ignores the inner error.
			return nil
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("outer transaction committed despite nested failure: %v", err)
		}
		var count int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", uncommittedID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("ignored nested failure committed PostgreSQL data")
		}
	})
}
