BEGIN;
CREATE TABLE accounts (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 person_id BIGINT NOT NULL REFERENCES persons(party_id),
 closed_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX accounts_one_open_per_person_uq
 ON accounts(person_id) WHERE closed_at IS NULL;
CREATE TABLE account_emails (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 account_id BIGINT NOT NULL REFERENCES accounts(id),
 email TEXT NOT NULL CHECK(btrim(email)<>''),
 verified_at TIMESTAMPTZ,
 is_primary BOOLEAN NOT NULL DEFAULT FALSE,
 UNIQUE(account_id,email)
);
CREATE UNIQUE INDEX account_emails_primary_uq ON account_emails(account_id) WHERE is_primary;
CREATE UNIQUE INDEX account_emails_login_key_uq ON account_emails(lower(btrim(email)));
CREATE TABLE account_phones (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 account_id BIGINT NOT NULL REFERENCES accounts(id),
 phone TEXT NOT NULL CHECK(btrim(phone)<>''),
 verified_at TIMESTAMPTZ, is_primary BOOLEAN NOT NULL DEFAULT FALSE,
 UNIQUE(account_id,phone)
);
CREATE UNIQUE INDEX account_phones_primary_uq ON account_phones(account_id) WHERE is_primary;
CREATE TABLE password_credentials (
 account_id BIGINT PRIMARY KEY REFERENCES accounts(id),
 password_hash TEXT NOT NULL,
 changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE sessions (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 account_id BIGINT NOT NULL REFERENCES accounts(id),
 token_hash TEXT NOT NULL UNIQUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 expires_at TIMESTAMPTZ NOT NULL,
 revoked_at TIMESTAMPTZ,
 CHECK (expires_at>created_at)
);
CREATE INDEX sessions_account_idx ON sessions(account_id);
CREATE TABLE email_verification_tokens (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 account_email_id BIGINT NOT NULL REFERENCES account_emails(id),
 token_hash TEXT NOT NULL UNIQUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 expires_at TIMESTAMPTZ NOT NULL,
 consumed_at TIMESTAMPTZ,
 CHECK (expires_at>created_at)
);
CREATE TABLE passkeys (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 account_id BIGINT NOT NULL REFERENCES accounts(id),
 credential_id TEXT NOT NULL UNIQUE,
 public_key BYTEA NOT NULL,sign_count BIGINT NOT NULL DEFAULT 0 CHECK(sign_count>=0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),last_used_at TIMESTAMPTZ
);
CREATE TABLE external_identities (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 account_id BIGINT NOT NULL REFERENCES accounts(id),
 issuer TEXT NOT NULL,subject TEXT NOT NULL,
 linked_at TIMESTAMPTZ NOT NULL DEFAULT now(),UNIQUE(issuer,subject)
);
CREATE TABLE service_accounts (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id BIGINT REFERENCES organizations(party_id),
 name TEXT NOT NULL,disabled_at TIMESTAMPTZ
);
CREATE TABLE service_account_keys (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 service_account_id BIGINT NOT NULL REFERENCES service_accounts(id),
 key_hash TEXT NOT NULL UNIQUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 expires_at TIMESTAMPTZ,revoked_at TIMESTAMPTZ
);
COMMIT;
