CREATE TABLE oauth2_credentials (
    issuer text NOT NULL CHECK (length(issuer) > 0),
    subject text NOT NULL CHECK (length(subject) > 0),
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    linked_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (issuer, subject)
);

CREATE INDEX oauth2_credentials_account_idx ON oauth2_credentials(account_id);

CREATE UNIQUE INDEX oauth2_credentials_account_issuer_uq ON oauth2_credentials(account_id,issuer);
