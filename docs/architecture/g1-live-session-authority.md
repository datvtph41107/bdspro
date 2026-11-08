# G1.2b — First-party live session authority at protected HTTP ingress

## Decision and implementation

This change adopts **PostgreSQL-backed live session authority** for the
ordinary first-party protected Gin API group. It reuses the existing
`models/auth.RequireActiveSession` model operation and the already signed
`Payload.SessionID`. No Redis denylist, extra token service, new migration,
new routing framework or shadow authentication registry is added.

HTTP ownership stays in `services/context`: the existing
`Authenticate` verifies JWT signature, purpose and expiration, followed by
`RequireSession` using Gin's own middleware chain. Only after both succeed
does a request get to its endpoint. The model checks that the persisted
session exists, belongs to the signed account, is not blocked and has not
expired. OAuth consent still uses its own transaction-serialized session
authority; third-party opaque OAuth access tokens are not first-party JWTs.

| Request/event | New protected HTTP request |
| --- | --- |
| Access JWT and active matching session | Passes admission; ordinary domain permissions still apply |
| `/tokens/revoke` committed | Prior session's access JWT is rejected (401) |
| Revoke-one or revoke-all committed | Affected session's access JWT is rejected (401) |
| Password recovery committed (deletes sessions) | Prior access JWT is rejected (401); new login may succeed |
| Token signed for another account's session | Rejected (401) |
| Legacy signed access JWT without session_id | Rejected (401), even if signature is valid |
| Expired/revoked/deleted persisted session | Rejected (401) |
| PostgreSQL/GetSession failure | Rejected (503), not silently accepted or converted to invalid credentials |
| Public login, password recovery request, token renewal/revocation | No new access-session gate; their existing checks remain |
| Delegated OAuth access to third-party applications | Unchanged |

The writer is the existing shared HTTP response helper. Never log raw JWTs,
refresh hashes, private SQL driver error values or client-provided credentials.
Expired/unbound/foreign sessions must be indistinguishable as invalid
access tokens, using `WWW-Authenticate: Bearer error="invalid_token"`.

## Concurrency contract — do not oversell revocation

**A protected HTTP request whose session check starts after the revocation
transaction committed is rejected.** A request that already cleared that
check before revocation may complete and commit a domain write afterward.

The Gin check is a single PostgreSQL read and does **not** hold a
`FOR SHARE` lock over every downstream operation. Requirements for
strict transaction-ordered changes of membership/ownership/credential
state must be enforced in the relevant model transaction with row locks.
Adding `FOR SHARE` only in HTTP context and releasing it before the
domain transaction cannot solve this race. See PostgreSQL 18 isolation:
https://www.postgresql.org/docs/18/transaction-iso.html
and explicit locking: https://www.postgresql.org/docs/18/explicit-locking.html.

## Availability, cost and migration considerations

Live session authority adds one `GetSession` PostgreSQL lookup for each
protected request, including reads that might otherwise avoid the database.
`GET /users/me` already fetches profile data; requiring a live session
adds a separate connection-pool acquisition. There is no allow-cache
that could continue to authorize a revoked session during Redis outages.

A PostgreSQL outage now makes *all* first-party protected endpoints return
503 after cryptographic JWT validation, even endpoints which otherwise
would be read-only and DB-independent. That is a conscious fail-closed
security trade-off, not a zero-cost optimization. The current
`GetSession` SQLC query reads the existing session row; a smaller indexed
projection may later be justified only by measured load.

Existing newer login and renewal paths issue `session_id`-bound access
JWTs. **Legacy unbound access JWTs are deliberately denied** on protected
routes immediately after rollout. Before deploying mixed-version servers,
confirm every active JWT issuer uses the bound format; drain older app
instances and communicate that legacy clients may need to sign in again.
Do not provide a compatibility bypass that restores the security gap.

The configured access-token TTL is still only required to be positive
and has no enforced maximum in `modules/setting`. Stolen access tokens
are revocable by blocking their underlying session, but there is no
per-JTI kill switch that leaves the session active. Account suspension
remains undefined and is not implemented here.

## Evidence gates

- `services/context/session_test.go`: valid, blocked, expired, missing,
  foreign, legacy and wrong-purpose tokens; unavailable PostgreSQL;
  verify one lookup for signed/active-capable tokens and no lookup for
  invalid cryptographic tokens.
- `routers/g1_session_authority_e2e_test.go`: real session revoke and
  password recovery via Gin/SQLC/PostgreSQL/Redis; old access denied and
  unrelated sessions preserved.
- `routers/g1_live_session_acceptance_test.go`: database outage fail-closed,
  unbound JWT, cross-account signed claim and a deterministic in-flight
  request which may complete after revocation.
- `routers/g1_session_authority_cost_test.go`: real PostgreSQL pool
  acquisition delta and exploratory in-process p50/p95/p99 (no production
  load or SLO claim). Expected added DB reads are one per protected request.
- Entire Go/race/vet/build, canonical migrations, SQLC, OpenAPI, worker
  crash recovery, Docker runtime and HTTP smoke must remain green.

## Follow-on acceptance

1. Define security-sensitive domain writes that require strict
   transaction-serialized session authority, not only HTTP admission.
2. Run representative production-like load under expected request rate,
   pool size and PostgreSQL outage before claiming a latency SLO.
3. Plan availability implications and legacy JWT rollout before enabling
   on production deployments. Fail closed must be observable in logs.
4. Evaluate refresh token rotation and account suspension independently;
   neither follows automatically from enforcing session authority.

Background: OWASP JWT and REST security cheat sheets discuss session
revocation and explicitly warn about stale authorization state:
https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_Cheat_Sheet.html
https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html
