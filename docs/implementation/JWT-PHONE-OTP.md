# JWT access authentication, authorization and phone OTP

## Why JWT here

JWT **is integrated**, not merely a lab. BDSPro still keeps a PostgreSQL
opaque session cookie for renewal, cookie-based clients and immediate
revocation. A signed **HS256 access JWT (15 minutes)** is issued on login
when `TOKEN_SYMMETRIC_KEY` is configured. The JWT has mandatory issuer
(`bdspro-auth`), audience (`bdspro-api`), subject (account ID),
issued-at, not-before, expiration, unique token ID and a `sid`
SHA-256 fingerprint of the session. No role, organization membership,
password or phone number is placed in claims.

Bearer authorization calls verify the signature and claims, then **check
the referenced session in PostgreSQL** on every request. This deliberately
retains immediate logout/logout-all revocation and closed-account behavior;
it does not deliver stateless lookup savings. Authorization for Org/Source
remains in SQL with active memberships/permissions. Invalid Authorization
does **not** fall back to a valid cookie.

HTTP login response adds `access_token`, `token_type: Bearer`,
`access_token_expires_at` without changing the opaque cookie.
`POST /auth/token/renew` requires the cookie plus existing CSRF controls;
it issues another short-lived access token without extending the session's
absolute 24-hour expiration. Both `POST /auth/logout` and
`POST /auth/logout-all` now accept a **validated Bearer JWT without a cookie**
and revoke the PostgreSQL session behind its signed `sid`. If both Bearer
and cookie credentials are present, Bearer takes precedence and these
endpoints do not clear or revoke an unrelated cookie session. Invalid
Bearer tokens never fall back to the cookie. Cookie-only logout and its
existing CSRF requirements remain unchanged; Bearer requests must still
obey the endpoint's browser-origin and request-proof rules.

JWTs are returned in JSON and NOT written
into persistent browser storage by the API. Browser clients should store
access tokens in memory; cookie remains HttpOnly, Secure outside development.

## Phone OTP verification

Two protected endpoints:

- `POST /auth/phone/start` JSON `{"phone":"+84901234567"}`
  returns 202 only after a real SMS sender accepted delivery.
- `POST /auth/phone/verify` JSON
  `{"phone":"+84901234567","code":"123456"}` returns 204
  after a successful, atomic one-time code consumption and phone verification.

Requires an existing logged-in account; **phone OTP is phone ownership
proof**, not standalone login, password reset, or proof of identity.
Phone numbers are required in E.164 format. A random six-digit OTP is
stored only as an HMAC-SHA256 digest using `PHONE_OTP_KEY`, account-phone
ID and random nonce. The code expires in **5 minutes**, allows **5
attempts**, has a **60-second issuance cooldown globally per destination**
and at most **5 challenges/24h per destination**, independent of the
requesting account. A separate **10 challenges/24h per account** cap
prevents rotating numbers to bypass an account's SMS budget. Issuing
a new code invalidates all earlier outstanding codes for that account/phone.
PostgreSQL locks the owner account, the destination (transaction-scoped
advisory lock) and the challenge to prevent concurrency bypasses; a
partial unique index prevents two
accounts claiming the same verified phone number.

The existing per-socket-IP admission limits are an additional layer:
6 / 15min start, 20 / 15min verify. OTP, SMS credentials and phone
numbers are never logged as raw values.

## Twilio SMS

The optional adapter `auth.TwilioSMS` sends to the fixed official
Twilio HTTPS Messaging API using configured Account SID, auth token and
From E.164 number. Enable with:

```env
PHONE_OTP_KEY=<independent-random-secret-of-at-least-32-bytes>
TWILIO_ACCOUNT_SID=AC...
TWILIO_AUTH_TOKEN=<secret>
TWILIO_FROM=+15017122661
```

Credentials are process environment, not committed defaults. If no
sender is configured, the endpoint returns 503 and **does not claim
SMS was sent**. Test doubles may intercept sending in CI; that
does NOT confirm live SMS carrier delivery or recipient ownership.

## Browser security and compatibility

Credentialed CORS allows the `Authorization` header only from the
configured/same origin, and CSRF checks still apply to cookie mutations.
Email/password and phone-only registration/login now coexist. An existing
Account can verify a phone through the authenticated endpoints; anonymous
phone-only registration creates the Account only after a verified one-time
challenge. See `G01-1-PHONE-ONLY-REGISTRATION.md` and
`PHONE-OTP-LOGIN.md` for the current flows.

SMS via PSTN is a restricted authenticator (SIM swap, porting, SS7).
This is an optional verification factor and MUST NOT be treated as
phishing-resistant MFA or sole authorization for high-value transfers.

## Limits / follow-up

- HS256 shared secret rotation needs explicit old-key overlap/key IDs
  before rolling out to multiple token-verifying services.
- JWT revocation is immediate because it reads PostgreSQL sessions;
  it does NOT reduce session lookup load.
- SMS API accepted != carrier delivered. Actual carrier/in-country support,
  costs, consent, delivery receipts and phone reassignment checks require
  a configured provider and operations work.
- Existing email conflict responses may still reveal account presence.
- Strict production deployment requires vetted CORS, strong JWT/OTP
  secrets, HTTPS, SMS sender and rate-limit store at shared edge scale.
- No AWS or background jobs introduced.

## Acceptance

CI must run migrations up/down/up, SQLC regeneration/source consistency,
Go tests/race, auth/phone/JWT integration tests, server start and Docker
Compose smoke. No merge on failing CI.
