-- name: CreatePasswordResetToken :exec
INSERT INTO password_reset_tokens (token_hash, account_id, expires_at)
VALUES (sqlc.arg(token_hash), sqlc.arg(account_id), now() + interval '15 minutes');

-- name: ConsumePasswordResetToken :one
DELETE FROM password_reset_tokens
WHERE token_hash = sqlc.arg(token_hash) AND expires_at > now()
RETURNING account_id;

-- name: DeletePasswordResetTokensForAccount :exec
DELETE FROM password_reset_tokens WHERE account_id = $1;

-- name: UpsertPasswordCredential :exec
INSERT INTO password_credentials (account_id, password_hash)
VALUES (sqlc.arg(account_id), sqlc.arg(password_hash))
ON CONFLICT (account_id) DO UPDATE
SET password_hash = EXCLUDED.password_hash, updated_at = now();

-- name: DeleteSessionsForAccount :exec
DELETE FROM sessions WHERE account_id = $1;
