package account

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
)

// GetConnections reads the current account's login events, newest first.
// Account ownership is enforced in SQL, never via a caller-supplied filter.
func GetConnections(ctx context.Context, store interface {
	ListAccountConnectionHistories(context.Context, int64) ([]db.ListAccountConnectionHistoriesRow, error)
}, accountID int64) ([]db.ListAccountConnectionHistoriesRow, error) {
	return store.ListAccountConnectionHistories(ctx, accountID)
}
