package organization_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This fixture directs the whole model transaction to one known PostgreSQL
// backend. It does not alter production transaction boundaries.
type creatorTxOnConn struct {
	db.Store
	conn *pgxpool.Conn
}

func (s creatorTxOnConn) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// An account whose delete transaction already owns the row lock must not
// create a new organization using a stale PersonID.
func TestOrganizationCreationWaitsForAccountDeletion(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires migrated disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing DB writes without BDS_TEST_MODE=disposable")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing DB writes outside bdspro_test")
	}
	cfg.MaxConns = max(cfg.MaxConns, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	store := db.NewStore(pool)
	actor := makeActor(t, ctx, store)
	name := "Creator lock " + uuid.NewString()

	deleter, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer deleter.Release()
	writer, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Release()

	deleterPID := int32(deleter.Conn().PgConn().PID())
	writerPID := int32(writer.Conn().PgConn().PID())
	tx, err := deleter.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "DELETE FROM accounts WHERE id=$1", actor.ID)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("expected one staged account deletion: %v %d", err, tag.RowsAffected())
	}

	pinned := creatorTxOnConn{Store: store, conn: writer}
	type outcome struct {
		organizationID int64
		err            error
	}
	done := make(chan outcome, 1)
	go func() {
		created, err := organization.CreateOrganization(ctx, pinned, actor.ID, name)
		if err != nil {
			done <- outcome{err: err}
			return
		}
		done <- outcome{organizationID: created.ID}
	}()

	// A real PostgreSQL backend lock is the acceptance condition.
	waiting := false
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for !waiting {
		select {
		case got := <-done:
			t.Fatalf("creation completed before account delete committed: %+v", got)
		case <-tick.C:
			if err := pool.QueryRow(ctx,
				"SELECT $1::integer = ANY(pg_blocking_pids($2::integer))",
				deleterPID, writerPID,
			).Scan(&waiting); err != nil {
				t.Fatal(err)
			}
		case <-deadline.C:
			t.Fatal("creator did not wait for the account deletion")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got.organizationID != 0 || !errors.Is(got.err, account_model.ErrAccountUnavailable) {
			t.Fatalf("deleted account was allowed to own organization: %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("creator stayed blocked after account deletion committed")
	}

	var count int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM organizations WHERE name=$1", name,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("deleted account produced an orphaned organization")
	}
}
