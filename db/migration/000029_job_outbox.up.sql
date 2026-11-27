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
