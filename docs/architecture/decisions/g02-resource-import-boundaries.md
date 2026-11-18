# G02-03 / G02-05 — Resource-owner import boundary (proposal)

Status: **PROPOSED / review requested**, not accepted as the final BDSPro subsystem architecture.
Baseline: `dev`, 2026-11-18. Scope: Web and Worker PostgreSQL/Redis process resources, not Organization or authentication business semantics.

## Problem and evidence

A resource owner currently imports process-wide setting and graceful orchestration, coupling its initialization to application mode, config state, and an initialized global graceful manager.

Observed source on baseline:
- `models/db/engine_init.go::InitEngine` read `setting.Database.DBSource` and registered `graceful.GetManager().RunAtTerminate(CloseEngine)`.
- `modules/conn/redis_init.go::InitRedis` read `setting.Redis.Address` and registered `RunAtTerminate(CloseRedis)`.
- Web: `cmd/web.go::serveInstalled → routers.InitWebInstalled → routers/common.InitDBEngine/InitRedis → owners`.
- Worker: `cmd/worker.go::serveWorker → startup.Run → owners`.
- Both owners already implement idempotent process-level cleanup, including a permanently closed state.
- `modules/graceful/manager.go::doShutdown` waits for managed activities before `doTerminate` invokes termination hooks. Hooks are concurrent: registration order is **not** a cleanup dependency order.

Gitea contrast: `go-gitea/gitea/routers/init.go::InitWebInstalled` explicitly orchestrates package initializers; Gitea's `services/actions/init.go` uses `setting.Actions.Enabled` and `graceful` inside its initializer. That is observed source behavior, not a mandate to copy its import graph.

CRM v4 contrast: `datvtph41107/bdspro-backend/crm-service/wire/wire_gen.go::InitializeApp` expresses a generated constructor graph and cleanup functions. Construction graph correctness alone does not establish activity drain or runtime readiness.

## Scoped target and implementation

The resource owners now accept their actual resource configuration explicitly:

```go
db.InitEngine(ctx, setting.Database.DBSource)
conn.InitRedis(ctx, setting.Redis.Address)
```

The Web composition boundary (`routers/common/db.go` and `routers/common/redis.go`) registers their corresponding `graceful.RunAtTerminate` hooks immediately after successful initialization; Worker composition (`cmd/worker.go`) does likewise inside its startup steps.

```text
cmd/web → routers/common → models/db, modules/conn
                  │               (no process setting/graceful imports)
                  └→ graceful + setting

cmd/worker → setting + graceful → models/db, modules/conn
```

This retains process-scoped ownership, rollback hooks, transaction selection, SQLC, Redis client sharing, transport setup, and existing mode-specific startup. It does not introduce new service interfaces, replace the graceful manager, or change business routes.

## Explicit architecture constraints

- **ARCH-RESOURCE-001:** `models/db` and `modules/conn` must not import `modules/setting` or `modules/graceful`. Resource owners receive inputs and expose cleanup; composition arranges process policy and lifetime.
- **ARCH-SERVICE-001:** `services/*` must not directly import `cmd/*` or `routers/*`; no reverse dependency from business operations into process/HTTP delivery.
- No blanket prohibition on `services/org → models/db` yet. Context-bound SQLC transactions and owner-check SQL are currently business-critical and must be preserved.
- `modules/setting → modules/ipactive` remains OPEN pending an ownership/duplication cost analysis. Avoid introducing mirror policy types merely to remove an import.
- Configuration enablement, resource ownership, long-running activity and runtime readiness are distinct contracts; none requires a universal `Subsystem` interface.

## Cost and comparison — static evidence, not production benchmarking

| Measure | Before | After | Meaning |
| --- | ---: | ---: | --- |
| Forbidden direct import edges, the two owners → setting/graceful | 4 | 0 | Stronger package independence, statically checkable |
| New direct imports at Web composition | 0 | 3 | Additional wiring visibility: graceful in DB adapter; graceful and setting in Redis adapter |
| New direct imports in Worker | 0 | 0 | Worker already imported setting and graceful |
| Existing Web/Worker process modes changed | 0 | 2 | Migration touches both startup paths |
| Business services/SQL/HTTP routes modified | 0 | 0 | Business semantics intentionally out of scope |

**Net targeted direct-import delta:** four removed and three added (minus one edge), before considering the newly added architecture-check tooling package. A smaller import count is *not* by itself evidence of lower production runtime cost.

**Added maintenance cost:** explicit DSN/address arguments at composition call sites; explicit termination registrations and a checker with unit tests; a dedicated CI job. **Potential value:** less global coupling and more obvious configuration/cleanup ownership; easier ability to unit-test resource initialization without setting/graceful bootstrap (requires follow-up test evidence).

**Not measured:** CPU, memory, PostgreSQL pool overhead, Redis connection count, request latency, production MTTR, shutdown latency, running Go application integration tests, or operational incident rate. No claims of improvements on these measures.

## Failure and compatibility matrix

| Failure injection | Required behavior | Evidence state |
| --- | --- | --- |
| PostgreSQL initialization fails | No published pool; no cleanup hook for failed initialization | Source-reviewed, integration test pending |
| Redis initialization fails after DB succeeds | `startup.Run` rolls back completed DB step | Source-reviewed, integration test pending |
| Handler construction fails after startup | Graceful termination hooks still close resources | Source-reviewed, integration test pending |
| SIGTERM during request or task | Managed activity drain occurs before termination cleanup | Source-reviewed; live protocol test pending |
| Redis disappears at runtime | Protected IP admission fails closed (HTTP 503) | Existing request handler source; live Redis test pending |
| Web/Worker with valid config | Startup remains behaviorally equivalent | **Full Go test suite pending** |

Risk: cleanup hooks must be registered exactly once after each successful process-owned resource initialization; this is now a composition responsibility. Both owners retain idempotent close semantics. Tests must verify no race between registration, shutdown initiation and active resource consumers.

## Verification

```bash
# Requires the Go toolchain declared in go.mod (go 1.27.3).
go test ./tools/archcheck
go run ./tools/archcheck
go test ./...
go test -race ./modules/graceful/... ./models/db/... ./modules/conn/... ./routers/common/...
```

The isolated checker unit tests were executed locally under Go 1.23.2 with `GO111MODULE=off go test -v` and passed (three top-level tests, including six subtests). The complete BDSPro suite could not be executed in that environment because the repository declares Go 1.27.3 and the full checkout/toolchain was unavailable. CI status at final PR head must be inspected before approval.

## Alternatives considered

1. **Retain global coupling:** cheapest code change, but process-global dependency continues inside resource owners. Reasonable baseline; less suitable for explicit reusable ownership.
2. **Move config + lifecycle registration to the current composition boundaries (this PR):** limited migration, no new framework, preserves existing startup sequencing. Requires wiring at Web and Worker paths.
3. **Introduce a generic DI/lifecycle container:** substantially more contracts, migrations and cognitive overhead; no evidence current scope warrants it.

## Review / rollback gates

Reviewers should validate:
1. Actual Go build/tests on matching toolchain and branch head.
2. Source search for all `db.InitEngine` and `conn.InitRedis` callers including tests and build-tagged source.
3. Web and Worker startup with PostgreSQL/Redis; injected failure and SIGTERM tests.
4. Graceful cleanup on errors after `startup.Run`.
5. Whether any new subsystem repeats these forbidden imports before expanding the rule set.

Rollback: revert this PR; no database migration, API or persistent data format change was introduced.

This proposal does **not** claim that the entire G02 research agenda or subsystem architecture has been accepted.
