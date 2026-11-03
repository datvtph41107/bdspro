# Modules — technical mechanisms, not a universal "core"

## How Gitea uses modules/

Gitea main includes many modules because its product executes Git operations,
serves multiple HTTP surfaces, manages issue/package content, renders a web UI,
uses several caches/queues and has extensive infrastructure adapters.
Presence in Gitea alone does **not** justify a BDSPro package.

Sample source:
- [Gitea modules/cache/cache.go](https://github.com/go-gitea/gitea/blob/main/modules/cache/cache.go): an initialized shared cache with specific lookup semantics.
- [Gitea modules/queue](https://github.com/go-gitea/gitea/tree/main/modules/queue): task transport and typed queue mechanisms.
- [Gitea modules/storage/storage.go](https://github.com/go-gitea/gitea/blob/main/modules/storage/storage.go): object storage interface with registered drivers.
- [Gitea modules/session/session.go](https://github.com/go-gitea/gitea/blob/main/modules/session/session.go): HTTP cookie/session middleware, which is NOT equivalent to JWT session records.
- [Gitea modules/repository](https://github.com/go-gitea/gitea/tree/main/modules/repository): Git-repository technical logic used by higher-level repository workflows.
- [Gitea modules/graceful](https://github.com/go-gitea/gitea/tree/main/modules/graceful), [proxy](https://github.com/go-gitea/gitea/tree/main/modules/proxy), [validation](https://github.com/go-gitea/gitea/tree/main/modules/validation): concrete process, outbound HTTP and validation mechanisms.
- [Gitea modules/setting](https://github.com/go-gitea/gitea/tree/main/modules/setting), [log](https://github.com/go-gitea/gitea/tree/main/modules/log): long-lived application infrastructure.

These modules are **not all generic utilities**: some are domain-specific
technical components (Git operations, markup, repository), others are cross-cutting
mechanisms (cache, settings, storage). Their value comes from owned behavior
and callers, not from being in a "core layer".

## Existing BDSPro modules

| Package | Code and responsibility | Consumers / invariant |
| --- | --- | --- |
| `setting` | `path.go`, `provider.go`, typed config; selection and validation | CLI web, worker, migrate; no DB connections |
| `log` | manager, formatted API, structured events, console/file writer lifecycle | HTTP, worker, CLI; errors must not leak credentials |
| `redis` | client setup, ping, cleanup on startup failure | HTTP throttling and worker |
| `queue` | Asynq Redis enqueuer, task processor/mux, queue-level logger | `cmd/worker` and outbox; **transport**, not job business |
| `auth` | bcrypt hash/verification | authentication service; no Gin or account repository |
| `token` | JWT maker, payload and access/refresh validation | services/auth, HTTP bearer admission |
| `secret` | cryptographic random hex bytes | verification/OTP proof generation; not a secrets manager |
| `errors` | application error identities and PostgreSQL classification helper | services produce identities; HTTP maps externally |
| `util` | random demo-data helpers | review for migration into test fixtures; not authorization/security |

The package `modules/queue` is **real**: it now owns the reusable Asynq
transport that was previously mixed into `services/worker`. The latter
continues to own persistence-to-queue outbox policy and task-to-business
adaptation. This is a *boundary change*, not just a directory rename.

```text
cmd.runWorker
  -> queue.NewRedisTaskProcessor (transport lifecycle, queues, mux, logs)
  -> queue.NewRedisEnqueuer (Redis transport)
  -> worker.NewOutboxDispatcher (SQL outbox processing)
  -> worker.NewVerifyEmailHandler (adapts Asynq to jobs)
  -> jobs.SendVerificationEmail (business operation)
```

Where should an interface live? Prefer the narrow consuming boundary.
`queue.TaskEnqueuer` currently captures a transport contract using native
`asynq.Task`; it is **not** a transport-agnostic general job interface.
Only introduce a higher-level task contract if replacing Asynq becomes a
real requirement and tests establish the benefit.

## Requested Gitea package candidates — admission criteria

| Name | Actual responsibility to prove | Decision today |
| --- | --- | --- |
| `cache`, `httpcache` | bounded cached data, invalidation, memory pressure; HTTP caching semantics | No general cache: Redis rate limiting is not a cache |
| `storage` | durable files, stream/URL policy, local/S3 lifecycle | No owner until dossier/asset files exist |
| `session` | cookie-based HTTP session middleware | Not introduced: JWT + DB sessions already have an explicit contract |
| `graceful` | multiple independent process lifecycles requiring coordinated stop/drain | Today context and native HTTP/Asynq shutdown; evaluate on failure injection |
| `proxy`, `httplib` | shared outbound proxy/timeout/SSRF controls | Only if OAuth/SMS clients need the same enforced policy |
| `repository` | common technical repository operation in Gitea is Git-specific | Do not create a generic SQL repository facade around SQLC |
| `public`, `web`, `templates` | static assets, rendered web utilities | Current two account pages and Swagger belong to top-level `templates/` |
| `private` | Gitea's internal HTTP client for Git/SSH subprocesses | No internal HTTP protocol in BDSPro |
| `options` | shipped static catalogs/license/gitignore data | No BDSPro static catalog yet |
| `paginator` | paging state and page-link rendering | Implement when a real endpoint defines stable paging semantics |
| `validation` | shared parsers/validators and domain-independent rules | Current Gin request binding and service invariants suffice |
| `structs` | shared API structs or independent transport contracts | Keep API requests/responses in routers until another consumer proves sharing |
| `test`, `testlogger` | fixture/log tools imported by test suites | Add only with real repeated test plumbing |
| `migration` | product/domain import-export operations in Gitea | Database schema migrations stay `models/db` + `db/migration` |
| `user`, `auth` | reusable identity **primitives** | bcrypt/JWT primitives exist; workflows remain in services/auth |
| `secret` | cryptography / encryption | RandomHex exists; rotation and storage need a real design |
| `util` | narrowly scoped genuinely reusable helpers | Avoid broad junk drawer; move test-only helpers when safe |

## A package is admitted only after five checks

1. Name a present caller and actual failure/use case.
2. Write the behavior and lifecycle it owns, including config/error policy.
3. Show the import direction; technical module must not import an HTTP handler.
4. State the native or existing mechanism it replaces and its maintenance cost.
5. Add a discriminating test (behavior/failure/cleanup), not just construction.

Do not introduce `modules/common` or `modules/core` as catch-all directories.
Gitea does not provide permission to create wrappers around existing
`net/http`, Gin, SQLC or Redis functionality that already works.
