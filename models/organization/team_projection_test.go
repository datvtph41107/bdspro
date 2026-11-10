package organization

import (
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
)

func TestTeamDomainProjectionAndListSemantics(t *testing.T) {
	created := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	row := db.Team{
		ID: 11, OrganizationID: 12, Name: "Backend", Description: "Services",
		CreatedAt: created, UpdatedAt: updated,
	}
	team := TeamFromDB(row)
	if team.ID != row.ID || team.OrgID != row.OrganizationID ||
		team.Name != row.Name || team.Description != row.Description ||
		team.CreatedAt != created || team.UpdatedAt != updated {
		t.Fatalf("SQLC Team projection changed: %+v", team)
	}
	if teamModelsFromDB(nil) != nil {
		t.Fatal("nil SQLC list must remain nil in domain")
	}
	empty := teamModelsFromDB([]db.Team{})
	if empty == nil || len(empty) != 0 {
		t.Fatalf("allocated empty Team list changed: %+v", empty)
	}
	got := teamModelsFromDB([]db.Team{row})
	if len(got) != 1 || *got[0] != *team {
		t.Fatalf("Team list projection changed: %+v", got)
	}
	row.Name = "Changed in persistence"
	if got[0].Name != "Backend" {
		t.Fatal("domain Team unexpectedly aliases SQLC data")
	}
}

func TestTeamMemberDomainProjectionAndListSemantics(t *testing.T) {
	joined := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	row := db.TeamMember{TeamID: 9, PersonID: 17, JoinedAt: joined}
	member := TeamMemberFromDB(row)
	if member.TeamID != row.TeamID || member.PersonID != row.PersonID || member.JoinedAt != joined {
		t.Fatalf("TeamMember projection changed: %+v", member)
	}
	if teamMembersFromDB(nil) != nil {
		t.Fatal("nil SQLC TeamMember list must remain nil in domain")
	}
	empty := teamMembersFromDB([]db.TeamMember{})
	if empty == nil || len(empty) != 0 {
		t.Fatalf("allocated empty TeamMember list changed: %+v", empty)
	}
	got := teamMembersFromDB([]db.TeamMember{row})
	if len(got) != 1 || *got[0] != *member {
		t.Fatalf("TeamMember list projection changed: %+v", got)
	}
	row.PersonID = 18
	if got[0].PersonID != 17 {
		t.Fatal("domain TeamMember unexpectedly aliases SQLC data")
	}
}
