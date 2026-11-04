package organization

import (
	"fmt"

	"github.com/datvtph41107/bdspro/modules/util"
)

var (
	ErrTeamNotFound              = fmt.Errorf("%w: team not found", util.ErrNotExist)
	ErrTeamNameInvalid           = fmt.Errorf("%w: invalid team name", util.ErrInvalidArgument)
	ErrTeamAlreadyExists         = fmt.Errorf("%w: team already exists", util.ErrAlreadyExist)
	ErrTeamOwnerRequired         = fmt.Errorf("%w: team owner required", util.ErrPermissionDenied)
	ErrTeamMemberExists          = fmt.Errorf("%w: team member already exists", util.ErrAlreadyExist)
	ErrTeamMemberNotFound        = fmt.Errorf("%w: team member not found", util.ErrNotExist)
	ErrTeamPaginationInvalid     = fmt.Errorf("%w: invalid team pagination", util.ErrInvalidArgument)
	ErrTeamInviteInvalid         = fmt.Errorf("%w: team invite invalid or expired", util.ErrInvalidArgument)
	ErrTeamInviteExists          = fmt.Errorf("%w: team invitation already exists", util.ErrAlreadyExist)
	ErrNameRequired              = fmt.Errorf("%w: organization name required", util.ErrInvalidArgument)
	ErrInvalidOrganization       = fmt.Errorf("%w: invalid organization", util.ErrInvalidArgument)
	ErrOrganizationNotFound      = fmt.Errorf("%w: organization not found", util.ErrNotExist)
	ErrOrganizationOwnerRequired = fmt.Errorf("%w: organization owner required", util.ErrPermissionDenied)
	ErrMembershipExists          = fmt.Errorf("%w: membership already exists", util.ErrAlreadyExist)
	ErrPersonUnavailable         = fmt.Errorf("%w: person unavailable", util.ErrNotExist)
)
