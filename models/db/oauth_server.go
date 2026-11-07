package db

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrOAuthGrantInvalid  = errors.New("invalid or expired OAuth grant")
	ErrOAuthGrantReplay   = errors.New("OAuth refresh token replay detected")
	ErrOAuthClientMissing = errors.New("OAuth client unavailable")
)

// OAuthGrant is an authorized subject and the immutable grant scope. It is
// not a bearer credential and never contains a secret.
type OAuthGrant struct {
	AccountID        int64
	Scope            string
	RefreshExpiresAt time.Time
}

type OAuthAccess struct {
	AccountID int64
	Scope     string
}

// OAuthAuthorizedClient represents an active account/client relationship.
// It is NOT an immutable record of historic consent.
type OAuthAuthorizedClient struct {
	ClientID uuid.UUID
	Name     string
}

type OAuthServerStore interface {
	CreateOAuthClient(context.Context, OauthClient) (OauthClient, error)
	GetOAuthClient(context.Context, uuid.UUID) (OauthClient, error)
	ListOAuthClients(context.Context, int64) ([]OauthClient, error)
	RevokeOAuthClient(context.Context, int64, uuid.UUID) (bool, error)
	GetOAuthConsentVersion(context.Context, int64, uuid.UUID) (int64, error)
	ApproveOAuthAuthCode(context.Context, uuid.UUID, int64, string, string, string, []byte, time.Time, int64, uuid.UUID) error
	RedeemOAuthAuthCode(context.Context, uuid.UUID, []byte, string, string, []byte, []byte, time.Time, time.Time, uuid.UUID) (OAuthGrant, error)
	RotateOAuthRefreshToken(context.Context, uuid.UUID, []byte, []byte, []byte, time.Time) (OAuthGrant, error)
	ResolveOAuthAccessToken(context.Context, []byte) (OAuthAccess, error)
	RevokeOAuthBearer(context.Context, uuid.UUID, []byte) error
	ListOAuthAuthorizations(context.Context, int64) ([]OAuthAuthorizedClient, error)
	RevokeOAuthAuthorization(context.Context, int64, uuid.UUID) error
}

func (s *SQLStore) CreateOAuthClient(ctx context.Context, input OauthClient) (OauthClient, error) {
	var result OauthClient
	err := s.pool.QueryRow(ctx, `INSERT INTO oauth_clients
		(id, owner_account_id, name, redirect_uri, secret_hash)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id,owner_account_id,name,redirect_uri,secret_hash,created_at,revoked_at`,
		input.ID, input.OwnerAccountID, input.Name, input.RedirectUri, input.SecretHash,
	).Scan(&result.ID, &result.OwnerAccountID, &result.Name, &result.RedirectUri,
		&result.SecretHash, &result.CreatedAt, &result.RevokedAt)
	return result, err
}

func (s *SQLStore) GetOAuthClient(ctx context.Context, id uuid.UUID) (OauthClient, error) {
	var client OauthClient
	err := s.pool.QueryRow(ctx, `SELECT id,owner_account_id,name,redirect_uri,secret_hash,created_at,revoked_at
		FROM oauth_clients WHERE id=$1 AND revoked_at IS NULL`, id).Scan(
		&client.ID, &client.OwnerAccountID, &client.Name, &client.RedirectUri,
		&client.SecretHash, &client.CreatedAt, &client.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return OauthClient{}, ErrOAuthClientMissing
	}
	return client, err
}

func (s *SQLStore) ListOAuthClients(ctx context.Context, owner int64) ([]OauthClient, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,owner_account_id,name,redirect_uri,secret_hash,created_at,revoked_at
		FROM oauth_clients WHERE owner_account_id=$1 AND revoked_at IS NULL
		ORDER BY created_at,id LIMIT 100`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	clients := make([]OauthClient, 0)
	for rows.Next() {
		var c OauthClient
		if err = rows.Scan(&c.ID, &c.OwnerAccountID, &c.Name, &c.RedirectUri, &c.SecretHash, &c.CreatedAt, &c.RevokedAt); err != nil {
			return nil, err
		}
		// The secret hash must never be returned to the API layer.
		c.SecretHash = nil
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

func (s *SQLStore) RevokeOAuthClient(ctx context.Context, owner int64, id uuid.UUID) (bool, error) {
	result, err := s.pool.Exec(ctx, `UPDATE oauth_clients SET revoked_at=now()
		WHERE id=$1 AND owner_account_id=$2 AND revoked_at IS NULL`, id, owner)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}

// RedeemOAuthAuthCode serializes contenders on one code_hash with row locking.
// An invalid verifier/redirect never consumes a code. The code is consumed
// only if access (and optional refresh) persistence commits in the same tx.
func (s *SQLStore) RedeemOAuthAuthCode(ctx context.Context, clientID uuid.UUID, codeHash []byte, redirect, challenge string, accessHash, refreshHash []byte, accessExpiry, refreshExpiry time.Time, family uuid.UUID) (OAuthGrant, error) {
	var grant OAuthGrant
	err := s.ExecTxRaw(ctx, func(tx pgx.Tx) error {
		// Read the immutable account lock key without a row lock, then
		// re-check the code under the shared account/client serialization.
		var accountID int64
		err := tx.QueryRow(ctx, `SELECT account_id FROM oauth_auth_codes
			WHERE code_hash=$1 AND client_id=$2 AND redirect_uri=$3`,
			codeHash, clientID, redirect).Scan(&accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOAuthGrantInvalid
		}
		if err != nil {
			return err
		}
		if err := lockOAuthAuthorization(ctx, tx, accountID, clientID); err != nil {
			return err
		}
		var lockedAccountID int64
		var scope, expected string
		err = tx.QueryRow(ctx, `SELECT a.account_id,a.scope,a.code_challenge
			FROM oauth_auth_codes a JOIN oauth_clients c ON c.id=a.client_id
            JOIN oauth_grants g ON g.account_id=a.account_id AND g.client_id=a.client_id
                AND g.revoked_at IS NULL AND g.scope=a.scope
			WHERE a.code_hash=$1 AND a.client_id=$2 AND a.redirect_uri=$3
			AND a.used_at IS NULL AND a.expires_at>now() AND c.revoked_at IS NULL
			FOR UPDATE OF a`, codeHash, clientID, redirect).Scan(&lockedAccountID, &scope, &expected)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && lockedAccountID != accountID) {
			return ErrOAuthGrantInvalid
		}
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(challenge)) != 1 {
			return ErrOAuthGrantInvalid
		}
		if _, err = tx.Exec(ctx, `UPDATE oauth_auth_codes SET used_at=now() WHERE code_hash=$1`, codeHash); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO oauth_access_tokens
			(token_hash,client_id,account_id,scope,expires_at)VALUES($1,$2,$3,$4,$5)`,
			accessHash, clientID, accountID, scope, accessExpiry); err != nil {
			return err
		}
		if len(refreshHash) > 0 {
			if _, err = tx.Exec(ctx, `INSERT INTO oauth_refresh_tokens
				(token_hash,access_token_hash,client_id,account_id,family_id,scope,expires_at)
				VALUES($1,$2,$3,$4,$5,$6,$7)`,
				refreshHash, accessHash, clientID, accountID, family, scope, refreshExpiry); err != nil {
				return err
			}
		}
		grant = OAuthGrant{AccountID: accountID, Scope: scope, RefreshExpiresAt: refreshExpiry}
		return nil
	})
	if err != nil {
		return OAuthGrant{}, err
	}
	return grant, nil
}

// A refresh token is one-use. Replaying a consumed token revokes the entire
// refresh family and all access tokens issued by that family.
func (s *SQLStore) RotateOAuthRefreshToken(ctx context.Context, clientID uuid.UUID, oldHash, newAccessHash, newRefreshHash []byte, accessExpiry time.Time) (OAuthGrant, error) {
	var grant OAuthGrant
	var replay bool
	err := s.ExecTxRaw(ctx, func(tx pgx.Tx) error {
		var family uuid.UUID
		var accountID int64
		// Look up immutable lock keys without a row lock. Neither read
		// grants authority: the state is checked again under both locks.
		err := tx.QueryRow(ctx, `SELECT t.family_id,t.account_id
			FROM oauth_refresh_tokens t JOIN oauth_clients c ON c.id=t.client_id
            JOIN oauth_grants g ON g.account_id=t.account_id AND g.client_id=t.client_id
                AND g.revoked_at IS NULL AND g.scope=t.scope
			WHERE t.token_hash=$1 AND t.client_id=$2 AND c.revoked_at IS NULL`,
			oldHash, clientID).Scan(&family, &accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOAuthGrantInvalid
		}
		if err != nil {
			return err
		}
		if err = lockOAuthAuthorization(ctx, tx, accountID, clientID); err != nil {
			return err
		}
		if err = lockOAuthFamily(ctx, tx, family); err != nil {
			return err
		}

		// Re-read under the family lock and lock the token row. The first
		// read is only a lock-key lookup, never an authorization decision.
		var lockedFamily uuid.UUID
		var expires time.Time
		var consumed, revoked pgtype.Timestamptz
		err = tx.QueryRow(ctx, `SELECT t.account_id,t.scope,t.family_id,t.expires_at,t.consumed_at,t.revoked_at
			FROM oauth_refresh_tokens t JOIN oauth_clients c ON c.id=t.client_id
            JOIN oauth_grants g ON g.account_id=t.account_id AND g.client_id=t.client_id
                AND g.revoked_at IS NULL AND g.scope=t.scope
			WHERE t.token_hash=$1 AND t.client_id=$2 AND c.revoked_at IS NULL
			FOR UPDATE OF t`, oldHash, clientID).Scan(&grant.AccountID, &grant.Scope, &lockedFamily, &expires, &consumed, &revoked)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (lockedFamily != family || grant.AccountID != accountID)) {
			return ErrOAuthGrantInvalid
		}
		if err != nil {
			return err
		}
		if consumed.Valid {
			replay = true
			return revokeOAuthFamily(ctx, tx, family)
		}
		if revoked.Valid || !expires.After(time.Now()) {
			return ErrOAuthGrantInvalid
		}
		if _, err = tx.Exec(ctx, `UPDATE oauth_refresh_tokens SET consumed_at=now() WHERE token_hash=$1`, oldHash); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO oauth_access_tokens
			(token_hash,client_id,account_id,scope,expires_at) VALUES($1,$2,$3,$4,$5)`,
			newAccessHash, clientID, grant.AccountID, grant.Scope, accessExpiry); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO oauth_refresh_tokens
			(token_hash,access_token_hash,client_id,account_id,family_id,scope,expires_at)
			VALUES($1,$2,$3,$4,$5,$6,$7)`,
			newRefreshHash, newAccessHash, clientID, grant.AccountID, family, grant.Scope, expires); err != nil {
			return err
		}
		grant.RefreshExpiresAt = expires
		return nil
	})
	if err != nil {
		return OAuthGrant{}, err
	}
	if replay {
		return OAuthGrant{}, ErrOAuthGrantReplay
	}
	return grant, nil
}

// lockOAuthFamily serializes refresh issuance and family revocation. The
// truncated SHA-256 key may collide only at the cost of extra serialization;
// no identity, credential or authority is inferred from the lock key. Always
// acquire it before token row locks. PostgreSQL releases it on commit/rollback.
func lockOAuthFamily(ctx context.Context, tx pgx.Tx, family uuid.UUID) error {
	digest := sha256.Sum256(family[:])
	key := int64(binary.BigEndian.Uint64(digest[:8]))
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1::bigint)`, key)
	return err
}

func revokeOAuthFamily(ctx context.Context, tx pgx.Tx, family uuid.UUID) error {
	if _, err := tx.Exec(ctx, `UPDATE oauth_refresh_tokens SET revoked_at=now()
		WHERE family_id=$1 AND revoked_at IS NULL`, family); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE oauth_access_tokens SET revoked_at=now()
		WHERE revoked_at IS NULL AND token_hash IN
			(SELECT access_token_hash FROM oauth_refresh_tokens WHERE family_id=$1)`, family)
	return err
}

// OAuth access tokens are opaque 256-bit random values, never internal JWTs.
// Every use consults current token, client, account and expiry authority.
func (s *SQLStore) ResolveOAuthAccessToken(ctx context.Context, hash []byte) (OAuthAccess, error) {
	var access OAuthAccess
	err := s.pool.QueryRow(ctx, `SELECT t.account_id,t.scope FROM oauth_access_tokens t
		JOIN oauth_clients c ON c.id=t.client_id
		JOIN accounts a ON a.id=t.account_id
        JOIN oauth_grants g ON g.account_id=t.account_id AND g.client_id=t.client_id
            AND g.revoked_at IS NULL AND g.scope=t.scope
		WHERE t.token_hash=$1 AND t.expires_at>now() AND t.revoked_at IS NULL
		AND c.revoked_at IS NULL`, hash).Scan(&access.AccountID, &access.Scope)
	if errors.Is(err, pgx.ErrNoRows) {
		return OAuthAccess{}, ErrOAuthGrantInvalid
	}
	return access, err
}

// Revoke is idempotent per RFC7009 and does not reveal whether the supplied
// credential exists. Revoking a refresh family also blocks its access tokens.
func (s *SQLStore) RevokeOAuthBearer(ctx context.Context, clientID uuid.UUID, hash []byte) error {
	return s.ExecTxRaw(ctx, func(tx pgx.Tx) error {
		var family uuid.UUID
		var accountID int64
		err := tx.QueryRow(ctx, `SELECT family_id,account_id FROM oauth_refresh_tokens
			WHERE token_hash=$1 AND client_id=$2`, hash, clientID).Scan(&family, &accountID)
		if err == nil {
			// All writes follow account/client -> family -> token row order.
			if err = lockOAuthAuthorization(ctx, tx, accountID, clientID); err != nil {
				return err
			}
			if err = lockOAuthFamily(ctx, tx, family); err != nil {
				return err
			}
			var lockedFamily uuid.UUID
			var lockedAccountID int64
			err = tx.QueryRow(ctx, `SELECT family_id,account_id FROM oauth_refresh_tokens
				WHERE token_hash=$1 AND client_id=$2 FOR UPDATE`, hash, clientID).Scan(&lockedFamily, &lockedAccountID)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // credential was removed during lock acquisition
			}
			if err != nil {
				return err
			}
			if lockedFamily != family || lockedAccountID != accountID {
				return ErrOAuthGrantInvalid
			}
			return revokeOAuthFamily(ctx, tx, family)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// Access-only revoke uses the same account/client serialization.
		err = tx.QueryRow(ctx, `SELECT account_id FROM oauth_access_tokens
			WHERE token_hash=$1 AND client_id=$2`, hash, clientID).Scan(&accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := lockOAuthAuthorization(ctx, tx, accountID, clientID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE oauth_access_tokens SET revoked_at=now()
			WHERE token_hash=$1 AND client_id=$2 AND account_id=$3 AND revoked_at IS NULL`, hash, clientID, accountID)
		return err
	})
}

// ExecTxRaw supports OAuth protocol transactions and atomic password recovery
// that revokes delegated OAuth grants. SQLC remains the default query layer.
func (s *SQLStore) ExecTxRaw(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin oauth transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
