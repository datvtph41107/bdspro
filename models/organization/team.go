// Package organization declares domain values and invariants independently
// of HTTP routers, SQLC query types and infrastructure session handling.
package organization

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/datvtph41107/bdspro/modules/util"
)

type Team struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

var (
	ErrInvalidTeamName = util.WithPublic(util.ErrInvalidArgument, "team name must contain 1–100 characters", "invalid_team_name", "team name must contain 1–100 characters")
	ErrTeamNotFound    = util.WithPublic(util.ErrNotExist, "team not found or inaccessible", "team_not_found", "team not found")
	ErrTeamForbidden   = util.WithPublic(util.ErrPermissionDenied, "organization owner required", "team_permission_denied", "organization owner permission required")
	ErrTeamExists      = util.WithPublic(util.ErrAlreadyExist, "team name already exists", "team_exists", "team name already exists in the organization")
	ErrTeamDescription = util.WithPublic(util.ErrInvalidArgument, "team description too long", "invalid_team_description", "team description must be at most 2000 bytes")
)

func NormalizeTeamName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	count := utf8.RuneCountInString(name)
	if count == 0 || count > 100 {
		return "", ErrInvalidTeamName
	}
	return name, nil
}
