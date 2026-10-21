-- Lecture 37 canonical baseline. Requires an EMPTY database, or the verified
-- backup/reset procedure in scripts/reset-bdspro-lecture37.sh.
CREATE TABLE parties (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);
CREATE TABLE persons (
    party_id BIGINT PRIMARY KEY REFERENCES parties(id)
);
CREATE TABLE accounts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    person_id BIGINT NOT NULL UNIQUE REFERENCES persons(party_id),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX accounts_email_login_uq ON accounts (lower(btrim(email)));
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    refresh_token TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    client_ip TEXT NOT NULL,
    is_blocked BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_account_idx ON sessions(account_id);
