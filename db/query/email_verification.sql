-- name: CreateEmailVerificationToken :exec
INSERT INTO email_verification_tokens (account_email_id, token_hash, expires_at)
VALUES (sqlc.arg(account_email_id), sqlc.arg(token_hash), now() + interval '30 minutes');

-- name: CreateVerificationResendToken :one
-- Resend is intentionally silent for missing/verified/closed accounts and throttled addresses.
INSERT INTO email_verification_tokens (account_email_id, token_hash, expires_at)
SELECT ae.id, sqlc.arg(token_hash)::text, now() + interval '30 minutes'
FROM account_emails ae
JOIN accounts a ON a.id = ae.account_id
WHERE lower(btrim(ae.email)) = sqlc.arg(email)::text
  AND ae.verified_at IS NULL AND a.closed_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM email_verification_tokens recent
    WHERE recent.account_email_id = ae.id AND recent.created_at > now() - interval '60 seconds'
  )
  AND (
    SELECT count(*) FROM email_verification_tokens daily
    WHERE daily.account_email_id = ae.id AND daily.created_at > now() - interval '24 hours'
  ) < 6
RETURNING account_email_id;

-- name: ConsumeEmailVerificationToken :one
UPDATE email_verification_tokens SET consumed_at = now()
WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
RETURNING account_email_id;

-- name: MarkAccountEmailVerified :exec
UPDATE account_emails SET verified_at = now()
WHERE id = $1 AND verified_at IS NULL;
