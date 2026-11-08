# G1.2a — First-party access JWT and session authority acceptance

Baseline `dev@9ca594ff80a31cf55ec8ff7127152a37a7600778`.
This checkpoint **documents and tests the currently deployed semantics**;
it does not silently introduce immediate access-token revocation.

## Source facts

- `services/auth.AuthService.issueSession` creates the refresh JWT whose ID
  identifies the persisted PostgreSQL session, then issues a separate signed
  access JWT with `SessionID` bound to that session.
- `services/context.Authenticate` checks JWT signature, access purpose,
  expiry and account ID. It does **not** read PostgreSQL session state.
- `/tokens/renew_access` checks the persisted refresh session, ownership,
  blocked status, expiry and refresh token digest before issuing another
  access JWT.
- `/tokens/revoke`, `DELETE /users/me/sessions/:id` and
  `/users/me/sessions/revoke_all` invalidate refresh session authority.
- Password recovery deletes first-party sessions and revokes OAuth grants
  and OAuth credentials transactionally; already-issued first-party
  stateless access JWTs survive until expiry.
- OAuth **authorization-code issuance** is not a generic first-party
  protected API route. It explicitly requires active session authority
  and serializes authorization with session revocation.

Do not confuse a signed SessionID claim with a live session assertion.
Likewise, a successful `/users/me` response usually reads PostgreSQL for
a profile; this is not evidence that its JWT session was checked.

## Concrete regression evidence

`routers/g1_session_authority_e2e_test.go` runs against migrated,
disposable PostgreSQL and Redis and the real Gin route tree:

1. Register account A and issue two sessions; issue one session for account B.
2. Revoke A's first refresh session via `/tokens/revoke`. Verify session row
   is blocked and that its refresh JWT cannot renew (401). The *already-issued*
   access JWT still reads `/users/me` (200) and **mutates** `/users/me/profile`
   (200, persisted), while the other sessions remain unaffected.
3. With the just-revoked first session's access JWT, call
   `/users/me/sessions/revoke_all` (204). This **currently succeeds** and
   invalidates account A's remaining refresh session; its access JWT still
   authorizes ordinary protected API endpoints. Account B remains intact.
4. Reset password through the HTTP endpoint. The old refresh JWT cannot
   renew; old password cannot authenticate. Yet old access JWT still reads
   `/users/me` and `/users/me/settings`; the new password issues a
   fresh session.

Tests are evidence of CURRENT behavior, not endorsement of a compromised
access JWT being able to revoke all refresh sessions.

## Security policy gate still open

A stolen access JWT, or a logged-out browser with leaked access credentials,
can perform any route that requires only current stateless JWT authentication
until JWT expiry. Configured access TTL sets the exposure bound and must be
reviewed per deployment. No canonical account-disabled state exists.

Three defensible choices need separate acceptance:

| Option | Revocation semantics | Dependency and operational cost |
| --- | --- | --- |
| A. Retain stateless JWT (current) | Access continues until expiry | No extra DB/Redis lookup in Authenticate; logout is not immediate |
| B. Live session authority at every protected request | Immediate on committed DB change | Per-request database check; fail closed on authority-store outage; legacy unbound JWT rollout needed |
| C. Targeted session authority / fresh proof for security-sensitive operations | Strict only on selected operations; reads may remain stateless | Explicit route inventory, split contracts, documented mixed semantics and test coverage |

**No universal policy is inferred solely from RFC 9700.** RFC 9700
addresses OAuth 2.0 authorization and token security, including refresh
rotation/sender constraints for public clients; these first-party signed
JWTs and OAuth opaque delegated credentials are separate BDSPro protocols.
OWASP JWT guidance discusses denylist and short TTL as potential controls,
not a free proof of immediate revocation.

- RFC 9700: https://www.rfc-editor.org/rfc/rfc9700.html
- OWASP JWT: https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_Cheat_Sheet

## Decision inputs before production enforcement change

1. Define **which** operations must stop after logout, revoke-all, password
   reset, stolen-token report, or eventual administrative account suspension.
   Consider user data mutation, team/invite management and OAuth client
   management separately from read-only profile operations.
2. Define maximum access revocation latency: zero seconds after database
   commit, a measured short grace period, or access TTL.
3. Choose PostgreSQL/Redis outage policy for authority resolution:
   always reject, allow within a bounded cache window, or explicit
   read-only degradation. Never claim immediate revocation while a stale
   allow-cache can authorize.
4. Define treatment of legacy access JWTs with no `SessionID`, signing
   key rotation, token issuer/audience, and rollout across mixed versions.
5. Measure p95/p99 and DB load under authenticated request rate; compare
   authority query vs existing profile/contact DB reads. Do not invent
   throughput or latency.
6. Add negative HTTP and concurrency tests for chosen policy, including
   a revoked session, a different account, password recovery, expired JWT,
   a stolen access JWT and a database outage.

**G1.2a is evidence consolidation only; G1.2b remains a security decision
gate.** Do not merge a global `GetSession` middleware or Redis denylist
without explicitly accepting its operational semantics.
