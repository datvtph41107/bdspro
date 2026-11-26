CREATE TABLE phone_continuation_challenges (
    id uuid PRIMARY KEY,
    phone_e164 text NOT NULL CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
    browser_digest bytea NOT NULL CHECK (octet_length(browser_digest)=32),
    code_mac bytea CHECK (octet_length(code_mac)=32),
    proof_digest bytea UNIQUE CHECK (octet_length(proof_digest)=32),
    expires_at timestamptz NOT NULL,
    resend_after timestamptz NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((code_mac IS NULL) <> (proof_digest IS NULL))
);
CREATE INDEX phone_continuation_expiry_idx ON phone_continuation_challenges(expires_at);
