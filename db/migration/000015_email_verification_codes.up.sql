CREATE TABLE email_verification_codes (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounts(id),
    email text NOT NULL,
    secret_code text NOT NULL,
    is_used boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    expired_at timestamptz NOT NULL DEFAULT (now() + interval '15 minutes')
);

CREATE INDEX email_verification_codes_account_idx ON email_verification_codes(account_id);
