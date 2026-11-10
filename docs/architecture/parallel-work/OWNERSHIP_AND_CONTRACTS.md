# Ownership Ledger, Interface Contracts, and Conflict Rules

**Principle:** independence is a property of **write sets + stable contracts**, not a chat title, directory appearance, or shared prompt.

## Write-set ledger (first parallel wave)

| Ownership | Sole code writer | Allowed paths during wave 1 | Out of scope |
|---|---|---|---|
| HTTP, routing, context | **A** | `modules/web/**`, `services/context/**`; route **fixtures** only through cross-owner lease | `models/**`, `cmd/**`, Auth/OAuth service implementation |
| DB and domain | **B** | `models/**` only in wave 1; `services/user/**` frozen until PR #218 overlap is reconciled and a coordinator lease is granted | `modules/web/**`, `services/context/**`, `routers/**`, `services/auth/**` |
| Runtime foundations | **C** | `cmd/**`, `modules/setting/**`, `modules/log/**`, `modules/cache/**`, `modules/queue/**`, `modules/graceful/**` | `routers/init.go`, `modules/web/**`, `models/**`, Auth/OAuth |
| API/Auth | **D** | **No production writes initially**. Later: `routers/api/v1/**`, agreed `routers/web/oauth/**`, `services/auth/**`, `modules/auth/**`, subject to gateway | A/B source or DB migration ownership |
| Integration, checks | **0** | `docs/architecture/parallel-work/**`, selected `.github/workflows/**`, `routers/init.go` and other shared contracts **only through approved integration** | unreviewed rewrites of worker-owned packages |

Path ownership is **provisional for each wave** and may be amended only by the coordinator in a reviewable contract PR. A worker may read every directory; a worker may not change other owners' files on its own branch merely to make tests green.

**Always coordinator-owned/shared:** `go.mod`/`go.sum`, `routers/init.go`, `.github/workflows/**`, canonical public API route files, SQLC generator configuration, migrations that cross model domains, `docs/architecture/parallel-work/README.md` and all shared contract docs. A claimed change to one of these needs a short RFC and explicit file lease.

**No shared writable branch.** Chat A continues PR #218 branch; B/C create independent work branches from the current `dev`; D starts as review-only. Do not make B/C branches from PR #218, or merge PR #218 directly into them while A is still changing its public contract.

## Logical dependency / call graph

`cmd` → `routers/init` → `modules/web` route setup → `services/context.APIContext` → `routers/api/v1` actions → `services/{domain}` → `models/{domain}` → `models/db` → SQLC/pgx/PostgreSQL.

Cross-cutting modules (`setting`, `log`, `cache`, `queue`, `graceful`) are *capabilities consumed by concrete call sites*, never empty copies of Gitea folders.

There is legitimate coupling: the HTTP context's `context.Context` carries DB/session authority and is consumed by models/services. The contract's **behavior** is shared; B must not change A's HTTP implementation, and A must not change B's transaction state machinery.

## Frozen boundary contracts for parallel development

### A ↔ B: request authority to DB
- Ordinary Go `context.Context` is the transport of cancellation/deadline and the request-local DB/verified-actor capability. Do not use Gin context inside models.
- Existing BDSPro `models/db.WithStore`, `StoreFromContext`, `WithTx`, `WithTx2`, `TxQueries` are the current migration surface; **inspect current code** before proposing consolidation.
- Authenticated **signed JWT alone does not authorize writes**. Actor and live session must agree; commit-time authority checks and nested transaction rollback-only must not regress.
- A cannot invent new DB transaction APIs. B cannot change the `APIContext.VerifiedActor` semantics or returned actor shape without a reviewed contract update.
- Contract tests should cover cancellation, deadlocks, rollback, session revocation and concurrent writes on both sides as appropriate.

### A ↔ D: typed routes and response
- Gitea-inspired target: framework-neutral `toHandlerProvider`, `middlewareProvider`, `wrapMiddlewareAndHandler`, registered typed `*APIContext`, one request-local scope, response-status short-circuit. Keep Gin as the one matcher.
- Existing public API paths/methods/status/cookies/redirects/ErrorCode are not to be silently changed.
- D may **document** new desired route signatures now, but cannot refactor `routers/api/v1/**` concurrently with A's router call-site migration. Integration coordinator grants a serialized edit window when contract is accepted.
- A may request an explicit lease to migrate D's route declarations **only for compilation** and must not refactor Auth/OAuth behavior under that lease.

### C ↔ A/D: process/bootstrap
- C may design and implement settings/logging/cache/queue/graceful process components in owned files.
- `routers/init.go` and dependency constructors are shared integration points; changes require coordinator-owned integration PR. C proposes a constructor/initializer contract and tests, does not inject it into A's live router branch.
- Redis unavailable, process shutdown, worker lease recovery and error policy are runtime acceptance invariants.

### B ↔ D: auth data and OAuth schema
- B owns `models/**` including SQLC, DB transactions and domain constraints; D owns OAuth route/action/auth-service semantics after its gate.
- Table/column, SQLC signature, actor/session semantics require a **contract change request** with DB and Auth reviewers; no unilateral renaming.
- Do not merge OAuth routes needing absent DB APIs. Use a milestone that declares DB capability stable first.

## Change request and lease

When a worker needs a file owned by another lane, write in its PR:

`RFC: boundary change`
- exact path(s), current owning lane, why the existing contract is insufficient
- old/new Go signatures and a reference to the Gitea source consumer
- migration strategy, tests, rollback/security implications
- proposed integration ordering, owner approval and deadline/window

Coordinator either (a) assigns work to the owner, (b) grants a time-bounded **file lease**, or (c) rejects/splits work. No implicit access just because the worker needs the change to compile.

## Source compatibility acceptance

Every PR must identify Gitea path + function + import relationship, BDSPro before/after, and **what differs for good reason** (Gin vs Chi; SQLC vs ORM). Reject copied packages/helpers without consumer, duplicate source of truth, and "similar name" claims.

This ledger should eventually be enforced by script/CI. The initial script is `tools/parallel/check-owned-paths.sh`, to be run by the worker with its lane and base branch. **Script checks changed paths only; it does NOT prove semantic correctness.**
