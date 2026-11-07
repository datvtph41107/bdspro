-- Persistent account/client consent, independent of token expiry.
-- version is an optimistic consent epoch: every approve/revoke advances it.
CREATE TABLE oauth_grants (
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    client_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    scope text NOT NULL CHECK (scope = 'profile:read'),
    version bigint NOT NULL CHECK (version > 0),
    approved_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, client_id)
);
CREATE INDEX oauth_grants_client_idx ON oauth_grants(client_id, account_id);
