CREATE TABLE audit_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint REFERENCES accounts(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('auth.login.rejected', 'auth.session.revoked', 'team.member.removed')),
    method text CHECK (method IN ('password', 'phone_otp', 'oidc')),
    reason text,
    subject_mac bytea CHECK (octet_length(subject_mac) = 32),
    ip_address text NOT NULL CHECK (length(ip_address) <= 45),
    created_at timestamptz NOT NULL DEFAULT now(),
    actor_account_id bigint,
    scope_type text,
    scope_ref text,
    target_type text,
    target_ref text,
    CONSTRAINT audit_events_payload_check CHECK (
        (
            action = 'auth.login.rejected' AND actor_account_id IS NULL AND
            scope_type IS NULL AND scope_ref IS NULL AND target_type IS NULL AND target_ref IS NULL AND
            method IS NOT NULL AND reason IS NOT NULL AND method IN ('password', 'phone_otp', 'oidc') AND
            subject_mac IS NOT NULL AND octet_length(subject_mac) = 32 AND
            ((method='password' AND reason IN ('unknown_account','password_unset','invalid_password')) OR
             (method='phone_otp' AND reason IN ('unknown_account','invalid_otp')) OR
             (method='oidc' AND reason IN ('invalid_identity','email_unverified','link_required')))
        ) OR (
            action IN ('auth.session.revoked','team.member.removed') AND
            actor_account_id IS NOT NULL AND actor_account_id > 0 AND account_id IS NULL AND
            method IS NULL AND reason IS NULL AND subject_mac IS NULL AND
            scope_type IS NOT NULL AND scope_ref IS NOT NULL AND target_type IS NOT NULL AND target_ref IS NOT NULL AND
            ((action='auth.session.revoked' AND scope_type='account' AND scope_ref <> '' AND target_type='session' AND target_ref <> '') OR
             (action='team.member.removed' AND scope_type='team' AND scope_ref <> '' AND target_type='person' AND target_ref <> ''))
        )
    )
);

CREATE INDEX audit_events_retention_idx ON audit_events(created_at, id);
