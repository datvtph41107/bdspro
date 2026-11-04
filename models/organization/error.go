package organization

import (
	"github.com/datvtph41107/bdspro/modules/util"
)

var (
	ErrNameRequired              = util.WithKind(util.ErrInvalidArgument, "organization name required")
	ErrInvalidOrganization       = util.WithKind(util.ErrInvalidArgument, "invalid organization")
	ErrOrganizationNotFound      = util.WithKind(util.ErrNotExist, "organization not found")
	ErrOrganizationOwnerRequired = util.WithKind(util.ErrPermissionDenied, "organization owner required")
	ErrMembershipExists          = util.WithKind(util.ErrAlreadyExist, "membership already exists")
	ErrPersonUnavailable         = util.WithKind(util.ErrNotExist, "person unavailable")
)
