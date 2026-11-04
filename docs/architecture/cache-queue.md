# R28 — Cache, queue and Redis resource ownership

Baseline: Gitea `ab295e62e4c7`; BDSPro R28 based on `dev` `9a52c3fb2ca5`.
This is an implementation decision, not a promise to copy all Gitea backends.

## Source evidence (read the implementations)

| Gitea | What it actually owns | BDSPro R28 |
| --- | --- | --- |
| [modules/nosql/manager_redis.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/nosql/manager_redis.go) | shared Redis client registry, reference counting and topology | `modules/conn/manager_redis.go`: process-scoped shared clients with one closing owner; no speculative reference counting |
| [modules/cache/string_cache.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/cache/string_cache.go) | cache semantics, configuration and backends | `modules/cache`: memory LRU, Redis and Memcached-backed string cache |
| [modules/cache/cache_redis.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/cache/cache_redis.go) | Redis adapter borrowing shared connection | Redis cache adapter borrows `redis.Cmdable`, never closes it |
| [modules/queue/base_redis.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/queue/base_redis.go) | Redis backend for Gitea's own queue engine | `modules/queue/base_redis.go`: Asynq-based transport, not a second queue engine |
| [modules/queue/workerqueue.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/queue/workerqueue.go) | worker pool, retries and shutdown | `modules/queue/workerqueue.go`: Asynq server and task handlers |
| [modules/setting/cache.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/setting/cache.go) | cache adapter and TTL settings | `modules/setting/cache.go` optional cache config |

## Package ownership and file naming

```text
cmd/web.go, cmd/worker.go
    -> modules/conn/manager_redis.go  (go-redis connection owner)
    -> modules/queue/base_redis.go       (Asynq producer: own client)
    -> modules/queue/workerqueue.go      (Asynq consumer: own connections)
modules/cache/
    cache.go         explicit backend selection
    string_cache.go  semantics and cache-aside loading
    cache_memory.go  bounded in-process LRU/TTL
    cache_memcache.go Memcached backend with timeout, absolute/relative TTL
    cache_redis.go   Redis adapter, borrowed client
    context.go       request-level cache with explicit invalidation
modules/queue/
    queue.go         task enqueuer contract (Asynq-shaped)
    manager.go       registered workers and deterministic shutdown
    base.go          backend enqueue/close contract and errors
    base_redis.go    Redis Asynq producer (durable task transport)
    base_channel.go  in-process FIFO bounded queue
    workerqueue.go   Asynq worker processor and handler mux
    channel_worker.go channel task handlers and worker goroutines
    config.go        queue weights and concurrency
    logger.go        logging adapter
```

The package is called `conn`, not `nosql` or `redis`. It owns shared Redis *connections*, **not** business authorization, cache invalidation or
queue retry policy. The Redis library is intentionally visible to some
low-level consumers; this is not a fabricated vendor-independent KV API.

## Redis resource lifecycle

1. The process constructs `connection.NewRedisManager()` in `cmd`.
2. `Client(ctx, addr)` checks the address, creates and PINGs one go-redis
   client, and shares it within that process. A failed PING is not cached.
3. Gin admission/OIDC state consumers borrow the client. A future Redis cache
   may borrow the same client; **none of them should call `Close`**.
4. `cmd` calls `manager.Close()` exactly once when the runtime stops.
5. Asynq is separate: it constructs/manages Redis connections internally from
   `asynq.RedisClientOpt`. A successful go-redis PING is only preflight;
   it cannot prove an Asynq worker will remain healthy.

Web and worker are distinct processes; they cannot share a process-local pool.
The R28 manager is not a distributed resource registry or reference-counted
manager. Future password/TLS/Sentinel/Cluster options must be coordinated
with **both** go-redis and Asynq; the current app.ini `[redis] ADDR` is
a plain host:port value.

## Cache semantics and consistency

- A hit returns `(value, true, nil)`.
- A key not found or expired returns `("", false, nil)`.
- A Redis read/write outage returns an **error**, never a fabricated miss.
- Callers decide whether to fail, bypass caching or serve stale data.
- Every value has a positive TTL. The memory backend uses bounded LRU eviction
  and lazy expiration; it does not promise persistence or cluster-wide coherence.
- `GetString` is a simple cache-aside helper without stampede suppression.
- No cache is initialized at startup until a real read-heavy consumer opts in.
- **Do not use `modules/cache` for auth rate limits, OTP attempts, OIDC
  one-time state, session revocation or any other security counter/proof.**
  Those operations require atomic/consistent state semantics.

Optional INI keys:

```ini
[cache]
ADAPTER = memory
HOST = 127.0.0.1:11211  ; only when adapter = memcache
PREFIX = cache:
MAX_ENTRIES = 10000
ITEM_TTL = 5m
```

`setting.LoadCacheSettings` reads these keys. `cache.NewStringCache`
selects the backend; when `ADAPTER = redis`, the caller supplies a borrowed
client obtained from `connection.RedisManager`. `LoadSettings` does not
implicitly start the cache because no production endpoint consumes it yet.

## Real cache backend behaviors

| Backend | Read semantics | Lifecycle and consistency |
| --- | --- | --- |
| Memory LRU (`cache_memory.go`) | Cache hit/miss, TTL, capacity-bound eviction | Process-local; cleared on restart; not shared across replicas |
| Redis (`cache_redis.go`) | `redis.Nil` is a miss; transport errors propagate | Borrows go-redis client from `modules/conn`; does not own its shutdown |
| Memcached (`cache_memcache.go`) | `memcache.ErrCacheMiss` is a miss; other errors propagate | Memcached TTL rules including absolute expiration above 30 days; bounded 2-second client timeout; actual Memcached integration test |
| Request context (`context.go`) | Typed, successful-only memoization, manual key invalidation | Lifetime limited to the request / 5 minutes; not a distributed cache or write-through mechanism |

`PutJSON` and `GetJSON` are serialization helpers on `StringCache`. Errors are never cached as successful values. Cache-aside loaders do not automatically suppress a thundering herd; callers must design invalidation and failure policy for their own data.

## Queue semantics and limitations

- Keep the **existing** `NewRedisEnqueuer`, `NewRedisTaskProcessor`,
  `TaskEnqueuer`, `Handle`, `Start`, `Shutdown`, queue names and task
  payload compatibility.
- Add `NewRedisTaskProcessorWithConfig` for validated queue weights and
  concurrency; `NewRedisTaskProcessor` retains existing defaults.
- `services/worker/OutboxDispatcher` owns SQL outbox claiming and Redis
  enqueue; `services/jobs` own individual job business operations.
- Redis enqueue and PostgreSQL commit are not atomic. The dispatcher currently
  performs enqueue *inside* a SQL transaction; its lock duration, duplicate
  delivery and multi-worker recovery need separate fault-injection acceptance.
- `base.go` gives two backends a common enqueue/close contract; it does **not**
  claim equivalent guarantees. Redis queues are backed by Asynq; in-memory
  `base_channel.go` provides bounded FIFO and explicit `ErrQueueFull`,
  `ErrQueueClosed`, and unsupported-option errors.
- `ChannelTaskQueue` runs local handlers with configurable goroutine count.
  It does not persist tasks or implement distributed locks, retries, delay,
  leasing or scheduled delivery. Never route committed outbox jobs to it.
- `Manager.Register/Get/Shutdown` owns explicitly registered worker lifetimes.
  The actual `cmd/worker.go` registers its Asynq processor; no global init()
  manager is needed.
- Do not replace Asynq with a copied Gitea worker engine. Channel is a distinct
  transient test/local backend and explicitly cannot claim Asynq delivery guarantees.

## Acceptance tests and unproven cases

| Contract | Evidence in this PR | Remaining integration requirement |
| --- | --- | --- |
| Missing Redis address and closed manager | `modules/conn/manager_redis_test.go` | Live Redis PING and restart/close behavior |
| Memory TTL, bounded LRU, cache-aside | `modules/cache/cache_test.go` | Memory pressure and concurrent load |
| Redis cache nil-client policy | `modules/cache/cache_test.go` + Redis integration test | Live Redis prefix, missing key, outage and TTL |
| Memcached key/TTL, hit/miss/expiration | `modules/cache/cache_memcache_test.go` | CI Memcached service; protocol failure policy |\n| Request context memoization and invalidate | `modules/cache/context_test.go` | Caller-managed invalidation after writes |\n| Cache config invalid adapter/TTL/size | `modules/setting/cache_test.go` | Environment override precedence |
| Asynq worker and queue config | Existing queue tests + `config_test.go` | Redis worker reconnect and graceful shutdown |\n| Channel queue bounded capacity, worker execution, concurrency | `modules/queue/base_channel_test.go` | No durability or retry claim |\n| Managed queue registration/close | `modules/queue/manager_test.go` | Worker shutdown under long task |
| Source ownership | `tests/architecture/ownership_test.go` | CI Go 1.27.3 / Docker / smoke tests |

No database migration or HTTP endpoint contract changes are intended in R28.
