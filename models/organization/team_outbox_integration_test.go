package organization_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	organization "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTeamInviteRecordsOutboxWithoutPlaintext(t *testing.T) {
	ctx, store := integrationStore(t)
	cfg, err := pgxpool.ParseConfig(os.Getenv("BDS_TEST_DB_URL"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" || os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing to use a non-disposable database")
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	owner := makeActor(t, ctx, store)
	recipient := makeActor(t, ctx, store)
	org, err := organization.CreateOrganization(ctx, store, owner.ID, "Invite outbox "+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.ID, recipient.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
		ID: recipient.ID, Email: recipient.Email,
	}); err != nil {
		t.Fatal(err)
	}
	team, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.ID, ActorAccountID: owner.ID, Name: "Operations",
	})
	if err != nil {
		t.Fatal(err)
	}
	const key = "01234567890123456789012345678901"
	issued, err := organization.CreateTeamInvite(ctx, store, owner.ID, team.ID, recipient.Email, time.Hour, key)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	var payloadText string
	err = pool.QueryRow(ctx, `
		SELECT count(*), COALESCE(max(payload::text), '')
		FROM job_outbox
		WHERE task_type='task:send_team_invite'
		  AND payload->>'invite_id' = $1
	`, issued.Invite.ID.String()).Scan(&count, &payloadText)
	if err != nil || count != 1 {
		t.Fatalf("missing durable delivery intent: count=%d err=%v", count, err)
	}
	if strings.Contains(payloadText, issued.Token) || strings.Contains(payloadText, recipient.Email) {
		t.Fatal("plaintext proof or recipient email leaked into outbox payload")
	}
	if !strings.Contains(payloadText, issued.Invite.ID.String()) {
		t.Fatal("outbox did not reference the invitation")
	}
	delivery, err := store.GetTeamInviteForDelivery(ctx, issued.Invite.ID)
	if err != nil || delivery.InvitedEmail != recipient.Email {
		t.Fatalf("worker cannot retrieve active invitation: %+v / %v", delivery, err)
	}
	if _, err := organization.CreateTeamInvite(ctx, store, owner.ID, team.ID, recipient.Email, time.Hour, key); !errors.Is(err, organization.ErrTeamInviteExists) {
		t.Fatalf("duplicate invitation accepted: %v", err)
	}
	if err := organization.RevokeTeamInvite(ctx, store, owner.ID, team.ID, issued.Invite.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetTeamInviteForDelivery(ctx, issued.Invite.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("revoked invitation remains deliverable: %v", err)
	}
}

// A separate dependency-free test must be able to prove malformed invitation
// requests never use the database; the actual transaction is exercised above.
func TestTeamInviteRequiresStrongSigningKey(t *testing.T) {
	_, err := organization.CreateTeamInvite(context.Background(), nil, 1, 2, "member@example.test", time.Hour, "weak")
	if err == nil {
		t.Fatal("weak invitation signing key accepted")
	}
}
