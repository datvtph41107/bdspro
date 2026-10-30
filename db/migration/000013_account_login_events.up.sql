-- A02: append-only password login evidence. Account may be unknown at attempt time.
-- No attempted email, plaintext secret, token or OTP is stored.
CREATE TABLE account_login_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint REFERENCES accounts(id) ON DELETE SET NULL,
    outcome text NOT NULL CHECK (outcome IN ('succeeded', 'rejected')),
    client_ip text NOT NULL CHECK (length(client_ip) <= 64),
    user_agent text NOT NULL CHECK (length(user_agent) <= 512),
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX account_login_events_account_time_idx
    ON account_login_events (account_id, occurred_at DESC, id DESC)
    WHERE account_id IS NOT NULL;
CREATE INDEX account_login_events_expiration_idx
    ON account_login_events (occurred_at);
