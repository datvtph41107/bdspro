ALTER TABLE accounts ADD COLUMN is_email_verified boolean NOT NULL DEFAULT false;
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
