-- Google 'sub' is the durable identity; email is not the account key.
CREATE TABLE google_identities (
  google_sub text PRIMARY KEY CHECK (length(google_sub) > 0),
  account_id bigint NOT NULL UNIQUE REFERENCES accounts(id),
  linked_at timestamptz NOT NULL DEFAULT now()
);
