# Authentication Abuse Protection — local Engineering Core

## Scope (no AWS, JWT, Redis, or database migration)

BDSPro now enforces bounded, process-local, mutex-protected **token buckets** at
HTTP admission, before JSON binding, Argon2id and database/SMTP work.

| Route | Socket-IP bucket | Independent identifier bucket |
|---|---|---|
| POST /auth/login | 30 / minute | 12 / 15 minutes per normalized email |
| POST /auth/register | 12 / hour | Not included in this stage |
| POST /auth/resend-verification | 12 / 15 minutes | PostgreSQL already enforces 60s cooldown + 6/24h per account |
| POST /auth/verify-email | 30 / 15 minutes | Not included in this stage |

All limits are token buckets: capacity is the initial burst; refill is
continuous over the listed window. 429 uses the existing safe JSON error
contract with `too_many_requests`, opaque message and `Retry-After` rounded
up to seconds. A denied request does not call Auth Service.

## Identity and trust

The per-identifier bucket hashes the lowercased, trimmed submitted email.
Login does not reveal which account exists; success and failure consume
the same quota. A generic IP limiter alone cannot stop distributed attempts
against the same account, which is why these are two **independent** buckets.

The per-client bucket uses `http.Request.RemoteAddr` and `netip` rather
than any user-controlled `X-Forwarded-For`, `Forwarded` or `X-Real-IP`
header. It groups IPv4-mapped IPv6 with IPv4. Unknown/malformed peer
addresses share one `unknown` bucket; they are never individually exempt.
Fingerprints are stored only in process memory, never written in request logs.

## Important limitations and trade-offs

- The limiter is **per process** and resets on restart. With multiple
  replicas, rate limits are multiplied; a shared Postgres/Redis/edge approach
  is needed before multi-instance or public production. This is a local
  defense-in-depth stage, **not a finished brute-force defense**.
- Behind Docker, ingress proxies, or NAT, many users can share one observed
  socket-IP. Never trust arbitrary forwarded-IP headers to "fix" it.
- An attacker can intentionally consume an email's temporary token budget,
  denying login to its real owner. Prefer adaptive risk controls and verified
  edge identity for future production design, not permanent account lockout.
- The bucket map is capped at 4096 entries per server. When exhausted by
  rotating identities/IPs, a new key is temporarily denied (fail closed);
  it is pruned when existing buckets reach full recovery. The cap prevents
  memory growth, but an attacker can still induce a temporary availability
  problem. Tune against real traffic before exposing publicly.
- Limiting successful and failed login attempts equally avoids a
  username/account-existence oracle and blocks repeated costly checks. It
  also means many legitimate logins can be temporarily throttled.
- This does not limit concurrent Argon2id operations, credential-stuffing
  across millions of rotating IPs, global traffic load, or account enumeration
  via existing registration HTTP responses. These are separate security gates.
- Database-backed email resend cooldown and daily cap remain the actual
  durable invariant; no second DB policy is introduced.

## Tests and acceptance

Unit tests use deterministic timestamps to prove refill/expiry, bounded
memory and concurrency. HTTP tests verify 429, Retry-After, JSON
request_id correlation, distinct IP/identity/endpoint policies, and spoofed
forwarded headers not bypassing limits. Existing register/login/logout,
email resend, RBAC, session, CORS and HTTP contracts must keep passing.
Full CI includes migrations, SQLC, Go tests/race and Docker Compose smoke.

Reference: OWASP Authentication Cheat Sheet and Credential Stuffing Prevention
Cheat Sheet (dual key throttling with careful lockout/DoS trade-offs).
