# R02 implementation plan — reviewable PR boundaries

## Branching and source integrity

1. Snapshot R01 `e1c3085321a26ee244809be80977155b2a10df1d` locally. The planning baseline is PR #91 (`refactor/gitea-ownership-r01`), **not** `dev`.
2. R01 is not on `dev` at planning time; do not assume that stacked PRs are merged. Capture the ancestry and intended base/head SHAs for each PR.
3. Create one local branch per PR. Never push/merge/rebase a remote branch without an explicit instruction.
4. Each PR states the **pinned** Gitea file/function, BDSPro before/after call path, contracts preserved, run commands and results.
5. Keep interim commits buildable. Prefer test → change → test/docs and small behavior-preserving diffs.

## PR sequence

| ID | Subject | Target source | Primary acceptance |
|---|---|---|---|
| R02-00 | Baseline and characterization | `docs/architecture`, route inventory, tests, benchmark harness | Source SHA and 34 legacy route pairs recorded; behavior tests selected. **This artifact starts here.** |
| R02-01 | Entry point and startup | `main.go`, `cmd/*`, `routers/init.go` | Correct init/close order, one dependency owner, no CLI regression. |
| R02-02 | Routing mechanism | `modules/web/*` | Chi group/mount, before/after, typed adaptation, 404/405, short-circuit, benchmark. |
| R02-03 | Context and response | `services/context/*`, `modules/reqctx` only if needed | Principal/request isolation, response tracking, cancellation and headers. |
| R02-04 | SQLC DB core | `models/db`, `db/migration`, `db/query` | One pgx transaction owner; rollback/cancellation/constraint behavior preserved. |
| R02-05 | Organization model | `models/organization`, `services/organization` | Create owner membership atomically, owner predicate on writes, 404/403 behavior preserved. |
| R02-06 | Account/identity model | `models/account`, credentials, Google/phone/verification | Password/OAuth signup, OTP wrong-attempt commit, replay/lockout, outbox unchanged. |
| R02-07 | API shapes and endpoints | `modules/structs`, `services/convert`, `routers/api/v1/*` | Every endpoint callable under its original external URL. |
| R02-08 | HTTP cutover | `routers/init.go`, remove old Gin runtime after parity | Same method/path/status/header/JSON/security; no parallel production routers. |
| R02-09 | Asynchronous infrastructure | `modules/queue`, `services/jobs`, `services/worker`, outbox | Stable task type/payload; recovery after Redis outages and worker restart. |
| R02-10 | Templates and OpenAPI | `templates`, Swagger generation/validation | Assets, HTML escaping, OpenAPI paths/response schema verified. |
| R02-11 | Operations | `modules/setting`, Dockerfile, Compose, Makefile, CI | DB 18, Redis 7, mailpit, API, worker, migration/readiness/shutdown. |
| R02-12 | Acceptance and cleanup | Integration/failure/performance suites; old code cleanup | No lost tests or duplicate mechanisms; documented before/after measurements. |

PR ordering may be split into smaller internal commits, but not merged into one unreviewable change. Any cross-cutting change must explicitly name the invariant it preserves.

## Mandatory decisions before each implementation PR

- **Ownership:** Why does the function belong in that file/package? Cite Gitea and BDSPro source.
- **Data contract:** Which SQL migration/SQLC query and transaction boundaries are authoritative?
- **API contract:** Which HTTP method/path/status/headers/body and authentication conditions are pinned?
- **Error contract:** Which public `code` maps to which persistence/business condition?
- **Failure contract:** What happens on PostgreSQL/Redis/worker failure, cancellation and shutdown?
- **Performance:** How many queries and allocations were there before/after? Do not silently add an extra permission query.
- **Tests:** Name tests that discriminate between the old and new implementations, including PostgreSQL integration tests.

## Acceptance stop rules

Stop promotion to the next PR when:

- Existing protected flows fail or require deleting a prior test to pass.
- The new router is green only with a reduced route inventory or weakened middleware semantics.
- SQLC+GORM write transactions are mixed without an explicit shared transaction proof.
- `go test ./...` passes solely because PostgreSQL tests skip.
- Build/run succeeds locally but Compose startup, mail delivery or graceful shutdown fails.
- The new implementation adds unexplained database round trips or significant hot-path overhead.

## Explicit exclusions

New business domains such as CRM, assets, deals and finance are **not** part of the mechanical R02 runtime migration. Their operations will use the R02 conventions once the reference workflows are accepted. New GORM usage is also excluded until a concrete query demonstrates need and correctness.

