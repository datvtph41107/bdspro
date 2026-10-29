-- name: LockPhoneOnlyEmailAccount :one
SELECT id FROM accounts WHERE id = $1 AND email IS NULL FOR UPDATE;

-- name: EmailLinkAddressTaken :one
SELECT EXISTS (SELECT 1 FROM accounts WHERE lower(btrim(email)) = lower(btrim($1))) AS taken;

-- name: UpsertEmailLinkChallenge :execrows
INSERT INTO email_link_challenges (account_id, email, token_hash, expires_at, next_send_at)
VALUES (sqlc.arg(account_id), sqlc.arg(email), sqlc.arg(token_hash),
        now() + interval '15 minutes', now() + interval '60 seconds')
ON CONFLICT (account_id) DO UPDATE SET
    email = EXCLUDED.email, token_hash = EXCLUDED.token_hash,
    expires_at = EXCLUDED.expires_at, next_send_at = EXCLUDED.next_send_at
WHERE email_link_challenges.next_send_at <= now();

-- name: ConsumeEmailLinkChallenge :one
DELETE FROM email_link_challenges
WHERE account_id = sqlc.arg(account_id)
  AND token_hash = sqlc.arg(token_hash) AND expires_at > now()
RETURNING email;

-- name: AttachVerifiedEmail :execrows
UPDATE accounts SET email = sqlc.arg(email), is_email_verified = TRUE
WHERE id = sqlc.arg(id) AND email IS NULL;
