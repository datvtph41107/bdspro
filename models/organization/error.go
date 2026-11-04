package organization

import "errors"

var (
	ErrNameRequired = errors.New("organization name required")
	ErrInvalidOrganization = errors.New("invalid organization")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrOrganizationOwnerRequired = errors.New("organization owner required")
	ErrMembershipExists = errors.New("membership already exists")
	ErrPersonUnavailable = errors.New("person unavailable")
)
