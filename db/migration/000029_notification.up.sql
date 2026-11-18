-- F40: persisted user inbox; realtime Pub/Sub is best-effort transport only.
CREATE TABLE notification (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    event_type text NOT NULL CHECK (length(event_type) BETWEEN 1 AND 128),
    source_type text NOT NULL CHECK (length(source_type) BETWEEN 1 AND 64),
    source_ref text NOT NULL CHECK (length(source_ref) BETWEEN 1 AND 128),
    status smallint NOT NULL DEFAULT 1 CHECK (status IN (1,2,3)),
    updated_by_account_id bigint,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(payload) = 'object'),
    dedupe_key text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    read_at timestamptz,
    CONSTRAINT notification_read_state_check
        CHECK (read_at IS NULL OR status IN (2,3))
);
CREATE UNIQUE INDEX notification_dedupe_uq
    ON notification(account_id, dedupe_key) WHERE dedupe_key IS NOT NULL;
CREATE INDEX notification_inbox_idx
    ON notification(account_id, status, updated_at DESC, id DESC);
CREATE INDEX notification_source_idx ON notification(source_type, source_ref);
