package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrOAuthConsentStale = errors.New("OAuth consent version is stale")
var ErrOAuthSessionInactive = errors.New("OAuth authorizing session is missing or revoked")

// GetOAuthConsentVersion is a read-only consent-preview operation. Zero means
// no prior decision; revoked rows retain their version as a durable tombstone.
func (s *SQLStore) GetOAuthConsentVersion(ctx context.Context, accountID int64, clientID uuid.UUID) (int64, error) {
	var version int64
	err := s.pool.QueryRow(ctx, `SELECT version FROM oauth_grants
        WHERE account_id=$1 AND client_id=$2`, accountID, clientID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return version, err
}

// ApproveOAuthAuthCode commits the user's consent decision AND authorization
// code together. A preview from before a revoke cannot be replayed afterward.
// Both approve and revoke hold the account/client advisory lock before rows.
func (s *SQLStore) ApproveOAuthAuthCode(ctx context.Context, clientID uuid.UUID,
	accountID int64, redirect, scope, challenge string, hash []byte,
	expires time.Time, expectedVersion int64, sessionID uuid.UUID) error {
	if expectedVersion < 0 || accountID <= 0 || clientID == uuid.Nil || scope != "profile:read" || sessionID == uuid.Nil {
		return ErrOAuthConsentStale
	}
	return s.ExecTxRaw(ctx, func(tx pgx.Tx) error {
		if err := lockOAuthAuthorization(ctx, tx, accountID, clientID); err != nil {
			return err
		}
		// RevokeOwnedSession, RevokeAllAccountSessions and password reset
		// UPDATE this row. FOR SHARE makes their transaction and this code
		// issuance linearizable: revoked sessions cannot create new codes.
		var lockedSession uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM sessions
			WHERE id=$1 AND account_id=$2 AND is_blocked=FALSE AND expires_at>now()
			FOR SHARE`, sessionID, accountID).Scan(&lockedSession)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOAuthSessionInactive
		}
		if err != nil {
			return err
		}
		var current int64
		err = tx.QueryRow(ctx, `SELECT version FROM oauth_grants
            WHERE account_id=$1 AND client_id=$2`, accountID, clientID).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			current = 0
		} else if err != nil {
			return err
		}
		if current != expectedVersion {
			return ErrOAuthConsentStale
		}

		// Verify client/account/redirect before recording consent. Foreign keys
		// remain the last line of defense if a concurrent owner deletes a client.
		var allowed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(
            SELECT 1 FROM oauth_clients c JOIN accounts a ON a.id=$2
            WHERE c.id=$1 AND c.revoked_at IS NULL AND c.redirect_uri=$3
              AND c.secret_hash IS NULL)`, clientID, accountID, redirect).Scan(&allowed); err != nil {
			return err
		}
		if !allowed {
			return ErrOAuthClientMissing
		}
		if current == 0 {
			_, err = tx.Exec(ctx, `INSERT INTO oauth_grants
                (account_id,client_id,scope,version,approved_at) VALUES($1,$2,$3,1,now())`, accountID, clientID, scope)
		} else {
			_, err = tx.Exec(ctx, `UPDATE oauth_grants SET scope=$3, version=version+1,
                approved_at=now(),revoked_at=NULL,updated_at=now()
                WHERE account_id=$1 AND client_id=$2`, accountID, clientID, scope)
		}
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO oauth_auth_codes
            (code_hash,client_id,account_id,redirect_uri,scope,code_challenge,expires_at)
            VALUES($1,$2,$3,$4,$5,$6,$7)`, hash, clientID, accountID, redirect, scope, challenge, expires)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrOAuthGrantInvalid
		}
		return nil
	})
}
