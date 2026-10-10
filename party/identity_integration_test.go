//go:build integration

package party

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These are real PostgreSQL tests, never an in-memory DB approximation.
// Run ONLY against a disposable database ending in _test, with migration 000006 applied.
func coreTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("CORE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("CORE_TEST_DATABASE_URL is required for core integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	var name string
	if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&name); err != nil {
		t.Fatalf("read database name: %v", err)
	}
	if !strings.HasSuffix(name, "_test") {
		t.Fatalf("unsafe test database %q: must end in _test", name)
	}
	var ready bool
	if err := pool.QueryRow(ctx,
		"SELECT to_regclass('parties') IS NOT NULL AND to_regclass('persons') IS NOT NULL AND to_regclass('organizations') IS NOT NULL",
	).Scan(&ready); err != nil || !ready {
		t.Fatalf("migration 000006 is required: ready=%v err=%v", ready, err)
	}
	return pool
}

func requireSQLState(t *testing.T, err error, state string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != state {
		t.Fatalf("got error %v; expected PostgreSQL SQLSTATE %s", err, state)
	}
}

func cleanupParty(t *testing.T, pool *pgxpool.Pool, id int64) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Errorf("cleanup: begin: %v", err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		for _, query := range []string{
			"DELETE FROM persons WHERE party_id = $1",
			"DELETE FROM organizations WHERE party_id = $1",
			"DELETE FROM parties WHERE id = $1",
		} {
			if _, err := tx.Exec(ctx, query, id); err != nil {
				t.Errorf("cleanup party %d: %v", id, err)
				return
			}
		}
		if err := tx.Commit(ctx); err != nil {
			t.Errorf("cleanup commit for party %d: %v", id, err)
		}
	})
}

func assertSubtype(t *testing.T, pool *pgxpool.Pool, id int64, person, organization int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var gotPerson, gotOrg int
	if err := pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM persons WHERE party_id = $1),
		  (SELECT count(*) FROM organizations WHERE party_id = $1)
	`, id).Scan(&gotPerson, &gotOrg); err != nil {
		t.Fatalf("count subtypes: %v", err)
	}
	if gotPerson != person || gotOrg != organization {
		t.Fatalf("party %d has person=%d org=%d; want person=%d org=%d",
			id, gotPerson, gotOrg, person, organization)
	}
}

func TestCreatePersonAndOrganization(t *testing.T) {
	pool := coreTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	personID, err := CreatePerson(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	cleanupParty(t, pool, personID)
	orgID, err := CreateOrganization(ctx, pool, "ABC")
	if err != nil {
		t.Fatal(err)
	}
	cleanupParty(t, pool, orgID)
	if personID == orgID {
		t.Fatal("two separate business identities share an ID")
	}
	assertSubtype(t, pool, personID, 1, 0)
	assertSubtype(t, pool, orgID, 0, 1)
}

func TestCommitWithoutSubtypeIsRejected(t *testing.T) {
	pool := coreTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	if err := tx.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&id); err != nil {
		t.Fatal(err)
	}
	requireSQLState(t, tx.Commit(ctx), "23514")
	var exists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM parties WHERE id=$1)", id).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("rejected party %d persisted", id)
	}
}

func TestCommitWithBothSubtypesIsRejected(t *testing.T) {
	pool := coreTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	if err := tx.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO persons (party_id) VALUES ($1)", id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO organizations (party_id, name) VALUES ($1, 'ABC')", id); err != nil {
		t.Fatal(err)
	}
	requireSQLState(t, tx.Commit(ctx), "23514")
}

func TestDeletingTheOnlySubtypeIsRejected(t *testing.T) {
	pool := coreTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := CreatePerson(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	cleanupParty(t, pool, id)
	_, err = pool.Exec(ctx, "DELETE FROM persons WHERE party_id=$1", id)
	requireSQLState(t, err, "23514")
	assertSubtype(t, pool, id, 1, 0)
}

func TestSwitchSubtypeWithinOneTransaction(t *testing.T) {
	pool := coreTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := CreatePerson(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	cleanupParty(t, pool, id)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "DELETE FROM persons WHERE party_id=$1", id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO organizations (party_id, name) VALUES ($1, 'ABC')", id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("replace subtype within one transaction: %v", err)
	}
	assertSubtype(t, pool, id, 0, 1)
}

func TestConcurrentCompetingSubtypeWritesDoNotCommitBoth(t *testing.T) {
	pool := coreTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	id, err := CreatePerson(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	cleanupParty(t, pool, id)

	// Tx1 converts Person to Organization and holds the parent row lock.
	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx1.Rollback(ctx) }()
	if _, err := tx1.Exec(ctx, "DELETE FROM persons WHERE party_id=$1", id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx1.Exec(ctx, "INSERT INTO organizations (party_id, name) VALUES ($1, 'ABC')", id); err != nil {
		t.Fatal(err)
	}

	competing := make(chan error, 1)
	go func() {
		tx2, err := pool.Begin(ctx)
		if err != nil {
			competing <- err
			return
		}
		defer func() { _ = tx2.Rollback(ctx) }()
		if _, err := tx2.Exec(ctx, "INSERT INTO organizations (party_id, name) VALUES ($1, 'XYZ')", id); err != nil {
			competing <- err
			return
		}
		competing <- tx2.Commit(ctx)
	}()

	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("winner commit: %v", err)
	}
	select {
	case err := <-competing:
		if err == nil {
			t.Fatal("two competing writers both committed a subtype")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			t.Fatalf("unexpected non-Postgres second writer failure: %v", err)
		}
	case <-ctx.Done():
		t.Fatal(fmt.Errorf("competing writer timed out: %w", ctx.Err()))
	}
	assertSubtype(t, pool, id, 0, 1)
}
