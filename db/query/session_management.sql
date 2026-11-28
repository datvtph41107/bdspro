-- name: LockSessionRevocationAccount :one
SELECT id FROM accounts
WHERE id = sqlc.arg(account_id)::bigint
FOR NO KEY UPDATE;

-- name: LockActingSession :one
WITH locked_session AS MATERIALIZED (
    SELECT id, is_blocked, expires_at FROM sessions
    WHERE id = sqlc.arg(session_id)::uuid
      AND account_id = sqlc.arg(account_id)::bigint
    FOR UPDATE
)
SELECT id FROM locked_session
WHERE is_blocked = FALSE AND expires_at > clock_timestamp();

-- name: ListActiveSessions :many
SELECT id, user_agent, client_ip, created_at, expires_at
FROM sessions
WHERE account_id = $1 AND is_blocked = FALSE AND expires_at > now()
ORDER BY created_at DESC, id;

-- name: RevokeOwnedSession :execrows
UPDATE sessions SET is_blocked = TRUE
WHERE id = sqlc.arg(id) AND account_id = sqlc.arg(account_id)
  AND is_blocked = FALSE AND expires_at > now();

-- name: RevokeAllAccountSessions :execrows
UPDATE sessions SET is_blocked = TRUE
WHERE account_id = $1 AND is_blocked = FALSE AND expires_at > now();

-- name: CreateRecentAuthProof :exec
INSERT INTO auth_proofs(digest,account_id,session_id,action,method,email_snapshot,expires_at)
VALUES (sqlc.arg(digest)::bytea,sqlc.arg(account_id)::bigint,sqlc.arg(session_id)::uuid,
    sqlc.arg(action)::text,sqlc.arg(method)::text,sqlc.arg(email_snapshot)::text,clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'));

-- name: ConsumeRecentAuthProof :one
WITH locked_proof AS MATERIALIZED (
    SELECT digest, expires_at FROM auth_proofs
    WHERE digest=sqlc.arg(digest)::bytea AND account_id=sqlc.arg(account_id)::bigint
      AND session_id=sqlc.arg(session_id)::uuid AND action=sqlc.arg(action)::text
    FOR UPDATE
)
DELETE FROM auth_proofs AS p
USING locked_proof AS locked
WHERE p.digest=locked.digest AND locked.expires_at>clock_timestamp()
RETURNING p.method,p.email_snapshot,p.google_subject;

-- name: DeleteAccountRecentAuthProofs :exec
DELETE FROM auth_proofs WHERE account_id=$1;

-- name: DeleteExpiredRecentAuthProofs :execrows
DELETE FROM auth_proofs WHERE digest IN (
    SELECT digest FROM auth_proofs WHERE expires_at<clock_timestamp()
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);

-- name: RotateSessionRefreshDigest :execrows
UPDATE sessions SET refresh_token=sqlc.arg(next_digest)::text
WHERE id=sqlc.arg(session_id)::uuid AND account_id=sqlc.arg(account_id)::bigint
AND refresh_token=sqlc.arg(previous_digest)::text AND NOT is_blocked AND expires_at>clock_timestamp();

-- name: CreateGoogleRecentAuthProof :one
INSERT INTO auth_proofs(digest,account_id,session_id,action,method,google_subject,expires_at)
SELECT sqlc.arg(digest)::bytea,sqlc.arg(account_id)::bigint,sqlc.arg(session_id)::uuid,sqlc.arg(action)::text,
    'google',sqlc.arg(google_subject)::text,LEAST(clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'),to_timestamp(sqlc.arg(auth_time)::bigint)+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'))
WHERE to_timestamp(sqlc.arg(auth_time)::bigint)>clock_timestamp()-(sqlc.arg(freshness_seconds)::integer * interval '1 second')
AND to_timestamp(sqlc.arg(auth_time)::bigint)<=clock_timestamp()+(sqlc.arg(future_skew_seconds)::integer * interval '1 second')
RETURNING expires_at;
