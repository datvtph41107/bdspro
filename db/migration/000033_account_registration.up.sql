ALTER TABLE accounts
    ADD COLUMN username text,
    ADD COLUMN status text NOT NULL DEFAULT 'active',
    ADD COLUMN terms_version text,
    ADD COLUMN terms_accepted_at timestamptz,
    ADD COLUMN marketing_opt_in boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT accounts_username_check CHECK (username IS NULL OR (username ~ '^[a-z0-9_-]{3,30}$' AND username NOT IN ('admin','system','support'))),
    ADD CONSTRAINT accounts_status_check CHECK (status IN ('active','suspended')),
    ADD CONSTRAINT accounts_terms_check CHECK ((terms_version IS NULL) = (terms_accepted_at IS NULL) AND (terms_version IS NULL OR length(terms_version) BETWEEN 1 AND 64));

CREATE UNIQUE INDEX accounts_username_uq ON accounts(username) WHERE username IS NOT NULL;

CREATE TABLE email_registration_challenges (
    id uuid PRIMARY KEY,
    email text NOT NULL CHECK (length(email) BETWEEN 3 AND 254 AND email=lower(btrim(email))),
    browser_digest bytea NOT NULL CHECK (octet_length(browser_digest)=32),
    code_mac bytea CHECK (octet_length(code_mac)=32),
    proof_digest bytea UNIQUE CHECK (octet_length(proof_digest)=32),
    expires_at timestamptz NOT NULL,
    resend_after timestamptz NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((code_mac IS NULL) <> (proof_digest IS NULL))
);
CREATE INDEX email_registration_expiry_idx ON email_registration_challenges(expires_at);
