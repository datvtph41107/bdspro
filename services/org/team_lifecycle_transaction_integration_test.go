package org

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

const changedTeamName = "Authorized Team Race"

func teamLifecycleMutation(ctx context.Context, store db.Store, actor teamCase, action string) error {
	ctx = teamActionContext(ctx, store, actor.accountID, actor.sessionID)
	switch action {
	case "create":
		return NewTeam(ctx, &org_model.Team{
			OrgID: actor.orgID, Name: changedTeamName, Description: "Test",
		})
	case "rename":
		return UpdateTeam(ctx, &org_model.Team{ID: actor.teamID, Name: changedTeamName})
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
				result <- teamLifecycleMutation(ctx, teamPinnedStore{Store: store, conn: writer}, actor, action)
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
				writeResult <- teamLifecycleMutation(ctx, teamPinnedStore{Store: store, conn: writer}, actor, action)
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
	// Gitea-style model inputs still reject invalid names before a Store is
	// required, and a missing/mismatched actor cannot authorize SQLC writes.
	if err := NewTeam(context.Background(), &org_model.Team{Name: "  "}); !errors.Is(err, org_model.ErrTeamNameInvalid) {
		t.Fatalf("blank name reached the database: %v", err)
	}
	if err := UpdateTeam(context.Background(), &org_model.Team{ID: 2, Name: "  "}); !errors.Is(err, org_model.ErrTeamNameInvalid) {
		t.Fatalf("blank rename reached the database: %v", err)
	}
	ctx, pool, store := membershipTestDB(t)
	actor := newTeamCase(t, ctx, store)
	if err := NewTeam(teamActionContext(ctx, store, actor.accountID, uuid.Nil),
		&org_model.Team{OrgID: actor.orgID, Name: changedTeamName}); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("missing session created team: %v", err)
	}
	if err := UpdateTeam(teamActionContext(ctx, store, actor.accountID+1, actor.sessionID),
		&org_model.Team{ID: actor.teamID, Name: changedTeamName}); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("foreign account renamed team: %v", err)
	}
	if err := NewTeam(teamActionContext(ctx, store, actor.accountID, actor.sessionID),
		&org_model.Team{OrgID: 9223372036854775807, Name: changedTeamName}); !errors.Is(err, org_model.ErrTeamOwnerRequired) {
		t.Fatalf("unauthorized team creation bypassed SQLC: %v", err)
	}
	verifyTeamLifecycleState(t, ctx, pool, actor, "create", false)
	verifyTeamLifecycleState(t, ctx, pool, actor, "rename", false)
}

func TestTeamLifecycleSuccessfulMutations(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	actor := newTeamCase(t, ctx, store)
	actionCtx := teamActionContext(ctx, store, actor.accountID, actor.sessionID)
	created := &org_model.Team{
		OrgID: actor.orgID, Name: "  Created By Service  ", Description: "Session checked",
	}
	if err := NewTeam(actionCtx, created); err != nil || created.Name != "Created By Service" {
		t.Fatalf("create team result %+v err=%v", created, err)
	}
	renamed := &org_model.Team{ID: created.ID, Name: "  Renamed By Service "}
	if err := UpdateTeam(actionCtx, renamed); err != nil || renamed.Name != "Renamed By Service" {
		t.Fatalf("rename team result %+v err=%v", renamed, err)
	}
	if err := NewTeam(actionCtx, &org_model.Team{OrgID: actor.orgID, Name: "RENAMED BY SERVICE"}); !errors.Is(err, org_model.ErrTeamAlreadyExists) {
		t.Fatalf("duplicate team name not rejected: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM teams WHERE id=$1 AND name=$2",
		created.ID, "Renamed By Service").Scan(&count); err != nil || count != 1 {
		t.Fatalf("updated team not committed: count=%d err=%v", count, err)
	}
}
