# ENG-GOV-01 — Core Ownership Matrix

**Status:** Engineering governance contract, adopted when merged to `dev`. This document describes the responsibility boundaries of the **existing** Engineering Core V1; it does not announce Business Core implementation or WSL acceptance.

**Canonical baseline:** BDSPro Go/Gin + PostgreSQL/sqlc + Redis/Asynq, as implemented on `dev`. Reference ideas: DRX Laravel's action-specific request naming, Java DRWash's shared capabilities, and TechSchool SimpleBank's direct execution/SQL transaction model. **Reference source is not BDSPro's authority.**

## 1. Ownership has four meanings

- **Capability owner:** the code area responsible for a behavior or policy (for example, `auth` owns credential authentication).
- **Data owner:** the mechanism that enforces durable state and invariants (often PostgreSQL constraints/SQL in `db/migration`, `db/query` and transaction code).
- **Execution owner:** the actual entry point executing the operation (route middleware, HTTP handler, service, Store or worker).
- **Change owner:** the developer implementing and a named reviewer evaluating the particular change. A package name is **not** a person, team or GitHub approval rule.

A request may cross all four, but an invariant needs a **primary place of enforcement**. When source and documentation disagree, inspect runtime and migrations, record the discrepancy, then update the intended contract via review; do not silently assume the document is code.

## 2. Current Engineering Core ownership

| Capability | Primary owner / source | Protected responsibility or invariant | Boundary and evidence |
| --- | --- | --- | --- |
| Bootstrap and runtime | `main.go`; `util/config.go` | Validate configuration, connect Postgres/Redis, run migrations, start worker/HTTP and report readiness | No HTTP/business rules. See startup tests, CI smoke and G0 WSL acceptance |
| Configuration and deployment | `app.env`, `util/config.go`, `Dockerfile`, `docker-compose.yaml`, `start.sh`, `wait-for.sh` | Explicit process-env override and known container services/volumes | No secrets committed; inspect env precedence, Compose, persistent volume ownership |
| HTTP routing and ingress | `api/server.go`, `api/request_limits.go` | Register endpoints, CORS, request ID, body cap, HTTP timeouts and direct-peer IP trust | No account ownership/SQL privilege decisions; see HTTP tests |
| Auth admission | `api/middleware_ipactive.go`, `api/middleware_prelogin.go`, `api/phone_otp.go` | `IPActiveSource` counts malformed attempts, `Pre*` validates input, `IPActive` applies action and identity quotas | Rate admission is **not** Authenticate or Authorize; Redis failure fails closed on active auth routes |
| Request observation | `api/middleware_request_monitor.go`, `api/middleware_suflogin.go` | Record safe denial/login outcomes without raw secrets; monitoring must not grant/deny business rights | Monitoring failures do not bypass authentication; see log/detection tests |
| Credential authentication | `auth/auth.go`, `auth/phone_otp.go`, `api/google.go` | Password/Google/OTP evidence before `IssueSession`; all methods resolve to existing account identity | Google routes require complete config, Phone OTP is development-only; email-verification eligibility is **not yet** a settled policy |
| Tokens and session lifecycle | `token/`, `auth.IssueSession`, `api/token.go`, `db/query/session.sql` | Access and refresh token type/expiry; durable refresh session and server-side renew/revoke checks | Current revoke blocks refresh session; issued access JWT survives until expiry |
| Identity persistence | `db/migration/`, `db/query/account.sql`, `db/sqlc/store.go` | `accounts.id` is the common account identity; Party/Person/Account registration commits in one PostgreSQL transaction | No duplicate account system per authentication provider; verify unique/foreign-key constraints in DB |
| Organization membership/authorization | `api/organization.go`, `db/query/organization.sql`, `db/sqlc` transaction helpers | Actor from verified JWT; active membership/owner predicates at SQL mutation time | `Authenticate` does not imply organization ownership; SQL predicates and constraints are enforcement |
| Email verification processing | `worker/`, `mail/`, `db/sqlc` verify-email transactions | Task lifecycle, retry, active verification record, single-use verification | Redis enqueue in `RegisterTx.AfterCreate` occurs **before** PG COMMIT; no atomic PG–Redis handoff is claimed |
| Database schema and code generation | `db/migration/*.sql`, `db/query/*.sql`, `sqlc.yaml`; generated `db/sqlc` | Migration history and SQL queries are the editable definitions; constraints own persistence invariants | Do not hand-edit generated sqlc code; existing handwritten Store/transactions in `db/sqlc` remain valid |
| CI and operational evidence | `.github/workflows/test.yml`, `scripts/`, `docs/operations/` | Produce evidence for build, migration, tests, HTTP smoke and runtime acceptance | A passing GitHub CI run is not a WSL, backup/restore or production certificate |

**Business Core:** Property, Listing, Inquiry and Transaction are **not implemented/accepted** by this matrix. Existing Organization access is an Application/Identity foundation, not proof that BDSPro's business domain has been modeled.

## 3. Request boundary — do not confuse admission, identity and rights

Actual password route in `api/server.go`:

```text
RequestIngress (global)
  -> RequestMonitor (global observation, reports on unwind)
  -> SufLogin (outcome observation, runs after handler too)
  -> IPActiveSource -> PreLogin -> IPActive
  -> loginUser -> auth.Login -> IssueSession -> SQL session
```

The global Gin registration order places `RequestMonitor` **before** `RequestIngress`; observation happens after downstream execution. The diagram above groups global responsibilities conceptually rather than asserting `RequestIngress` executes first. For exact call order, inspect `api/server.go` and [request-lifecycle.md](request-lifecycle.md).

Protected organization routes use `Authenticate` to obtain account identity and SQL membership/owner conditions to decide authority. Routes for Google/Phone OTP have different middleware orders and conditional registration; **there is no universal `PreLogin`/IPActive chain for every route.**

## 4. Ownership review rules

1. State the **actor, action, resource and durable state** before choosing a package.
2. Keep transport concerns in `api`: bind/validate HTTP input, map errors, handle response and request admission; handlers may call Store directly for a simple use case **as existing code does**.
3. Put authentication and session issuance in the existing `auth`/`token` contract; do not implement a second token-issuing path in a middleware/controller.
4. Keep data atomicity, constraints and authorization predicates at the database boundary when consistency with a mutation is required.
5. Place workers/delivery integrations behind their existing responsibility; do not equate successful Redis enqueue with successful email delivery.
6. An interface, new package, separate Go module or generic service layer requires a demonstrated second implementation, testing need or genuine responsibility boundary; it is not mandatory for CRUD.
7. Changes that cross capabilities must identify **one coordinating operation**, every affected owner and the tests that protect each boundary; do not duplicate the same rule in several layers.

## 5. Open decisions are not hidden guarantees

- **G0:** WSL Compose, backup/restore and persistent runtime evidence remain to be confirmed; see [G0 acceptance](../operations/g0-runtime-acceptance.md).
- **G1:** confirm route-by-route request admission limits, reverse-proxy trust and denial behavior.
- **G2:** decide PostgreSQL–Redis delivery guarantees after failure-injection evidence; see [Register/Asynq reliability](../operations/register-asynq.md).
- **G3:** decide policy for unverified-email accounts and session invalidation; real Google browser/SMS provider E2E is outside current CI.

Change process and review requirements: [development-contract.md](development-contract.md). Decision and rejected alternatives: [ADR 0001](decisions/0001-core-ownership.md).
