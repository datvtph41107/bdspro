# Services — current operation ownership

Baseline: BDSPro `dev` R28, after the models/OIDC cut-over. Gitea reference:
[services/auth](https://github.com/go-gitea/gitea/tree/main/services/auth),
[services/org](https://github.com/go-gitea/gitea/tree/main/services/org),
[services/issue](https://github.com/go-gitea/gitea/tree/main/services/issue).

| Package | Owns | Does not own |
| --- | --- | --- |
| `services/auth` | registration, login, OAuth2/OIDC client, phone OTP, token renewal and session orchestration | Gin parsing or HTTP status codes |
| `services/org` | organization/team writes, acting-session checks and cross-model transactions | HTTP response formats or SQLC generation |
| `services/convert` | mapping model/service values to `modules/structs` API responses | persistent constraints or transport routing |
| `services/jobs` | email verification, password reset and invitation job operations | Asynq transport and Redis connections |
| `services/worker` | SQL outbox dispatch and Asynq-to-job adapters | generic Asynq connection/processor |
| `services/mail` | SMTP sender integration | account authorization |
| `services/notification` | phone delivery implementations | Gin admission policy |

Organization and Team orchestration now lives in `services/org`, following
Gitea's distinction between `services/org` and `models/organization`.
The service coordinates verified acting sessions, SQLC transactions and
cross-model writes; `models/organization` remains the owner of entities,
constraints and account-scoped persistence invariants.

## Request, transaction and delivery

```text
POST /users
 -> routers/api/v1/user.AuthHandler.Register
 -> services/auth.AuthService.Register
 -> modules/auth.HashPassword
 -> models/account.Register -> models/db.Store.ExecTx
 -> PostgreSQL commit: account + password credential + outbox intent

worker process:
 -> services/worker.OutboxDispatcher
 -> modules/queue.TaskEnqueuer (Asynq, Redis)
 -> services/worker task adapters -> services/jobs -> services/mail
```

`models/db.ExecTx` owns begin/commit/rollback. Domain models own atomic
persistence invariants. A service owns business operations and cross-model or
cross-transport orchestration when that work is real, not when it only forwards
a single SQLC call.

Redis client lifetime is owned by `modules/conn` for go-redis consumers.
Asynq owns separate internal Redis connections; `modules/queue` provides its
worker/producer integration. See [cache/queue ownership](cache-queue.md).

## Naming and evidence standard

Prefer `Register`, `Renew`, `StartBind`, `VerifyBind`, `Dispatch`
over generic `Execute` or a monolithic `resources.go`.
A new service must name callers, transactional boundary, side effects,
failure policy and a test which fails without the new behavior.

Current gaps are explicit: `services/auth.StartEmailLink` sends SMTP
synchronously after storing a proof, unlike other outbox-backed email flows;
the outbox dispatcher holds a PostgreSQL transaction while enqueueing to
Redis. These are **not** fixed by R28's module reorganization.
