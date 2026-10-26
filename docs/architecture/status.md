# Implementation state and unresolved decisions

**Base:** `dev@0cb3d050`, 2026-10-26. This ledger must be updated with actual merge/CI outcomes, not simply a desired directory picture.

| Work area | Current architecture branch | Completion caveat |
|---|---|---|
| `main.go` | Delegates to `bootstrap.Run` | Some startup composition still uses legacy package interfaces |
| `bootstrap/` | Startup, migration, active runtime and legacy transport code separated | Legacy gRPC/gateway not active in the default run |
| `routes/` | Gin auth/API/Google/Phone declarations centralized | Explicit handlers are assembled once; no Laravel-style `init()` magic |
| `app/http/requests/` | Body/Query DTOs extracted from former `api/` types | Some old handlers still in `api/` |
| `app/http/binding/` | Single JSON/Query/Path bind mechanism | Feature-by-feature rollout needed |
| `app/http/controllers/api/` | Organization actions extracted | Auth/Google/OTP actions still need migration |
| `app/http/middleware/` | Authenticate/AccountID moved | IPActive and monitoring remain in `api/` |
| `app/http/responses/` | JSON success/error writer created | Not all handlers converted |
| `app/http/resources/` | Session/user public DTOs | Only current DTOs moved |
| `app/errors/` | Single public error catalog + PostgreSQL classifier | Domain-specific errors still live elsewhere; catalog not yet comprehensive |
| `app/services/` | Organization Create/List operation relocated from root package | `CreateOrganizationTx` business flow remains in Store |
| `jobs/` | Verification-email job action separated from Asynq runtime | Other jobs do not exist yet |
| `worker/` | Owns Asynq scheduling/execution mechanics | Existing retry semantics preserved |
| `db/query/` and `db/sqlc/` | Existing SQLC implementation still in place | Google/Phone/verification now use generated SQLC queries; Store has one pool owner; transaction/workflow layers still require review |
| `tests/` | Existing tests retained for compatibility gates | User requested requalification/removal of legacy tests; do not claim deletion or replacement |

### Explicit not-yet-accepted work

- Complete `db/query` SQLC migrations, generate code, remove duplicate PhoneOTPStore and verify OTP replay/cooldown/lock/atomicity.
- Move service-level Create Organization/Register/Email/OTP transaction workflow carefully, preserving security semantics.
- Finish controller migration for auth/session/Google/Phone; group admin controllers only when admin routes exist.
- Move remaining middleware/ingress out of `api/` without changing order/fail-closed behavior.
- Finish consistent response/error catalog integration and project-level request/response contracts.
- Review filter/pagination only when a real endpoint needs them; do not create a speculative builder.
- Audit and redesign test suite **before** deleting tests that currently prove critical invariants.
- Update CI evidence and merge acceptance only after all gates are green.

### Non-goals

No Laravel Facade emulation; no reflection controller lookup; no hidden DI container; no generalized repository per table; no new ORM; no auto-created admin/realtime endpoints; no forced microservice split; no schema rewrites or automatic removal of migrations.
