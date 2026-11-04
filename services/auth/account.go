package auth

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
)

// CurrentAccount reads persisted identity instead of echoing a JWT claim.
// All credentials converge on the same account representation.
func (s *AuthService) CurrentAccount(ctx context.Context, id int64) (db.Account, error) {
	return account_model.GetByID(ctx, s.store, id)
}
