package convert

import (
	"testing"
	"time"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
)

func TestTeamConversionUsesDomainModels(t *testing.T) {
	created := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	team := &organization_model.Team{
		ID: 11, OrgID: 12, Name: "Backend", Description: "Services",
		CreatedAt: created, UpdatedAt: updated,
	}
	apiTeam := ToTeam(team)
	if apiTeam.ID != team.ID || apiTeam.OrganizationID != team.OrgID ||
		apiTeam.Name != team.Name || apiTeam.Description != team.Description ||
		apiTeam.CreatedAt != team.CreatedAt || apiTeam.UpdatedAt != team.UpdatedAt {
		t.Fatalf("Team API contract changed: %+v", apiTeam)
	}
	teams := ToTeams([]*organization_model.Team{team})
	if len(teams) != 1 || teams[0] != apiTeam {
		t.Fatalf("Team API list contract changed: %+v", teams)
	}
	if ToTeams(nil) == nil {
		t.Fatal("Team API list formerly emitted [] for nil SQLC list")
	}
	member := &organization_model.TeamMember{
		TeamID: 11, PersonID: 14, JoinedAt: created,
	}
	apiMember := ToTeamMember(member)
	if apiMember.TeamID != member.TeamID ||
		apiMember.PersonID != member.PersonID || apiMember.JoinedAt != member.JoinedAt {
		t.Fatalf("TeamMember API contract changed: %+v", apiMember)
	}
	members := ToTeamMembers([]*organization_model.TeamMember{member})
	if len(members) != 1 || members[0] != apiMember {
		t.Fatalf("TeamMember API list contract changed: %+v", members)
	}
	if ToTeamMembers(nil) == nil {
		t.Fatal("TeamMember API list formerly emitted [] for nil SQLC list")
	}
}
