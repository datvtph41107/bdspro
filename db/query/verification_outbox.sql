-- name: CreateVerificationOutbox :exec
INSERT INTO verification_outbox (account_id) VALUES ($1);

-- name: ListPendingVerificationOutbox :many
SELECT id, account_id
FROM verification_outbox
WHERE queued_at IS NULL
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkVerificationOutboxQueued :exec
UPDATE verification_outbox
SET queued_at = now()
WHERE id = $1 AND queued_at IS NULL;
