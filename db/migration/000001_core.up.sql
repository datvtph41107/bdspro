-- Canonical fresh-install schema. No import, data upgrade, mirror, renamed objects or triggers.
-- Only run on an EMPTY database. Existing schema_migrations v1..v21 require explicit conversion.

CREATE TABLE "parties" (
  "id" bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);

CREATE TABLE "persons" (
  "party_id" bigint PRIMARY KEY REFERENCES parties(id)
);

CREATE TABLE "accounts" (
  "id" bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  "person_id" bigint NOT NULL UNIQUE REFERENCES persons(party_id),
  "email" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "is_email_verified" boolean NOT NULL DEFAULT false
);

CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "account_id" bigint NOT NULL REFERENCES accounts(id),
  "refresh_token" text NOT NULL,
  "user_agent" text NOT NULL,
  "client_ip" text NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "expires_at" timestamptz NOT NULL
);




CREATE UNIQUE INDEX "accounts_email_login_uq" ON "accounts" (lower(btrim("email")));

CREATE INDEX "sessions_account_idx" ON "sessions" ("account_id");

CREATE TABLE verify_emails (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  account_id bigint NOT NULL REFERENCES accounts(id),
  email text NOT NULL,
  secret_code text NOT NULL,
  is_used boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  expired_at timestamptz NOT NULL DEFAULT (now() + interval '15 minutes')
);
CREATE INDEX verify_emails_account_idx ON verify_emails(account_id);

CREATE TABLE organizations (
  party_id bigint PRIMARY KEY REFERENCES parties(id),
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE organization_members (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations(party_id),
  person_id bigint NOT NULL REFERENCES persons(party_id),
  role text NOT NULL CHECK (role IN ('owner', 'member')),
  joined_at timestamptz NOT NULL DEFAULT now(),
  ended_at timestamptz
);

CREATE UNIQUE INDEX organization_members_one_open_uq
  ON organization_members (organization_id, person_id) WHERE ended_at IS NULL;

CREATE INDEX organization_members_person_idx
  ON organization_members (person_id);

CREATE TABLE phone_credentials (
  account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
  phone_e164 text NOT NULL UNIQUE CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  verified_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE phone_otp_challenges (
  phone_e164 text NOT NULL CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  purpose text NOT NULL CHECK (purpose IN ('bind', 'login')),
  account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  code_mac bytea NOT NULL CHECK (octet_length(code_mac) = 32),
  dev_code text,
  expires_at timestamptz NOT NULL,
  next_send_at timestamptz NOT NULL,
  failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),
  attempt_window_started_at timestamptz NOT NULL DEFAULT now(),
  locked_until timestamptz NOT NULL DEFAULT 'epoch'::timestamptz,
  PRIMARY KEY (phone_e164, purpose)
);
CREATE INDEX phone_otp_challenges_account_idx ON phone_otp_challenges(account_id);

CREATE TABLE job_outbox (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    task_type text NOT NULL CHECK (length(task_type) > 0),
    payload jsonb NOT NULL,
    queue_name text NOT NULL DEFAULT 'default' CHECK (length(queue_name) > 0),
    max_retry integer NOT NULL DEFAULT 10 CHECK (max_retry >= 0),
    run_at timestamptz NOT NULL DEFAULT now(),
    task_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    queued_at timestamptz
);

CREATE INDEX job_outbox_pending_idx ON job_outbox(id) WHERE queued_at IS NULL;

CREATE TABLE password_credentials (
    account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    password_hash text NOT NULL CHECK (length(password_hash) > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE phone_signup_challenges (
  phone_e164 text PRIMARY KEY CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  code_mac bytea NOT NULL CHECK (octet_length(code_mac)=32),
  dev_code text,
  expires_at timestamptz NOT NULL,
  next_send_at timestamptz NOT NULL,
  failed_attempts integer NOT NULL DEFAULT 0,
  locked_until timestamptz NOT NULL DEFAULT 'epoch'::timestamptz
);

-- Email-based reset is single-use and time-bounded. Store only token hashes.
CREATE TABLE password_reset_tokens (
  token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash)=32),
  account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX password_reset_tokens_account_idx ON password_reset_tokens(account_id);

CREATE TABLE email_link_challenges (
    account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    email text NOT NULL CHECK (email = lower(btrim(email)) AND length(email) > 3),
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    expires_at timestamptz NOT NULL,
    next_send_at timestamptz NOT NULL
);
CREATE INDEX email_link_challenges_expiry_idx ON email_link_challenges(expires_at);

CREATE TABLE teams (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations(party_id) ON DELETE CASCADE,
  name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 100),
  description text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX teams_org_name_uq ON teams (organization_id, lower(btrim(name)));
CREATE INDEX teams_org_idx ON teams (organization_id, id);

CREATE TABLE team_members (
  team_id bigint NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  person_id bigint NOT NULL REFERENCES persons(party_id),
  joined_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (team_id, person_id)
);

CREATE INDEX team_members_person_idx ON team_members (person_id);

-- Store only a hash of the invitation proof, never its raw token.
CREATE TABLE team_invites (
  id uuid PRIMARY KEY,
  team_id bigint NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  invited_email text NOT NULL,
  invited_by_account_id bigint NOT NULL REFERENCES accounts(id),
  token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
  expires_at timestamptz NOT NULL,
  accepted_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (accepted_at IS NULL OR revoked_at IS NULL)
);

CREATE UNIQUE INDEX team_invites_open_uq
  ON team_invites (team_id, lower(btrim(invited_email)))
  WHERE accepted_at IS NULL AND revoked_at IS NULL;
CREATE INDEX team_invites_team_idx ON team_invites (team_id, created_at);

CREATE TABLE oauth2_credentials (
    issuer text NOT NULL CHECK (length(issuer) > 0),
    subject text NOT NULL CHECK (length(subject) > 0),
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    linked_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (issuer, subject)
);
CREATE INDEX oauth2_credentials_account_idx ON oauth2_credentials(account_id);
CREATE UNIQUE INDEX oauth2_credentials_account_issuer_uq ON oauth2_credentials(account_id,issuer);

CREATE TABLE person_profiles (
    person_id bigint PRIMARY KEY REFERENCES persons(party_id) ON DELETE CASCADE,
    display_name text NOT NULL DEFAULT '' CHECK (length(display_name) <= 120),
    first_name text NOT NULL DEFAULT '' CHECK (length(first_name) <= 120),
    last_name text NOT NULL DEFAULT '' CHECK (length(last_name) <= 120),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE account_settings (
    account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    locale text NOT NULL DEFAULT 'vi' CHECK (locale IN ('vi', 'en')),
    timezone text NOT NULL DEFAULT 'Asia/Ho_Chi_Minh' CHECK (timezone IN ('Asia/Ho_Chi_Minh', 'UTC')),
    currency text NOT NULL DEFAULT 'VND' CHECK (currency IN ('VND', 'USD')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE account_connection_histories (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('login')),
    ip_address text NOT NULL CHECK (length(ip_address) <= 45),
    user_agent text NOT NULL CHECK (length(user_agent) <= 2048),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX account_connection_histories_account_recent_idx
    ON account_connection_histories(account_id, created_at DESC, id DESC);

CREATE INDEX account_connection_histories_retention_idx
    ON account_connection_histories(created_at, id);

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

-- BDSPro is now an OAuth2 authorization server as well as an OIDC consumer.
CREATE TABLE oauth_clients (
    id uuid PRIMARY KEY,
    owner_account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    redirect_uri text NOT NULL CHECK (char_length(redirect_uri) BETWEEN 1 AND 2048),
    secret_hash bytea CHECK (octet_length(secret_hash) = 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);
CREATE INDEX oauth_clients_owner_idx ON oauth_clients(owner_account_id,id);

CREATE TABLE oauth_auth_codes (
    code_hash bytea PRIMARY KEY CHECK (octet_length(code_hash) = 32),
    client_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    redirect_uri text NOT NULL,
    scope text NOT NULL,
    code_challenge text NOT NULL CHECK (char_length(code_challenge) = 43),
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX oauth_auth_codes_expiry_idx ON oauth_auth_codes(expires_at);

CREATE TABLE oauth_access_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    client_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    scope text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX oauth_access_tokens_account_idx ON oauth_access_tokens(account_id,client_id);
CREATE INDEX oauth_access_tokens_expiry_idx ON oauth_access_tokens(expires_at);

CREATE TABLE oauth_refresh_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    access_token_hash bytea NOT NULL REFERENCES oauth_access_tokens(token_hash) ON DELETE CASCADE,
    client_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    family_id uuid NOT NULL,
    scope text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX oauth_refresh_tokens_family_idx ON oauth_refresh_tokens(family_id);
CREATE INDEX oauth_refresh_tokens_expiry_idx ON oauth_refresh_tokens(expires_at);
