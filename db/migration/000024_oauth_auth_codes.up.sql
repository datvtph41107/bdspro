CREATE TABLE oauth_auth_codes (
    code_hash bytea PRIMARY KEY CHECK (octet_length(code_hash) = 32),
    client_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    redirect_uri text NOT NULL,
    scope text NOT NULL,
    code_challenge text NOT NULL CHECK (char_length(code_challenge) = 43),
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX oauth_auth_codes_expiry_idx ON oauth_auth_codes(expires_at);
