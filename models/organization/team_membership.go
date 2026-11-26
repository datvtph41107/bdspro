package organization

import (
	"time"

	"github.com/datvtph41107/bdspro/modules/util"
)

type TeamMembership struct {
	TeamID   string    `json:"team_id"`
	PersonID string    `json:"person_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type TeamInvitation struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

var (
	ErrTeamMemberNotFound = util.WithPublic(util.ErrNotExist, "team member missing or ineligible", "team_member_not_found", "team member not found or ineligible")
	ErrTeamMemberExists   = util.WithPublic(util.ErrAlreadyExist, "team member already exists", "team_member_exists", "person is already a team member")
	ErrTeamInviteInvalid  = util.WithPublic(util.ErrInvalidArgument, "invalid or unavailable invitation", "invalid_team_invite", "invitation is invalid or unavailable")
	ErrTeamInviteExists   = util.WithPublic(util.ErrAlreadyExist, "open team invitation exists", "team_invite_exists", "an open invitation already exists")
	ErrTeamInviteNotFound = util.WithPublic(util.ErrNotExist, "team invite not found or unavailable", "team_invite_not_found", "invitation not found or unavailable")
	ErrTeamInviteDuration = util.WithPublic(util.ErrInvalidArgument, "invalid team invitation expiry", "invalid_invite_expiry", "invitation lifetime must be between 300 and 86400 seconds")
)

func TeamInviteDuration(seconds int) (time.Duration, error) {
	if seconds == 0 {
		seconds = 86400
	}
	if seconds < 300 || seconds > 86400 {
		return 0, ErrTeamInviteDuration
	}
	return time.Duration(seconds) * time.Second, nil
}
