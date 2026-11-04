# Modules — technical capability and backend ownership

The module name describes a **mechanism** with explicit callers and lifecycle,
not an arbitrary dependency or a copy of Gitea's folder list.

## Gitea reference and BDSPro decision

Gitea separates
[cache](https://github.com/go-gitea/gitea/tree/main/modules/cache),
[queue](https://github.com/go-gitea/gitea/tree/main/modules/queue) and
[nosql/manager_redis.go](https://github.com/go-gitea/gitea/blob/main/modules/nosql/manager_redis.go):
cache semantics, queue semantics, and shared connection ownership.
BDSPro calls the latter `kvstore`, because no NoSQL database manager or
LevelDB backend exists.

| BDSPro module | Responsibility | Consumer and constraints |
| --- | --- | --- |
| `kvstore` | `manager_redis.go`: verified shared process-level go-redis clients, explicit close | cmd web/worker, Gin security, OIDC state; no auth policy here |
| `cache` | bounded memory LRU and Redis string cache; TTL, miss/error distinction | opt-in future consumers; **not used for security state** |
| `queue` | Asynq producer, worker processor, task registration, log adapter, queue weights | cmd worker and services/worker; **not** business job policy |
| `setting` | app.ini config, paths, typed settings, cache defaults | cmd, cache construction, service configuration |
| `log` | structured logs, console/file writers and close behavior | runtime and adapters; redact credentials |
| `auth` | password hash/check primitives | services/auth, no Gin or persistence |
| `token` | signed access/refresh token maker | auth operations and HTTP admission |
| `secret` | cryptographic random/derived proof primitives | verification and jobs |
| `validation` | shared email/password input rules | routers and services, not domain authorization |
| `structs` | shared external API option/response structures | routers and services/convert |
| `util` | narrow error categories and shared helpers | models/services; no generic app-wide policy |

**Removed:** `modules/redis`. Its only behavior was client constructor +
PING, now owned by `modules/kvstore/manager_redis.go`. This is a package
ownership correction, not Redis removal. `modules/errors` was already
retired; do not document it as current.

## File organization

```text
modules/kvstore/
    manager_redis.go
modules/cache/
    cache.go
    string_cache.go
    cache_memory.go
    cache_redis.go
modules/queue/
    queue.go
    config.go
    base_redis.go
    workerqueue.go
    logger.go
modules/setting/
    cache.go
    provider.go
    config.go
```

The queue is not Gitea's specialized worker-queue engine because BDSPro
already runs Asynq. The cache adapter borrows Redis clients and cannot close
them. Asynq creates separate internal Redis connections; sharing one go-redis
`*Client` does not automatically share Asynq's internal connection pool.
Details, tests and limitations: [R28 cache/queue ownership](cache-queue.md).

## Additional Gitea module candidates

- `modules/web`: Gitea wraps Chi routing, request binding and middleware.
  BDSPro keeps Gin directly; a second router engine has no demonstrated value.
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
