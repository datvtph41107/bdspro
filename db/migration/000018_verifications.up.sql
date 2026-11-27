CREATE TABLE verifications (
    kind text NOT NULL CHECK (kind IN ('email', 'phone')),
    id uuid NOT NULL,
    destination text NOT NULL,
    browser_digest bytea NOT NULL CHECK (octet_length(browser_digest) = 32),
    code_mac bytea CHECK (octet_length(code_mac) = 32),
    proof_digest bytea CHECK (octet_length(proof_digest) = 32),
    expires_at timestamptz NOT NULL,
    resend_after timestamptz NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, id),
    UNIQUE (kind, proof_digest),
    CHECK ((code_mac IS NULL) <> (proof_digest IS NULL)),
    CHECK ((kind = 'email' AND length(destination) BETWEEN 3 AND 254 AND destination = lower(btrim(destination)))
        OR (kind = 'phone' AND destination ~ '^\+[1-9][0-9]{7,14}$'))
);
CREATE INDEX verifications_expiry_idx ON verifications(kind, expires_at);
