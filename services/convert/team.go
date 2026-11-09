package convert

import (
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
)

// ToTeamModel is the domain-model to public-API conversion. Existing DB
// conversions stay until every Team endpoint has migrated.
func ToTeamModel(team *organization_model.Team) structs.Team {
	return structs.Team{
		ID:             team.ID,
		OrganizationID: team.OrgID,
		Name:           team.Name,
		Description:    team.Description,
		CreatedAt:      team.CreatedAt,
		UpdatedAt:      team.UpdatedAt,
	}
}

func ToTeam(team db.Team) structs.Team {
	return structs.Team{
		ID: team.ID, OrganizationID: team.OrganizationID, Name: team.Name,
		Description: team.Description, CreatedAt: team.CreatedAt, UpdatedAt: team.UpdatedAt,
	}
}

func ToTeams(teams []db.Team) []structs.Team {
	result := make([]structs.Team, 0, len(teams))
	for _, team := range teams {
		result = append(result, ToTeam(team))
	}
	return result
}

func ToTeamMember(member db.TeamMember) structs.TeamMember {
	return structs.TeamMember{
		TeamID: member.TeamID, PersonID: member.PersonID, JoinedAt: member.JoinedAt,
	}
}

func ToTeamMembers(members []db.TeamMember) []structs.TeamMember {
	out := make([]structs.TeamMember, 0, len(members))
	for _, member := range members {
		out = append(out, ToTeamMember(member))
	}
	return out
}
