# Google Sign-In — BDSPro (backend v1)

## Scope and ownership

Google is an **authentication provider**. Google `sub` uniquely identifies an external Google identity linked to one internal AccountID. All sign-in methods share the same JWT, session and organization authorization model.

A new Google subject can create a passwordless BDSPro account when its signed ID token contains a verified email. Existing accounts are **never linked automatically by email**. Password credentials, if present, live in `password_credentials`. Google sign-up does not enqueue a verification-email job because the identity provider has already verified the email.

## Google Cloud OAuth client

1. Create/select a Google Cloud project; configure Google Auth Platform Branding and Audience.
2. Create a **Web application** OAuth client. During testing, add the desired Google test users.
3. Add this exact **Authorized redirect URI**:
   `http://localhost:8080/users/google/callback`
4. Supply credentials as process environment (do not commit secrets):

```bash
export GOOGLE_CLIENT_ID='your-client-id.apps.googleusercontent.com'
export GOOGLE_CLIENT_SECRET='your-client-secret'
export GOOGLE_REDIRECT_URI='http://localhost:8080/users/google/callback'
make server
```

The application registers Google routes only when all three environment variables are present. Production redirect URIs must be HTTPS. Do not set Google credentials in checked-in `app.env` or Dockerfile.

For Compose, inject credentials securely into the `api` service at deployment; this repository deliberately does not include example real secrets.

## API flow

**Link** — existing email/password account:

1. Obtain BDSPro access JWT via `POST /users/login`.
2. From the *same browser* that will navigate to Google, request:
   `POST /users/me/google/start`, with `Authorization: Bearer <access-token>` and JSON `{"password":"your-password"}`.
3. The response supplies `authorization_url`; browser receives a short-lived HttpOnly state cookie. Navigate that same browser to the URL.
4. After Google consent, Google redirects to `/users/google/callback`. Callback returns `{"linked":true}` only when the Google `sub` belongs to that same BDSPro account.

**Sign up or sign in** — Google account:

1. Open `GET /users/google/login/start` in a browser.
2. The server redirects to Google; Google redirects back to `/users/google/callback`.
3. The callback validates the Google ID token. A linked subject signs in; a new subject with a verified email creates a Party, Person, Account and Google identity atomically, then creates the session. A new subject whose email already belongs to an account is rejected with HTTP 409 and must link after signing in. A new subject without verified email cannot register.
4. The callback returns BDSPro JWT/session JSON, never raw Google tokens.

This backend-only version returns session JSON to the browser; a production web frontend must arrange a safe completion UX and token storage policy (e.g. HttpOnly secure first-party cookies), rather than copying tokens to URLs, browser history or external redirects.

## Security properties

- Authorization Code Flow with PKCE S256, state/nonce generated with crypto/rand.
- Redis stores a five-minute challenge containing PKCE verifier, nonce and operation/account; GETDEL consumes it exactly once.
- Short-lived host-only SameSite=Lax, HttpOnly state cookie binds callback to the initiating browser; Secure is enabled for HTTPS.
- Google ID token signature verified against Google JWKS, with RS256, audience, issuer, expiry and nonce checks. No trust in email text for account linking.
- Password re-entry is required when linking an existing account.
- Google refresh/access tokens and credentials are not saved in the application database.
- Secrets, access tokens, authorization codes and callback query strings must never be placed in application logs.
- Google credentials absent: the existing email/password endpoints remain operational.

## Verification

```bash
go test -count=1 ./...
go vet ./...
go build ./...
git diff --check
```

On a disposable test database, apply migrations through 000008. Exercise passwordless Google creation, identity collisions, existing-email conflicts, linking and replay rejection. Test the end-to-end browser flow only after configuring a real Google OAuth client, authorized redirect URI and test Google account; unit tests do not impersonate Google's identity service.

Do not run SQL migrations/tests against an irreplaceable database. No production deployment is asserted by this document.
