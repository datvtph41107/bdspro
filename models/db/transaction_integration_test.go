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
