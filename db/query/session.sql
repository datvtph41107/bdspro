-- name: CreateLoginSession :one
INSERT INTO sessions (account_id, token_hash, expires_at)
VALUES (sqlc.arg(account_id), sqlc.arg(token_hash), now() + interval '24 hours')
RETURNING expires_at;

-- name: GetSessionAccount :one
SELECT a.id FROM sessions se JOIN accounts a ON a.id = se.account_id
WHERE se.token_hash = $1 AND se.revoked_at IS NULL AND se.expires_at > now()
  AND a.closed_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = now()
WHERE token_hash = $1 AND revoked_at IS NULL;
