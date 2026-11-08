package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const teamInviteTestKey = "01234567890123456789012345678901"

type inviteRaceCase struct {
	teamCase
	email        string
	inviteID     uuid.UUID
	token        string
	actorID      int64
	actorSession uuid.UUID
}

func newInviteRaceCase(t *testing.T, ctx context.Context, pool *pgxpool.Pool, store db.Store, operation string) inviteRaceCase {
	t.Helper()
	d := newTeamCase(t, ctx, store)
	var recipientID int64
	var email string
	if err := pool.QueryRow(ctx, "SELECT id, email FROM accounts WHERE person_id=$1",
		d.recipientPersonID).Scan(&recipientID, &email); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{ID: recipientID, Email: email}); err != nil {
		t.Fatal(err)
	}
	test := inviteRaceCase{
		teamCase: d, email: email, actorID: d.accountID, actorSession: d.sessionID,
	}
	if operation == "accept" || operation == "revoke" {
		created, err := org_model.CreateTeamInvite(ctx, store, d.accountID, d.teamID,
			email, time.Hour, teamInviteTestKey)
		if err != nil {
			t.Fatal(err)
		}
		test.inviteID, test.token = created.Invite.ID, created.Token
	}
	if operation == "accept" {
		test.actorID = recipientID
		test.actorSession = uuid.New()
		if _, err := store.CreateSession(ctx, db.CreateSessionParams{
			ID: test.actorSession, AccountID: recipientID, RefreshToken: "test-" + uuid.NewString(),
			UserAgent: "g1-2c5", ClientIp: "203.0.113.23", ExpiresAt: time.Now().Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}
	return test
}

func applyTeamInvite(ctx context.Context, svc *TeamService, d inviteRaceCase, operation string) error {
	switch operation {
	case "issue":
		_, err := svc.CreateTeamInvite(ctx, d.actorID, d.actorSession,
			d.teamID, d.email, time.Hour, teamInviteTestKey)
		return err
	case "accept":
		_, err := svc.AcceptTeamInvite(ctx, d.actorID, d.actorSession, d.token)
		return err
	case "revoke":
		return svc.RevokeTeamInvite(ctx, d.actorID, d.actorSession, d.teamID, d.inviteID)
	default:
		panic("unknown test operation")
	}
}

func assertInviteState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, d inviteRaceCase, operation string, mutationCommitted bool) {
	t.Helper()
	switch operation {
	case "issue":
		var n, outbox int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM team_invites WHERE team_id=$1", d.teamID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM job_outbox j JOIN team_invites i
			ON j.payload->>'invite_id' = i.id::text WHERE i.team_id=$1 AND j.task_type='task:send_team_invite'`,
			d.teamID).Scan(&outbox); err != nil {
			t.Fatal(err)
		}
		expected := 0
		if mutationCommitted {
			expected = 1
		}
		if n != expected || outbox != expected {
			t.Fatalf("issue proof/outbox mismatch: invites=%d jobs=%d expected=%d", n, outbox, expected)
		}
	case "accept":
		expected := 0
		if mutationCommitted {
			expected = 1
		}
		if n := membershipCount(t, ctx, pool, d.teamCase); n != expected {
			t.Fatalf("accepted membership count=%d want=%d", n, expected)
		}
		var accepted bool
		if err := pool.QueryRow(ctx, "SELECT accepted_at IS NOT NULL FROM team_invites WHERE id=$1", d.inviteID).Scan(&accepted); err != nil {
			t.Fatal(err)
		}
		if accepted != mutationCommitted {
			t.Fatalf("proof consumed=%v want=%v", accepted, mutationCommitted)
		}
	case "revoke":
		var revoked bool
		if err := pool.QueryRow(ctx, "SELECT revoked_at IS NOT NULL FROM team_invites WHERE id=$1", d.inviteID).Scan(&revoked); err != nil {
			t.Fatal(err)
		}
		if revoked != mutationCommitted {
			t.Fatalf("revoked proof=%v want=%v", revoked, mutationCommitted)
		}
	}
}

// Revocation obtains the session lock first. The invitation transaction
// cannot commit after observing a revoked actor, even when the request was
// admitted by HTTP before the revoke began.
func TestTeamInviteCannotOutliveRevokedActingSession(t *testing.T) {
	for _, operation := range []string{"issue", "accept", "revoke"} {
		t.Run(operation, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			d := newInviteRaceCase(t, ctx, pool, store, operation)
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
			if _, err := tx.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", d.actorSession); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- applyTeamInvite(ctx, NewTeamService(teamPinnedStore{Store: store, conn: writer}), d, operation)
			}()
			waitTeamBlocking(t, ctx, pool, revoker, writer, done)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if !errors.Is(err, auth_model.ErrSessionUnavailable) {
					t.Fatalf("revoked actor authorized invitation %s: %v", operation, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("invite mutation did not resume after revocation")
			}
			assertInviteState(t, ctx, pool, d, operation, false)
		})
	}
}

// The inverse: a transaction that has locked the actor session is entitled
// to commit its invited membership/outbox/proof update before a later revoke.
// Resource locking establishes this interleaving using pg_blocking_pids.
func TestTeamInviteCommitPrecedesRevocation(t *testing.T) {
	for _, operation := range []string{"issue", "accept", "revoke"} {
		t.Run(operation, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			d := newInviteRaceCase(t, ctx, pool, store, operation)
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
			if operation == "issue" {
				var rowID int64
				if err := tx.QueryRow(ctx, `SELECT m.id FROM organization_members m
					JOIN accounts a ON a.person_id=m.person_id
					WHERE a.id=$1 AND m.organization_id=$2 AND m.ended_at IS NULL
					FOR UPDATE OF m`, d.accountID, d.orgID).Scan(&rowID); err != nil {
					t.Fatal(err)
				}
			} else {
				var proofID uuid.UUID
				if err := tx.QueryRow(ctx, "SELECT id FROM team_invites WHERE id=$1 FOR UPDATE",
					d.inviteID).Scan(&proofID); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan error, 1)
			go func() {
				done <- applyTeamInvite(ctx, NewTeamService(teamPinnedStore{Store: store, conn: writer}), d, operation)
			}()
			waitTeamBlocking(t, ctx, pool, blocker, writer, done)
			revokeDone := make(chan error, 1)
			go func() {
				revokeDone <- auth_model.RevokeOwnedSession(ctx,
					teamPinnedStore{Store: store, conn: revoker}, d.actorID,
					d.actorSession, d.actorSession)
			}()
			waitTeamBlocking(t, ctx, pool, writer, revoker, revokeDone)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("valid %s did not commit: %v", operation, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("invitation writer did not commit")
			}
			select {
			case err := <-revokeDone:
				if err != nil {
					t.Fatalf("later session revocation failed: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("later session revoke did not complete")
			}
			assertInviteState(t, ctx, pool, d, operation, true)
		})
	}
}

func TestTeamInviteServiceRejectsInvalidActorAndProof(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newInviteRaceCase(t, ctx, pool, store, "issue")
	svc := NewTeamService(store)
	if _, err := svc.CreateTeamInvite(ctx, d.actorID, uuid.Nil, d.teamID,
		d.email, time.Hour, teamInviteTestKey); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("nil actor session issued invitation: %v", err)
	}
	if _, err := svc.CreateTeamInvite(ctx, d.actorID+1, d.actorSession, d.teamID,
		d.email, time.Hour, teamInviteTestKey); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("foreign actor/session issued invitation: %v", err)
	}
	if _, err := svc.AcceptTeamInvite(ctx, d.actorID, uuid.Nil, "not hex"); !errors.Is(err, org_model.ErrTeamInviteInvalid) {
		t.Fatalf("malformed proof should be rejected before DB lookup: %v", err)
	}
	if err := svc.RevokeTeamInvite(ctx, d.actorID, uuid.Nil, d.teamID, uuid.New()); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("nil actor revoked invitation: %v", err)
	}
	assertInviteState(t, ctx, pool, d, "issue", false)
}
