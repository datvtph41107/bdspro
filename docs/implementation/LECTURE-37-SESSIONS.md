# Lecture #37 — Session Lifecycle (BDSPro adaptation)

TechSchool SimpleBank Lecture #37 teaches managing long-running sessions
using a refresh token. BDSPro currently uses **PostgreSQL-backed opaque
session cookies**, not JWT/PASETO access and refresh token pairs. This is
an intentional adaptation of session lifetime and revocation concepts,
not a claim that JWT refresh tokens have been implemented.

## Session contract

- Login still verifies email and Argon2id password, then issues a fresh
  32-byte random opaque token, stored only as its SHA-256 digest.
- Sessions still expire after **24 hours** (absolute expiry, not sliding).
- On each successful login, the oldest active sessions are revoked until
  at most **5 active sessions** remain for that account. Selection is
  deterministic: `created_at DESC, id DESC`.
- Expired and already-revoked rows remain as audit history; they are not
  counted as active sessions. No new schema or migration.
- A login transaction takes an account row `FOR UPDATE` before inserting
  the session and pruning. This serializes concurrent logins.
- **POST /auth/logout-all** requires the existing session cookie and CSRF
  proof under the existing browser security contract. It locks the same
  account row and revalidates the caller's session before updating all
  of the account's sessions (including the caller). Success is empty
  HTTP 204 and clears the current browser's cookie.
- Expired/revoked token is HTTP 401. Database failure is HTTP 503,
  preserves the browser cookie on failure, and exposes no DB diagnostics.
- A future correct login is allowed after logout-all.

## Concurrency and scope

Account-wide actions follow lock order `accounts FOR UPDATE` -> `sessions`.
A login and logout-all transaction therefore have an unambiguous order.
Requests that authenticated *before* a revocation commit may finish
according to their already-admitted request context; later authentication
requests fail. The API does not cancel in-progress business work.

A single-session logout still revokes just one token (old API contract).
This route does not acquire an account-wide lock; it may invalidate
the caller's token in parallel with logout-all, and logout-all revalidates
its token under the account lock rather than relying on a prior HTTP check.

The cap is an explicit initial product policy, not a security standard.
It may log out older devices unexpectedly. Review per-device session
management and user-visible notices before production. Old revoked
sessions will accumulate in PostgreSQL; retention/purge policy is a
separate maintenance decision. No refresh token, JWT, AWS or Redis added.

## Acceptance

PostgreSQL integration tests prove eviction of oldest tokens, at most five
active sessions after concurrent logins, logout-all invalidating all
tokens, re-login, and account row locking. HTTP tests prove cookie/CSRF,
401 vs 503, cookie-clearing only on success and response body contracts.
CI must pass SQLC generated diff, gofmt, migrations, Go tests/race and
Docker Compose smoke. This is a source-only Engineering Core milestone.
