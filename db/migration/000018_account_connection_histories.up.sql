-- Account, not Person or Party, owns authentication history.
-- Keep the observed session identifier even after an account session is pruned:
-- deliberately no foreign key to sessions, which may be deleted on password reset.
CREATE TABLE account_connection_histories (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('login')),
    ip_address text NOT NULL CHECK (length(ip_address) <= 45),
    user_agent text NOT NULL CHECK (length(user_agent) <= 2048),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX account_connection_histories_account_recent_idx
    ON account_connection_histories(account_id, created_at DESC, id DESC);
