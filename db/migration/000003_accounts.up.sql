CREATE TABLE "accounts" (
  "id" bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  "person_id" bigint NOT NULL UNIQUE REFERENCES persons(party_id),
  "email" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "is_email_verified" boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX "accounts_email_login_uq" ON "accounts" (lower(btrim("email")));
