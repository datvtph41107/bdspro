package organization

import "errors"

var (
	ErrTeamNotFound = errors.New("team not found")
	ErrTeamNameInvalid = errors.New("invalid team name")
	ErrTeamAlreadyExists = errors.New("team already exists")
	ErrTeamOwnerRequired = errors.New("team owner required")
	ErrTeamMemberExists = errors.New("team member already exists")
	ErrTeamMemberNotFound = errors.New("team member not found")
	ErrTeamPaginationInvalid = errors.New("invalid team pagination")
	ErrTeamInviteInvalid = errors.New("team invite invalid or expired")
	ErrTeamInviteExists = errors.New("team invitation already exists")
	ErrNameRequired              = errors.New("organization name required")
	ErrInvalidOrganization       = errors.New("invalid organization")
	ErrOrganizationNotFound      = errors.New("organization not found")
	ErrOrganizationOwnerRequired = errors.New("organization owner required")
	ErrMembershipExists          = errors.New("membership already exists")
	ErrPersonUnavailable         = errors.New("person unavailable")
)
