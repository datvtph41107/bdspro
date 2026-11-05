-- Refuse to silently delete security evidence introduced after 000020.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM audit_events WHERE action <> 'auth.login.rejected') THEN
        RAISE EXCEPTION 'cannot roll back audit actions while non-login evidence exists';
    END IF;
END $$;

ALTER TABLE audit_events
    DROP CONSTRAINT audit_events_payload_check,
    DROP CONSTRAINT audit_events_action_check,
    DROP COLUMN target_ref,
    DROP COLUMN target_type,
    DROP COLUMN scope_ref,
    DROP COLUMN scope_type,
    DROP COLUMN actor_account_id;

ALTER TABLE audit_events
    ALTER COLUMN method SET NOT NULL,
    ALTER COLUMN reason SET NOT NULL,
    ALTER COLUMN subject_mac SET NOT NULL;

ALTER TABLE audit_events
    ADD CONSTRAINT audit_events_action_check CHECK (action IN ('auth.login.rejected')),
    ADD CONSTRAINT audit_events_login_reason_check CHECK (
        (method = 'password' AND reason IN ('unknown_account', 'password_unset', 'invalid_password')) OR
        (method = 'phone_otp' AND reason IN ('unknown_account', 'invalid_otp')) OR
        (method = 'oidc' AND reason IN ('invalid_identity', 'email_unverified', 'link_required'))
    );
