-- Enqueue intent is committed with the account, before any Redis side effect.
CREATE TABLE verification_outbox (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  account_id bigint NOT NULL UNIQUE REFERENCES accounts(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  queued_at timestamptz
);

CREATE INDEX verification_outbox_pending_idx
  ON verification_outbox(id) WHERE queued_at IS NULL;
