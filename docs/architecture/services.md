# Services — business operations and work execution

Gitea has `services/auth`, `services/repository`,
`services/issues`, `services/actions`, `services/org` and others
because their application owns real domain workflows. A service module
coordinates business policy, persistence and effects; it does not simply
forward a request to SQLC.

## Existing BDSPro ownership

| Package | Owns | Does not own |
| --- | --- | --- |
| `services/auth` | account registration, passwords, sessions, Google/phone authentication | Gin parsing or HTTP status codes |
| `services/organization` | organization registration and membership rules | router setup |
| `services/jobs` | email verification/reset operations invoked after durable intents | task retry queues or Redis |
| `services/worker` | SQL outbox dispatch policy and Asynq-to-business adapters | generic Asynq connection/processor |
| `services/mail` | SMTP sender integration | account authorization policy |
| `services/notification` | SMS/Zalo/local phone code delivery | HTTP route admission |

```text
POST /users (routers/api)
  -> auth.AuthService.Register(ctx, email, password)
       -> authcrypto.HashPassword
       -> models/db.RegisterTx
            -> accounts + password_credentials + job_outbox
  commit
  -> worker.OutboxDispatcher
       -> queue.TaskEnqueuer (Asynq)
       -> worker.NewVerifyEmailHandler
            -> jobs.SendVerificationEmail -> mail.EmailSender
```

**Transaction owner:** `models/db.ExecTx` starts and commits the transaction.
The service decides when an atomic operation is needed. An Asynq call inside
a PostgreSQL transaction has independent failure semantics: Redis enqueue
and PostgreSQL commit cannot be one atomic commit. Worker replicas, retries,
task ID conflicts and SIGTERM need separate proof.

## Naming standard

Prefer verbs that name operations (`Register`, `Renew`,
`StartBind`, `VerifyBind`, `Dispatch`) and packages named by domain or
technical capability (`auth`, `organization`, `jobs`).
Avoid `common_service`, mega-`resources.go` or
`BusinessProcessorFactory` without demonstrated multiplicity.
A service's `New...` constructor must connect real collaborators and
invariants, not add another delegation layer.

When account and organization logic share a DB transaction, do not create
an imaginary cross-domain repository interface before tracing the write.
Use the existing SQLC transaction and constrain the operation at the
service owner. GORM is not added merely to change model syntax.

Gitea evidence:
[services/auth](https://github.com/go-gitea/gitea/tree/main/services/auth),
[services/issue](https://github.com/go-gitea/gitea/tree/main/services/issue),
[services/repository](https://github.com/go-gitea/gitea/tree/main/services/repository).
