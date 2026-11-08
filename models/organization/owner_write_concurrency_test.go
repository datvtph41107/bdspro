package organization_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This test pins the authorized write to one PostgreSQL connection so that
// pg_blocking_pids can establish a real row-lock wait, not a timing guess.
type ownerWriteOnConn struct {
	db.Store
	conn *pgxpool.Conn
}

func (s ownerWriteOnConn) UpdateOrganization(ctx context.Context, arg db.UpdateOrganizationParams) (db.Organization, error) {
	return db.New(s.conn).UpdateOrganization(ctx, arg)
}

func (s ownerWriteOnConn) AddOrganizationMember(ctx context.Context, arg db.AddOrganizationMemberParams) (db.OrganizationMember, error) {
	return db.New(s.conn).AddOrganizationMember(ctx, arg)
}

func TestOwnerRevocationSerializesOrganizationWrites(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires migrated disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes without BDS_TEST_MODE=disposable")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing database writes outside bdspro_test")
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

	makeAccount := func(t *testing.T) db.Account {
		t.Helper()
		personID, err := db.WithTxResult(ctx, store, func(q *db.Queries) (int64, error) {
			return db.CreatePersonWithParty(ctx, q)
		})
		if err != nil {
			t.Fatal(err)
		}
		account, err := store.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: personID,
			Email:    uuid.NewString() + "@example.test",
		})
		if err != nil {
			t.Fatal(err)
		}
		return account
	}

	for _, kind := range []string{"rename", "add_member"} {
		t.Run(kind, func(t *testing.T) {
			owner := makeAccount(t)
			target := makeAccount(t)
			orgName := "Owner lock " + uuid.NewString()
			org, err := organization.CreateOrganization(ctx, store, owner.ID, orgName)
			if err != nil {
				t.Fatal(err)
			}
			members, err := store.ListOrganizationMembersForOwner(ctx, db.ListOrganizationMembersForOwnerParams{
				OrganizationID: org.PartyID,
				ActorAccountID: owner.ID,
			})
			if err != nil || len(members) != 1 {
				t.Fatalf("expected exactly one owner membership: %v %v", members, err)
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
			revokerPID := int32(revoker.Conn().PgConn().PID())
			writerPID := int32(writer.Conn().PgConn().PID())

			tx, err := revoker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if _, err := tx.Exec(ctx,
				"UPDATE organization_members SET ended_at=now(), owner_person_id=NULL WHERE id=$1",
				members[0].ID,
			); err != nil {
				t.Fatal(err)
			}

			pinned := ownerWriteOnConn{Store: store, conn: writer}
			result := make(chan error, 1)
			go func() {
				switch kind {
				case "rename":
					_, err := organization.UpdateOrganization(ctx, pinned, owner.ID, org.PartyID, "unlawful rename")
					result <- err
				case "add_member":
					_, err := organization.AddOrgUser(ctx, pinned, owner.ID, org.PartyID, target.PersonID)
					result <- err
				}
			}()

			// Wait for proof of a real PostgreSQL lock dependency. Merely
			// sleeping is not a valid concurrency acceptance test.
			seenWait := false
			poll := time.NewTicker(20 * time.Millisecond)
			defer poll.Stop()
			timeout := time.NewTimer(5 * time.Second)
			defer timeout.Stop()
			for !seenWait {
				select {
				case outcome := <-result:
					t.Fatalf("owner-authorized %s completed before revocation: %v", kind, outcome)
				case <-poll.C:
					if err := pool.QueryRow(ctx,
						"SELECT $1::integer = ANY(pg_blocking_pids($2::integer))",
						revokerPID, writerPID,
					).Scan(&seenWait); err != nil {
						t.Fatal(err)
					}
				case <-timeout.C:
					t.Fatalf("%s did not wait on the revoking owner membership", kind)
				}
			}

			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case outcome := <-result:
				if !errors.Is(outcome, organization.ErrOrganizationOwnerRequired) {
					t.Fatalf("revoked owner %s should be denied, got %v", kind, outcome)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("owner write did not finish after membership revocation")
			}

			var actualName string
			if err := pool.QueryRow(ctx, "SELECT name FROM organizations WHERE party_id=$1", org.PartyID).Scan(&actualName); err != nil {
				t.Fatal(err)
			}
			if actualName != orgName {
				t.Fatalf("revoked owner changed organization name to %q", actualName)
			}
			var activeAdded int
			if err := pool.QueryRow(ctx,
				"SELECT count(*) FROM organization_members WHERE organization_id=$1 AND person_id=$2 AND ended_at IS NULL",
				org.PartyID, target.PersonID,
			).Scan(&activeAdded); err != nil {
				t.Fatal(err)
			}
			if activeAdded != 0 {
				t.Fatal("revoked owner added a new member")
			}
		})
	}
}
