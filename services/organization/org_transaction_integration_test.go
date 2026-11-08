package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type organizationRaceCase struct {
	teamCase
	createdName    string
	initialName    string
	updatedName    string
	targetPersonID int64
	actorPersonID  int64
}

func newOrganizationRaceCase(t *testing.T, ctx context.Context, pool *pgxpool.Pool, store db.Store) organizationRaceCase {
	t.Helper()
	actor := newTeamCase(t, ctx, store)
	var originalName string
	var actorPersonID int64
	if err := pool.QueryRow(ctx, "SELECT name FROM organizations WHERE party_id=$1", actor.orgID).
		Scan(&originalName); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT person_id FROM accounts WHERE id=$1", actor.accountID).
		Scan(&actorPersonID); err != nil {
		t.Fatal(err)
	}
	personID, err := db.WithTxResult(ctx, store, func(q *db.Queries) (int64, error) {
		return db.CreatePersonWithParty(ctx, q)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAccount(ctx, db.CreateAccountParams{
		PersonID: personID, Email: uuid.NewString() + "@example.test",
	}); err != nil {
		t.Fatal(err)
	}
	return organizationRaceCase{
		teamCase: actor, createdName: "Org lifecycle " + uuid.NewString(),
		initialName: originalName, updatedName: "Renamed " + uuid.NewString(),
		targetPersonID: personID, actorPersonID: actorPersonID,
	}
}

func applyOrganizationWrite(ctx context.Context, store db.Store, d organizationRaceCase, operation string) error {
	switch operation {
	case "create":
		_, err := CreateOrganization(ctx, store, d.accountID, d.sessionID, d.createdName)
		return err
	case "update":
		_, err := UpdateOrganization(ctx, store, d.accountID, d.sessionID, d.orgID, d.updatedName)
		return err
	case "add_member":
		_, err := AddOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.targetPersonID)
		return err
	}
	panic("unsupported organization write")
}

func verifyOrganizationWrite(t *testing.T, ctx context.Context, pool *pgxpool.Pool, d organizationRaceCase, operation string, committed bool) {
	t.Helper()
	switch operation {
	case "create":
		var count, owners int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM organizations WHERE name=$1", d.createdName).
			Scan(&count); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM organization_members m
			JOIN organizations o ON o.party_id=m.organization_id
			WHERE o.name=$1 AND m.person_id=$2 AND m.role='owner' AND m.ended_at IS NULL`,
			d.createdName, d.actorPersonID).Scan(&owners); err != nil {
			t.Fatal(err)
		}
		want := 0
		if committed {
			want = 1
		}
		if count != want || owners != want {
			t.Fatalf("organization/owner commit mismatch: org=%d owners=%d want=%d", count, owners, want)
		}
	case "update":
		var name string
		if err := pool.QueryRow(ctx, "SELECT name FROM organizations WHERE party_id=$1", d.orgID).
			Scan(&name); err != nil {
			t.Fatal(err)
		}
		want := d.initialName
		if committed {
			want = d.updatedName
		}
		if name != want {
			t.Fatalf("organization name=%q want=%q", name, want)
		}
	case "add_member":
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM organization_members
			WHERE organization_id=$1 AND person_id=$2 AND role='member' AND ended_at IS NULL`,
			d.orgID, d.targetPersonID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		want := 0
		if committed {
			want = 1
		}
		if count != want {
			t.Fatalf("new member count=%d want=%d", count, want)
		}
	}
}

// Session revocation wins. Each write must wait for the actual acting-session
// PostgreSQL row and then fail without a persisted organization effect.
func TestOrganizationWritesCannotOutliveRevokedSession(t *testing.T) {
	for _, operation := range []string{"create", "update", "add_member"} {
		t.Run(operation, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			d := newOrganizationRaceCase(t, ctx, pool, store)
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
			if _, err := tx.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", d.sessionID); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- applyOrganizationWrite(ctx, teamPinnedStore{Store: store, conn: writer}, d, operation) }()
			waitTeamBlocking(t, ctx, pool, revoker, writer, done)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if !errors.Is(err, auth_model.ErrSessionUnavailable) {
					t.Fatalf("revoked session authorized %s: %v", operation, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("organization write did not resume")
			}
			verifyOrganizationWrite(t, ctx, pool, d, operation, false)
		})
	}
}

// Write wins. For create, block the new owner membership's Person FK row;
// create must have already locked account then session before reaching it.
// For update/add-member, block the owner membership row FOR UPDATE.
func TestOrganizationWritesCommitBeforeRevocation(t *testing.T) {
	for _, operation := range []string{"create", "update", "add_member"} {
		t.Run(operation, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			d := newOrganizationRaceCase(t, ctx, pool, store)
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
			if operation == "create" {
				var personID int64
				if err := tx.QueryRow(ctx,
					"SELECT party_id FROM persons WHERE party_id=$1 FOR UPDATE",
					d.actorPersonID).Scan(&personID); err != nil {
					t.Fatal(err)
				}
			} else {
				var memberID int64
				if err := tx.QueryRow(ctx, `SELECT m.id FROM organization_members m
					JOIN accounts a ON a.person_id=m.person_id
					WHERE a.id=$1 AND m.organization_id=$2 AND m.ended_at IS NULL
					FOR UPDATE OF m`, d.accountID, d.orgID).Scan(&memberID); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan error, 1)
			go func() { done <- applyOrganizationWrite(ctx, teamPinnedStore{Store: store, conn: writer}, d, operation) }()
			waitTeamBlocking(t, ctx, pool, blocker, writer, done)
			revokeDone := make(chan error, 1)
			go func() {
				revokeDone <- auth_model.RevokeOwnedSession(ctx,
					teamPinnedStore{Store: store, conn: revoker},
					d.accountID, d.sessionID, d.sessionID)
			}()
			waitTeamBlocking(t, ctx, pool, writer, revoker, revokeDone)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("organization %s rejected before revocation: %v", operation, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("organization writer stuck after resource release")
			}
			select {
			case err := <-revokeDone:
				if err != nil {
					t.Fatalf("revoke after %s failed: %v", operation, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("revoker stuck after organization commit")
			}
			verifyOrganizationWrite(t, ctx, pool, d, operation, true)
		})
	}
}

// Password reset acquires accounts FOR UPDATE BEFORE DeleteSessionsForAccount.
// An organization create must wait at account FOR KEY SHARE (not hold session
// and then wait at account); otherwise these transactions could deadlock.
func TestOrganizationCreationAccountLockPrecedesSessionForPasswordReset(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOrganizationRaceCase(t, ctx, pool, store)
	reset, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reset.Release()
	writer, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Release()
	tx, err := reset.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var id int64
	if err := tx.QueryRow(ctx, "SELECT id FROM accounts WHERE id=$1 FOR UPDATE",
		d.accountID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- createOrganizationRaceError(ctx, teamPinnedStore{Store: store, conn: writer}, d) }()
	waitTeamBlocking(t, ctx, pool, reset, writer, done)
	// This follows credential recovery's account-first lock order and
	// deletes the actor session without waiting on the organization writer.
	if _, err := tx.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1", d.accountID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, auth_model.ErrSessionUnavailable) {
			t.Fatalf("account-first reset did not invalidate organization create: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("creation blocked after reset commit")
	}
	verifyOrganizationWrite(t, ctx, pool, d, "create", false)
}

func createOrganizationRaceError(ctx context.Context, store db.Store, d organizationRaceCase) error {
	_, err := CreateOrganization(ctx, store, d.accountID, d.sessionID, d.createdName)
	return err
}

func TestOrganizationServiceAuthorizationAndPreflight(t *testing.T) {
	// No DB access when the name is known invalid.
	if _, err := CreateOrganization(context.Background(), nil, 10, uuid.Nil, " "); !errors.Is(err, org_model.ErrNameRequired) {
		t.Fatalf("blank creation reached database: %v", err)
	}
	if _, err := UpdateOrganization(context.Background(), nil, 10, uuid.Nil, 2, " "); !errors.Is(err, org_model.ErrNameRequired) {
		t.Fatalf("blank update reached database: %v", err)
	}
	ctx, pool, store := membershipTestDB(t)
	d := newOrganizationRaceCase(t, ctx, pool, store)
	if _, err := CreateOrganization(ctx, store, d.accountID, uuid.Nil, d.createdName); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("nil actor session created organization: %v", err)
	}
	if _, err := UpdateOrganization(ctx, store, d.accountID+1, d.sessionID, d.orgID, d.updatedName); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("foreign actor/session renamed organization: %v", err)
	}
	if _, err := AddOrganizationMember(ctx, store, d.accountID, uuid.Nil, d.orgID, d.targetPersonID); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("nil actor session added organization member: %v", err)
	}
	if _, err := CreateOrganization(ctx, store, 9223372036854775807, d.sessionID, d.createdName); !errors.Is(err, account_model.ErrAccountUnavailable) {
		t.Fatalf("nonexistent account created organization: %v", err)
	}
	var recipientID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM accounts WHERE person_id=$1", d.teamCase.recipientPersonID).
		Scan(&recipientID); err != nil {
		t.Fatal(err)
	}
	recipientSession := uuid.New()
	if _, err := store.CreateSession(ctx, db.CreateSessionParams{
		ID: recipientSession, AccountID: recipientID,
		RefreshToken: "test-" + uuid.NewString(), UserAgent: "g1-2c7",
		ClientIp: "203.0.113.43", ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateOrganization(ctx, store, recipientID, recipientSession, d.orgID, d.updatedName); !errors.Is(err, org_model.ErrOrganizationOwnerRequired) {
		t.Fatalf("ordinary member changed organization name: %v", err)
	}
	if _, err := AddOrganizationMember(ctx, store, recipientID, recipientSession, d.orgID, d.targetPersonID); !errors.Is(err, org_model.ErrOrganizationOwnerRequired) {
		t.Fatalf("ordinary member invited organization member: %v", err)
	}
	verifyOrganizationWrite(t, ctx, pool, d, "create", false)
	verifyOrganizationWrite(t, ctx, pool, d, "update", false)
	verifyOrganizationWrite(t, ctx, pool, d, "add_member", false)
}

func TestOrganizationServiceAuthorizedWrites(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOrganizationRaceCase(t, ctx, pool, store)
	created, err := CreateOrganization(ctx, store, d.accountID, d.sessionID, " "+d.createdName+" ")
	if err != nil || created.Name != d.createdName {
		t.Fatalf("organization create=%+v err=%v", created, err)
	}
	updated, err := UpdateOrganization(ctx, store, d.accountID, d.sessionID, d.orgID, d.updatedName)
	if err != nil || updated.Name != d.updatedName {
		t.Fatalf("organization update=%+v err=%v", updated, err)
	}
	member, err := AddOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.targetPersonID)
	if err != nil || member.PersonID != d.targetPersonID || member.Role != "member" {
		t.Fatalf("added member=%+v err=%v", member, err)
	}
	verifyOrganizationWrite(t, ctx, pool, d, "create", true)
	verifyOrganizationWrite(t, ctx, pool, d, "update", true)
	verifyOrganizationWrite(t, ctx, pool, d, "add_member", true)
	if _, err := AddOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.targetPersonID); !errors.Is(err, org_model.ErrMembershipExists) {
		t.Fatalf("duplicate membership bypassed SQLC: %v", err)
	}
}
