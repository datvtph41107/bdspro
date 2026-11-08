package organization

import (
	"errors"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrTeamNotFound               = util.WithPublic(util.ErrNotExist, "team not found", "organization.team_not_found", "team not found")
	ErrTeamNameInvalid            = util.WithPublic(util.ErrInvalidArgument, "invalid team name", "organization.team_name_invalid", "invalid team name")
	ErrTeamAlreadyExists          = util.WithPublic(util.ErrAlreadyExist, "team already exists", "organization.team_exists", "team already exists")
	ErrTeamOwnerRequired          = util.WithPublic(util.ErrPermissionDenied, "team owner required", "organization.team_owner_required", "organization owner required")
	ErrTeamMemberExists           = util.WithPublic(util.ErrAlreadyExist, "team member already exists", "organization.team_member_exists", "team member already exists")
	ErrTeamMemberNotFound         = util.WithPublic(util.ErrNotExist, "team member not found", "organization.team_member_not_found", "team member not found")
	ErrTeamPaginationInvalid      = util.WithPublic(util.ErrInvalidArgument, "invalid team pagination", "organization.team_pagination_invalid", "invalid team pagination")
	ErrTeamInviteInvalid          = util.WithPublic(util.ErrInvalidArgument, "team invite invalid or expired", "organization.team_invite_invalid", "invalid or expired invitation")
	ErrTeamInviteExists           = util.WithPublic(util.ErrAlreadyExist, "team invitation already exists", "organization.team_invite_exists", "team invitation already exists")
	ErrNameRequired               = util.WithPublic(util.ErrInvalidArgument, "organization name required", "http.name_required", "name required")
	ErrInvalidOrganization        = util.WithPublic(util.ErrInvalidArgument, "invalid organization", "http.invalid_organization", "invalid organization")
	ErrOrganizationNotFound       = util.WithPublic(util.ErrNotExist, "organization not found", "organization.not_found", "organization not found")
	ErrOrganizationOwnerRequired  = util.WithPublic(util.ErrPermissionDenied, "organization owner required", "organization.owner_required", "organization owner required")
	ErrMembershipExists           = util.WithPublic(util.ErrAlreadyExist, "membership already exists", "organization.membership_exists", "membership already exists")
	ErrOrganizationMemberNotFound = util.WithPublic(util.ErrNotExist, "organization member not found or not removable", "organization.member_not_found", "organization member not found")
	ErrPersonUnavailable          = util.WithPublic(util.ErrNotExist, "person unavailable", "organization.person_unavailable", "person unavailable")
)

// isPGConstraint recognizes an exact database invariant owned by this model.
// SQLSTATE alone is not sufficient: unrelated constraints remain internal.
func isPGConstraint(err error, state, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == state && pgErr.ConstraintName == constraint
}
