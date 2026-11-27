CREATE TABLE account_login_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('login')),
    ip_address text NOT NULL CHECK (length(ip_address) <= 45),
    user_agent text NOT NULL CHECK (length(user_agent) <= 2048),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX account_login_events_account_recent_idx
    ON account_login_events(account_id, created_at DESC, id DESC);

CREATE INDEX account_login_events_retention_idx
    ON account_login_events(created_at, id);
