CREATE TABLE oauth_clients (
    id uuid PRIMARY KEY,
    owner_account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    redirect_uri text NOT NULL CHECK (char_length(redirect_uri) BETWEEN 1 AND 2048),
    secret_hash bytea CHECK (octet_length(secret_hash) = 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);

CREATE INDEX oauth_clients_owner_idx ON oauth_clients(owner_account_id,id);
