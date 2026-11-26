package auth

import (
	"context"
	"errors"
	"strconv"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ListSessions(ctx context.Context, actor authmodel.Actor) ([]authmodel.SessionInfo, error) {
	if actor.AccountID <= 0 || actor.SessionID == uuid.Nil {
		return nil, util.ErrUnauthenticated
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveSessions(ctx, actor.AccountID)
	if err != nil {
		return nil, err
	}
	items := make([]authmodel.SessionInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, authmodel.SessionInfo{
			ID: row.ID, UserAgent: row.UserAgent, ClientIP: row.ClientIp,
			CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt,
			Current: row.ID == actor.SessionID,
		})
	}
	return items, nil
}

func RevokeSession(ctx context.Context, actor authmodel.Actor, target uuid.UUID) error {
	if actor.AccountID <= 0 || actor.SessionID == uuid.Nil {
		return util.ErrUnauthenticated
	}
	if target == uuid.Nil {
		return authmodel.ErrSessionNotFound
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}
		if _, err = q.LockSessionRevocationAccount(txCtx, actor.AccountID); errors.Is(err, pgx.ErrNoRows) {
			return util.ErrUnauthenticated
		} else if err != nil {
			return err
		}
		if err := requireActiveAccount(txCtx, q, actor.AccountID); err != nil {
			return err
		}
		if _, err = q.LockActingSession(txCtx, db.LockActingSessionParams{
			AccountID: actor.AccountID, SessionID: actor.SessionID,
		}); errors.Is(err, pgx.ErrNoRows) {
			return util.ErrUnauthenticated
		} else if err != nil {
			return err
		}
		n, err := q.RevokeOwnedSession(txCtx, db.RevokeOwnedSessionParams{
			ID: target, AccountID: actor.AccountID,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return authmodel.ErrSessionNotFound
		}
		return q.InsertAuditAction(txCtx, db.InsertAuditActionParams{
			Action: "auth.session.revoked", ActorAccountID: actor.AccountID,
			ScopeType: "account", ScopeRef: strconv.FormatInt(actor.AccountID, 10),
			TargetType: "session", TargetRef: target.String(),
		})
	})
}
