package convert

import (
	"testing"
	"time"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
)

func TestOrganizationMemberConversionUsesDomainModel(t *testing.T) {
	joined := time.Date(2026, 11, 9, 11, 0, 0, 0, time.UTC)
	ended := joined.Add(time.Minute)
	domain := &organization_model.Member{
		ID: 1, OrganizationID: 2, PersonID: 3,
		Role: organization_model.RoleMember, JoinedAt: joined, EndedAt: &ended,
	}
	got := ToOrganizationMember(domain)
	if got.ID != 1 || got.OrganizationID != 2 || got.PersonID != 3 ||
		got.Role != organization_model.RoleMember || got.JoinedAt != joined ||
		got.EndedAt == nil || !got.EndedAt.Equal(ended) {
		t.Fatalf("public membership contract changed: %+v", got)
	}
	active := ToActiveOrganizationMembers([]*organization_model.Member{{
		ID: 4, OrganizationID: 2, PersonID: 5, Role: organization_model.RoleOwner,
		JoinedAt: joined,
	}})
	if len(active) != 1 || active[0].ID != 4 || active[0].OrganizationID != 2 ||
		active[0].PersonID != 5 || active[0].Role != organization_model.RoleOwner ||
		active[0].JoinedAt != joined {
		t.Fatalf("public active membership contract changed: %+v", active)
	}
}
