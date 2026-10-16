-- name: CreateEmailVerificationToken :exec
INSERT INTO email_verification_tokens (account_email_id, token_hash, expires_at)
VALUES (sqlc.arg(account_email_id), sqlc.arg(token_hash), now() + interval '30 minutes');

-- name: ConsumeEmailVerificationToken :one
UPDATE email_verification_tokens SET consumed_at = now()
WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
RETURNING account_email_id;

-- name: MarkAccountEmailVerified :exec
UPDATE account_emails SET verified_at = now()
WHERE id = $1 AND verified_at IS NULL;
