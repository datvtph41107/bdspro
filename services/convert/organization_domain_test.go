package convert

import (
	"testing"
	"time"

	org_model "github.com/datvtph41107/bdspro/models/organization"
)

func TestOrganizationConverterOnlyAcceptsDomainModel(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	model := &org_model.Organization{ID: 31, Name: "Canonical", CreatedAt: now}
	dto := ToOrganization(model)
	if dto.PartyID != model.ID || dto.Name != model.Name || dto.CreatedAt != now {
		t.Fatalf("organization API projection changed: %+v", dto)
	}
	access := org_model.Access{Organization: *model, MembershipID: 41, Role: org_model.RoleOwner, JoinedAt: now}
	view := ToOrganizationAccess(access)
	if view.PartyID != 31 || view.Name != "Canonical" || view.CreatedAt != now ||
		view.Membership.ID != 41 || view.Membership.Role != org_model.RoleOwner ||
		view.Membership.JoinedAt != now {
		t.Fatalf("organization access API projection changed: %+v", view)
	}
}
