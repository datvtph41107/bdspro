BEGIN;

-- An E.164 phone can be unverified on multiple accounts, but it can only
-- become verified on one. Verification is completed in a locked transaction.
CREATE UNIQUE INDEX account_phones_verified_number_uq
ON account_phones (phone) WHERE verified_at IS NOT NULL;

CREATE TABLE phone_verification_challenges (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_phone_id BIGINT NOT NULL REFERENCES account_phones(id),
    nonce TEXT NOT NULL,
    otp_mac TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0 AND attempts <= 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '5 minutes'),
    consumed_at TIMESTAMPTZ,
    CHECK (expires_at > created_at)
);
CREATE INDEX phone_challenges_recent_idx
ON phone_verification_challenges (account_phone_id, created_at DESC);

COMMIT;
