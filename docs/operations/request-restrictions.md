# Request Restrictions — BDSPro HTTP/Gin P0

## Scope and intent

A request is accepted only within HTTP resource limits and the relevant authentication / authorization / business constraints. This change limits untrusted traffic on the **active Gin HTTP server** without creating an API gateway or a new security framework.

## Incoming request boundary

- `Gin.SetTrustedProxies(nil)`: `ClientIP()` is the direct connection address, not a client-supplied `X-Forwarded-For` / `X-Real-IP`. **If a reverse proxy is added later**, configure only its actual proxy IP ranges and sanitize the forwarding headers at that proxy. Until then, all clients behind an unconfigured proxy may share the proxy's IP-based limit.
- `Content-Length > 1 MiB`: immediately `413`. Chunked/streamed JSON is limited with `http.MaxBytesReader`; its decoder failure maps to `413`.
- Server `ReadHeaderTimeout=5s`, `ReadTimeout=15s`, `WriteTimeout=30s`, `IdleTimeout=60s`, `MaxHeaderBytes=16KiB`. These are protective boundaries, not an exhaustive connection-level DoS defense.
- Authorization is unchanged: middleware authenticates access JWT, and organization SQL guards object-level access with membership/owner conditions.
- Do not confuse CORS with a security boundary for nonbrowser clients.

## Redis authentication attempt limits

Counters are shared across API replicas. Every request reaching the relevant auth operation consumes one attempt, **whether its credentials are right or wrong**. Atomic Lua INCR + EXPIRE uses two independent fixed windows: direct connection IP and normalized identity (email, verification record ID or token). Values are SHA-256 hashed in Redis keys; no passwords, OTPs or raw tokens are stored as counter keys. A counter TTL starts on the first attempt; repeated denial does not extend it.

| Endpoint | Operation | Per direct IP | Per normalized identity |
|---|---|---|---|
| POST /users | register | 10 per hour | 3 per email per hour |
| POST /users/login | login | 30 per minute | 10 per email per minute |
| GET /users/verify_email | verify_email | 60 per minute | 10 per email verification record ID per minute |
| POST /tokens/renew_access | refresh | 60 per minute | 20 per refresh token per minute |
| POST /tokens/revoke | revoke | 60 per minute | 20 per refresh token per minute |

These are initial protective limits, not measured production capacity targets; consider legitimate shared-IP users, accessibility, NAT, abuse patterns and support procedures before production.

**Outcomes:** below threshold, request continues unchanged; exceeded limit, `429 Too Many Requests` with a coarse `Retry-After` value and no credential details; Redis unavailable, `503 Service Unavailable` (fail closed). Malformed JSON still returns `400`, with body limit enforced. Redis is already mandatory for BDSPro runtime. The current `main.go` explicitly invokes `EnableAuthRateLimiting(redisClient)` before the Gin server is started. Unit-test constructors without Redis retain existing direct handler tests, with dedicated Redis integration tests under `REDIS_TEST_ADDRESS`.

**Limitation:** rate limiting is not equivalent to account lockout, bot detection, distributed attack protection, fraud prevention or SMS cost control. It may throttle legitimate users on shared networks. Put volumetric attacks behind an edge/proxy with suitable connection controls.

## Verification

GitHub CI already provides a disposable PostgreSQL 18 and Redis 7 service.

```bash
go test -count=1 ./api
go test -race -count=1 ./api
go vet ./...
go build ./...
```

Local integration tests can opt in using `REDIS_TEST_ADDRESS=127.0.0.1:6379`. They use random rate-key namespaces and do not flush the Redis database.

Test cases: forged forwarding headers do not change Gin ClientIP, Content-Length and chunked oversized bodies both return 413, Redis outage returns 503, exceeded attempts return 429, concurrent counters cannot exceed the allowance, existing JWT/owner/DB flows remain unaffected.

## Explicitly deferred

OTP-specific resend cooldown and atomic challenge attempts must be implemented alongside the OTP endpoints (not present on dev). Google authorization state and PKCE in draft PR #38 need a separate abuse limit and integration after that PR is merged. No automatic merges with local WSL changes or previous draft PRs.

## Operational safety

Protect `REDIS_ADDRESS` from public access, do not log credentials/OTP/refresh tokens, and preserve existing `request_id` structured logging. This change does not provide production WAF, distributed-IP bot defenses, global connection limits or full load-testing evidence.
