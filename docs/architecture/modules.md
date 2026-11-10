# Modules — technical capability and backend ownership

The module name describes a **mechanism** with explicit callers and lifecycle,
not an arbitrary dependency or a copy of Gitea's folder list.

## Gitea reference and BDSPro decision

Gitea separates
[cache](https://github.com/go-gitea/gitea/tree/main/modules/cache),
[queue](https://github.com/go-gitea/gitea/tree/main/modules/queue) and
[nosql/manager_redis.go](https://github.com/go-gitea/gitea/blob/main/modules/nosql/manager_redis.go):
cache semantics, queue semantics, and shared connection ownership.
BDSPro calls the latter `conn`, because no NoSQL database manager or
LevelDB backend exists.

| BDSPro module | Responsibility | Consumer and constraints |
| --- | --- | --- |
| `conn` | `manager_redis.go`: verified shared process-level go-redis clients, explicit close | cmd web/worker, Gin security, OIDC state; no auth policy here |
| `json` | strict JSON serialization helpers used by existing callers | OAuth2 state/provider payloads and Asynq; Gin responses remain separate |
| `cache` | **NOT PRESENT** after P06; no production consumers justified a backend | add only for a proven read-heavy consumer with invalidation and failure policy |
| `queue` | Asynq Redis producer/processor, worker Manager and validated queue config | cmd worker and services/worker durable SQL outbox; no channel backend |
| `setting` | app.ini config paths, command-specific typed settings, worker queue validation | cmd and services; no cache settings loader |
| `log` | structured logs, console/file writers and close behavior | runtime and adapters; redact credentials |
| `auth` | password hash/check and JWT access/refresh maker/claims | services/auth, services/context, HTTP startup; no DB sessions here |
| `timeutil` | SecondsCeil (Redis rate limit TTL) and Expired (JWT and DB sessions) | services/security, models/auth, modules/auth; no second DB timestamp type |
| `secret` | cryptographic random/derived proof primitives | verification and jobs |
| `validation` | shared email/password input rules | routers and services, not domain authorization |
| `structs` | shared external API option/response structures | routers and services/convert |
| `util` | shared error categories, WithKind and WithCause with safe public messages | models and services/auth; test fixture random helpers are not cryptographic |

**Removed:** `modules/redis`. Its only behavior was client constructor +
PING, now owned by `modules/conn/manager_redis.go`. This is a package
ownership correction, not Redis removal. `modules/errors` was already
retired; do not document it as current.

## File organization

```text
modules/conn/
    manager_redis.go
modules/json/
    json.go
    jsonv2.go
    json_test.go
modules/auth/
    password.go
    jwt.go
    jwt_payload.go
    token.go
modules/timeutil/
    duration.go
    expiry.go
modules/util/
    error.go
    random.go
modules/queue/
    queue.go
    manager.go
    base.go
    base_redis.go
    workerqueue.go
    config.go
    logger.go
modules/setting/
    provider.go
    config.go
    queue.go
    worker.go
    database.go
```

BDSPro uses Asynq with a PostgreSQL outbox and does not copy Gitea's specialized worker-queue engine. P06 removed both the non-durable channel backend and unused cache package. Asynq creates separate internal Redis connections; sharing one go-redis `*Client` does not automatically share Asynq's internal connection pool.
Current contracts, tests and limitations: [P07 queue, outbox and Redis ownership](cache-queue.md).
JSON semantics, Gitea evidence and compatibility: [JSON serialization policy](json.md).

## Additional Gitea module candidates

- `modules/web`: Gitea wraps Chi routing, request binding and middleware.
  BDSPro now has one non-generic `web.Router` compiling actions onto one Gin matcher; `Server.Handler` serves `CORS(WrapIngress())`.
- `modules/graceful`, `modules/process`: add only when `cmd` context and
  native HTTP/Asynq shutdown fail a concrete lifecycle acceptance test.
- `modules/storage`, `modules/httpcache`, `modules/paginator`: add with
  actual persisted files, HTTP cache policy or stable pagination contract.
- `modules/repository` and `modules/git`: Gitea-specific Git repository
  mechanisms; neither means a SQLC repository pattern.
- `models/db`, `models/organization`, and Gitea `models/perm/access`
  remain persistence/domain ownership, **not** a generic `modules/perm`.

## Admission rule

Before adding a module, state its caller, backend, lifecycle owner, failure
semantics, configuration source and discriminating test. No empty future
directories, duplicate queue engines or cache fallback for OTP/rate-limits.

## P06 ownership update

The removed `modules/cache/**`, `modules/setting/cache*` and `modules/queue/{base_channel,channel_worker}*` implementations are **not available** at P07 integration. Do not document or import them as active features. A future cache or in-process queue must have real consumers, an accepted failure policy and a new ownership decision. Runtime queue defaults/validation reside in `modules/setting/queue.go` and `modules/queue/config.go`.

## R30 acceptance

JWT signing moved from `modules/token` into `modules/auth` without changing
stored refresh JWTs. See [JWT, JSON, timeutil and safe error decisions](r30-auth-json-timeutil.md).

Gitea's `timeutil.TimeStamp` is not copied because SQLC already owns
PostgreSQL `time.Time` representation. `json` exposes only operations that
existing production callers need. `util.WithCause` hides driver messages while
preserving `errors.Is`/`errors.As` for diagnosis.
