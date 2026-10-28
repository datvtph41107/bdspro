# Authentication & Phone Verification — acceptance gate

This document describes intended current behavior. It does not claim that a
third-party SMS gateway or Zalo account has delivered a real message.

## Identity rules

- AccountID is the application's immutable principal. Google identity is
  matched by verified provider subject (`sub`), not by email text alone.
- Google-only users are not forced to set a BDSPro password. A password
  registration with an already-registered email cannot create a second account.
- Password users may explicitly link Google after password confirmation.
- A verified primary phone belongs to at most one account. Binding is not
  changing; an existing phone must not be silently overwritten.
- Phone OTP only verifies a phone. The phone-only login endpoints are removed.
- Non-verified email accounts may inspect their own AccountID but business
  organization routes and phone binding require verified email.
- An email verification URL is only displayed by GET. POST consumes the token.

## Local development acceptance (disposable DB only)

Start the API, PostgreSQL and Redis in the usual Docker Compose development
stack. Open Mailpit at `http://localhost:8025` for email verification.

1. Register an email/password account via `POST /users`. Observe a new row in
   `accounts` and exactly one `password_credentials` row.
2. Open the Mailpit verification URL in a browser. GET must present a confirmation
   form **without** setting `is_email_verified=true`. Submit the POST form;
   then the account must be marked verified. Submitting again must fail.
3. Sign in using `POST /users/login`. Confirm that the returned access token
   can read `GET /users/me`. Before email verification the account must not
   access `/organizations`; after verification it can proceed subject to
   organization authorization.
4. A linked Google user logs in via `GET /users/google/login/start`. Compare
   `GET /users/me` AccountID with the original ID; it must not change.
5. For a password account, start verification with:
   `POST /users/me/phone/start`, Bearer access JWT, JSON:
   `{"phone":"+84912345678","password":"current-password"}`.
6. For an already linked Google account without password, use
   `POST /users/me/phone/google/start`, Bearer access JWT, JSON:
   `{"phone":"+84912345678"}`. Open the returned
   `authorization_url` in the **same browser** that initiated the request.
   The callback must verify that the returned Google subject is linked to
   the *same* AccountID and return HTTP 202 only then.
7. In an isolated development database only, inspect the active challenge:

   ```sql
   SELECT phone_e164, purpose, dev_code, expires_at, failed_attempts
   FROM phone_otp_challenges WHERE account_id = 3;
   ```

   Submit `POST /users/me/phone/verify` with the Account's Bearer token and
   JSON `{"phone":"+84912345678","code":"<six digits>"}`.
8. Confirm `phone_credentials` contains the AccountID and phone. A second
   Account cannot claim this phone even if concurrent requests race.
9. Repeat with wrong, expired, reused and rate-limited codes. A second bind
   attempt to change the original account's phone must fail. The dedicated
   phone-change/recovery operation is **not yet implemented**.
10. `POST /users/phone/login/start` and `/verify` must both return 404.

Use disposable numbers and accounts. Never publish real JWTs, verification
secrets, Google authorization codes, phone OTPs, or provider API tokens.

## Phone delivery contracts

`PHONE_DELIVERY=local` is allowed only in development; the code is written
as `dev_code` into a disposable PostgreSQL database for manual testing.

For production choose exactly one delivery channel:

- `PHONE_DELIVERY=sms`, `SMS_GATEWAY_URL` (HTTPS), `SMS_GATEWAY_TOKEN`.
  The operator-managed gateway contract is POST JSON
  `{"phone":"+84...","message":"Your verification code is 123456"}`
  with `Authorization: Bearer ...`. **This is a gateway contract, not a
  promise of direct integration with any named carrier.** An actual SMS
  provider adapter and account must implement it.
- `PHONE_DELIVERY=zalo`, `ZALO_ACCESS_TOKEN`, `ZALO_TEMPLATE_ID`,
  `ZALO_CODE_FIELD` (default `otp`). ZBS requires an approved template,
  OA permissions, end-user eligibility and an actively refreshed access
  token. This implementation sends the template; token refresh and provider
  webhooks are not automated here.
- `PHONE_DELIVERY=disabled` or no configured provider disables phone routes
  outside development.

The application sends OTP synchronously only after persisting a challenge.
`code_mac` is retained; `dev_code` is empty outside development. A provider
response does not mark the phone as verified. An ambiguous provider timeout can
leave a challenge until cooldown/expiry; it does not grant verification.

## Release gate

CI: SQLC regeneration must be unchanged, migrations must apply to a disposable
PostgreSQL database, Go tests must pass under race detector, vet and build.

Manual release acceptance requires a real browser Google OAuth run, mail
delivery with a configured sender, and delivery to a controlled physical number
through each enabled production channel. Passing CI alone is not that proof.

Deferred: phone change/recovery, automatic Zalo token refresh, delivery-status
webhooks, real carrier integrations and transaction-specific step-up policies.
