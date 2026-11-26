CREATE TABLE oauth_refresh_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    access_token_hash bytea NOT NULL REFERENCES oauth_access_tokens(token_hash) ON DELETE CASCADE,
    client_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    family_id uuid NOT NULL,
    scope text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX oauth_refresh_tokens_family_idx ON oauth_refresh_tokens(family_id);

CREATE INDEX oauth_refresh_tokens_expiry_idx ON oauth_refresh_tokens(expires_at);
