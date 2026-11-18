package auth

import (
	"context"
	"errors"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
)

// ListSessions returns only sessions of the caller's verified account.
// Account identity is derived from the authenticated Actor, never from a
// request path, query parameter or user-supplied JSON document.
func ListSessions(ctx context.Context, doer *authmodel.Actor) ([]db.ListActiveSessionsRow, error) {
	if ctx == nil {
		return nil, errors.New("auth: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if doer == nil || doer.AccountID <= 0 || doer.SessionID == ([16]byte{}) {
		return nil, util.ErrUnauthenticated
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	return q.ListActiveSessions(ctx, doer.AccountID)
}
