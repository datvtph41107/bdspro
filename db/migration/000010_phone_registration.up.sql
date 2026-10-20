BEGIN;

-- Anonymous phone registration exists before there is an Account.
-- The random challenge handle is stored by SHA-256 digest; OTP by keyed HMAC.
-- This table cannot create identities or grant a Session by itself.
CREATE TABLE phone_registration_challenges (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    phone TEXT NOT NULL CHECK (phone ~ '^\+[1-9][0-9]{7,14}$'),
    challenge_hash TEXT NOT NULL UNIQUE,
    nonce TEXT NOT NULL,
    otp_mac TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '5 minutes'),
    consumed_at TIMESTAMPTZ,
    CHECK (expires_at > created_at)
);

CREATE INDEX phone_registration_recent_idx
ON phone_registration_challenges (phone, created_at DESC);

COMMIT;
