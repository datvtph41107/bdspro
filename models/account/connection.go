package account

import (
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

// Connection is an account-scoped login-history model. AccountID stays at the
// model persistence boundary; the SQL query already enforces ownership.
type Connection struct {
	ID        int64
	SessionID uuid.UUID
	Action    string
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}

func connectionsFromDB(rows []db.ListAccountConnectionHistoriesRow) []Connection {
	if rows == nil {
		return nil
	}
	connections := make([]Connection, 0, len(rows))
	for _, row := range rows {
		connections = append(connections, Connection{
			ID:        row.ID,
			SessionID: row.SessionID,
			Action:    row.Action,
			IPAddress: row.IpAddress,
			UserAgent: row.UserAgent,
			CreatedAt: row.CreatedAt,
		})
	}
	return connections
}
