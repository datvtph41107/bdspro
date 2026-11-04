package convert

import (
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/structs"
)

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
