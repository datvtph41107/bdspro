BEGIN;

-- A login challenge is NOT an authenticated session. It can only be
-- associated with an already verified phone on an existing account.
CREATE TABLE phone_login_challenges (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_phone_id BIGINT NOT NULL REFERENCES account_phones(id),
    challenge_hash TEXT NOT NULL UNIQUE,
    nonce TEXT NOT NULL,
    otp_mac TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '5 minutes'),
    consumed_at TIMESTAMPTZ,
    CHECK (expires_at > created_at)
);
CREATE INDEX phone_login_challenges_recent_idx
    ON phone_login_challenges(account_phone_id, created_at DESC);

COMMIT;
