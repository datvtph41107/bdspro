package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"errors"
	audit_model "github.com/datvtph41107/bdspro/models/audit"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/timeutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// SessionStore provides session reads, revocations and the shared SQLC transaction primitive.
// Authentication owns the session + connection-history commit invariant.
type SessionStore interface {
	db.Transactioner
	GetSession(context.Context, uuid.UUID) (db.Session, error)
	ListActiveSessions(context.Context, int64) ([]db.ListActiveSessionsRow, error)
	RevokeOwnedSession(context.Context, db.RevokeOwnedSessionParams) (int64, error)
	RevokeAllAccountSessions(context.Context, int64) (int64, error)
}

// When expectedPasswordHash is supplied, the credential must still match
// inside the session transaction. The session FK holds KEY SHARE on accounts;
// recovery's account FOR UPDATE lock serializes password reset with issuance.
func NewSession(ctx context.Context, store SessionStore, params db.CreateSessionParams, expectedPasswordHash ...string) (db.Session, error) {
	if len(expectedPasswordHash) > 1 {
		return db.Session{}, ErrInvalidCredentials
	}
	return db.WithTx2(db.WithStore(ctx, store), func(txCtx context.Context) (db.Session, error) {
		q, txErr := db.TxQueries(txCtx)
		if txErr != nil {
			return db.Session{}, txErr
		}
		session, err := q.CreateSession(txCtx, params)
		if err != nil {
			return db.Session{}, err
		}
		if err := q.CreateAccountConnectionHistory(txCtx, db.CreateAccountConnectionHistoryParams{
			AccountID: params.AccountID,
			SessionID: session.ID,
			IpAddress: params.ClientIp,
			UserAgent: params.UserAgent,
		}); err != nil {
			return db.Session{}, err
		}
		if len(expectedPasswordHash) == 1 {
			current, err := q.GetPasswordHash(txCtx, params.AccountID)
			if errors.Is(err, pgx.ErrNoRows) {
				return db.Session{}, ErrInvalidCredentials
			}
			if err != nil {
				return db.Session{}, err
			}
			if subtle.ConstantTimeCompare([]byte(current), []byte(expectedPasswordHash[0])) != 1 {
				return db.Session{}, ErrInvalidCredentials
			}
		}
		return session, nil
	})
}

func ValidRefreshSession(session db.Session, accountID int64, refreshToken string, now time.Time) bool {
	return !session.IsBlocked && session.AccountID == accountID &&
		MatchesRefreshToken(session.RefreshToken, refreshToken) && !timeutil.Expired(now, session.ExpiresAt)
}

func ListSessions(ctx context.Context, accountID int64) ([]db.ListActiveSessionsRow, error) {
	store, err := db.StoreFromContext(ctx)
	if err != nil { return nil, err }
	return store.ListActiveSessions(ctx, accountID)
}

// RevokeOwnedSession commits only while the session which initiated this
// action is still active. Always lock the account before the acting session:
// concurrent A->B and B->A revocations otherwise create a row-lock cycle.
// Target ownership and state are enforced by the SQLC UPDATE predicate.
func RevokeOwnedSession(ctx context.Context, accountID int64, actingSessionID, targetSessionID uuid.UUID) error {
	if accountID <= 0 || actingSessionID == uuid.Nil {
		return ErrSessionUnavailable
	}
	if targetSessionID == uuid.Nil {
		return ErrSessionNotFound
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, txErr := db.TxQueries(txCtx)
		if txErr != nil {
			return txErr
		}
		if _, err := q.LockSessionRevocationAccount(txCtx, accountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSessionUnavailable
			}
			return err
		}
		if _, err := q.LockActingSession(txCtx, db.LockActingSessionParams{
			SessionID: actingSessionID, AccountID: accountID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSessionUnavailable
			}
			return err
		}
		// Mutation and security evidence belong to the same transaction.
		// Missing/foreign/previously-revoked targets share one domain error.
		affected, err := q.RevokeOwnedSession(txCtx, db.RevokeOwnedSessionParams{
			ID: targetSessionID, AccountID: accountID,
		})
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrSessionNotFound
		}
		return audit_model.RecordSessionRevoked(txCtx, q, accountID, targetSessionID)
	})
}

// RevokeAllSessions is authorized at commit time, not only at the HTTP
// boundary. An account-level NO KEY UPDATE lock serializes concurrent
// revoke-all requests; the acting session is then locked before any updates.
func RevokeAllSessions(ctx context.Context, accountID int64, actingSessionID uuid.UUID) error {
	if accountID <= 0 || actingSessionID == uuid.Nil {
		return ErrSessionUnavailable
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, txErr := db.TxQueries(txCtx)
		if txErr != nil {
			return txErr
		}
		if _, err := q.LockSessionRevocationAccount(txCtx, accountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSessionUnavailable
			}
			return err
		}
		if _, err := q.LockActingSession(txCtx, db.LockActingSessionParams{
			SessionID: actingSessionID, AccountID: accountID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSessionUnavailable
			}
			return err
		}
		_, err := q.RevokeAllAccountSessions(txCtx, accountID)
		return err
	})
}

const refreshTokenDigestPrefix = "sha256:"

// RefreshTokenDigest stores a one-way fingerprint of a high-entropy signed JWT.
// Clients receive the original bearer token; database sessions persist only this
// fingerprint for newly issued credentials.
func RefreshTokenDigest(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return refreshTokenDigestPrefix + hex.EncodeToString(digest[:])
}

// MatchesRefreshToken supports both fingerprinted sessions and legacy raw JWT
// rows during a bounded migration window. JWT validity and account/session
// authority are checked separately by the caller before this comparison.
func MatchesRefreshToken(stored, raw string) bool {
	if stored == "" || raw == "" {
		return false
	}
	expected := raw
	if strings.HasPrefix(stored, refreshTokenDigestPrefix) {
		expected = RefreshTokenDigest(raw)
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(expected)) == 1
}

// RequireActingSession holds the caller's persisted session row FOR UPDATE
// through a protected write transaction. Call only inside the same transaction
// as the subsequent mutation; HTTP admission is not a commit-time guarantee.
func RequireActingSession(ctx context.Context, q *db.Queries, accountID int64, sessionID uuid.UUID) error {
	if accountID <= 0 || sessionID == uuid.Nil {
		return ErrSessionUnavailable
	}
	_, err := q.LockActingSession(ctx, db.LockActingSessionParams{
		AccountID: accountID, SessionID: sessionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionUnavailable
	}
	return err
}

// RequireActiveSession validates persisted session authority for session-bound
// access JWTs, including first-party protected API admission and OAuth consent.
func RequireActiveSession(ctx context.Context, reader interface {
	GetSession(context.Context, uuid.UUID) (db.Session, error)
}, accountID int64, sessionID uuid.UUID, now time.Time) error {
	if accountID <= 0 || sessionID == uuid.Nil {
		return ErrSessionUnavailable
	}
	session, err := reader.GetSession(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionUnavailable
	}
	if err != nil {
		return err
	}
	if session.ID != sessionID || session.AccountID != accountID || session.IsBlocked ||
		!session.ExpiresAt.After(now) {
		return ErrSessionUnavailable
	}
	return nil
}
