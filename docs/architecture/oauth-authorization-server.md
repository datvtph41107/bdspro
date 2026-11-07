# OAuth2 authorization-server contract

BDSPro operates a restricted OAuth2 Authorization Code + PKCE S256 server for registered public clients. It separately consumes OIDC identities from external providers through `oauth2_credentials`; these are not BDSPro-issued grants.

`/oauth/authorize` requires an existing BDSPro JWT and explicit consent. `/oauth/token` supports only `authorization_code` and `refresh_token`, returns opaque random credentials, stores SHA256 digests, and rotates refresh tokens once per use. `/oauth/userinfo` requires a live OAuth access token scoped `profile:read`. Internal JWTs cannot be substituted for OAuth access tokens, or vice versa.

Refresh rotation, refresh-token replay revocation, and explicit family revocation use the same transaction-scoped PostgreSQL family advisory lock. A preliminary lookup determines the immutable family key; the token is always reread and checked after acquiring that lock. Acquiring a token row lock first would risk deadlock. This prevents a concurrent rotation from inserting a new refresh token after a revoke-family UPDATE has taken its statement snapshot. Do not replace the family lock with token-only `FOR UPDATE`; they protect different invariants.

Data tables: `oauth_clients`, `oauth_auth_codes`, `oauth_access_tokens`, `oauth_refresh_tokens`. All four are distinct from `sessions` and `oauth2_credentials`. Each table is created directly in its own ordered up/down migration, with no mirror, trigger or rename.

Account authorization management (R38, restricted API): authenticated `GET /users/me/oauth/authorizations` lists up to 100 currently active application/account relationships derived from an unused code, unrevoked access token or live unconsumed refresh token. `DELETE /users/me/oauth/authorizations/:client_id` idempotently invalidates that account's unused codes and both token types, without revoking the client application or any other user. An app's creator is not automatically an authorizer. New explicit consent may authorize a client again. All issuing/rotating operations and account-level revocation share the `(account_id, client_id)` transaction advisory lock, then the refresh-family lock where applicable. A request already processing before revocation may finish; revocation prevents subsequent credential verification. These endpoints are **not** historical consent records, have no persistent grant, and are not a complete browser OAuth experience.

The authorization endpoint currently requires an API Bearer JWT rather than implementing browser cookie-based login and HTML consent. This is a restricted API-oriented implementation, not a complete third-party browser OAuth user experience. Only `profile:read` is supported. Under the current privacy-minimized contract this scope returns only the subject identifier from `/oauth/userinfo`, never `email` or `email_verified`. A separate contact-data scope, human-readable consent disclosure, and per-user authorization lifecycle must be accepted before expanding those claims; this endpoint is not an OpenID Connect UserInfo provider. Confidential clients, password grant, arbitrary resources and an OpenID Provider discovery surface are not implemented.

**Migration compatibility:** This 26-file canonical ledger is not equivalent to the previous migration versions 1–21. Never force a version or use it against existing data. Provision a new database and conduct a verified data transfer as a separately accepted cutover.

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
