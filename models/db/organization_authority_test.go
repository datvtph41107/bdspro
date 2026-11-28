package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestOrganizationAuthority(t *testing.T) {
	source := os.Getenv("BDSPRO_ORGANIZATION_AUTHORITY_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires disposable migrated BDSPRO_ORGANIZATION_AUTHORITY_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	locker, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer locker.Close(context.Background())
	waiter, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer waiter.Close(context.Background())

	setup, err := locker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Rollback(context.Background())
	q := sqlc.New(setup)
	createAccount := func() (int64, int64, string) {
		t.Helper()
		person, err := q.CreateParty(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := q.CreatePerson(ctx, person); err != nil {
			t.Fatal(err)
		}
		email := uuid.NewString() + "@example.invalid"
		account, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{PersonID: person, Email: email})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := q.MarkEmailVerified(ctx, sqlc.MarkEmailVerifiedParams{ID: account.ID, Email: email}); err != nil {
			t.Fatal(err)
		}
		return person, account.ID, email
	}
	ownerPerson, owner, _ := createAccount()
	memberPerson, recipient, email := createAccount()
	orgID, err := q.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateOrganization(ctx, sqlc.CreateOrganizationParams{PartyID: orgID, Name: "fixture"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateOrganizationMember(ctx, sqlc.CreateOrganizationMemberParams{OrganizationID: orgID, PersonID: ownerPerson, Role: "owner"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateOrganizationMember(ctx, sqlc.CreateOrganizationMemberParams{OrganizationID: orgID, PersonID: memberPerson, Role: "member"}); err != nil {
		t.Fatal(err)
	}
	team, err := q.CreateTeamForOwner(ctx, sqlc.CreateTeamForOwnerParams{OrganizationID: orgID, ActorAccountID: owner, Name: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, statement := range []struct {
			sql string
			id  int64
		}{
			{"DELETE FROM teams WHERE organization_id=$1", orgID},
			{"DELETE FROM organization_members WHERE organization_id=$1", orgID},
			{"DELETE FROM organizations WHERE party_id=$1", orgID},
			{"DELETE FROM accounts WHERE id=$1", owner},
			{"DELETE FROM accounts WHERE id=$1", recipient},
			{"DELETE FROM persons WHERE party_id=$1", ownerPerson},
			{"DELETE FROM persons WHERE party_id=$1", memberPerson},
			{"DELETE FROM parties WHERE id=$1", ownerPerson},
			{"DELETE FROM parties WHERE id=$1", memberPerson},
			{"DELETE FROM parties WHERE id=$1", orgID},
		} {
			if _, err := locker.Exec(cleanup, statement.sql, statement.id); err != nil {
				t.Error("fixture cleanup failed", err)
			}
		}
	}()
	q = sqlc.New(locker)
	newInvite := func(lifetime time.Duration) sqlc.CreateTeamInviteForOwnerParams {
		t.Helper()
		sum := sha256.Sum256([]byte(uuid.NewString()))
		return sqlc.CreateTeamInviteForOwnerParams{ID: uuid.New(), TeamID: team.ID, ActorAccountID: owner, InvitedEmail: email, TokenHash: sum[:], ExpiresAt: time.Now().Add(lifetime)}
	}
	releaseAfterWait := func(block pgx.Tx, deadline time.Time) <-chan error {
		result := make(chan error, 1)
		lockerPID, waiterPID := int32(locker.PgConn().PID()), int32(waiter.PgConn().PID())
		go func() {
			result <- func() error {
				defer block.Rollback(context.Background())
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for {
					var waiting bool
					if err := block.QueryRow(ctx, "SELECT $2::integer=ANY(pg_blocking_pids($1::integer))", waiterPID, lockerPID).Scan(&waiting); err != nil {
						return err
					}
					if waiting {
						break
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-ticker.C:
					}
				}
				if !deadline.IsZero() {
					if _, err := block.Exec(ctx, "SELECT pg_sleep(GREATEST(0,EXTRACT(EPOCH FROM ($1::timestamptz-clock_timestamp()))::double precision+0.05))", deadline); err != nil {
						return err
					}
				}
				return block.Commit(ctx)
			}()
		}()
		return result
	}

	t.Run("invitation expiry and replay", func(t *testing.T) {
		stale := newInvite(5 * time.Second)
		if _, err := q.CreateTeamInviteForOwner(ctx, stale); err != nil {
			t.Fatal(err)
		}
		block, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer block.Rollback(context.Background())
		if _, err := block.Exec(ctx, "SELECT id FROM team_invites WHERE id=$1 FOR UPDATE", stale.ID); err != nil {
			t.Fatal(err)
		}
		tx, err := waiter.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		tq := sqlc.New(tx)
		binding := sqlc.ConsumeTeamInviteParams{TokenHash: stale.TokenHash, AccountID: recipient}
		if _, err := tq.FindTeamInviteOrganization(ctx, sqlc.FindTeamInviteOrganizationParams{TokenHash: stale.TokenHash, AccountID: recipient}); err != nil {
			t.Fatal("invite must be live before wait", err)
		}
		if _, err := tq.LockOrganizationRoleChanges(ctx, orgID); err != nil {
			t.Fatal(err)
		}
		released := releaseAfterWait(block, stale.ExpiresAt)
		_, err = tq.ConsumeTeamInvite(ctx, binding)
		if releaseErr := <-released; releaseErr != nil {
			t.Fatal("invite row wait not verified", releaseErr)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("expired invite accepted", err)
		}
		// The transaction began before expiry. Cleanup must use live time too.
		if err := tq.RevokeExpiredTeamInvites(ctx, sqlc.RevokeExpiredTeamInvitesParams{TeamID: team.ID, InvitedEmail: email}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		var revoked bool
		if err := locker.QueryRow(ctx, "SELECT revoked_at IS NOT NULL AND accepted_at IS NULL FROM team_invites WHERE id=$1", stale.ID).Scan(&revoked); err != nil || !revoked {
			t.Fatal("expired invite not revoked", err)
		}
		live := newInvite(time.Hour)
		if _, err := q.CreateTeamInviteForOwner(ctx, live); err != nil {
			t.Fatal(err)
		}
		_, err = q.CreateTeamInviteForOwner(ctx, newInvite(time.Hour))
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "team_invites_open_uq" {
			t.Fatal("duplicate open invite not rejected by expected constraint", err)
		}
		if n, err := q.RevokeTeamInviteForOwner(ctx, sqlc.RevokeTeamInviteForOwnerParams{InviteID: live.ID, TeamID: team.ID, ActorAccountID: owner}); err != nil || n != 1 {
			t.Fatal("revoke failed", n, err)
		}
		binding.TokenHash = live.TokenHash
		if _, err := q.ConsumeTeamInvite(ctx, binding); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("revoked invite accepted", err)
		}
		live = newInvite(time.Hour)
		if _, err := q.CreateTeamInviteForOwner(ctx, live); err != nil {
			t.Fatal(err)
		}
		tx, err = waiter.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		tq = sqlc.New(tx)
		if _, err := tq.LockOrganizationRoleChanges(ctx, orgID); err != nil {
			t.Fatal(err)
		}
		binding.TokenHash = live.TokenHash
		teamID, err := tq.ConsumeTeamInvite(ctx, binding)
		if err != nil || teamID != team.ID {
			t.Fatal("valid invite denied", err)
		}
		if _, err := tq.InsertInvitedTeamUser(ctx, sqlc.InsertInvitedTeamUserParams{TeamID: teamID, AccountID: recipient}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := q.ConsumeTeamInvite(ctx, binding); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("invite replay accepted", err)
		}
		if n, err := q.RevokeTeamInviteForOwner(ctx, sqlc.RevokeTeamInviteForOwnerParams{InviteID: live.ID, TeamID: team.ID, ActorAccountID: owner}); err != nil || n != 0 {
			t.Fatal("accepted invite revoked", n, err)
		}
		t.Log("PASS: expiry after lock, cleanup at live time, duplicate/revoke/replay denied")
	})

	t.Run("membership removal and rejoin", func(t *testing.T) {
		// Remove the team link so an invitation can target the original membership.
		if _, err := q.RemoveTeamUserForOwner(ctx, sqlc.RemoveTeamUserForOwnerParams{TeamID: team.ID, ActorAccountID: owner, PersonID: memberPerson}); err != nil {
			t.Fatal(err)
		}
		old := newInvite(time.Hour)
		if _, err := q.CreateTeamInviteForOwner(ctx, old); err != nil {
			t.Fatal(err)
		}
		if _, err := q.AddTeamUserForOwner(ctx, sqlc.AddTeamUserForOwnerParams{TeamID: team.ID, ActorAccountID: owner, PersonID: memberPerson}); err != nil {
			t.Fatal(err)
		}
		tx, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		tq := sqlc.New(tx)
		if _, err := tq.LockOrganizationRoleChanges(ctx, orgID); err != nil {
			t.Fatal(err)
		}
		if _, err := tq.EndOrdinaryOrganizationMemberForOwner(ctx, sqlc.EndOrdinaryOrganizationMemberForOwnerParams{OrganizationID: orgID, ActorAccountID: owner, PersonID: memberPerson}); err != nil {
			t.Fatal(err)
		}
		if n, err := tq.DeleteOrgMemberTeamLinks(ctx, sqlc.DeleteOrgMemberTeamLinksParams{OrganizationID: orgID, PersonID: memberPerson}); err != nil || n != 1 {
			t.Fatal("team link cleanup failed", n, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := q.AddTeamUserForOwner(ctx, sqlc.AddTeamUserForOwnerParams{TeamID: team.ID, ActorAccountID: owner, PersonID: memberPerson}); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("ended member re-added to team", err)
		}
		if _, err := q.CreateOrganizationMember(ctx, sqlc.CreateOrganizationMemberParams{OrganizationID: orgID, PersonID: memberPerson, Role: "member"}); err != nil {
			t.Fatal(err)
		}
		if _, err := q.ConsumeTeamInvite(ctx, sqlc.ConsumeTeamInviteParams{TokenHash: old.TokenHash, AccountID: recipient}); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("old invite admitted new membership", err)
		}
		t.Log("PASS: removal clears team links and old invite cannot bind to rejoined member")
	})

	t.Run("eligible owner rollback and serialization", func(t *testing.T) {
		if _, err := q.PromoteOrganizationMember(ctx, sqlc.PromoteOrganizationMemberParams{OrganizationID: orgID, PersonID: memberPerson}); err != nil {
			t.Fatal(err)
		}
		block, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer block.Rollback(context.Background())
		bq := sqlc.New(block)
		if _, err := bq.LockOrganizationRoleChanges(ctx, orgID); err != nil {
			t.Fatal(err)
		}
		if _, err := bq.DemoteOrganizationOwner(ctx, sqlc.DemoteOrganizationOwnerParams{OrganizationID: orgID, PersonID: ownerPerson}); err != nil {
			t.Fatal(err)
		}
		if n, err := bq.CountEligibleOrganizationOwners(ctx, orgID); err != nil || n != 1 {
			t.Fatal("first demotion lost eligible owner", n, err)
		}
		tx, err := waiter.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		tq := sqlc.New(tx)
		released := releaseAfterWait(block, time.Time{})
		_, err = tq.LockOrganizationRoleChanges(ctx, orgID)
		if releaseErr := <-released; releaseErr != nil {
			t.Fatal("organization wait not verified", releaseErr)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tq.DemoteOrganizationOwner(ctx, sqlc.DemoteOrganizationOwnerParams{OrganizationID: orgID, PersonID: memberPerson}); err != nil {
			t.Fatal(err)
		}
		if n, err := tq.CountEligibleOrganizationOwners(ctx, orgID); err != nil || n != 0 {
			t.Fatal("last-owner guard fixture invalid", n, err)
		}
		// Mirrors the service's ensureEligibleOwner error and outer rollback.
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		if n, err := q.CountEligibleOrganizationOwners(ctx, orgID); err != nil || n != 1 {
			t.Fatal("rollback did not preserve last owner", n, err)
		}
		t.Log("PASS: serialized owner changes and last-owner rollback")
	})
}
