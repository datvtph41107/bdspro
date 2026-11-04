package organization

// Stored role names are constrained by the organization_members table.
// They identify persisted membership roles, not HTTP permission metadata.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)
