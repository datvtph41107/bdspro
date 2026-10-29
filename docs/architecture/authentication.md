# Three-method Authentication — Current Contract

## One account, three independent proofs

- **Email + password:** `POST /users` then email verification and `POST /users/login`. A duplicate normalized email returns `409 auth.email_registered`.
- **Continue with Google:** `GET /users/google/login/start` and callback. The verified Google subject is unique. Existing matching email is **not silently linked**. A new Google account has no password credential.
- **Phone OTP:** `POST /users/phone/register/start` then `/verify` returns access/refresh tokens. Login uses `POST /users/phone/login/start` and `/verify`. All phone numbers are E.164, and `phone_credentials.phone_e164` is `UNIQUE`. Registration conflicts return `409 auth.phone_already_registered`. Unknown phone login starts return 202 without revealing account existence.

All three methods issue the same AccountID-based JWT and PostgreSQL refresh session.
Phone-only accounts deliberately have NULL email; no fake address or password
is invented. VerifiedContact permits either verified email or verified phone
for protected organization operations.

## Linking Google to an existing phone-only account

First sign in using phone OTP, then request another OTP for that same bound number with `POST /users/phone/login/start`. Call authenticated `POST /users/me/google/start` with `{"phone_code":"<fresh code>"}`, then complete the Google browser flow. The OTP is consumed, bound to the AccountID and cannot be replayed. Existing password accounts may instead use `{"password":"<current password>"}`. Linking Google subject never merges accounts solely by email.

Linking Google to a phone-only account currently **does not add its verified Google email** to `accounts.email`. Adding an email to that account needs a separate explicit, verified linking operation.

## Establishing a password after Google sign-up

Google never automatically provides a BDSPro password. The user must prove
control of the verified email:

1. `POST /users/password/reset/request` with `{"email":"user@example.com"}`.
2. Read the local Mailpit message in development; follow its 15-minute reset link.
3. `POST /users/password/reset` with `{"token":"...","password":"new-strong-password"}` or submit the HTML reset form.
4. `POST /users/login` using that email and new password.

The reset token is random, SHA-256 hashed in PostgreSQL, single-use and
consumed inside the same transaction that upserts the password credential and
deletes the account's existing refresh sessions. Unknown/unverified email
requests return the same 202 as known email addresses; do not enumerate accounts.
Access JWTs already issued remain valid until their natural expiry.

## Local test sequence

Run PostgreSQL 18, Redis 7 and Mailpit. Apply migrations 1–9 using `make
migrateup`, then start `go run .`.

1. Phone registration start: POST `/users/phone/register/start`,
   `{"phone":"+84912345678"}` → 202.
2. Read OTP via a **development-only** SQL query:
   `SELECT dev_code FROM phone_signup_challenges WHERE phone_e164='+84912345678';`
3. Registration verify: POST `/users/phone/register/verify`,
   `{"phone":"+84912345678","code":"123456"}` (use actual dev_code) → 201 tokens.
4. Login start: POST `/users/phone/login/start` → 202.
5. Read `dev_code` in `phone_otp_challenges` for purpose `login`; verify at
   `/users/phone/login/verify` → 200 tokens.
6. Repeat registration using same phone → 409 without a duplicate Account.
7. Reset password using Mailpit and repeat token consumption → 400.

Local SMS uses `notification.Local` and is never an actual carrier delivery.
For production, supply a tested SMS or Zalo delivery provider and HTTPS secrets.
The current email reset dispatch is synchronous SMTP; failure handling and
durability must be measured before a production readiness claim.

## Not yet included

- Linking a verified email to a phone-only account; auto-merging existing
  accounts is prohibited without proving control of both.
- Immediate invalidation of already-issued access JWTs.
- Live Google/SMS-provider end-to-end acceptance and backup/restore drills.
- Password reset email durable queueing / provider failure retry.
