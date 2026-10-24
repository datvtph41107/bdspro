-- Development-only phone OTP until a delivery provider is configured.
-- Account identity remains accounts.id; the verified phone must be unique.
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
