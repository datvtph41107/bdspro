-- One durable handoff record per job occurrence, independent of its task type.
-- Migrate existing verification intents including their queued state and TaskID.
CREATE TABLE job_outbox (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    task_type text NOT NULL CHECK (length(task_type) > 0),
    payload jsonb NOT NULL,
    queue_name text NOT NULL DEFAULT 'default' CHECK (length(queue_name) > 0),
    max_retry integer NOT NULL DEFAULT 10 CHECK (max_retry >= 0),
    run_at timestamptz NOT NULL DEFAULT now(),
    task_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    queued_at timestamptz
);

CREATE INDEX job_outbox_pending_idx ON job_outbox(id) WHERE queued_at IS NULL;

INSERT INTO job_outbox (
    id, task_type, payload, queue_name, max_retry, run_at, task_id, created_at, queued_at
)
OVERRIDING SYSTEM VALUE
SELECT
    id, 'task:send_verify_email',
    jsonb_build_object('account_id', account_id),
    'critical', 10, created_at + interval '10 seconds',
    'verify-email-outbox-' || id, created_at, queued_at
FROM verification_outbox;

SELECT setval(
    pg_get_serial_sequence('job_outbox', 'id'),
    COALESCE((SELECT max(id) FROM job_outbox), 1),
    EXISTS(SELECT 1 FROM job_outbox)
);

DROP TABLE verification_outbox;
