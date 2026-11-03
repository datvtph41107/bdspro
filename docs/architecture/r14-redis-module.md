# R14 — Redis rate mechanisms and Gin middleware

Gitea reference: `modules/cache`, `modules/queue` and `routers/common` at commit `12b20c7`.

Technical Redis Lua counters and stable SHA-256 hashing are now in `modules/ratelimit`. The module has **no Gin or HTTP response dependency**.

`routers/api/middleware` still owns client IP extraction, Redis key namespaces, source and credential quotas, HTTP 429/503 on admission failure, logging, and fail-open behavior **only** for monitoring. The Redis Lua programs and their return semantics are unchanged: TTL set on first increment, both credential and source counters increase even on denial, and repeat-denial monitoring does not block requests.

The historical `bdspro:...` Redis key prefixes are deliberately preserved as persisted state. No key migration, SQL change, ORM, framework wrapper or route change.

Tests cover stable hashing, atomic increments, TTL preservation and pair quota behavior. Go 1.27.3 CI must confirm middleware tests, PostgreSQL/Redis, 34-route/OpenAPI contract, build/race, Docker and HTTP smoke.
