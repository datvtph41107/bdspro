CREATE TABLE phone_credentials (
  account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
  phone_e164 text NOT NULL UNIQUE CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  verified_at timestamptz NOT NULL DEFAULT now()
);
