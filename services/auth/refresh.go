package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/db/sqlc"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func RefreshSession(ctx context.Context, raw string) (LoginResult, error) {
	proof, err := token.ReadRefreshCredential(setting.Security.TokenSymmetricKey, raw)
	if err != nil {
		return LoginResult{}, util.ErrUnauthenticated
	}
	var reused bool
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, row, err := lockRefreshSession(txCtx, proof.SessionID)
		if err != nil {
			return LoginResult{}, err
		}
		if !token.MatchesRefreshDigest(raw, row.RefreshToken) {
			if err := revokeRefreshRow(txCtx, q, row); err != nil {
				return LoginResult{}, err
			}
			reused = true
			return LoginResult{}, nil
		}
		next, err := token.NewRefreshCredential(setting.Security.TokenSymmetricKey, row.ID, proof.Remember)
		if err != nil {
			return LoginResult{}, err
		}
		result, err := sessionResult(txCtx, row.AccountID, row.ID, row.ExpiresAt, next, proof.Remember)
		if err != nil {
			return LoginResult{}, err
		}
		n, err := q.RotateSessionRefreshDigest(txCtx, sqlc.RotateSessionRefreshDigestParams{
			NextDigest: token.RefreshDigest(next), SessionID: row.ID,
			AccountID: row.AccountID, PreviousDigest: row.RefreshToken,
		})
		if err != nil {
			return LoginResult{}, err
		}
		if n != 1 {
			return LoginResult{}, util.ErrUnauthenticated
		}
		return result, nil
	})
	if err == nil && reused {
		return LoginResult{}, util.ErrUnauthenticated
	}
	return result, err
}

func lockRefreshSession(ctx context.Context, sessionID uuid.UUID) (*sqlc.Queries, sqlc.Session, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, sqlc.Session{}, err
	}
	row, err := q.GetSession(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, sqlc.Session{}, util.ErrUnauthenticated
	}
	if err != nil {
		return nil, sqlc.Session{}, err
	}
	if _, err = q.LockSessionRevocationAccount(ctx, row.AccountID); errors.Is(err, pgx.ErrNoRows) {
		return nil, sqlc.Session{}, util.ErrUnauthenticated
	} else if err != nil {
		return nil, sqlc.Session{}, err
	}
	if err := requireActiveAccount(ctx, q, row.AccountID); err != nil {
		return nil, sqlc.Session{}, err
	}
	if _, err = q.LockActingSession(ctx, sqlc.LockActingSessionParams{AccountID: row.AccountID, SessionID: row.ID}); errors.Is(err, pgx.ErrNoRows) {
		return nil, sqlc.Session{}, util.ErrUnauthenticated
	} else if err != nil {
		return nil, sqlc.Session{}, err
	}
	row, err = q.GetSession(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, sqlc.Session{}, util.ErrUnauthenticated
	}
	if err != nil {
		return nil, sqlc.Session{}, err
	}
	if row.IsBlocked || !row.ExpiresAt.After(time.Now()) {
		return nil, sqlc.Session{}, util.ErrUnauthenticated
	}
	return q, row, nil
}

func revokeRefreshRow(ctx context.Context, q *sqlc.Queries, row sqlc.Session) error {
	n, err := q.RevokeOwnedSession(ctx, sqlc.RevokeOwnedSessionParams{ID: row.ID, AccountID: row.AccountID})
	if err != nil {
		return err
	}
	if n != 1 {
		return util.ErrUnauthenticated
	}
	return q.InsertAuditAction(ctx, sqlc.InsertAuditActionParams{
		Action: "auth.session.revoked", ActorAccountID: row.AccountID,
		ScopeType: "account", ScopeRef: strconv.FormatInt(row.AccountID, 10),
		TargetType: "session", TargetRef: row.ID.String(),
	})
}

func LogoutRefresh(ctx context.Context, raw string) error {
	proof, err := token.ReadRefreshCredential(setting.Security.TokenSymmetricKey, raw)
	if err != nil {
		return util.ErrUnauthenticated
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, row, err := lockRefreshSession(txCtx, proof.SessionID)
		if err != nil {
			return err
		}
		if !token.MatchesRefreshDigest(raw, row.RefreshToken) {
			return util.ErrUnauthenticated
		}
		return revokeRefreshRow(txCtx, q, row)
	})
}
