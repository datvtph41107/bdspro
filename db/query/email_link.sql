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

-- name: StartCredentialLink :execrows
INSERT INTO contact_link_challenges(id,account_id,session_id,kind,destination,code_mac,expires_at,resend_after)
VALUES (sqlc.arg(id)::uuid,sqlc.arg(account_id)::bigint,sqlc.arg(session_id)::uuid,sqlc.arg(kind)::text,
    sqlc.arg(destination)::text,sqlc.arg(code_mac)::bytea,clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'),clock_timestamp()+(sqlc.arg(cooldown_seconds)::integer * interval '1 second'))
ON CONFLICT (account_id,kind) DO UPDATE SET
    id=EXCLUDED.id,session_id=EXCLUDED.session_id,destination=EXCLUDED.destination,code_mac=EXCLUDED.code_mac,
    expires_at=EXCLUDED.expires_at,resend_after=EXCLUDED.resend_after,
    failed_attempts=CASE WHEN contact_link_challenges.expires_at<=clock_timestamp() THEN 0 ELSE contact_link_challenges.failed_attempts END
WHERE contact_link_challenges.resend_after<=clock_timestamp()
AND (contact_link_challenges.failed_attempts<sqlc.arg(max_failed_attempts)::integer OR contact_link_challenges.expires_at<=clock_timestamp());

-- name: LockCredentialLink :one
SELECT destination,code_mac,expires_at,failed_attempts FROM contact_link_challenges
WHERE id=sqlc.arg(id)::uuid AND account_id=sqlc.arg(account_id)::bigint
AND session_id=sqlc.arg(session_id)::uuid AND kind=sqlc.arg(kind)::text FOR UPDATE;

-- name: ReadCredentialLinkDestination :one
SELECT destination FROM contact_link_challenges
WHERE id=sqlc.arg(id)::uuid AND account_id=sqlc.arg(account_id)::bigint
AND session_id=sqlc.arg(session_id)::uuid AND kind=sqlc.arg(kind)::text;

-- name: FailCredentialLink :exec
UPDATE contact_link_challenges SET failed_attempts=failed_attempts+1 WHERE id=sqlc.arg(id)::uuid AND failed_attempts<sqlc.arg(max_failed_attempts)::integer;

-- name: DeleteAccountCredentialLinks :exec
DELETE FROM contact_link_challenges WHERE account_id=$1;

-- name: DeleteCredentialLink :exec
DELETE FROM contact_link_challenges WHERE id=$1;

-- name: DeleteExpiredCredentialLinks :execrows
DELETE FROM contact_link_challenges WHERE id IN (
    SELECT id FROM contact_link_challenges WHERE expires_at<clock_timestamp()-(sqlc.arg(retention_minutes)::integer * interval '1 minute')
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);

-- name: SetVerifiedAccountEmail :execrows
UPDATE accounts SET email=sqlc.arg(email)::text,is_email_verified=true
WHERE id=sqlc.arg(account_id)::bigint AND status='active';
