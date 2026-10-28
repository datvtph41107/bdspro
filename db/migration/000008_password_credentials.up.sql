-- A password is an optional sign-in credential, not a property of every account.
CREATE TABLE password_credentials (
    account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    password_hash text NOT NULL CHECK (length(password_hash) > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO password_credentials (account_id, password_hash)
SELECT id, password_hash FROM accounts;

ALTER TABLE accounts DROP COLUMN password_hash;
