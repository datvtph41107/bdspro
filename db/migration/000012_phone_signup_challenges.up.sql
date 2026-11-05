CREATE TABLE phone_signup_challenges (
  phone_e164 text PRIMARY KEY CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  code_mac bytea NOT NULL CHECK (octet_length(code_mac)=32),
  dev_code text,
  expires_at timestamptz NOT NULL,
  next_send_at timestamptz NOT NULL,
  failed_attempts integer NOT NULL DEFAULT 0,
  locked_until timestamptz NOT NULL DEFAULT 'epoch'::timestamptz
);
