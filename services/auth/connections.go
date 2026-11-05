package auth

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
)

// GetConnections matches DRX UserService::getUserAccessHistories while
// retaining BDSPro's Account identity and SQLC ownership predicates.
func (s *AuthService) GetConnections(ctx context.Context, accountID int64) ([]db.ListAccountConnectionHistoriesRow, error) {
	return account_model.GetConnections(ctx, s.store, accountID)
}
