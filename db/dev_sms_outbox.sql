-- DEVELOPMENT ONLY. Apply manually to an isolated local PostgreSQL database.
-- Not included in db/migration: production must never create a plaintext OTP sink.
-- WARNING: Treat these rows as temporary secrets. Never expose via an HTTP endpoint.
CREATE TABLE IF NOT EXISTS dev_sms_outbox (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    phone TEXT NOT NULL,
    otp_code TEXT NOT NULL CHECK (otp_code ~ '^[0-9]{6}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '5 minutes'),
    CHECK (expires_at > created_at)
);
