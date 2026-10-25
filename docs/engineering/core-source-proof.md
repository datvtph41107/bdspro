# ENG-CORE — Source Proof & Coverage Matrix

**Canonical audited baseline:** `datvtph41107/bdspro` branch `dev` commit **`6f47f4971fba49ba8ce9479adbfb9739c244f620`** on 25/10/2026. Git commit's **tree object**: `a6c20e726b1e4fb1617393269fc1093faadd389a`. Git recursive tree was **not truncated**. This document is an **audit of that immutable baseline**, not a declaration that every file is conformant, secure, or production-deployed.

**Evidence source:** [full 180-file blob inventory](source-baseline-6f47f49.tree) and [read-only proof script](../../scripts/check-core-source-baseline.sh). The manifest records each Git mode, **blob SHA**, and exact path; it includes source, generated code, tests, docs, configs, scripts, and historical SQL. Reproduce with `sh scripts/check-core-source-baseline.sh` on a **full-history Git checkout**. This checks Git object identity; it does not execute features or certify source behavior.

## 1. All files exist, but that is only the first level of proof

| Evidence level | What can be asserted | Current source of truth |
| --- | --- | --- |
| **E0 — Provenance** | Exact file/path/blob at one immutable source commit | Git commit + tree + 180-line manifest |
| **E1 — Structural** | Package/function/interface/query exists; dependency owner can be inspected | Handwritten Go, generated SQLC, migrations, proto; specific source refs below |
| **E2 — Verified behavior** | A named test actually executes and passes for a bounded invariant | 52 tracked `_test.go` files; CI logs (not just test names) |
| **E3 — Runtime smoke** | Disposable CI boots an HTTP server with PostgreSQL/Redis/Mailpit and checks narrow HTTP behaviors | `.github/workflows/test.yml`, push [CI #119](https://github.com/datvtph41107/bdspro/actions/runs/40196527668), **SUCCESS** |
| **E4 — Operational acceptance** | Running WSL stack, real external provider, data volume/recovery/deploy meets explicit runbook criteria | `scripts/accept-g0.sh` exists; **not established by this source audit** |

**Inventory from Git tree:** 208 Git tree entries = **28 tree/directory entries + 180 blobs/files**. Among blobs: **116 Go** (52 `_test.go`, 64 other Go), **15 SQL** (10 immutable migration up/down files and 5 sqlc query files), **3 proto**, **27 files under `docs/`**. Generated Go and mock files are included; no statement of manual review of every generated line is implied.

**CI evidence nuance:** push CI #119 on exact baseline passed migrations, Go tests, build + Docker Compose validation, HTTP smoke, go vet, and race detector. “Protect migration history” is skipped on the *push event* by design; PR #59's CI #118 exercised that PR-only check and passed. A green `go test ./...` may include skipped integration tests when their environment is unavailable; specific guarded tests require checking logs for `RUN/PASS`.

## 2. Coverage of every source area (all 180 tracked files)

The mapping below partitions **every tracked path by top-level folder/root**. Counts come from the pinned Git tree, not hand-estimated LOC.

| Root / tracked files | Engineering ownership and actual source evidence | Maturity / important limit |
| --- | --- | --- |
| **Repository root — 15** | `main.go` composition, startup/migrate/shutdown; `app.env`, `go.mod/go.sum`, `sqlc.yaml`, `Makefile`, `Dockerfile`, `docker-compose.yaml`, `start.sh`, `wait-for.sh`, root tests, .gitignore/.dockerignore/README | **E1–E3** for CI path. No guarantee of local WSL parity; root `.env`/.env.example are **not tracked** in the pinned tree |
| **api/ — 40** | `server.go` route + request lifecycle; `request_limits.go` ingress; `middleware_*.go` admission/IPActive/RequestMonitor/SufLogin/Authenticate; `request_binding.go`, `actor.go`, `request_params.go`, `postgres_error.go`; handlers for user/token/verify_email/organization/Google/Phone; HTTP tests | **E1–E3** for routes under CI, not exhaustive provider E2E. Baseline drift: `verify_email.go` passed Gin context to Store (remediated in this audit PR, separately tested) |
| **auth/ — 11** | `auth.go`: Register/Login/IssueSession; `verification_scheduler.go`: consumer-owned scheduling port; `google_token.go`: OIDC verification; `phone_otp.go`: dev-only OTP logic; unit/contract tests | **E1–E2**; Google external OIDC and SMS delivery not proven; OTP consumption before session persistence is existing failure contract |
| **organization/ — 2** | `operations.go`: CreateForAccount/ListForAccount resolve Account→Person outside Gin, call Store; `operations_test.go` | **E1–E2**; explicit application module for multi-step operations, not mandatory for every CRUD |
| **db/ — 39** | `db/migration/000001..000005` up/down; `db/query/{account,party,organization,session,verify_email}.sql`; handwritten `db/sqlc/store.go` + transaction/Google/Phone; sqlc generated `*.sql.go`, `models.go`, `querier.go`, mock store, PG tests | **E1–E2/E3** in CI PG. Constraints and membership owner predicates are DB-owned; no Property/Listing schema |
| **token/ — 4** | JWT maker/payload/interface and tests; access/refresh separation | **E1–E2**. Access JWT remains valid after refresh-session revocation until expiry |
| **worker/ — 10** | `distributor.go`, `processor.go`, `task_send_verify_email.go`, `verification_scheduler.go`; mock, unit/integration tests | **E1–E2/E3** bounded Redis/Asynq/Mailpit; enqueue before PostgreSQL COMMIT is **not atomic** |
| **mail/ — 1** | `sender.go` SMTP adapter | **E1**, exercised by CI integration; external production SMTP not proven |
| **util/ — 8** | Config via Viper + process overrides, logger, password and random helpers with tests | **E1–E2**. `app.env` mandatory for current `LoadConfig`; no claim of `.env.example` tracked |
| **gapi/ — 9** | gRPC server/handlers and tests for Register/Login/VerifyEmail/UpdateOrganization, logger | **E1–E2**; these compile but `main.run()` starts **Gin HTTP only**. RPC runtime not part of active boot path |
| **pb/ — 3** | Generated protobuf/grpc/gateway Go code | **E1**, build proof only; no active gRPC/Gateway listener |
| **proto/ — 3** | Proto BDSPro and two Google API proto references | **E1** source; regenerability needs protoc/plugin/toolchain check separately |
| **doc/ — 3** | DBML, static swagger JSON, statik generated file | **E0–E1**; cannot claim OpenAPI up-to-date solely from file existence |
| **scripts/ — 3** | `accept-g0.sh` read-only G0; `test-integration.sh` disposable PostgreSQL; `check-migration-history.sh` immutable migration check | **E1–E3** script syntax/CI; G0 on actual user environment unverified |
| **.github/ — 2** | PR template + `workflows/test.yml` with PG18, Redis7, Mailpit, migrations, tests, smoke, vet/race | **E3** exact CI run; no claim branch protection/CODEOWNERS automatically configured |
| **docs/ — 27** | Technical lectures, DRX/SimpleBank mappings, ENG-GOV/ARC/BIZ/ENG-CORE contracts, ADR, operational runbooks | **E0–E1**: documentation and decisions, *not* executable behavior |

**File count check:** 15 + 40 + 11 + 2 + 39 + 4 + 10 + 1 + 8 + 9 + 3 + 3 + 3 + 3 + 2 + 27 = **180**. The pinned manifest is the exhaustive per-file ledger; this table is the exhaustive **owner/category coverage**, not a manual audit assertion about 180 file bodies.

## 3. ENG-CORE rules actually represented in production source

| Rule / maturity | Canonical owner and source | Runtime path/contract | Verification |
| --- | --- | --- | --- |
| **ENG-CORE-01/Bootstrap** (source + CI) | `main.go`, `util/config.go`, `app.env` | Load config → connect PG → migrations → connect Redis → worker + Gin HTTP → shutdown | `main_test.go`, config tests; CI #119 |
| **ENG-CORE-HTTP Ingress** (source + CI) | `api/server.go`, `middleware_*.go`, `request_limits.go` | net/http → Gin → RequestMonitor/RequestIngress → per-route Source IPActive/Pre*/identity IPActive/Authenticate → handler | `api/middleware_pipeline_test.go`, `api/admission_evidence_test.go`, `api/http_runtime_test.go`, `api/request_limits_test.go` |
| **ENG-CORE-04/Reusable HTTP** (source + test) | `api/request_binding.go`, `actor.go`, `user.go` | typed JSON, JWT actor, shared login response | `api/application_foundation_test.go` |
| **ENG-CORE-05/Errors & Params** (source + test) | `api/request_params.go`, `postgres_error.go`, Organization/Register handlers | positive resource ID + driver error introspection, **per-operation** HTTP mapping | `api/api_error_contract_test.go` |
| **ENG-CORE-06/Application Operations** (source + PG test) | `organization/operations.go`; `db/sqlc/tx_create_organization.go` | HTTP Create/List → Account→Person → Store; AddMember/Update still direct owner-scoped SQL | `organization/operations_test.go`, `api/organization_business_integration_test.go` |
| **ENG-CORE-07/Dependency Direction** (source + AST guard) | `auth/verification_scheduler.go`, `worker/verification_scheduler.go`, `api/server.go`, `gapi/server.go` | HTTP and dormant RPC call `auth.Register`; auth expresses verification intent; worker chooses Asynq opts | `auth/dependency_boundary_test.go`, scheduler tests, PG/Redis handoff |
| **ENG-CORE-08/Session Integrity** (source + CI PG) | `auth/auth.go`, `token/`, `api/token.go`, `db/query/session.sql` | Password/Google/Phone → `IssueSession` → PG session; Renew/Revoke uses refresh session | `auth/session_contract_test.go`, `api/session_lifecycle_integration_test.go`, `api/session_revoke_contract_test.go` |
| **ENG-CORE-09/Feature Delivery** (document + executable regression) | `docs/engineering/feature-delivery-standard.md`, `api/feature_delivery_contract_test.go` | business contract → route/input/actor → operation or SQL → response → evidence | Four named tests RUN/PASS on CI #118/#119 |
| **BIZ-01/BIZ-02/Organization Authority** (source + CI PG) | `db/migration/000003_*.sql`, `db/query/organization.sql` and `db/sqlc` | active `role='owner'` grants Organization mutation; partial unique open memberships | `api/organization_business_integration_test.go`, `db/sqlc/organization_ownership_integrity_test.go` |
| **Operational Contracts** (source + CI smoke) | `Dockerfile`, Compose, `scripts/*.sh`, `.github/workflows/test.yml` | local PG18/Redis7/Mailpit/Gin; disposable CI checks build, migrations, HTTP smoke | push CI #119; G0 local runbook has not been executed in this audit |

**Example one end-to-end business chain:** `POST /organizations` → `Authenticate` creates verified Account actor → `api.createOrganization` normalizes input → `organization.CreateForAccount` gets PersonID → `db.SQLStore.CreateOrganizationTx` persists Party/Organization/active owner membership in one PG transaction. Authorization for `PATCH /organizations/:id` and `POST /organizations/:id/members` is a resource-specific SQL predicate, not a JWT role.

**Example async chain:** `POST /users` → PreRegister/IPActive → `auth.Register` → `db.RegisterTx` with `AfterCreate` → `auth.VerificationTaskScheduler` → `worker.VerificationScheduler` → Redis/Asynq → worker handler → `GetOrCreateVerifyEmailTx` → SMTP. It has two independent durable systems: PostgreSQL and Redis. No atomic handoff or exactly-once guarantee exists.

## 4. Gaps and non-claims (no false equivalence)

| Finding | Evidence in pinned source | Assessment / action |
| --- | --- | --- |
| **HTTP request context drift** | `api/verify_email.go` passed `*gin.Context` into `VerifyEmailTx` | **Confirmed code issue; fixed in this audit PR** to `ctx.Request.Context()` with a context-propagation regression test |
| **Source files ≠ full behavioral proof** | 180 blobs, 52 Go test files, CI #119 green | E0/E1 exhaustive **inventory**; tests E2 scoped, CI E3 narrow. No 100% coverage or comprehensive security proof claimed |
| **gRPC/Gateway dormant** | `main.run()` starts `runGinServer` and worker, not `runGrpcServer/runGatewayServer` | Source and compilation yes; active runtime no. Future transport activation requires auth/security/runtime acceptance |
| **PostgreSQL–Redis atomicity gap** | `db.RegisterTx.AfterCreate` enqueues before PG COMMIT | Proven partial-failure risk; outbox/alternate delivery contract requires separate decision |
| **Phone OTP login failure** | `auth.PhoneOTP.VerifyLogin` consumes challenge before `IssueSession` | Current replay-safe behavior can require a new OTP after persistence failure; no policy change approved |
| **Email verified login gate** | `auth.Login` checks password and issues session without testing `IsEmailVerified` | Existing behavior; do not silently call it a security bug or impose an unapproved policy |
| **Property/Listing legal ownership** | No such migrations/commands in tracked `db/`, `api/`, `organization/` | Not implemented; Organization role='owner' is management authority, not proof of legal title |
| **Operational/third-party acceptance** | `scripts/accept-g0.sh`, Google and OTP docs exist | No local WSL G0 result or external Google/SMS provider E2E in this audit |
| **Generated docs parity** | `proto/`, `pb/`, `doc/swagger/` tracked | Build proves Go compiles, **not** generator version parity or freshness of swagger artifacts |
| **Tracked environment template** | `app.env` tracked; `.env.example` absent from pinned Git tree | Do not claim separate .env.example is present without future source evidence |

## 5. How to reproduce the source proof

With a full Git checkout containing baseline `6f47f49`:

```sh
git fetch origin --prune
git switch dev
git pull --ff-only origin dev

# Exact 180-file snapshot vs Git object's content-addressed file identities
sh scripts/check-core-source-baseline.sh

# Current source build/tests independently of the immutable snapshot
go test ./...
go build ./...
go vet ./...
docker compose config --quiet

# CI evidence, not automatic WSL acceptance:
# https://github.com/datvtph41107/bdspro/actions/runs/40196527668
```

**Caution:** `go test ./...` may skip protected integration tests without their disposable PostgreSQL/Redis/Mailpit env. Follow `scripts/test-integration.sh` for isolated PG, or use CI disposable dependencies for full protected scenarios. Do not point test env at the real `bdspro` Compose volume. G0 is a separate read-only acceptance route requiring running Compose services.

### What this audit proves and does not prove

1. **Proves exhaustive tracked-source identity at baseline** (every blob/path/hash, verifiable Git objects).
2. **Proves source-backed ownership tracing for each repository area** and representative critical execution chains.
3. **Proves CI on baseline is green** and names scoped tests; further changes require a new passing CI run.
4. **Does not prove** every line exercised, 100% security/compliance, local runtime, third-party provider E2E, data recovery, or a future domain not yet implemented.

**Governance acceptance:** Update the coverage matrix when changing a capability owner or material invariant; record gaps candidly; do not require rewriting the pinned snapshot after every normal PR. To snapshot a later milestone, use that milestone's own Git commit/tree and create a **new immutable** ledger rather than modifying this historical baseline.
