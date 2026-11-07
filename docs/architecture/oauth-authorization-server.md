# OAuth2 authorization-server contract

BDSPro operates a restricted OAuth2 Authorization Code + PKCE S256 server for registered public clients. It separately consumes OIDC identities from external providers through `oauth2_credentials`; these are not BDSPro-issued grants.

`/oauth/authorize` requires an existing BDSPro JWT and explicit consent. `/oauth/token` supports only `authorization_code` and `refresh_token`, returns opaque random credentials, stores SHA256 digests, and rotates refresh tokens once per use. `/oauth/userinfo` requires a live OAuth access token scoped `profile:read`. Internal JWTs cannot be substituted for OAuth access tokens, or vice versa.

Refresh rotation, refresh-token replay revocation, and explicit family revocation use the same transaction-scoped PostgreSQL family advisory lock. A preliminary lookup determines the immutable family key; the token is always reread and checked after acquiring that lock. Acquiring a token row lock first would risk deadlock. This prevents a concurrent rotation from inserting a new refresh token after a revoke-family UPDATE has taken its statement snapshot. Do not replace the family lock with token-only `FOR UPDATE`; they protect different invariants.

Data tables: `oauth_clients`, `oauth_auth_codes`, `oauth_access_tokens`, `oauth_refresh_tokens`, `oauth_grants`. The first four are distinct from `sessions` and `oauth2_credentials`. Each table is created directly in its own ordered up/down migration, with no mirror, trigger or rename.

Account authorization management (R38.2, restricted API): `oauth_grants` is the durable account/client/scope approval record. GET `/oauth/authorize` is read-only and returns `consent_version`; the explicit authenticated approve POST must echo this integer. The server locks account/client and checks the expected version before committing consent and code issuance atomically. A revoke increments the version and retains a tombstone, so an approval based on a prior preview cannot silently re-authorize. `GET /users/me/oauth/authorizations` lists active grants even when access tokens expire; `DELETE /users/me/oauth/authorizations/:client_id` revokes the grant and related tokens/codes for only that account. The application owner is not automatically an authorizer.

**Single write owner:** `SQLStore.CreateOAuthAuthCode` has been removed. The only application path to issue a code is `ApproveOAuthAuthCode`, which commits the versioned consent and code in one transaction. Integration tests use the same atomic approval helper. PostgreSQL concurrency acceptance remains mandatory. An integer version is not a browser CSRF proof or a one-time consent challenge. Client-owner disable and HTTP browser consent are still separate work.

When [server] PUBLIC_ORIGIN is configured, GET /oauth/authorize supports HTML browser consent using a signed HttpOnly SameSite=Lax session-reference cookie; a PostgreSQL sessions lookup on every browser authorization request rejects revoked/expired sessions. GET /oauth/login and POST /oauth/login use AuthService.Login and the existing session model. Each cookie-authenticated consent POST requires both the trusted Origin/CSRF proof and the request-bound consent_proof; POST /oauth/logout revokes the same SQL-backed session. The existing Authorization: Bearer JWT API contract is preserved without ambient cookie fallback on malformed bearer credentials. Keep PUBLIC_ORIGIN HTTPS outside loopback development. Browser sign-in currently supports email/password accounts only. Mobile/phone-first and external OIDC browser sign-in require a separate supported login option. Browser OAuth requires valid Host forwarding from a trusted ingress, not trust in arbitrary forwarded headers.

The authorization endpoint previously required an API Bearer JWT rather than implementing browser cookie-based login and HTML consent. This is a restricted API-oriented implementation, not a complete third-party browser OAuth user experience. Only `profile:read` is supported. Under the current privacy-minimized contract this scope returns only the subject identifier from `/oauth/userinfo`, never `email` or `email_verified`. A separate contact-data scope, human-readable consent disclosure, and per-user authorization lifecycle must be accepted before expanding those claims; this endpoint is not an OpenID Connect UserInfo provider. Confidential clients, password grant, arbitrary resources and an OpenID Provider discovery surface are not implemented.

**Migration compatibility:** This 27-file canonical ledger is not equivalent to the previous migration versions 1–21. Never force a version or use it against existing data. Provision a new database and conduct a verified data transfer as a separately accepted cutover.

Evidence: `services/oauth/oauth_test.go` tests S256, exact redirect and code use; `models/db/oauth_server_integration_test.go` tests PostgreSQL single-use, concurrent exchange and refresh replay; `routers/api/v1/oauth/oauth_test.go` tests grant rejection and token-purpose separation. Production acceptance additionally requires consent UX, retention, scope/resource policy and security load testing.

**OAuth record cleanup:** The PostgreSQL-only `bdspro cleanup` command prunes
expired authorization codes and refresh tokens before expired access tokens.
Every sweep is bounded to 2,000 rows per table; operators must monitor backlog
and schedule repeated runs. An expired access token must NOT be deleted while a
refresh-token row still references it: the existing foreign key uses
`ON DELETE CASCADE`, and premature deletion would remove a live refresh token.
Consumed refresh-token rows are preserved until their fixed expiry so that
replay still revokes the active family. The retention sweep does not claim
immediate erasure or production data-retention compliance.

**Concurrency test evidence:** The PostgreSQL integration tests must observe the
exact advisory lock in `pg_locks` with `granted=false` before releasing their
blocking transaction. Timing-only non-completion checks cannot prove a request
reached a lock wait. These tests still do not replace fault-injection or
production lock-wait monitoring.

**Rolling-upgrade constraint:** all pre-existing OAuth tokens/codes from the 26-table version lack durable grants. This candidate intentionally rejects those tokens once the new grant checks deploy. Operators must plan a coordinated cutover and client reauthorization; do not apply this patch as an invisible in-place schema change.

## R38.3a — Request-bound consent proof (API only)

The authenticated `GET /oauth/authorize` response includes a five-minute
`consent_proof` HMAC binding the account, client, redirect, scope, PKCE,
`state` and consent epoch. `POST /oauth/authorize` with `approve=true`
requires both `consent_version` and the proof. The proof key is domain-
separated from the application's JWT secret; the proof is neither an OAuth
access token nor a login credential. The versioned grant transaction in R38.2
rejects successful approval replay and stale versions.

**Not a browser OAuth session:** API Bearer JWTs are still required on both
routes. The proof cannot replace server-side browser authentication, a
same-origin CSRF defense for cookie-authenticated forms, reauthentication,
or accessible UI showing the application's identity and exact disclosures.
Do not expose this as a completed HTML consent/login feature. Separate browser
routes must be designed and tested before third-party client release.

**Browser CSP interoperability:** the consent HTML form POSTs to /oauth/authorize, which redirects to the exact registered client callback on success. Chromium may apply CSP form-action to this redirect; therefore the consent page allows `'self'` and **only that registered callback origin** in form-action, while the browser login page stays restricted to `'self'`. Do not use a wildcard or an unvalidated Origin/Host header. See MDN's form-action redirect note and Django OAuth Toolkit's documented OAuth authorization form behavior.

## Browser login via the existing OIDC provider (R38.4)

If `[server] PUBLIC_ORIGIN` is configured and `[oauth2] REDIRECT_URI` equals **exactly** `PUBLIC_ORIGIN + /users/oauth2/callback`, browser consent also presents "Continue with identity provider". GET `/oauth/login/oidc/start?next=<encoded /oauth/authorize path>` starts the same OIDC Authorization Code + PKCE S256 + nonce workflow as the existing `/users/oauth2/login/start`. It stores the strictly validated local return path in the existing one-time Redis state under purpose `login`; no second provider callback or duplicate OIDC verifier is introduced. The existing `GET /users/oauth2/callback` performs state-cookie comparison, consumes Redis state, exchanges the code, validates ID token, then calls `AuthService.LoginOIDC`. Only when the pending login has a return path does the HTTP adapter issue an HttpOnly browser-session cookie (backed by `sessions`) and 303 redirect to local consent; ordinary API OIDC login retains its JSON JWT response. No JWT is returned to the browser completion path. 

With a different OIDC callback origin, the browser OIDC link/route stays disabled to avoid unusable cross-host cookies; password-based browser login and existing API OIDC remain available. The browser callback **does not** automatically associate an external identity with an existing email account; linking remains its own authenticated flow. There is only one pending OIDC state cookie per browser at a time, so starting a second OIDC operation supersedes the previous pending browser operation, which must fail closed.

## Session-bound authorization authority (R38.5)

Access JWTs created by `AuthService.issueSession` and `AuthService.Renew` now include an authenticated `session_id` claim identifying the existing PostgreSQL refresh session. `id` remains the unique JWT identifier and is not repurposed. The existing JWT signature and `account_id` and token-type verification do not change.

**OAuth authorization is deliberately stricter than ordinary stateless API access.** Both GET and POST `/oauth/authorize` require a live session for Bearer access JWTs. Cookie-based browser authorization already resolves its signed session reference from PostgreSQL. A missing, blocked, expired or cross-account session returns 401; PostgreSQL outages return 503, not a false invalid-credential response. A legacy access JWT without `session_id` remains valid for ordinary endpoints until its normal expiry, but MUST NOT authorize a third-party client: users must sign in again or renew a valid refresh token for a session-bound access JWT. This is a deliberate, narrowly scoped change at the third-party consent boundary, not a systemwide JWT revocation mechanism.

For **approval POST**, a preliminary session read is not sufficient. `ApproveOAuthAuthCode` takes the account/client transaction advisory lock, then locks the live `sessions` row with `FOR SHARE` before it writes the consent grant and authorization code. Session revocation takes an incompatible row lock, so the two transactions are serialized; an already committed revocation cannot be followed by an authorization commit based on that session. The DB checks current expiry using `clock_timestamp()` to avoid a stale transaction-start time during lock waits. No new table, middleware stack or Redis revocation registry is introduced.

**Scope limitations:** other account APIs retain the existing stateless access-token expiry policy; their already-issued JWTs are not retroactively invalidated by a session revoke. Existing OAuth access/refresh grants remain governed by their separate OAuth grant revocation policy. The session requirement protects **new** third-party authorization issuance, not all previously exchanged credentials.
