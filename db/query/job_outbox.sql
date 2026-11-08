-- name: RecordEmailVerificationJob :exec
INSERT INTO job_outbox (task_type, payload, queue_name, max_retry, run_at)
VALUES (
    'task:send_verify_email',
    jsonb_build_object('account_id', sqlc.arg(account_id)::bigint),
    'critical', 10, now() + interval '10 seconds'
);

-- name: ListPendingJobOutbox :many
SELECT id, task_type, payload, queue_name, max_retry, run_at, task_id
FROM job_outbox
WHERE queued_at IS NULL
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkJobOutboxQueued :exec
UPDATE job_outbox
SET queued_at = now()
WHERE id = $1 AND queued_at IS NULL;

-- name: RecordPasswordResetJob :exec
-- Snapshot the account's recovery version under its row lock.
-- Never serialize a reset proof into the outbox.
INSERT INTO job_outbox (task_type, payload, queue_name, max_retry, run_at)
SELECT 'task:send_password_reset',
       jsonb_build_object('account_id', id, 'reset_version', password_reset_version),
       'critical', 10, now()
FROM accounts
WHERE id = sqlc.arg(account_id)::bigint AND is_email_verified
FOR SHARE;

-- name: RecordTeamInviteJob :exec
-- Only the invite UUID is durable. The proof is derived inside the worker
-- using the configured signing key, never persisted in plaintext.
INSERT INTO job_outbox (task_type, payload, queue_name, max_retry, run_at)
VALUES (
    'task:send_team_invite',
    jsonb_build_object('invite_id', sqlc.arg(invite_id)::uuid),
    'critical', 10, now()
);
