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
INSERT INTO job_outbox (task_type, payload, queue_name, max_retry, run_at)
VALUES ('task:send_password_reset',
        jsonb_build_object('account_id', sqlc.arg(account_id)::bigint),
        'critical', 10, now());
