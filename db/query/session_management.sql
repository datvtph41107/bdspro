-- name: LockSessionRevocationAccount :one
SELECT id FROM accounts
WHERE id = sqlc.arg(account_id)::bigint
FOR NO KEY UPDATE;

-- name: LockActingSession :one
SELECT id FROM sessions
WHERE id = sqlc.arg(session_id)::uuid
  AND account_id = sqlc.arg(account_id)::bigint
  AND is_blocked = FALSE AND expires_at > clock_timestamp()
FOR UPDATE;

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
INSERT INTO recent_auth_proofs(digest,account_id,session_id,action,method,email_snapshot,expires_at)
VALUES (sqlc.arg(digest)::bytea,sqlc.arg(account_id)::bigint,sqlc.arg(session_id)::uuid,
    sqlc.arg(action)::text,sqlc.arg(method)::text,sqlc.arg(email_snapshot)::text,clock_timestamp()+interval '10 minutes');

-- name: ConsumeRecentAuthProof :one
DELETE FROM recent_auth_proofs
WHERE digest=sqlc.arg(digest)::bytea AND account_id=sqlc.arg(account_id)::bigint
AND session_id=sqlc.arg(session_id)::uuid AND action=sqlc.arg(action)::text AND expires_at>clock_timestamp()
RETURNING method,email_snapshot,google_subject;

-- name: DeleteAccountRecentAuthProofs :exec
DELETE FROM recent_auth_proofs WHERE account_id=$1;

-- name: DeleteExpiredRecentAuthProofs :execrows
DELETE FROM recent_auth_proofs WHERE digest IN (
    SELECT digest FROM recent_auth_proofs WHERE expires_at<clock_timestamp()
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);

-- name: RotateSessionRefreshDigest :execrows
UPDATE sessions SET refresh_token=sqlc.arg(next_digest)::text
WHERE id=sqlc.arg(session_id)::uuid AND account_id=sqlc.arg(account_id)::bigint
AND refresh_token=sqlc.arg(previous_digest)::text AND NOT is_blocked AND expires_at>clock_timestamp();

-- name: CreateGoogleRecentAuthProof :execrows
INSERT INTO recent_auth_proofs(digest,account_id,session_id,action,method,google_subject,expires_at)
SELECT sqlc.arg(digest)::bytea,sqlc.arg(account_id)::bigint,sqlc.arg(session_id)::uuid,sqlc.arg(action)::text,
    'google',sqlc.arg(google_subject)::text,LEAST(clock_timestamp()+interval '10 minutes',to_timestamp(sqlc.arg(auth_time)::bigint)+interval '10 minutes')
WHERE to_timestamp(sqlc.arg(auth_time)::bigint)>clock_timestamp()-interval '5 minutes'
AND to_timestamp(sqlc.arg(auth_time)::bigint)<=clock_timestamp()+interval '30 seconds';
