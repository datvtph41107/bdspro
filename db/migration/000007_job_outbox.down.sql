-- The legacy schema cannot represent other task types or multiple
-- verification attempts per account. Refuse a lossy rollback.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM job_outbox
        WHERE task_type <> 'task:send_verify_email'
           OR (payload->>'account_id') IS NULL
    ) OR EXISTS (
        SELECT payload->>'account_id' FROM job_outbox
        GROUP BY payload->>'account_id' HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'job_outbox cannot be losslessly downgraded';
    END IF;
END $$;

CREATE TABLE verification_outbox (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint NOT NULL UNIQUE REFERENCES accounts(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    queued_at timestamptz
);
CREATE INDEX verification_outbox_pending_idx
    ON verification_outbox(id) WHERE queued_at IS NULL;

INSERT INTO verification_outbox(id, account_id, created_at, queued_at)
OVERRIDING SYSTEM VALUE
SELECT id, (payload->>'account_id')::bigint, created_at, queued_at
FROM job_outbox;

SELECT setval(
    pg_get_serial_sequence('verification_outbox', 'id'),
    COALESCE((SELECT max(id) FROM verification_outbox), 1),
    EXISTS(SELECT 1 FROM verification_outbox)
);

DROP TABLE job_outbox;
