package organization

import (
	"errors"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrTeamNotFound              = util.WithKind(util.ErrNotExist, "team not found")
	ErrTeamNameInvalid           = util.WithKind(util.ErrInvalidArgument, "invalid team name")
	ErrTeamAlreadyExists         = util.WithKind(util.ErrAlreadyExist, "team already exists")
	ErrTeamOwnerRequired         = util.WithKind(util.ErrPermissionDenied, "team owner required")
	ErrTeamMemberExists          = util.WithKind(util.ErrAlreadyExist, "team member already exists")
	ErrTeamMemberNotFound        = util.WithKind(util.ErrNotExist, "team member not found")
	ErrTeamPaginationInvalid     = util.WithKind(util.ErrInvalidArgument, "invalid team pagination")
	ErrTeamInviteInvalid         = util.WithKind(util.ErrInvalidArgument, "team invite invalid or expired")
	ErrTeamInviteExists          = util.WithKind(util.ErrAlreadyExist, "team invitation already exists")
	ErrNameRequired              = util.WithKind(util.ErrInvalidArgument, "organization name required")
	ErrInvalidOrganization       = util.WithKind(util.ErrInvalidArgument, "invalid organization")
	ErrOrganizationNotFound      = util.WithKind(util.ErrNotExist, "organization not found")
	ErrOrganizationOwnerRequired = util.WithKind(util.ErrPermissionDenied, "organization owner required")
	ErrMembershipExists          = util.WithKind(util.ErrAlreadyExist, "membership already exists")
	ErrPersonUnavailable         = util.WithKind(util.ErrNotExist, "person unavailable")
)

// isPGConstraint recognizes an exact database invariant owned by this model.
// SQLSTATE alone is not sufficient: unrelated constraints remain internal.
func isPGConstraint(err error, state, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == state && pgErr.ConstraintName == constraint
}
