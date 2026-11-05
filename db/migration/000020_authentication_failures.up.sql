-- Rejected authentication proofs only. Successful logins have their own
-- atomic session / account_connection_histories transaction.
CREATE TABLE authentication_failures (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint REFERENCES accounts(id) ON DELETE CASCADE,
    method text NOT NULL CHECK (method IN ('password', 'phone_otp', 'oidc')),
    reason text NOT NULL,
    credential_mac bytea NOT NULL CHECK (octet_length(credential_mac) = 32),
    ip_address text NOT NULL CHECK (length(ip_address) <= 45),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT authentication_failures_reason_check CHECK (
        (method = 'password' AND reason IN ('unknown_account', 'password_unset', 'invalid_password')) OR
        (method = 'phone_otp' AND reason IN ('unknown_account', 'invalid_otp')) OR
        (method = 'oidc' AND reason IN ('invalid_identity', 'email_unverified', 'link_required'))
    )
);

-- History is scoped to an account only when identity is known.
CREATE INDEX authentication_failures_account_recent_idx
    ON authentication_failures(account_id, occurred_at DESC, id DESC)
    WHERE account_id IS NOT NULL;

-- Correlate credential-targeted attempts without retaining raw email or phone.
CREATE INDEX authentication_failures_credential_recent_idx
    ON authentication_failures(credential_mac, occurred_at DESC, id DESC);

-- The account-scoped indexes cannot serve global expiry scans efficiently.
CREATE INDEX authentication_failures_retention_idx
    ON authentication_failures(occurred_at, id);
