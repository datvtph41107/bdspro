-- Audit evidence is separate from successful Account connection history.
-- A missing account identity is representable without inventing a user.
CREATE TABLE audit_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint REFERENCES accounts(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('auth.login.rejected')),
    method text NOT NULL CHECK (method IN ('password', 'phone_otp', 'oidc')),
    reason text NOT NULL,
    subject_mac bytea NOT NULL CHECK (octet_length(subject_mac) = 32),
    ip_address text NOT NULL CHECK (length(ip_address) <= 45),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_events_login_reason_check CHECK (
        (method = 'password' AND reason IN ('unknown_account', 'password_unset', 'invalid_password')) OR
        (method = 'phone_otp' AND reason IN ('unknown_account', 'invalid_otp')) OR
        (method = 'oidc' AND reason IN ('invalid_identity', 'email_unverified', 'link_required'))
    )
);

-- One index actually used by global expiry; avoid unused write-amplifying indexes.
CREATE INDEX audit_events_retention_idx ON audit_events(created_at, id);
