-- F40: unified audit recorder schema; action definitions belong to Go.
-- This replaces old per-action constraint migrations on a clean database.
CREATE TABLE audit_event (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    action text NOT NULL CHECK (length(action) BETWEEN 3 AND 128),
    actor_id bigint,
    actor_name text NOT NULL DEFAULT '',
    actor_credential text NOT NULL DEFAULT '',
    impersonator_id bigint,
    impersonator_name text NOT NULL DEFAULT '',
    scope_type text NOT NULL CHECK (length(scope_type) BETWEEN 1 AND 64),
    scope_ref text NOT NULL DEFAULT '',
    scope_name text NOT NULL DEFAULT '',
    target_type text NOT NULL DEFAULT '',
    target_ref text NOT NULL DEFAULT '',
    origin text NOT NULL DEFAULT 'system'
        CHECK (length(origin) BETWEEN 1 AND 32),
    message text NOT NULL DEFAULT '' CHECK (length(message) <= 255),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(metadata) = 'object'),
    ip_address inet,
    subject_account_id bigint,
    subject_mac bytea CHECK (subject_mac IS NULL OR octet_length(subject_mac) = 32),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_event_created_idx ON audit_event(created_at DESC, id DESC);
CREATE INDEX audit_event_actor_idx ON audit_event(actor_id, created_at DESC, id DESC);
CREATE INDEX audit_event_scope_idx ON audit_event(scope_type, scope_ref, created_at DESC);
CREATE INDEX audit_event_action_idx ON audit_event(action, created_at DESC);
