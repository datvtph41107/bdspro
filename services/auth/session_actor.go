package auth

import (
	"context"
	"errors"
	"time"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type sessionReader interface {
	GetSession(context.Context, uuid.UUID) (db.Session, error)
}

// ResolveSessionActor verifies a server-authenticated session identifier
// against the live PostgreSQL session before producing a trusted Actor.
//
// IMPORTANT: sessionID must be obtained only after verifying an authenticated
// cookie/session-reference proof. Never accept an arbitrary user-supplied
// UUID as authentication. This function is not an HTTP authentication endpoint.
func ResolveSessionActor(ctx context.Context, sessionID uuid.UUID) (*authmodel.Actor, error) {
	if ctx == nil {
		return nil, errors.New("auth: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sessionID == uuid.Nil {
		return nil, util.ErrUnauthenticated
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	return resolveSessionActor(ctx, q, sessionID, time.Now())
}

func resolveSessionActor(
	ctx context.Context,
	reader sessionReader,
	sessionID uuid.UUID,
	now time.Time,
) (*authmodel.Actor, error) {
	if reader == nil || sessionID == uuid.Nil {
		return nil, util.ErrUnauthenticated
	}
	s, err := reader.GetSession(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, util.ErrUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	if s.ID != sessionID || s.AccountID <= 0 ||
		s.IsBlocked || !s.ExpiresAt.After(now) {
		return nil, util.ErrUnauthenticated
	}
	return &authmodel.Actor{AccountID: s.AccountID, SessionID: s.ID}, nil
}
