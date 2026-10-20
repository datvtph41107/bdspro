package organization

import (
	"context"
	"time"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
)

// Workspace is a current navigation choice, never a cached grant of authority.
// Membership and action permissions are rechecked by each organization service.
type Workspace struct {
	OrganizationID   int64     `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	MembershipID     int64     `json:"membership_id"`
	JoinedAt         time.Time `json:"joined_at"`
}

func (s *Service) ListWorkspaces(ctx context.Context, accountID int64) ([]Workspace, error) {
	if err := requireAccount(accountID); err != nil {
		return nil, err
	}
	rows, err := dbsqlc.New(s.Pool).ListActiveAccountWorkspaces(ctx, accountID)
	if err != nil {
		return nil, err
	}
	result := make([]Workspace, 0, len(rows))
	for _, row := range rows {
		result = append(result, Workspace{
			OrganizationID:   row.OrganizationID,
			OrganizationName: row.OrganizationName,
			MembershipID:     row.MembershipID,
			JoinedAt:         row.JoinedAt,
		})
	}
	return result, nil
}
