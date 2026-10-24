# ENG-GOV-01 — Development and Change Contract

**Applies to:** all new changes targeting BDSPro `dev`. The aim is understandable ownership, minimal implementation and reproducible evidence, **not** more interfaces, approval bureaucracy or architecture layers.

Read [Core Ownership Matrix](core-ownership.md) before a cross-module change. For ingress and middleware naming, use [Request Lifecycle](request-lifecycle.md). For runtime risk, use the [G0 runbook](../operations/g0-runtime-acceptance.md).

## 1. Workflow every developer can follow

```text
Problem -> Intent -> Responsibility/Owner -> Invariant
        -> Small implementation -> Evidence -> Review -> Acceptance
```

1. **Observe.** Read the relevant live route, call chain, SQL definition, migration and test. Record current behavior; do not infer it from file names or older reference projects.
2. **Choose the owner.** Identify capability owner, durable data owner, entrypoint and change reviewer. Confirm whether the change affects other authentication methods, routes or actors.
3. **State the invariant.** Define what must be true, including invalid input, revoked access, concurrent requests and failure paths if relevant.
4. **Change the smallest scope.** Reuse existing Gin/Go and sqlc boundaries; avoid new abstractions without a specific reason.
5. **Prove behavior.** Test valid and denied cases, DB constraints/transactions, observability, retries/concurrency as applicable. Distinguish mock tests from disposable integration and real WSL runtime.
6. **Review a focused PR.** Use the [PR template](../../.github/PULL_REQUEST_TEMPLATE.md) to state impact and evidence, including intentionally untested areas.
7. **Accept precisely.** CI success supports source acceptance. WSL runtime, external provider E2E and data recovery require **separate** evidence where relevant. Record merged SHA and outstanding risks.

## 2. Coding responsibilities (current code, not a forced layered architecture)

| Boundary | It owns | It must not pretend to own |
| --- | --- | --- |
| Middleware in `api/` | Request admission, shape validation, JWT verification, safe observation | Organization owner rights, DB transaction, issuing tokens after response |
| HTTP handler in `api/` | Route-specific input, actor extraction, status/response, orchestration of a simple operation | Durable invariants that SQL/transaction must enforce |
| `auth/` | Credentials/provider evidence and shared session issuance | Business-resource ownership or request transport |
| `token/` | JWT payload, signing and validation | Refresh session database status, SQL resource permission |
| `db/query/*.sql` | Explicit persisted queries and SQL authorization predicates | Transport HTTP codes |
| `db/migration/*.sql` | Versioned tables, FKs, constraints, indexes | Runtime response mapping |
| Handwritten `db/sqlc/*.go` | Store transactions and cross-query atomic operations | Hand changes to generated sqlc artifacts |
| `worker/` and `mail/` | Async task execution and delivery adapters | Exactly-once delivery claims or unrelated synchronous authorization |
| `main.go` / `util/config.go` | Bootstrap, env, startup/shutdown | Business flows |

**Service is conditional, not compulsory:** `auth.Login` is a real shared application behavior. Organization handlers currently call Store/query methods directly; inserting an additional service layer solely for uniformity is not accepted as a design justification.

**DRX naming with true semantics:** `PreLogin` only validates request shape; `SufLogin` only reports an outcome. `IPActive` gates quota, `Authenticate` validates identity, SQL decides `Authorize`. Do not rename a method to an aspirational concept that its code does not implement.

## 3. Source and database changes

- Work from the intended `dev` SHA on a focused feature branch. Inspect local `git status` and committed/uncommitted changes **before** syncing. Never force-push `dev` or reset local changes to make a test pass.
- Keep migrations forward-only. Do not rewrite an applied migration; create a numbered new migration with a compatible rollback where possible. Check existing data, nullability, unique/FK constraints, concurrent writes and migration ownership.
- Edit `db/query/*.sql` then regenerate using `make sqlc`; generated files must match inputs. Do not manually change generated query code. Handwritten transaction files are explicitly owned code.
- State whether a change is **data-preserving**, **backward-compatible** and **safe for existing volumes**. Docker PostgreSQL 14 data cannot be mounted directly into 18; `docker compose down -v` is not part of routine work.
- Configuration must make file defaults/process-env precedence clear. No live secrets, OTP, verification links or bearer tokens in examples, PR descriptions, logs or commits.
- New public endpoints must be registered intentionally; document HTTP method/path, middleware chain, auth requirements, error mapping and negative test.
- No new AWS, Kubernetes, microservice, outbox, custom framework or interface without a documented need and evidence. This is not a permanent ban; it is a proportional-design rule.

## 4. Evidence proportional to change

| Change type | Minimum useful evidence |
| --- | --- |
| Documentation-only/PR template | Paths and links resolve; read existing source; diff inspection for incorrect guarantees; CI run result |
| Middleware/security | Unit tests for ordering/denial/non-leakage; Redis integration for concurrent quota when changed |
| Token/auth/session | Valid/invalid/expired/revoked paths; real session DB behavior; check effects on password, Google and Phone OTP |
| Database/schema/authorization | Fresh disposable DB migration, positive/negative SQL access, concurrent or integrity checks if state can race |
| Asynq/email | Disposable Redis/Postgres + mocked or Mailpit SMTP, retry/failure behavior; do not claim atomic handoff without proof |
| Docker/startup/operations | Compose validation, runtime smoke and WSL acceptance as applicable; preserve existing persistent data |
| Future business change | Explicit actor/action/resource, state transitions, data owner and invariants; a thin end-to-end test for the selected use case |

Useful existing checks (run only where prerequisites exist):

```sh
go test ./...
go build ./...
go vet ./...
docker compose config --quiet
```

CI runs migrations and integration tests against disposable service containers. Some local DB and Mailpit E2E tests are deliberately skipped without CI configuration. The isolated local DB test script `sh scripts/test-integration.sh` starts a **separate disposable Postgres on 127.0.0.1:15433** and must not be confused with the persisted application DB. `sh scripts/accept-g0.sh` inspects an **already running** Compose stack and does not start/recreate it.

## 5. Review decision

A reviewer checks these questions, not the number of directories added:

- Was the actual source read, and does the selected owner match the behavior?
- Does the change duplicate or weaken an existing invariant, especially across password/Google/OTP and organization membership?
- Are failures, malformed requests, unauthorized actors and concurrency covered as needed?
- Can CI evidence be reproduced? Is any claimed WSL/provider/production verification genuinely observed?
- Did an unrelated feature, migration, environment assumption or generated-code change slip into scope?

**No `CODEOWNERS` or branch-protection requirements are installed by this document.** PR authors must identify who reviewed important cross-capability changes; the actual reviewer is assigned through the team's GitHub process when one exists. Never fabricate a reviewer or claim an unconfigured gate is enforced automatically.

## 6. Real example: policy for unverified email

Current `auth.Login` checks password and creates a session without testing `IsEmailVerified`. This is **observed behavior, not a newly approved policy**.

If a product requirement proposes blocking unverified accounts:

1. Record the intended allowed/forbidden actions: login, link provider, renew tokens, create organization.
2. Decide if the restriction is credential-dependent or account-wide; inspect `auth.Login`, `auth.IssueSession`, Google OIDC and Phone OTP.
3. Pick **one primary enforcement owner** and keep SQL/resource checks coherent.
4. Test unverified/verified users on every relevant path; ensure a new provider cannot bypass the decision.
5. Merge only after policy acceptance. Do **not** add a quick `PreLogin` check as a substitute for a cross-provider decision.

Architecture-impacting choices belong in a compact decision record using the [ADR format](decisions/0001-core-ownership.md). Small refactors and typo fixes do not require a new ADR.
