# G01.1 — Phone-only registration and authentication

**Scope:** one new account registration proof, not MFA and not automatic
linking of another user's phone. No cloud provider or deployment involved.

## Source and intent

Before this gate, `accounts` already allowed optional `account_emails` and
`password_credentials`, but `auth.Service.Register` always created both.
`phone_login_challenges` require a previously verified Account phone;
`phone_verification_challenges` require an authenticated Account.

**The new flow must first prove phone possession and only then create an Account.**

A phone number and an Account are not equivalent to a legally verified person.
Phone ownership can change; recovery and step-up protection for sensitive
operations remain G01.2/G08 responsibilities.

## API

All POSTs require `Content-Type: application/json` and the current browser
request proof (`X-BDSPro-Request: 1`) when no trusted Origin/fetch metadata
is provided. The same abuse limiter protects known and unknown identifiers.

1. `POST /auth/phone/register/start`, body `{"phone":"+84901234567"}`
   - Well-formed phone: 202 with
     `{"challenge_token":"<random opaque token>","expires_in_seconds":300}`.
   - If phone is already verified on any Account, or the shared SMS quota
     is exhausted, return an **unpersisted fake token** with the same response
     shape, without sending an SMS.
   - No Account, Person, Party or session exists yet. Provider send failure
     is logged without phone/code and returns the same public 202 shape.
   - Invalid E.164: 400; delivery unavailable: 503; infra fail: 503.

2. `POST /auth/phone/register/verify`, body
   `{"challenge_token":"<value above>","code":"123456"}`
   - Wrong, expired, replayed or fake proof: 401 with no session.
   - Correct proof: the transaction consumes the challenge and inserts
     Party (PERSON), Person, Account, verified primary AccountPhone,
     and the existing 24-hour Session; returns **the same** cookie/JWT
     response as email/password and phone login.
   - If a verified phone became bound to another Account, return 409
     only after verifying a valid registration proof; no session issued.
   - The phone string must already be E.164 (no country guessing).

`POST /auth/phone/login/start` and `/verify` continue to serve existing
verified phone owners. Email/password login remains unchanged.
No automatic Organization membership, email or password is created.

## Invariants and SQL

- Migration `000010` creates anonymous `phone_registration_challenges`,
  with SHA-256 challenge handle hash, HMAC-SHA256 OTP MAC (domain separated
  by `register|` and bound to phone+nonce), five-minute expiry, five
  attempts, consumed marker.
- OTP plaintext never enters that challenge table. Only the isolated
  local-only `dev_sms_outbox` transport may temporarily contain plaintext
  for development; it must never be installed into production.
- Same per-phone PostgreSQL advisory lock is used for issuance and
  registration redemption; verified phone uniqueness
  `account_phones_verified_number_uq` prevents a race with the previously
  implemented authenticated phone verification route.
- Global send policy is shared among **phone registration, phone login and
  authenticated phone enrollment**: no more than one send per phone each
  60 seconds and five per rolling 24-hour window. Existing account-specific
  login/enrollment quotas remain. Sending provider failures do not refund
  quota automatically.
- Incorrect code attempts **commit** and consume attempt budget; correct
  code consumption + all Account writes + Session creation share a single
  PostgreSQL transaction. Repeated redemption is rejected.
- There is still a small timing side-channel between a known and unknown
  phone despite uniform response. A future multi-instance edge must add a
  distributed public abuse policy; current in-memory HTTP limiter alone
  is not global.

## Local workflow without real SMS

Use the checked-out `dev` branch once the feature is merged, apply
versioned migrations and enable development delivery:

```sh
make migrateup DB_URL='postgresql://root:secret@localhost:5432/bdspro?sslmode=disable'
# Set up db/dev_sms_outbox.sql manually on disposable LOCAL development DB:
docker exec -i postgres psql -U root -d bdspro < db/dev_sms_outbox.sql

DEV_SMS_OUTBOX=true PHONE_OTP_KEY="$(openssl rand -hex 32)" make server
```

Use **the same stable** PHONE_OTP_KEY across startup and verification; rotating
the key invalidates unconsumed codes.

```http
POST /auth/phone/register/start
Content-Type: application/json
X-BDSPro-Request: 1

{"phone":"+84901234567"}
```

Query the latest OTP in the local database:

```sql
SELECT phone, otp_code, expires_at
FROM dev_sms_outbox
WHERE expires_at > now()
ORDER BY id DESC LIMIT 1;
```

Send that code with the challenge_token to
`POST /auth/phone/register/verify`, then `GET /auth/me` with returned
cookie/JWT. Logout revokes the session as usual.

**Do not use these commands against a database containing real user data.**

## Acceptance

Tests must prove that no Account is created at start or for failed proofs;
only one Account can be created by concurrent verification; HMAC not plaintext
is stored; no email/password/Org membership is manufactured; replay, expiry,
five guesses and unknown phone fail; verified new phone logs into the exact
same Account later through the original phone login route.

CI must pass migration `000010` up/down/up, regenerated sqlc diff, Go
tests/race and local Docker Compose smoke. G01.1 isn't proof that all G01
account recovery, credential linking, passkeys, or G02–G10 are complete.
