# Phone OTP login — development SMS as a database outbox

Production and development share **one** authentication state machine.

- Auth challenges are PostgreSQL rows with an **HMAC**, random nonce,
  attempt count, five-minute expiration, single-use confirmation, and
  account-locked session issuance. Neither client nor PostgreSQL challenge
  table contains a plaintext code.
- `SMSSender` is a delivery boundary. In production, configure a real
  delivery provider. In development, the opt-in database outbox replaces
  only the SMS transport and contains plaintext OTP **only temporarily**.
- **Do not migrate `dev_sms_outbox` into production.** Its schema is
  deliberately separate from versioned application migrations, and
  `DEV_SMS_OUTBOX=true` is rejected unless `ENVIRONMENT=development`.
- The development outbox does not make SMS delivery claims. Acceptance
  means the local database row was recorded, not a real SMS was sent.

## Local use

After applying versioned migrations to the correct local development DB,
provision the disposable outbox explicitly:

```bash
# The connection URL must point to your isolated local development database.
psql "$DB_SOURCE" -f db/dev_sms_outbox.sql
export DEV_SMS_OUTBOX=true
export PHONE_OTP_KEY="a-unique-local-random-secret-at-least-32-bytes"
make server
```

The SQL command is optional in Docker-only installations: use a PostgreSQL
client inside the local database container, passing in this same SQL script.
Do not run this provisioning against production.

A previously registered Account must verify its phone through the existing
authenticated `/auth/phone/start` and `/auth/phone/verify` endpoints.
The new anonymous login accepts **verified, unique, open-account numbers** only.

```http
POST /auth/phone/login/start
Content-Type: application/json
X-BDSPro-Request: 1

{"phone":"+84901234567"}
```

Returns HTTP 202 with `challenge_token` even for unknown numbers,
without issuing any login credentials.

Read the most recent development-only SMS row, then submit:

```sql
SELECT phone, otp_code, expires_at
FROM dev_sms_outbox
WHERE expires_at > now()
ORDER BY id DESC LIMIT 1;
```

```http
POST /auth/phone/login/verify
Content-Type: application/json
X-BDSPro-Request: 1

{"challenge_token":"<value from start>","code":"<code from dev_sms_outbox>"}
```

Only a valid, unused challenge can create the normal session/JWT.
SMS codes are one factor, not MFA; this does not prove legal identity.
Do not build or expose a read-OTP API, log raw codes, or use this
outbox with sensitive/non-disposable user records. Drop the dev table
if copying the database to a non-development environment.
