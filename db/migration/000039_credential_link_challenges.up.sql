CREATE TABLE credential_link_challenges (
    id uuid PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('email','phone')),
    destination text NOT NULL CHECK (length(destination) BETWEEN 3 AND 254),
    code_mac bytea NOT NULL CHECK (octet_length(code_mac)=32),
    expires_at timestamptz NOT NULL,
    resend_after timestamptz NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    UNIQUE(account_id,kind),
    CHECK ((kind='email' AND destination=lower(btrim(destination))) OR
           (kind='phone' AND destination ~ '^\+[1-9][0-9]{7,14}$'))
);
CREATE INDEX credential_link_challenges_expiry_idx ON credential_link_challenges(expires_at);
