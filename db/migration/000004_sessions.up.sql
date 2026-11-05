CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "account_id" bigint NOT NULL REFERENCES accounts(id),
  "refresh_token" text NOT NULL,
  "user_agent" text NOT NULL,
  "client_ip" text NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "expires_at" timestamptz NOT NULL
);

CREATE INDEX "sessions_account_idx" ON "sessions" ("account_id");
