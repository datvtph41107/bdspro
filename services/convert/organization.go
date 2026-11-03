package convert

import (
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
)

func ToOrganization(v db.Organization) structs.Organization {
	return structs.Organization{PartyID: v.PartyID, Name: v.Name, CreatedAt: v.CreatedAt}
}

// PermissionsForRole is public display metadata, never an authorization source.
func PermissionsForRole(role string) []string {
	switch role {
	case "owner":
		return []string{"organization.view", "organization.update", "organization.members.view", "organization.members.add"}
	case "member":
		return []string{"organization.view"}
	default:
		return []string{}
	}
}

func ToOrganizationAccess(v organization_model.Access) structs.OrganizationAccess {
	return structs.OrganizationAccess{
		PartyID: v.Organization.PartyID, Name: v.Organization.Name, CreatedAt: v.Organization.CreatedAt,
		Membership:  structs.OrganizationMembership{ID: v.MembershipID, Role: v.Role, JoinedAt: v.JoinedAt},
		Permissions: PermissionsForRole(v.Role),
	}
}

func ToOrganizationAccessList(list []organization_model.Access) []structs.OrganizationAccess {
	out := make([]structs.OrganizationAccess, 0, len(list))
	for _, item := range list {
		out = append(out, ToOrganizationAccess(item))
	}
	return out
}

func ToOrganizationMember(v db.OrganizationMember) structs.OrganizationMember {
	result := structs.OrganizationMember{
		ID: v.ID, OrganizationID: v.OrganizationID, PersonID: v.PersonID,
		Role: v.Role, JoinedAt: v.JoinedAt,
	}
	if v.EndedAt.Valid {
		value := v.EndedAt.Time
		result.EndedAt = &value
	}
	return result
}

func ToActiveOrganizationMembers(rows []db.ListOrganizationMembersForOwnerRow) []structs.ActiveOrganizationMember {
	out := make([]structs.ActiveOrganizationMember, 0, len(rows))
	for _, v := range rows {
		out = append(out, structs.ActiveOrganizationMember{
			ID: v.ID, OrganizationID: v.OrganizationID, PersonID: v.PersonID, Role: v.Role, JoinedAt: v.JoinedAt,
		})
	}
	return out
}
