CREATE TABLE email_link_challenges (
    account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    email text NOT NULL CHECK (email = lower(btrim(email)) AND length(email) > 3),
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    expires_at timestamptz NOT NULL,
    next_send_at timestamptz NOT NULL
);
CREATE INDEX email_link_challenges_expiry_idx ON email_link_challenges(expires_at);
