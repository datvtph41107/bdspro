package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func verifyRemovedMember(t *testing.T, ctx context.Context, pool *pgxpool.Pool, d teamCase, removed bool) {
	t.Helper()
	var active, teams int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM organization_members
		WHERE organization_id=$1 AND person_id=$2 AND ended_at IS NULL`,
		d.orgID, d.recipientPersonID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM team_members tm
		JOIN teams t ON t.id=tm.team_id
		WHERE t.organization_id=$1 AND tm.person_id=$2`,
		d.orgID, d.recipientPersonID).Scan(&teams); err != nil {
		t.Fatal(err)
	}
	wantActive, wantTeam := 1, 1
	if removed {
		wantActive, wantTeam = 0, 0
	}
	if active != wantActive || teams != wantTeam {
		t.Fatalf("removed=%v active=%d team_links=%d expected=%d/%d", removed, active, teams, wantActive, wantTeam)
	}
}

func TestRemoveOrdinaryOrganizationMemberDoesNotRestoreTeamOrInviteOnRejoin(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newTeamCase(t, ctx, store)
	recipient, err := store.GetAccountByID(ctx, accountIDForPerson(t, ctx, pool, d.recipientPersonID))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
		ID: recipient.ID, Email: recipient.Email,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := org_model.AddTeamUser(ctx, store, d.accountID, d.teamID, d.recipientPersonID); err != nil {
		t.Fatal(err)
	}
	otherTeam, err := org_model.CreateTeam(ctx, store, org_model.CreateTeamOptions{
		OrganizationID: d.orgID, ActorAccountID: d.accountID, Name: "Invites " + uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	oldInvite, err := org_model.CreateTeamInvite(ctx, store, d.accountID, otherTeam.ID,
		recipient.Email, time.Hour, teamInviteTestKey)
	if err != nil {
		t.Fatal(err)
	}
	verifyRemovedMember(t, ctx, pool, d, false)
	if err := RemoveOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.recipientPersonID); err != nil {
		t.Fatal(err)
	}
	verifyRemovedMember(t, ctx, pool, d, true)
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events
		WHERE action='organization.member.removed' AND actor_account_id=$1
		  AND scope_type='organization' AND scope_ref=$2
		  AND target_type='person' AND target_ref=$3`,
		d.accountID, fmt.Sprint(d.orgID), fmt.Sprint(d.recipientPersonID)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("missing atomic organization removal audit: %d", auditCount)
	}
	if _, err := store.GetTeamInviteForDelivery(ctx, oldInvite.Invite.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("removed recipient invitation still deliverable: %v", err)
	}
	if _, err := org_model.AddOrgUser(ctx, store, d.accountID, d.orgID, d.recipientPersonID); err != nil {
		t.Fatal(err)
	}
	// A rejoin must not restore grants from the previous membership generation.
	var leftover int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM team_members WHERE team_id=$1 AND person_id=$2`,
		d.teamID, d.recipientPersonID).Scan(&leftover); err != nil || leftover != 0 {
		t.Fatalf("stale team membership restored: count=%d err=%v", leftover, err)
	}
	if _, err := org_model.AcceptTeamInvite(ctx, store, recipient.ID, oldInvite.Token); !errors.Is(err, org_model.ErrTeamInviteInvalid) {
		t.Fatalf("prior-generation invitation became reusable: %v", err)
	}
	if _, err := store.GetTeamInviteForDelivery(ctx, oldInvite.Invite.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("prior-generation mail intent became deliverable: %v", err)
	}
	fresh, err := org_model.CreateTeamInvite(ctx, store, d.accountID, otherTeam.ID,
		recipient.Email, time.Hour, teamInviteTestKey)
	if err != nil {
		t.Fatalf("stale open invitation blocked a fresh membership generation: %v", err)
	}
	if _, err := org_model.AcceptTeamInvite(ctx, store, recipient.ID, fresh.Token); err != nil {
		t.Fatalf("new generation invitation cannot be redeemed: %v", err)
	}
	if err := RemoveOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.recipientPersonID); err != nil {
		t.Fatalf("second generation removal: %v", err)
	}
}

func accountIDForPerson(t *testing.T, ctx context.Context, pool *pgxpool.Pool, personID int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx, "SELECT id FROM accounts WHERE person_id=$1", personID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRemoveOrganizationMemberRejectsOwnerOutsiderReplayAndRevokedActor(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newTeamCase(t, ctx, store)
	var ownerPersonID int64
	if err := pool.QueryRow(ctx, "SELECT person_id FROM accounts WHERE id=$1", d.accountID).Scan(&ownerPersonID); err != nil {
		t.Fatal(err)
	}
	if err := RemoveOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, ownerPersonID); !errors.Is(err, org_model.ErrOrganizationMemberNotFound) {
		t.Fatalf("owner role was removable: %v", err)
	}
	var owners int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM organization_members
		WHERE organization_id=$1 AND role='owner' AND ended_at IS NULL`, d.orgID).Scan(&owners); err != nil || owners != 1 {
		t.Fatalf("last owner lost: owners=%d err=%v", owners, err)
	}
	outsiderID := accountIDForPerson(t, ctx, pool, d.recipientPersonID)
	sessionID := uuid.New()
	if _, err := store.CreateSession(ctx, db.CreateSessionParams{
		ID: sessionID, AccountID: outsiderID,
		RefreshToken: "test-" + uuid.NewString(), UserAgent: "org-member", ClientIp: "203.0.113.42",
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveOrganizationMember(ctx, store, outsiderID, sessionID, d.orgID, d.recipientPersonID); !errors.Is(err, org_model.ErrOrganizationMemberNotFound) {
		t.Fatalf("regular member revoked membership: %v", err)
	}
	if err := RemoveOrganizationMember(ctx, store, d.accountID, uuid.Nil, d.orgID, d.recipientPersonID); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("nil actor authorized: %v", err)
	}
	if err := RemoveOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.recipientPersonID); err != nil {
		t.Fatalf("owner removal: %v", err)
	}
	if err := RemoveOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.recipientPersonID); !errors.Is(err, org_model.ErrOrganizationMemberNotFound) {
		t.Fatalf("replayed removal result: %v", err)
	}
	if _, err := store.RevokeOwnedSession(ctx, db.RevokeOwnedSessionParams{
		ID: d.sessionID, AccountID: d.accountID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveOrganizationMember(ctx, store, d.accountID, d.sessionID, d.orgID, d.recipientPersonID); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("revoked actor reached mutation: %v", err)
	}
}

var errAuditUnavailable = errors.New("audit storage unavailable in test")

type auditRejectDB struct{ db.DBTX }

func (w auditRejectDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "-- name: InsertAuditAction") {
		return pgconn.CommandTag{}, errAuditUnavailable
	}
	return w.DBTX.Exec(ctx, sql, args...)
}

type auditRejectStore struct {
	db.Store
	pool *pgxpool.Pool
}

func (s auditRejectStore) ExecTx(ctx context.Context, action func(*db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := action(db.New(auditRejectDB{DBTX: tx})); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func TestOrganizationMemberRemovalAuditFailureRollsBack(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newTeamCase(t, ctx, store)
	if _, err := org_model.AddTeamUser(ctx, store, d.accountID, d.teamID, d.recipientPersonID); err != nil {
		t.Fatal(err)
	}
	broken := auditRejectStore{Store: store, pool: pool}
	if err := RemoveOrganizationMember(ctx, broken, d.accountID, d.sessionID, d.orgID, d.recipientPersonID); !errors.Is(err, errAuditUnavailable) {
		t.Fatalf("audit failure not propagated: %v", err)
	}
	verifyRemovedMember(t, ctx, pool, d, false)
}

func TestConcurrentTeamInsertCannotOutliveOrganizationRemoval(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newTeamCase(t, ctx, store)
	teamWriter, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer teamWriter.Release()
	remover, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer remover.Release()
	tx, err := teamWriter.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := db.New(tx).AddTeamUserForOwner(ctx, db.AddTeamUserForOwnerParams{
		TeamID: d.teamID, ActorAccountID: d.accountID, PersonID: d.recipientPersonID,
	}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- RemoveOrganizationMember(ctx, teamPinnedStore{Store: store, conn: remover}, d.accountID, d.sessionID, d.orgID, d.recipientPersonID)
	}()
	waitTeamBlocking(t, ctx, pool, teamWriter, remover, done)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("member removal did not finish after team commit")
	}
	verifyRemovedMember(t, ctx, pool, d, true)
}
