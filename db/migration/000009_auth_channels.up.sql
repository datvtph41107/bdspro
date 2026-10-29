-- Phone-only identities have no email address; keep nullable email unique when present.
ALTER TABLE accounts ALTER COLUMN email DROP NOT NULL;

-- A registration challenge precedes account creation: no orphan Account is made
-- until possession of the phone has been proved by a valid one-time code.
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
