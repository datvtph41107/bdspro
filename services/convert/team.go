package convert

import (
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
)

// ToTeam converts the model owned by models/organization to the public API.
// This is the sole Team converter; no generated SQLC type crosses the boundary.
func ToTeam(team *organization_model.Team) structs.Team {
	return structs.Team{
		ID:             team.ID,
		OrganizationID: team.OrgID,
		Name:           team.Name,
		Description:    team.Description,
		CreatedAt:      team.CreatedAt,
		UpdatedAt:      team.UpdatedAt,
	}
}

// ToTeams converts domain models from the SQLC-authorized model read.
// The HTTP codec retains the existing empty-list JSON behavior.
func ToTeams(teams []*organization_model.Team) []structs.Team {
	result := make([]structs.Team, 0, len(teams))
	for _, team := range teams {
		result = append(result, ToTeam(team))
	}
	return result
}

func ToTeamMember(member *organization_model.TeamMember) structs.TeamMember {
	return structs.TeamMember{
		TeamID: member.TeamID, PersonID: member.PersonID, JoinedAt: member.JoinedAt,
	}
}

func ToTeamMembers(members []*organization_model.TeamMember) []structs.TeamMember {
	out := make([]structs.TeamMember, 0, len(members))
	for _, member := range members {
		out = append(out, ToTeamMember(member))
	}
	return out
}
