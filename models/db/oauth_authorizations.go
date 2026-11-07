package db

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// oauthAuthorizationKey is a namespaced, deterministic advisory-lock key.
// Collisions only add serialization; they never grant authorization.
func oauthAuthorizationKey(accountID int64, clientID uuid.UUID) int64 {
	var account [8]byte
	binary.BigEndian.PutUint64(account[:], uint64(accountID))
	h := sha256.New()
	_, _ = h.Write([]byte("oauth.authorization.account_client.v1"))
	_, _ = h.Write(account[:])
	_, _ = h.Write(clientID[:])
	return int64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))
}

// All token/code issuance for the same account/client pair must acquire this
// lock before any row lock; refreshing also acquires the family lock second.
func lockOAuthAuthorization(ctx context.Context, tx pgx.Tx, accountID int64, clientID uuid.UUID) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1::bigint)`, oauthAuthorizationKey(accountID, clientID))
	return err
}

// ListOAuthAuthorizations lists current authorizations, not historical consent.
// The owner of an OAuth client is deliberately NOT considered an authorizer.
func (s *SQLStore) ListOAuthAuthorizations(ctx context.Context, accountID int64) ([]OAuthAuthorizedClient, error) {
	rows, err := s.pool.Query(ctx, `SELECT c.id, c.name
		FROM oauth_clients c
		WHERE c.revoked_at IS NULL AND (
			EXISTS (SELECT 1 FROM oauth_auth_codes ac
				WHERE ac.account_id=$1 AND ac.client_id=c.id
				  AND ac.used_at IS NULL AND ac.expires_at>now())
			OR EXISTS (SELECT 1 FROM oauth_access_tokens a
				WHERE a.account_id=$1 AND a.client_id=c.id
				  AND a.revoked_at IS NULL AND a.expires_at>now())
			OR EXISTS (SELECT 1 FROM oauth_refresh_tokens r
				WHERE r.account_id=$1 AND r.client_id=c.id
				  AND r.consumed_at IS NULL AND r.revoked_at IS NULL AND r.expires_at>now())
		)
		ORDER BY c.created_at DESC, c.id LIMIT 100`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OAuthAuthorizedClient, 0)
	for rows.Next() {
		var entry OAuthAuthorizedClient
		if err := rows.Scan(&entry.ClientID, &entry.Name); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

// RevokeOAuthAuthorization revokes all live credentials and unused codes for
// THIS account/client pair. It never revokes the application or other users.
// Idempotent by design: unknown and already-revoked pairs return nil.
func (s *SQLStore) RevokeOAuthAuthorization(ctx context.Context, accountID int64, clientID uuid.UUID) error {
	return s.ExecTxRaw(ctx, func(tx pgx.Tx) error {
		if err := lockOAuthAuthorization(ctx, tx, accountID, clientID); err != nil {
			return err
		}
		// No revoked_at on auth codes in the canonical schema. Unused codes
		// are removed, so an unredeemed code cannot mint tokens after revoke.
		if _, err := tx.Exec(ctx, `DELETE FROM oauth_auth_codes
			WHERE account_id=$1 AND client_id=$2 AND used_at IS NULL`, accountID, clientID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE oauth_refresh_tokens SET revoked_at=now()
			WHERE account_id=$1 AND client_id=$2 AND revoked_at IS NULL`, accountID, clientID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE oauth_access_tokens SET revoked_at=now()
			WHERE account_id=$1 AND client_id=$2 AND revoked_at IS NULL`, accountID, clientID)
		return err
	})
}
