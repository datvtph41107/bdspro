package org

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func requirePGConstraint(t *testing.T, err error, state, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != state || pgErr.ConstraintName != constraint {
		t.Fatalf("expected SQLSTATE %s / constraint %s; got %v", state, constraint, err)
	}
}

func requireOwnerPersonLink(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orgID, personID int64, linked bool) {
	t.Helper()
	var ownerPerson sql.NullInt64
	if err := pool.QueryRow(ctx, `SELECT owner_person_id FROM organization_members
		WHERE organization_id=$1 AND person_id=$2 AND ended_at IS NULL`,
		orgID, personID).Scan(&ownerPerson); err != nil {
		t.Fatal(err)
	}
	if linked {
		if !ownerPerson.Valid || ownerPerson.Int64 != personID {
			t.Fatalf("owner_person_id=%v, expected person %d", ownerPerson, personID)
		}
	} else if ownerPerson.Valid {
		t.Fatalf("non-owner retains account-deletion restriction: %v", ownerPerson)
	}
}

// This FK is independent of HTTP and of service authorization. Out-of-band
// SQL may not leave a stored owner whose Person no longer has an Account.
func TestOwnerAccountDeleteBlockedByDatabaseForeignKey(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOwnerRoleCase(t, ctx, pool, store, true)
	requireOwnerPersonLink(t, ctx, pool, d.orgID, d.actorPersonID, true)
	requireOwnerPersonLink(t, ctx, pool, d.orgID, d.recipientPersonID, false)

	// The session FK must not mask the new owner-specific FK.
	if _, err := pool.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1", d.accountID); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, "DELETE FROM accounts WHERE id=$1", d.accountID)
	requirePGConstraint(t, err, "23001", "organization_members_owner_account_fkey")
	requireOwnerCount(t, ctx, pool, d.orgID, 1)
	requireOwnerPersonLink(t, ctx, pool, d.orgID, d.actorPersonID, true)

	_, err = pool.Exec(ctx, `UPDATE organization_members SET owner_person_id=NULL
		WHERE organization_id=$1 AND person_id=$2`, d.orgID, d.actorPersonID)
	requirePGConstraint(t, err, "23514", "organization_members_owner_link_check")

	_, err = pool.Exec(ctx, `UPDATE organization_members SET owner_person_id=$1
		WHERE organization_id=$2 AND person_id=$3`,
		d.recipientPersonID, d.orgID, d.actorPersonID)
	requirePGConstraint(t, err, "23514", "organization_members_owner_link_check")
	requireOwnerCount(t, ctx, pool, d.orgID, 1)
}

func TestOwnerAccountLinkRejectsForgedAndOrphanedOwnerButAllowsOrdinaryPerson(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOwnerRoleCase(t, ctx, pool, store, true)
	orphan, err := db.WithTxResult(ctx, store, func(q *db.Queries) (int64, error) {
		return db.CreatePersonWithParty(ctx, q)
	})
	if err != nil {
		t.Fatal(err)
	}
	// Both the direct SQL attempt and the canonical SQLC create must reject
	// creating an owner whose Person has no account record.
	_, err = pool.Exec(ctx, `INSERT INTO organization_members
		(organization_id, person_id, role, owner_person_id)
		VALUES ($1, $2, 'owner', $2)`, d.orgID, orphan)
	requirePGConstraint(t, err, "23503", "organization_members_owner_account_fkey")

	_, err = pool.Exec(ctx, `INSERT INTO organization_members
		(organization_id, person_id, role)
		VALUES ($1, $2, 'owner')`, d.orgID, orphan)
	requirePGConstraint(t, err, "23514", "organization_members_owner_link_check")

	_, err = store.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
		OrganizationID: d.orgID, PersonID: orphan, Role: org_model.RoleOwner,
	})
	requirePGConstraint(t, err, "23503", "organization_members_owner_account_fkey")

	added, err := org_model.AddOrgUser(ctx, store, d.accountID, d.orgID, orphan)
	if err != nil || added.Role != org_model.RoleMember {
		t.Fatalf("unlinked person can still be ordinary member: %+v %v", added, err)
	}
	requireOwnerPersonLink(t, ctx, pool, d.orgID, orphan, false)
	_, err = pool.Exec(ctx, `UPDATE organization_members SET role='owner'
		WHERE organization_id=$1 AND person_id=$2`, d.orgID, orphan)
	requirePGConstraint(t, err, "23514", "organization_members_owner_link_check")
}

// An actual transfer must change the FK in the same commit as the owner
// roles. The original owner can then be deleted once the unrelated session
// references are cleaned up. This checks exact FK release, not only roles.
func TestOwnerAccountLinkFollowsTransferAndDemotion(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOwnerRoleCase(t, ctx, pool, store, true)
	if _, err := TransferOrganizationOwner(ctx, store, d.accountID, d.sessionID,
		d.orgID, d.recipientPersonID); err != nil {
		t.Fatal(err)
	}
	requireOwnerPersonLink(t, ctx, pool, d.orgID, d.actorPersonID, false)
	requireOwnerPersonLink(t, ctx, pool, d.orgID, d.recipientPersonID, true)

	// Only the former owner is a plain member now; its account FK is released.
	if _, err := pool.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1", d.accountID); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, "DELETE FROM accounts WHERE id=$1", d.accountID)
	if err != nil {
		t.Fatalf("former owner's account should no longer be protected by owner FK: %v", err)
	}
	requireOwnerCount(t, ctx, pool, d.orgID, 1)
	requireRole(t, ctx, pool, d.orgID, d.actorPersonID, org_model.RoleMember)
	_, err = pool.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1", d.recipientAccountID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, "DELETE FROM accounts WHERE id=$1", d.recipientAccountID)
	requirePGConstraint(t, err, "23001", "organization_members_owner_account_fkey")
	requireOwnerPersonLink(t, ctx, pool, d.orgID, d.recipientPersonID, true)
}

// Account deletion locks the recipient first. Promotion cannot slip past a
// concurrent deletion and leave a newly orphaned owner.
func TestOwnerPromotionCannotOutliveAccountDeletion(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOwnerRoleCase(t, ctx, pool, store, true)
	deleter, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer deleter.Release()
	promoter, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer promoter.Release()
	tx, err := deleter.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1", d.recipientAccountID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM accounts WHERE id=$1", d.recipientAccountID); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := PromoteOrganizationOwner(ctx,
			teamPinnedStore{Store: store, conn: promoter},
			d.accountID, d.sessionID, d.orgID, d.recipientPersonID)
		done <- err
	}()
	waitTeamBlocking(t, ctx, pool, deleter, promoter, done)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, org_model.ErrOwnerCandidateInvalid) {
			t.Fatalf("promotion authorized a deleted account: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("promotion stuck after account deletion")
	}
	requireRole(t, ctx, pool, d.orgID, d.recipientPersonID, org_model.RoleMember)
	requireOwnerCount(t, ctx, pool, d.orgID, 1)
}
