-- Canonical Lecture 37 BDSPro baseline. New databases only.
-- Existing bdspro databases at migration v1 need no rerun.
-- Layout matches SimpleBank: tables, relations, indexes.
-- Account means login identity, not a bank balance account.

CREATE TABLE "parties" (
  "id" bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);

CREATE TABLE "persons" (
  "party_id" bigint PRIMARY KEY
);

CREATE TABLE "accounts" (
  "id" bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  "person_id" bigint NOT NULL UNIQUE,
  "email" text NOT NULL,
  "password_hash" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "account_id" bigint NOT NULL,
  "refresh_token" text NOT NULL,
  "user_agent" text NOT NULL,
  "client_ip" text NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "expires_at" timestamptz NOT NULL
);

ALTER TABLE "persons" ADD FOREIGN KEY ("party_id") REFERENCES "parties" ("id");

ALTER TABLE "accounts" ADD FOREIGN KEY ("person_id") REFERENCES "persons" ("party_id");

ALTER TABLE "sessions" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

CREATE UNIQUE INDEX "accounts_email_login_uq" ON "accounts" (lower(btrim("email")));

CREATE INDEX "sessions_account_idx" ON "sessions" ("account_id");
