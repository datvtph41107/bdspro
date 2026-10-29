# BDSPro authentication: three methods and development verification

The application has **one account identity** (`accounts.id`), and three supported ways to authenticate. A Google identity, a verified phone credential and a password credential are distinct proofs. They must not silently create links between existing accounts.

## A. Email and password

```http
POST /users
Content-Type: application/json

{"email":"developer@example.test","password":"SufficientlyStrong123"}
```

The response is HTTP 201. Duplicate normalized emails return HTTP 409 `auth.email_registered`. Registration creates an unverified email account and a durable verification job. Use the Mailpit inbox at `http://127.0.0.1:8025` when running the Docker Compose development stack; confirm email using the verification page or POST before protected business operations.

```http
POST /users/login
Content-Type: application/json

{"email":"developer@example.test","password":"SufficientlyStrong123"}
```

A successful login returns access and refresh tokens backed by one session.

## B. Google OAuth (Continue with Google)

Set `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI` together. Open `GET /users/google/login/start` in a **browser**; browser cookies/state, PKCE and nonce protect the callback. After authorization, `GET /users/google/callback` returns the ordinary login response.

New verified Google subjects create a **passwordless** account. **Google does not provide or generate a BDSPro password**. The user can set a BDSPro password through verified-email password reset below. Existing accounts are not auto-linked simply because the email matches; signing into the existing account and explicitly linking Google is a separate operation.

## C. Phone-only signup and OTP login

Phone input uses E.164, e.g. `+84901234567`, not `0901234567`. One normalized phone belongs to at most one account, enforced by `phone_credentials.phone_e164 UNIQUE`.

```http
POST /users/phone/register/start
Content-Type: application/json

{"phone":"+84901234567"}
```

HTTP 202 means the challenge was accepted. An already-owned phone returns HTTP 409 `auth.phone_already_registered`.

To inspect the **development-only** code from the disposable development database (never a production database):

```sh
docker compose exec postgres psql -U root -d bdspro -c \
  "SELECT phone_e164, dev_code, expires_at FROM phone_signup_challenges WHERE phone_e164='+84901234567';"
```

```http
POST /users/phone/register/verify
Content-Type: application/json

{"phone":"+84901234567","code":"123456"}
```

Replace `123456` with the actual development code. Successful HTTP 201 creates Party, Person, Account and PhoneCredential atomically, and issues the usual access and refresh tokens. The account has **no email address or password** at creation.

For a returning user:

```http
POST /users/phone/login/start
Content-Type: application/json

{"phone":"+84901234567"}
```

Login start always responds HTTP 202 for an unknown well-formed phone to avoid disclosing account existence. For a registered phone, the development code is in `phone_otp_challenges` with `purpose='login'`:

```sh
docker compose exec postgres psql -U root -d bdspro -c \
  "SELECT dev_code, expires_at FROM phone_otp_challenges WHERE phone_e164='+84901234567' AND purpose='login';"
```

```http
POST /users/phone/login/verify
Content-Type: application/json

{"phone":"+84901234567","code":"123456"}
```

A valid OTP returns HTTP 200 and the standard login response. Codes expire, attempts are limited and a successfully consumed code cannot be used again. Provider SMS/Zalo delivery requires separately configured credentials; development `Local` does not send real messages.

Existing email/Google accounts may bind a verified phone via authenticated `/users/me/phone/*` endpoints; a phone-only account is never automatically merged into another account.

## D. Password setup and recovery

```http
POST /users/password/reset/request
Content-Type: application/json

{"email":"developer@example.test"}
```

For a verified email, a one-time 15-minute link is sent through the configured SMTP provider. Use Mailpit locally. Google-only accounts may use this flow to set their **first** local password. The request normally returns HTTP 202 without confirming whether an email belongs to an account.

Open the reset link in the browser or use Postman:

```http
POST /users/password/reset
Content-Type: application/json

{"token":"ONE_TIME_TOKEN_FROM_EMAIL","password":"AnotherStrong123"}
```

A valid token is consumed once. The password credential is inserted/updated, and outstanding refresh sessions are revoked atomically. Previously issued access JWTs remain valid until their normal expiration.

## Development acceptance

Run `docker compose up --build` with a disposable local PostgreSQL instance and Redis; never test OTP against production user data. Check `GET /health`, `GET /swagger/openapi.json` and Swagger UI at `GET /swagger/`. Run `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...`.

Integration tests require `BDS_TEST_MODE=disposable` and `BDS_TEST_DB_URL` explicitly pointing to `bdspro_test` with migrations applied. A green CI result does not prove a live Google login, external SMS provider delivery, provider outage handling or production rollout.

### Current limitations to track

- Phone-only accounts cannot yet add an email/password through an authenticated email-link flow.
- Google linking for passwordless phone-only accounts requires an explicit reauthentication design; it must not auto-link by email.
- SMTP password-reset delivery is currently synchronous and is not yet fully outbox-backed; provider failure behavior must be audited for account enumeration and retries.
- The OpenAPI contract lists optional Google/phone routes even in environments where the matching provider is disabled.

## E. Link Google to an existing phone-only account

A phone-only account has no BDSPro password. To link Google **to that same AccountID**, first sign in with phone OTP, then request a fresh OTP for its registered phone using `POST /users/phone/login/start`.

With the authenticated access token from phone login and the *fresh* OTP, call:

```http
POST /users/me/google/start
Authorization: Bearer ACCESS_TOKEN
Content-Type: application/json

{"phone_code":"123456"}
```

A successful response contains a Google authorization URL. Complete that flow in a browser. The callback verifies Google's subject and links it to the **authenticated account**, not to an email-address match. The OTP cannot be replayed. Accounts already having a password may instead send `{"password":"CURRENT_PASSWORD"}`. Providing neither or both proofs fails validation.

**Limitation:** Linking Google to a phone-only account does not yet attach a verified email address to the phone-only Account record. Email/password login for that account requires a separate verified-email linking flow. Do not turn Google subject ownership into an implicit email-based account merge.
