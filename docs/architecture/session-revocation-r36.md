# R36.0 — Access and refresh token revocation contract

> **Historical R36 behavior (2026-11-05).** Ordinary protected first-party
> HTTP routes now require a live PostgreSQL session under G1.2b; signed JWTs
> from revoked/deleted sessions are rejected on newly admitted requests.
> See [g1-live-session-authority.md](g1-live-session-authority.md).
> The remainder describes the earlier stateless baseline for audit comparison.


## Confirmed source behavior (2026-11-05)

BDSPro currently issues two purpose-separated HS256 JWTs through
`modules/auth`. `services/auth.AuthService.issueSession` persists a refresh
session plus account connection history in one PostgreSQL transaction before
returning both JWTs. `RefreshTokenDigest` protects the stored refresh bearer
token from direct database replay.

* `POST /tokens/revoke`: verifies the provided refresh JWT and blocks its
  matching session; a repeated revoke is not success.
* `POST /users/me/sessions/revoke_all`: blocks all live refresh sessions for
  the authenticated account.
* `DELETE /users/me/sessions/:id`: blocks an owned refresh session only.
* Password reset deletes persisted sessions after changing credentials inside
  the model transaction, so prior refresh JWTs cannot renew.
* `POST /tokens/renew_access` validates refresh signature, purpose, expiry,
  session ownership, token digest and blocked state.
* `routers/api/middleware.Authenticate` checks only the access JWT's
  signature, purpose and expiry. It **does not** check session rows or live
  account status. Existing access JWTs remain usable until expiry after
  any of the operations above.

The current documented development access TTL is 15 minutes, but actual
deployments can configure `ACCESS_TOKEN_DURATION`. This is a *bounded
stateless access policy*, not immediate access-token revocation. Do not call
`/tokens/revoke` a universal logout from every device, and do not promise
account disable/password reset blocks already issued access JWTs immediately.

## Why no automatic new auth middleware?

Adding `GetSession` or Redis lookup on every request would change both
security semantics and availability/cost: DB/Redis outage would become a
potential protected-route outage, and high-volume routes incur new network
calls. Access JWTs presently have independent JTI values, not a bound
refresh-session ID, so checking the current session also requires a protocol
change. A cache without correct invalidation may not deliver immediate
revocation. Gitea/DRX folder names are not a substitute for this decision.

Before making immediate revocation the accepted contract, define:
1. Required revocation scope: current token/session, all sessions, password
   reset, account disable and stolen access token.
2. Maximum accepted revocation latency in seconds, and behavior when the
   session-authority store is unavailable.
3. Whether old JWTs remain compatible through the rollout.
4. Per-request read/caching budget, p95/p99 latency and load-test evidence.
5. Contract tests proving a revoked access token gets 401 while a different
   account and new valid session still work.

## Existing evidence and remaining work

`services/auth.TestAuthLoginRenewRevoke` proves that revoke denies renewal
while the previous access JWT still verifies. `middleware.TestAuthenticateAccessOnly`
proves access/refresh purpose separation. These are explicit regression tests
of the *current* policy, not evidence for immediate revocation.

Rotating refresh JWTs on each renewal is also a separate protocol change:
today renewal issues a new **access** JWT only and the same refresh JWT may
renew repeatedly until expiry or revocation. Replay detection and concurrency
semantics must be reviewed before changing it.
