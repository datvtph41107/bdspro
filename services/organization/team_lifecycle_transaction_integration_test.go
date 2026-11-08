package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const changedTeamName = "Authorized Team Race"

func teamLifecycleMutation(ctx context.Context, svc *TeamService, actor teamCase, action string) error {
	switch action {
	case "create":
		_, err := svc.CreateTeam(ctx, actor.sessionID, org_model.CreateTeamOptions{
			OrganizationID: actor.orgID, ActorAccountID: actor.accountID,
			Name: changedTeamName, Description: "Test",
		})
		return err
	case "rename":
		_, err := svc.RenameTeam(ctx, actor.accountID, actor.sessionID, actor.teamID, changedTeamName)
		return err
	default:
		panic("unexpected lifecycle action")
	}
}

func verifyTeamLifecycleState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, actor teamCase, action string, committed bool) {
	t.Helper()
	switch action {
	case "create":
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM teams WHERE organization_id=$1 AND name=$2",
			actor.orgID, changedTeamName).Scan(&count); err != nil {
			t.Fatal(err)
		}
		expected := 0
		if committed {
			expected = 1
		}
		if count != expected {
			t.Fatalf("created teams=%d expected=%d", count, expected)
		}
	case "rename":
		var name string
		if err := pool.QueryRow(ctx, "SELECT name FROM teams WHERE id=$1", actor.teamID).Scan(&name); err != nil {
			t.Fatal(err)
		}
		expected := "Backend"
		if committed {
			expected = changedTeamName
		}
		if name != expected {
			t.Fatalf("team name=%q expected=%q", name, expected)
		}
	}
}

// Session revocation wins: Create and Rename must wait for the revocation
// row lock, then reject without leaving any persisted team change.
func TestTeamLifecycleCannotOutliveRevokedSession(t *testing.T) {
	for _, action := range []string{"create", "rename"} {
		t.Run(action, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			actor := newTeamCase(t, ctx, store)
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
			if _, err := tx.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", actor.sessionID); err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() {
				result <- teamLifecycleMutation(ctx, NewTeamService(teamPinnedStore{Store: store, conn: writer}), actor, action)
			}()
			waitTeamBlocking(t, ctx, pool, revoker, writer, result)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-result:
				if !errors.Is(err, auth_model.ErrSessionUnavailable) {
					t.Fatalf("revoked session authorized team %s: %v", action, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("team action did not resume after session revocation")
			}
			verifyTeamLifecycleState(t, ctx, pool, actor, action, false)
		})
	}
}

// Write wins: the service locks the acting session first and waits for the
// organization's owner-membership FOR SHARE lock. Subsequent revocation
// waits for the acting session and may only commit after the team write.
func TestTeamLifecycleCommitPrecedesRevocation(t *testing.T) {
	for _, action := range []string{"create", "rename"} {
		t.Run(action, func(t *testing.T) {
			ctx, pool, store := membershipTestDB(t)
			actor := newTeamCase(t, ctx, store)
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
				FOR UPDATE OF m`, actor.accountID, actor.orgID).Scan(&membershipID); err != nil {
				t.Fatal(err)
			}
			writeResult := make(chan error, 1)
			go func() {
				writeResult <- teamLifecycleMutation(ctx, NewTeamService(teamPinnedStore{Store: store, conn: writer}), actor, action)
			}()
			waitTeamBlocking(t, ctx, pool, blocker, writer, writeResult)
			revokeResult := make(chan error, 1)
			go func() {
				revokeResult <- auth_model.RevokeOwnedSession(ctx,
					teamPinnedStore{Store: store, conn: revoker},
					actor.accountID, actor.sessionID, actor.sessionID)
			}()
			waitTeamBlocking(t, ctx, pool, writer, revoker, revokeResult)
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-writeResult:
				if err != nil {
					t.Fatalf("team %s failed before revocation: %v", action, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("team writer did not resume")
			}
			select {
			case err := <-revokeResult:
				if err != nil {
					t.Fatalf("session revocation failed after team %s: %v", action, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("session revoker did not resume")
			}
			verifyTeamLifecycleState(t, ctx, pool, actor, action, true)
		})
	}
}

func TestTeamLifecycleServiceRejectsInvalidActorAndName(t *testing.T) {
	// Invalid name must be rejected without accessing the store, whether
	// or not the actor and session would be authorized.
	svcWithoutDB := NewTeamService(nil)
	if _, err := svcWithoutDB.CreateTeam(context.Background(), uuid.Nil,
		org_model.CreateTeamOptions{Name: "  "}); !errors.Is(err, org_model.ErrTeamNameInvalid) {
		t.Fatalf("blank name reached the database: %v", err)
	}
	if _, err := svcWithoutDB.RenameTeam(context.Background(), 1, uuid.Nil, 2,
		"  "); !errors.Is(err, org_model.ErrTeamNameInvalid) {
		t.Fatalf("blank rename reached the database: %v", err)
	}
	ctx, pool, store := membershipTestDB(t)
	actor := newTeamCase(t, ctx, store)
	svc := NewTeamService(store)
	if _, err := svc.CreateTeam(ctx, uuid.Nil, org_model.CreateTeamOptions{
		OrganizationID: actor.orgID, ActorAccountID: actor.accountID,
		Name: changedTeamName,
	}); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("missing session created team: %v", err)
	}
	if _, err := svc.RenameTeam(ctx, actor.accountID+1, actor.sessionID,
		actor.teamID, changedTeamName); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("foreign account renamed team: %v", err)
	}
	if _, err := svc.CreateTeam(ctx, actor.sessionID, org_model.CreateTeamOptions{
		OrganizationID: 9223372036854775807, ActorAccountID: actor.accountID,
		Name: changedTeamName,
	}); !errors.Is(err, org_model.ErrTeamOwnerRequired) {
		t.Fatalf("unauthorized team creation bypassed SQLC: %v", err)
	}
	verifyTeamLifecycleState(t, ctx, pool, actor, "create", false)
	verifyTeamLifecycleState(t, ctx, pool, actor, "rename", false)
}

// Confirm the model and service keep the same output mapping on the
// authorized path, including trimmed names and SQLC unique constraints.
func TestTeamLifecycleSuccessfulMutations(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	actor := newTeamCase(t, ctx, store)
	svc := NewTeamService(store)
	created, err := svc.CreateTeam(ctx, actor.sessionID, org_model.CreateTeamOptions{
		OrganizationID: actor.orgID, ActorAccountID: actor.accountID,
		Name: "  Created By Service  ", Description: "Session checked",
	})
	if err != nil || created.Name != "Created By Service" {
		t.Fatalf("create team result %+v err=%v", created, err)
	}
	renamed, err := svc.RenameTeam(ctx, actor.accountID, actor.sessionID,
		created.ID, "  Renamed By Service ")
	if err != nil || renamed.Name != "Renamed By Service" {
		t.Fatalf("rename team result %+v err=%v", renamed, err)
	}
	if _, err := svc.CreateTeam(ctx, actor.sessionID, org_model.CreateTeamOptions{
		OrganizationID: actor.orgID, ActorAccountID: actor.accountID,
		Name: "RENAMED BY SERVICE",
	}); !errors.Is(err, org_model.ErrTeamAlreadyExists) {
		t.Fatalf("duplicate team name not rejected: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM teams WHERE id=$1 AND name=$2",
		created.ID, "Renamed By Service").Scan(&count); err != nil || count != 1 {
		t.Fatalf("updated team not committed: count=%d err=%v", count, err)
	}
}
