//go:build integration

package party

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Only use a dedicated, disposable *_test database with 000001 applied.
// Identity rows are deliberately NOT deleted during test cleanup: deletion
// is prohibited by the business invariant. Destroy the disposable DB after CI.
func coreTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("CORE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("CORE_TEST_DATABASE_URL must identify a disposable database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	var name string
	if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&name); err != nil {
		t.Fatalf("database identity: %v", err)
	}
	if !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to run core integration tests against %q", name)
	}
	var ready bool
	if err := pool.QueryRow(ctx, `SELECT
		to_regclass('public.parties') IS NOT NULL
		AND to_regclass('public.persons') IS NOT NULL
		AND to_regclass('public.organizations') IS NOT NULL`).Scan(&ready); err != nil || !ready {
		t.Fatalf("000001 baseline missing: ready=%v err=%v", ready, err)
	}
	return pool
}

func expectSQLState(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("got %v; want PostgreSQL SQLSTATE %s", err, code)
	}
}

func assertKind(t *testing.T, pool *pgxpool.Pool, id int64, person, org int) {
	t.Helper()
	var p, o int
	err := pool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM persons WHERE party_id = $1),
		(SELECT count(*) FROM organizations WHERE party_id = $1)`, id).Scan(&p, &o)
	if err != nil || p != person || o != org {
		t.Fatalf("party %d got persons=%d organizations=%d err=%v; expected %d/%d", id, p, o, err, person, org)
	}
}

func TestCreatesExactlyOneImmutableKind(t *testing.T) {
	pool := coreTestPool(t)
	ctx := context.Background()
	personID, err := CreatePerson(ctx, pool)
	if err != nil { t.Fatal(err) }
	orgID, err := CreateOrganization(ctx, pool, "ABC")
	if err != nil { t.Fatal(err) }
	if personID == orgID { t.Fatal("distinct parties share identity") }
	assertKind(t, pool, personID, 1, 0)
	assertKind(t, pool, orgID, 0, 1)
}

func TestCommitBarePartyRejected(t *testing.T) {
	pool := coreTestPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	if err := tx.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&id); err != nil { t.Fatal(err) }
	expectSQLState(t, tx.Commit(ctx), "23514")
	var found bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM parties WHERE id=$1)", id).Scan(&found); err != nil { t.Fatal(err) }
	if found { t.Fatalf("bare Party %d was committed", id) }
}

func TestAddingOppositeSubtypeRejected(t *testing.T) {
	pool := coreTestPool(t)
	ctx := context.Background()
	id, err := CreatePerson(ctx, pool)
	if err != nil { t.Fatal(err) }
	_, err = pool.Exec(ctx, "INSERT INTO organizations(party_id,name) VALUES ($1,'ABC')", id)
	expectSQLState(t, err, "23514")
	assertKind(t, pool, id, 1, 0)
}

func TestDualSubtypeBeforeCommitRejected(t *testing.T) {
	pool := coreTestPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	if err := tx.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&id); err != nil { t.Fatal(err) }
	if _, err := tx.Exec(ctx, "INSERT INTO persons(party_id) VALUES ($1)", id); err != nil { t.Fatal(err) }
	_, err = tx.Exec(ctx, "INSERT INTO organizations(party_id,name) VALUES ($1,'ABC')", id)
	expectSQLState(t, err, "23514")
	if err := tx.Rollback(ctx); err != nil { t.Fatal(err) }
	var found bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM parties WHERE id=$1)", id).Scan(&found); err != nil { t.Fatal(err) }
	if found { t.Fatalf("failed transaction persisted party %d", id) }
}

func TestSubtypeCannotBeDeletedOrChanged(t *testing.T) {
	pool := coreTestPool(t)
	ctx := context.Background()
	id, err := CreatePerson(ctx, pool)
	if err != nil { t.Fatal(err) }
	otherID, err := CreatePerson(ctx, pool)
	if err != nil { t.Fatal(err) }
	_, err = pool.Exec(ctx, "DELETE FROM persons WHERE party_id=$1", id)
	expectSQLState(t, err, "23514")
	_, err = pool.Exec(ctx, "UPDATE persons SET party_id=$2 WHERE party_id=$1", id, otherID)
	expectSQLState(t, err, "23514")
	_, err = pool.Exec(ctx, "DELETE FROM parties WHERE id=$1", id)
	// ON DELETE RESTRICT uses SQLSTATE 23001 (restrict_violation), not 23503.
	expectSQLState(t, err, "23001")
	assertKind(t, pool, id, 1, 0)
}

func TestEvenOneTransactionCannotSwitchKind(t *testing.T) {
	pool := coreTestPool(t)
	ctx := context.Background()
	id, err := CreatePerson(ctx, pool)
	if err != nil { t.Fatal(err) }
	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, "DELETE FROM persons WHERE party_id=$1", id)
	expectSQLState(t, err, "23514")
	if err := tx.Rollback(ctx); err != nil { t.Fatal(err) }
	assertKind(t, pool, id, 1, 0)
}

func TestCompetingSubtypeWriteWhilePartyLocked(t *testing.T) {
	pool := coreTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := CreatePerson(ctx, pool)
	if err != nil { t.Fatal(err) }
	locker, err := pool.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer func() { _ = locker.Rollback(ctx) }()
	if _, err := locker.Exec(ctx, "SELECT 1 FROM parties WHERE id=$1 FOR UPDATE", id); err != nil { t.Fatal(err) }

	start := make(chan struct{})
	competing := make(chan error, 1)
	go func() {
		<-start
		_, e := pool.Exec(ctx, "INSERT INTO organizations(party_id,name) VALUES ($1,'XYZ')", id)
		competing <- e
	}()
	close(start)
	// Release parent lock, then competing statement must observe original kind.
	if err := locker.Commit(ctx); err != nil { t.Fatal(err) }
	select {
	case err := <-competing:
		expectSQLState(t, err, "23514")
	case <-ctx.Done():
		t.Fatalf("competing write did not finish: %v", ctx.Err())
	}
	assertKind(t, pool, id, 1, 0)
}
