package organization

import (
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestOrganizationAccessProjection(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	raw := db.Organization{PartyID: 31, Name: "Canonical", CreatedAt: now}
	model := OrganizationFromDB(raw)
	if model.ID != raw.PartyID || model.Name != raw.Name || model.CreatedAt != now {
		t.Fatalf("SQLC Organization mapping lost fields: %+v", model)
	}
	access := Access{Organization: *model, MembershipID: 7, Role: RoleOwner}
	copy := access.OrganizationModel()
	copy.Name = "mutated"
	if access.Organization.Name != "Canonical" {
		t.Fatal("Access leaks a mutable persistence alias")
	}
}

func TestTeamInviteDomainProjection(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	raw := db.TeamInvite{
		ID: id, TeamID: 4, InvitedEmail: "member@example.test",
		InvitedByAccountID: 5, TokenHash: []byte("secret"),
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	got := TeamInviteFromDB(raw)
	if got.ID != id || got.TeamID != raw.TeamID ||
		got.InvitedEmail != raw.InvitedEmail || got.InvitedByAccountID != raw.InvitedByAccountID ||
		got.ExpiresAt != raw.ExpiresAt || got.CreatedAt != raw.CreatedAt {
		t.Fatalf("SQLC TeamInvite mapping lost metadata: %+v", got)
	}
}
