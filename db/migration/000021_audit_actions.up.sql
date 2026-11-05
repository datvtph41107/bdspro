-- Evolve the existing security audit table; never create per-action history tables.
-- Retain all R35.1 rejected-login rows and their scoped HMAC correlation.
ALTER TABLE audit_events
    ADD COLUMN actor_account_id bigint REFERENCES accounts(id) ON DELETE SET NULL,
    ADD COLUMN scope_type text,
    ADD COLUMN scope_ref text,
    ADD COLUMN target_type text,
    ADD COLUMN target_ref text;

ALTER TABLE audit_events
    ALTER COLUMN method DROP NOT NULL,
    ALTER COLUMN reason DROP NOT NULL,
    ALTER COLUMN subject_mac DROP NOT NULL;

ALTER TABLE audit_events DROP CONSTRAINT audit_events_action_check;
ALTER TABLE audit_events DROP CONSTRAINT audit_events_login_reason_check;

ALTER TABLE audit_events
    ADD CONSTRAINT audit_events_action_check CHECK (
        action IN ('auth.login.rejected', 'auth.session.revoked', 'team.member.removed')
    ),
    ADD CONSTRAINT audit_events_payload_check CHECK (
        (
            action = 'auth.login.rejected'
            AND actor_account_id IS NULL
            AND scope_type IS NULL AND scope_ref IS NULL
            AND target_type IS NULL AND target_ref IS NULL
            AND method IN ('password', 'phone_otp', 'oidc')
            AND subject_mac IS NOT NULL AND octet_length(subject_mac) = 32
            AND (
                (method = 'password' AND reason IN ('unknown_account', 'password_unset', 'invalid_password')) OR
                (method = 'phone_otp' AND reason IN ('unknown_account', 'invalid_otp')) OR
                (method = 'oidc' AND reason IN ('invalid_identity', 'email_unverified', 'link_required'))
            )
        )
        OR
        (
            action IN ('auth.session.revoked', 'team.member.removed')
            AND actor_account_id > 0
            AND account_id IS NULL
            AND method IS NULL AND reason IS NULL AND subject_mac IS NULL
            AND (
                (action = 'auth.session.revoked'
                    AND scope_type = 'account' AND scope_ref <> ''
                    AND target_type = 'session' AND target_ref <> '')
                OR
                (action = 'team.member.removed'
                    AND scope_type = 'team' AND scope_ref <> ''
                    AND target_type = 'person' AND target_ref <> '')
            )
        )
    );
