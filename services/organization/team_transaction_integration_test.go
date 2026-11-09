package organization

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// All tests operate only on a disposable migrated bdspro_test database.
func membershipTestDB(t *testing.T) (context.Context, *pgxpool.Pool, *db.SQLStore) {
	t.Helper()
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing non-disposable database writes")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing a non-test PostgreSQL database")
	}
	cfg.MaxConns = max(cfg.MaxConns, 6)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, db.NewStore(pool)
}

type teamCase struct {
	accountID, orgID, teamID, recipientPersonID int64
	sessionID                                   uuid.UUID
}

func newTeamCase(t *testing.T, ctx context.Context, store db.Store) teamCase {
	t.Helper()
	newAccount := func() db.Account {
		t.Helper()
		a, err := db.WithTxResult(ctx, store, func(q *db.Queries) (db.Account, error) {
			personID, err := db.CreatePersonWithParty(ctx, q)
			if err != nil {
				return db.Account{}, err
			}
			account, err := q.CreateAccount(ctx, db.CreateAccountParams{
				PersonID: personID, Email: uuid.NewString() + "@example.test",
			})
			if err != nil {
				return db.Account{}, err
			}
			return account, q.CreateAccountSettings(ctx, account.ID)
		})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	owner, recipient := newAccount(), newAccount()
	org, err := organization_model.CreateOrganization(ctx, store, owner.ID, "session team "+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := organization_model.AddOrgUser(ctx, store, owner.ID, org.PartyID, recipient.PersonID); err != nil {
		t.Fatal(err)
	}
	team, err := organization_model.CreateTeam(ctx, store, organization_model.CreateTeamOptions{
		OrganizationID: org.PartyID, ActorAccountID: owner.ID, Name: "Backend",
	})
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	if _, err := store.CreateSession(ctx, db.CreateSessionParams{
		ID: sessionID, AccountID: owner.ID, RefreshToken: "test-" + uuid.NewString(),
		UserAgent: "g1-2c4", ClientIp: "203.0.113.22", ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	return teamCase{accountID: owner.ID, orgID: org.PartyID, teamID: team.ID,
		recipientPersonID: recipient.PersonID, sessionID: sessionID}
}

// Pin the entire service transaction to a known PostgreSQL backend.
type teamPinnedStore struct {
	db.Store
	conn *pgxpool.Conn
}

func (s teamPinnedStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
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

func waitTeamBlocking(t *testing.T, ctx context.Context, pool *pgxpool.Pool, blocker, waiter *pgxpool.Conn, done <-chan error) {
	t.Helper()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case err := <-done:
			t.Fatalf("operation returned before lock released: %v", err)
		case <-tick.C:
			var blocked bool
			if err := pool.QueryRow(ctx,
				"SELECT $1::integer=ANY(pg_blocking_pids($2::integer))",
				int32(blocker.Conn().PgConn().PID()), int32(waiter.Conn().PgConn().PID()),
			).Scan(&blocked); err != nil {
				t.Fatal(err)
			}
			if blocked {
				return
			}
		case <-timeout.C:
			t.Fatal("expected PostgreSQL lock wait was not observed")
		}
	}
}

func teamMembershipMutation(ctx context.Context, svc *TeamService, data teamCase, kind string) error {
	// Exercise new context-scoped service operations on the same pinned SQL
	// connection as the legacy receiver; lock-order assertions remain intact.
	ctx = db.WithStore(auth_model.WithActor(ctx, auth_model.Actor{
		AccountID: data.accountID, SessionID: data.sessionID,
	}), svc.store)
	if kind == "add" {
		_, err := AddTeamMember(ctx, data.teamID, data.recipientPersonID)
		return err
	}
	return RemoveTeamMember(ctx, data.teamID, data.recipientPersonID)
}

func membershipCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, data teamCase) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM team_members WHERE team_id=$1 AND person_id=$2",
		data.teamID, data.recipientPersonID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// A separate transaction revokes the acting session first. The service must
// wait for its session row and then reject WITHOUT modifying membership.
func TestTeamMembershipCannotOutliveActingSession(t *testing.T) {
	for _, kind := range []string{"add", "remove"} {
		t.Run(kind, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			data := newTeamCase(t, ctx, store)
			if kind == "remove" {
				if _, err := organization_model.AddTeamUser(ctx, store, data.accountID, data.teamID, data.recipientPersonID); err != nil {
					t.Fatal(err)
				}
			}
			revoker, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer revoker.Release()
			writer, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Release()
			tx, err := revoker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if _, err := tx.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", data.sessionID); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- teamMembershipMutation(ctx, NewTeamService(teamPinnedStore{Store: store, conn: writer}), data, kind)
			}()
			waitTeamBlocking(t, ctx, pool, revoker, writer, done)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if !errors.Is(err, auth_model.ErrSessionUnavailable) {
					t.Fatalf("revoked actor unexpectedly authorized: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("membership transaction failed to resume")
			}
			expected := 0
			if kind == "remove" {
				expected = 1
			}
			if got := membershipCount(t, ctx, pool, data); got != expected {
				t.Fatalf("stale actor modified membership: got=%d expected=%d", got, expected)
			}
		})
	}
}

// Reverse race: stage the member write after it has acquired its session
// lock but before owner-scoped SQLC can get FOR SHARE OF membership. Revoke
// must then wait on the writer, which is allowed to commit first.
func TestTeamMembershipCommitPrecedesRevocation(t *testing.T) {
	for _, kind := range []string{"add", "remove"} {
		t.Run(kind, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			data := newTeamCase(t, ctx, store)
			if kind == "remove" {
				if _, err := organization_model.AddTeamUser(ctx, store, data.accountID, data.teamID, data.recipientPersonID); err != nil {
					t.Fatal(err)
				}
			}
			blocker, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer blocker.Release()
			writer, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Release()
			revoker, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer revoker.Release()
			tx, err := blocker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			var membershipID int64
			if err := tx.QueryRow(ctx, `SELECT m.id FROM organization_members m
				JOIN accounts a ON a.person_id=m.person_id
				WHERE a.id=$1 AND m.organization_id=$2 AND m.ended_at IS NULL
				FOR UPDATE OF m`, data.accountID, data.orgID).Scan(&membershipID); err != nil {
				t.Fatal(err)
			}
			writeDone := make(chan error, 1)
			go func() {
				writeDone <- teamMembershipMutation(ctx, NewTeamService(teamPinnedStore{Store: store, conn: writer}), data, kind)
			}()
			waitTeamBlocking(t, ctx, pool, blocker, writer, writeDone)
			revokeDone := make(chan error, 1)
			go func() {
				revokeDone <- auth_model.RevokeOwnedSession(ctx, teamPinnedStore{Store: store, conn: revoker}, data.accountID, data.sessionID, data.sessionID)
			}()
			waitTeamBlocking(t, ctx, pool, writer, revoker, revokeDone)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-writeDone:
				if err != nil {
					t.Fatalf("valid write did not commit first: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("writer did not commit after lock release")
			}
			select {
			case err := <-revokeDone:
				if err != nil {
					t.Fatalf("session revoke failed after write: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("revoker did not complete after write")
			}
			expected := 1
			if kind == "remove" {
				expected = 0
			}
			if got := membershipCount(t, ctx, pool, data); got != expected {
				t.Fatalf("unexpected committed membership after %s: got=%d expected=%d", kind, got, expected)
			}
		})
	}
}

func TestTeamMembershipServiceRequiresSignedActorAndOwner(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	data := newTeamCase(t, ctx, store)
	svc := NewTeamService(store)
	if _, err := svc.AddTeamUser(ctx, data.accountID, uuid.Nil, data.teamID, data.recipientPersonID); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("nil session authorized: %v", err)
	}
	if _, err := svc.AddTeamUser(ctx, data.accountID+1, data.sessionID, data.teamID, data.recipientPersonID); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("mismatched session/account authorized: %v", err)
	}
	if _, err := svc.AddTeamUser(ctx, data.accountID, data.sessionID, data.teamID, data.recipientPersonID); err != nil {
		t.Fatalf("valid member addition rejected: %v", err)
	}
	if _, err := svc.AddTeamUser(ctx, data.accountID, data.sessionID, data.teamID, data.recipientPersonID); !errors.Is(err, organization_model.ErrTeamMemberExists) {
		t.Fatalf("duplicate membership not rejected: %v", err)
	}
	if err := svc.RemoveTeamUser(ctx, data.accountID, data.sessionID, data.teamID, data.recipientPersonID); err != nil {
		t.Fatalf("valid removal failed: %v", err)
	}
	if n := membershipCount(t, ctx, pool, data); n != 0 {
		t.Fatalf("remove did not commit: %d", n)
	}
	if _, err := svc.AddTeamUser(ctx, data.accountID, data.sessionID, 9223372036854775807, data.recipientPersonID); !errors.Is(err, organization_model.ErrTeamOwnerRequired) {
		t.Fatalf("missing team escaped owner predicate: %v", err)
	}
}
