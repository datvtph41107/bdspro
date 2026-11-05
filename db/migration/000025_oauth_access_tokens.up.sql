CREATE TABLE oauth_access_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    client_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    scope text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX oauth_access_tokens_account_idx ON oauth_access_tokens(account_id,client_id);

CREATE INDEX oauth_access_tokens_expiry_idx ON oauth_access_tokens(expires_at);
