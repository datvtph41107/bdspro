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
| `json` | only Marshal, Unmarshal and NewDecoder backed by strict JSON v2 | cache, OAuth2 state/provider payloads, Asynq; Gin responses remain separate |
| `cache` | memory LRU, Redis, Memcached, JSON and request-context memoization; TTL/invalidation/miss/error | opt-in future consumers; **not used for security state** |
| `queue` | Asynq Redis producer/worker, bounded transient channel backend, worker Manager, logging and config | cmd worker and services/worker; transient channel has **no retry/durability** |
| `setting` | app.ini config, paths, typed settings, cache defaults | cmd, cache construction, service configuration |
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
modules/cache/
    cache.go
    string_cache.go
    cache_memory.go
    cache_redis.go
    cache_memcache.go
    context.go
modules/queue/
    queue.go
    manager.go
    base.go
    base_redis.go
    base_channel.go
    workerqueue.go
    channel_worker.go
    config.go
    logger.go
modules/setting/
    cache.go
    provider.go
    config.go
```

The Redis queue is not Gitea's specialized worker-queue engine because BDSPro
already runs Asynq. The channel backend has independent, deliberately non-durable semantics and is not used for committed jobs. The cache adapter borrows Redis clients and cannot close
them. Asynq creates separate internal Redis connections; sharing one go-redis
`*Client` does not automatically share Asynq's internal connection pool.
Details, tests and limitations: [R28 cache/queue ownership](cache-queue.md).
JSON semantics, Gitea evidence and compatibility: [JSON serialization policy](json.md).

## Additional Gitea module candidates

- `modules/web`: Gitea owns Chi grouping, binding and middleware conversion.
  BDSPro uses a typed `Router[T]` over Gin Groups plus request-local form storage
  and `BindJSON[T]`; Gin remains the only middleware executor. A method called
  `BeforeRouting` or `AfterRouting` would need to prove its *actual Gin ordering*
  before being added, because copying Chi names does not reproduce Chi behavior.
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

## R30 acceptance

JWT signing moved from `modules/token` into `modules/auth` without changing
stored refresh JWTs. See [JWT, JSON, timeutil and safe error decisions](r30-auth-json-timeutil.md).

Gitea's `timeutil.TimeStamp` is not copied because SQLC already owns
PostgreSQL `time.Time` representation. `json` exposes only operations that
existing production callers need. `util.WithCause` hides driver messages while
preserving `errors.Is`/`errors.As` for diagnosis.
