DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM team_invites WHERE recipient_account_id IS NOT NULL) OR
       EXISTS (SELECT 1 FROM audit_events WHERE action IN ('team.member.added','team.invite.created','team.invite.revoked','team.invite.accepted')) THEN
        RAISE EXCEPTION 'team invitation or audit history requires explicit rollback plan';
    END IF;
END $$;
ALTER TABLE team_invites DROP CONSTRAINT team_invites_recipient_binding_check;
ALTER TABLE team_invites DROP COLUMN recipient_account_id, DROP COLUMN recipient_membership_id;
ALTER TABLE audit_events DROP CONSTRAINT audit_events_action_check;
ALTER TABLE audit_events DROP CONSTRAINT audit_events_payload_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_action_check
  CHECK (action IN ('auth.login.rejected','auth.session.revoked','team.member.removed','organization.owner.promoted','organization.owner.demoted','organization.owner.transferred','organization.member.removed','auth.password.changed','auth.password.reset','auth.account.registered','auth.password.set','auth.credential.removed','auth.sessions.revoked_all','auth.credential.linked'));
ALTER TABLE audit_events ADD CONSTRAINT audit_events_payload_check CHECK (
(
            action = 'auth.login.rejected' AND actor_account_id IS NULL AND
            scope_type IS NULL AND scope_ref IS NULL AND target_type IS NULL AND target_ref IS NULL AND
            method IS NOT NULL AND reason IS NOT NULL AND method IN ('password', 'phone_otp', 'oidc') AND
            subject_mac IS NOT NULL AND octet_length(subject_mac) = 32 AND
            ((method='password' AND reason IN ('unknown_account','password_unset','invalid_password')) OR
             (method='phone_otp' AND reason IN ('unknown_account','invalid_otp')) OR
             (method='oidc' AND reason IN ('invalid_identity','email_unverified','link_required')))
        ) OR (
            action IN ('auth.session.revoked','team.member.removed','organization.member.removed','organization.owner.promoted','organization.owner.demoted','organization.owner.transferred','auth.password.changed','auth.password.reset','auth.account.registered','auth.password.set','auth.credential.removed','auth.sessions.revoked_all','auth.credential.linked') AND
            actor_account_id IS NOT NULL AND actor_account_id > 0 AND account_id IS NULL AND
            method IS NULL AND reason IS NULL AND subject_mac IS NULL AND
            scope_type IS NOT NULL AND scope_ref IS NOT NULL AND target_type IS NOT NULL AND target_ref IS NOT NULL AND
            ((action='auth.session.revoked' AND scope_type='account' AND scope_ref <> '' AND target_type='session' AND target_ref <> '') OR
             (action IN ('auth.password.changed','auth.password.reset','auth.password.set') AND scope_type='account' AND scope_ref <> '' AND target_type='password' AND target_ref=scope_ref) OR
             (action IN ('auth.account.registered','auth.sessions.revoked_all') AND scope_type='account' AND scope_ref <> '' AND target_type='account' AND target_ref=scope_ref) OR
             (action IN ('auth.credential.removed','auth.credential.linked') AND scope_type='account' AND scope_ref <> '' AND target_type IN ('email','phone','google','password') AND target_ref=scope_ref) OR
             (action='team.member.removed' AND scope_type='team' AND scope_ref <> '' AND target_type='person' AND target_ref <> '') OR
             (action='organization.member.removed' AND scope_type='organization' AND scope_ref <> '' AND target_type='person' AND target_ref <> '') OR
             (action IN ('organization.owner.promoted','organization.owner.demoted','organization.owner.transferred') AND scope_type='organization' AND scope_ref <> '' AND target_type='person' AND target_ref <> ''))
        )
);
