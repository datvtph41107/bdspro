CREATE TABLE password_reset_tokens (
  token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash)=32),
  account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX password_reset_tokens_account_idx ON password_reset_tokens(account_id);
