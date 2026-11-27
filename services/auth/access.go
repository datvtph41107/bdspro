package auth

import (
	"context"
	"errors"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ResolveAccessToken(ctx context.Context, encoded string) (*auth_model.Actor, error) {
	payload, err := token.ProcessMaker().VerifyToken(encoded, token.TokenTypeAccess)
	if err != nil || payload.SessionID == uuid.Nil {
		return nil, util.ErrUnauthenticated
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	session, err := q.GetSession(ctx, payload.SessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, util.ErrUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	if session.AccountID != payload.AccountID || session.IsBlocked || !session.ExpiresAt.After(time.Now()) {
		return nil, util.ErrUnauthenticated
	}
	if err := requireActiveAccount(ctx, q, session.AccountID); err != nil {
		return nil, err
	}
	return &auth_model.Actor{AccountID: session.AccountID, SessionID: session.ID}, nil
}

// LockActorForMutation requires an active transaction and locks the account
// before validating its active session. Keep this order for credential mutations.
func LockActorForMutation(ctx context.Context, actor auth_model.Actor) (*db.Queries, error) {
	if actor.AccountID <= 0 || actor.SessionID == uuid.Nil || !db.InTransaction(ctx) {
		return nil, util.ErrUnauthenticated
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	_, err = q.LockSessionRevocationAccount(ctx, actor.AccountID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, util.ErrUnauthenticated
	case err != nil:
		return nil, err
	}
	if err := requireActiveAccount(ctx, q, actor.AccountID); err != nil {
		return nil, err
	}
	_, err = q.LockActingSession(ctx, db.LockActingSessionParams{AccountID: actor.AccountID, SessionID: actor.SessionID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, util.ErrUnauthenticated
	case err != nil:
		return nil, err
	}
	return q, nil
}
