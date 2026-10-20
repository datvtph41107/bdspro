# Phone OTP login — one Account, two credential paths

## Status and scope

Implements an **optional login method**, not MFA or auto-registration.
A phone MUST already belong to an open Account with `account_phones.verified_at`
set. The same Account is reached by email/password and verified phone/SMS OTP;
no organization context or role appears in JWT or in the anonymous challenge.

### HTTP contract

1. `POST /auth/phone/login/start` with `{"phone":"+84901234567"}`.
   An eligible number receives a 6-digit SMS. HTTP 202 response ALWAYS has
   `{"challenge_token":"<opaque-random-handle>","expires_in_seconds":300}`
   for a syntactically valid number, including unknown/closed/unverified or
   destination-throttled numbers. Fake handles are deliberately not stored.
   An SMS delivery attempt failure is logged WITHOUT the phone or OTP; the
   public result remains 202. Missing provider configuration returns 503
   without disclosing Account status.
2. `POST /auth/phone/login/verify` with
   `{"challenge_token":"<opaque-handle>","code":"123456"}`.
   Correct, unexpired, single-use OTP returns the exact existing login
   response including `account_id`, HttpOnly session cookie and access JWT
   when enabled. Wrong, expired, used or fake handles return 401.
   No cookie/JWT is issued on an invalid OTP. All requests must pass the
   existing browser-origin and login CSRF policy.

Identifiers are E.164; this milestone does not normalize user-local
`090...` numbers into international form. A country-aware UI may do that
explicitly; never silently guess a dialing prefix.

### Account, session and security rules

- `Account -> Person` remains the source of identity; memberships/permissions
  remain in PostgreSQL, evaluated on business operations.
- Session creation and challenge consumption are atomic, after the Account
  lock and OTP check. Invalid attempts are committed, capped at five.
- The challenge is a 32-byte cryptographically random *handle*, stored only
  as SHA-256. It is **not a session**. The six-digit OTP is stored only
  as HMAC-SHA256 with a separate namespace prefix (`login|`), account_phone
  ID and random nonce, using the existing PHONE_OTP_KEY.
- TTL 5 minutes, one active challenge per phone, 60-second destination
  cooldown shared with phone verification, 5 SMS sends/24 hours per
  destination across login and enrollment (on the login issuance path),
  10 phone-login sends/24 hours per Account. An authenticated enrollment
  endpoint still enforces its own previously agreed independent quotas.
- Existing per-client admission limits also apply; the limiter is in-memory
  and does not replace PostgreSQL quotas or a shared edge limiter.
- A valid password still logs in regardless of whether the Account has a
  verified phone. Phone login is **not MFA** and MUST NOT be promoted to
  a second factor of the password flow without explicit pending-auth policy.
- Access through a phone login still has the same 24-hour absolute session
  expiration, max-five active-session cap and immediate PostgreSQL revocation.

### Caveats, limitations and next decisions

SMS is vulnerable to SIM swap, number recycling and interception; don't
treat it as phishing-resistant. Recovery, controlled phone replacement and
higher-assurance authorization for tenant administrators are later policies.
This feature does **not** create accounts for unknown numbers, change
email/password registration, enable SMS-only reset or grant tenant authority.
An SMS provider API accepting a message does not prove handset delivery.

### Acceptance gate

Migration 000009 UP-DOWN-UP; SQLC generation consistency; all Go and race
tests, actual PostgreSQL integration for unknown number, unverified phone,
wrong/expired/replayed OTP, simultaneous redemption, session/JWT behavior,
and Docker Compose build/start. SMS is faked during CI: no assertion that
a carrier delivered any message. No AWS required.
