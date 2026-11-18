# Gitea-aligned BDSPro Core: integration acceptance ledger

Baseline: `refactor/remove-asynq-outbox` at `bb0bc5a37e1dd42dd55bba1e76fc12b2e3e54b79`.
This document is an honest implementation inventory, **not** a declaration that every requested feature is implemented.

## Principles
1. Gitea-style handlers express HTTP orchestration, services express business decisions, and models/sqlc own PostgreSQL invariants.
2. Resource ownership: db owns pgx, conn owns Redis, PubSub owns subscriptions, mailer owns SMTP, phone transport owns provider clients, HTTP layer owns connection sessions.
3. Mutation and required audit records share a transaction. Transient notify runs after commit. Durable messages require persisted work before request success.
4. Request auth uses a verified Actor; account and organization IDs from untrusted headers/query params never establish authority.
5. No generic EventBus, DI container, or worker platform without a concrete business requirement.

## Implemented on this branch
- `services/websocket/stream.go`: preserve the first buffered event instead of silently consuming it in pre-snapshot readiness inspection.
- `services/websocket/stream_regression_test.go`: deterministic regression for that loss.
- `models/audit/retention.go`: SQLC-backed bounded `DeleteOldAuditEvents` operation.
- `services/audit/retention.go`: one bounded sweep (20 x 1,000 max), hourly registration with the existing cron subsystem.
- `cmd/web.go`: starts the audit retention scheduler after PostgreSQL and HTTP handler initialization; `graceful.RunWithCancel` owns drain before DB engine termination.
- `services/audit/retention_test.go`: partial batch, hard limit, database failure, invalid configuration.
- `services/auth/password_proof.go`: credential verification using existing SQLC account/hash queries, generic failure for unknown/wrong/passwordless credentials and dummy bcrypt comparison on absent records. It intentionally does **not** issue tokens.
- `services/auth/password_proof_test.go`: fake SQLC reader tests for valid/wrong/absent credentials and database failure.
- `routers/api/v1/auth/refresh.go`, `modules/structs/auth_refresh.go`: POST `/api/v1/users/refresh` exchanges an opaque refresh secret plus session UUID, guarded by `ipactive.RenewAccess` and `Cache-Control: no-store`. Transactional rotation additionally takes the account-level lock shared with revocation workflows; PostgreSQL concurrent replay acceptance remains pending.
- `services/auth/password_login.go`, `routers/api/v1/auth/password.go`, `modules/structs/auth.go`: rate-limited POST `/api/v1/users/login` using VerifyPasswordProof and IssueSession. The route rejects requests when Redis-backed admission is unavailable and sends `Cache-Control: no-store` on success.
- `models/audit/session_issue.go`: required `auth.session.issued` record in the transaction creating session and login history. Audit insert failure prevents commit and token return.
- `routers/api/v1/api_login_test.go`: validates public login fails closed when admission is missing.
- `services/auth/session_actor.go` and tests: live DB session authority for server-verified session references; revoked, expired, unknown or mismatched sessions are rejected.
- `services/auth/access.go`: access JWT verification now reuses the same session authority and checks account-ID agreement.
- `.github/workflows/test.yml`: SQLC generator pinned to v1.31.1 matching committed generated output.
- `services/auth/session_issue.go`: trusted successful-credential callers issue access JWT plus opaque refresh credentials, persisting the refresh digest and login history atomically.
- `services/auth/session_refresh.go`: single-use refresh credential rotates under PostgreSQL transaction; old refresh session is blocked, replacement is created before commit. Refresh HTTP endpoint uses the same rotation service.
- `services/auth/session_revoke.go` and `models/audit/session.go`: transaction-bound owned-session revocation with required typed audit record.
- `routers/api/v1/auth/session.go`: authenticated bearer self-logout handler, routed at `POST /api/v1/users/me/logout`.
- `routers/api/v1/auth/sessions.go`: authenticated `GET /api/v1/users/me/sessions` scoped exclusively by verified `ctx.Doer.AccountID`.
- `scripts/demo-auth-api.sh` (wired into `.github/workflows/test.yml`): disposable PostgreSQL/Redis HTTP E2E for password login (valid/invalid), refresh token digest, issuance audit, own-session listing, single-use refresh/replay rejection, old access revocation, self-logout and required revoke audit. Infrastructure CI PASS at `52b0901f25d43188eb333b22f122f94944ef0e09`.
- Unit tests cover invalid session/revoke/refresh proofs, opaque refresh randomness, audit transaction requirement, login cancellation and cron registration. Security concurrency and provider-failure tests remain mandatory.

## Missing — blockers to marking the broad requested scope as complete

| Area | Required behavior before acceptance | Status |
| --- | --- | --- |
| Email/password registration | account + credential + verification job commit atomically; idempotent retries; duplicate races | NOT IMPLEMENTED |
| Password login | verified proof -> session record/history/required audit -> access/refresh tokens | HTTP + DISPOSABLE POSTGRES/REDIS E2E PASS; BRUTE-FORCE AND FAILURE ACCEPTANCE PENDING |
| Refresh/logout | atomic refresh rotation, revoke/session concurrency and authenticated endpoints | DISPOSABLE HTTP/DB SUCCESS + REPLAY/LOGOUT TESTS PASS; CONCURRENT TX FAULT TESTS PENDING |
| Phone signup/login | E.164, persisted OTP challenge MAC, limits, single-use, anti-enumeration, provider send | SQLC ONLY |
| Google OIDC login | state/PKCE/nonce, issuer/audience/signature, callback, issuer+sub link, sessions | CONFIG/SQLC ONLY |
| Authenticated user context | endpoint coverage, session authority and error contract | BEARER + OWN SESSIONS CHECKED VIA E2E; BROWSER COOKIE / WEBSOCKET PENDING |
| Organization/Team authorization | exhaustive permissions and cross-tenant isolation tests | EXISTING ROUTES, NOT FULLY ACCEPTED |
| Audit recording | typed records and required same-transaction inserts on critical mutations | SESSION ISSUANCE/REVOCATION TRANSACTIONAL; OTHER DOMAIN MUTATIONS PENDING |
| Mailer | SMTP provider, template, durable verification/reset delivery, restart/retry | CONFIG ONLY |
| Phone delivery | vetted SMS/Zalo adapters, timeout and provider failure tests | CONFIG ONLY |
| Notification inbox | account-scoped persistence and mark-read | EXISTS, NOT E2E |
| Pub/Sub and realtime | fanout, cancellation; authenticated browser upgrade, snapshot/ordering, reconnect | PARTIAL |
| Cron | scheduler lifecycle and bounded job contracts | AUDIT RETENTION WIRED TO WEB GRACEFUL; INJECTION/SHUTDOWN TEST PENDING |
| Durable worker | persisted jobs with leasing, retry, idempotency, dead-letter policy | NOT IMPLEMENTED |
| Operations | disposable Postgres/Redis, SMTP test server, OIDC issuer, SMS fakes, SIGTERM stress, metrics | NOT ACCEPTED |

## Next implementation order
A. Auth/session authority and account/organization/permission acceptance.
B. Audit typed recording at transactional mutation boundaries.
C. Durable delivery policy, Mailer & Phone OTP services and provider fakes.
D. Google OIDC login and explicit identity linking.
E. Cron wiring, retention, durable workers, notification pipeline.
F. WebSocket authenticated endpoint and ordering/snapshot recovery.
G. Integration/security/failure tests and runbooks.

## Test commands (pending execution by compatible Go toolchain and disposable services)

```sh
gofmt -w services/auth/password_proof.go services/auth/password_proof_test.go services/audit/retention.go services/audit/retention_test.go models/audit/retention.go services/websocket/stream.go services/websocket/stream_regression_test.go
go test ./services/auth ./services/audit ./services/websocket ./services/notify ./services/pubsub
go test -race ./...
go vet ./...
go build ./...
```

A passing unit test does not establish production readiness; integration, security and failure tests remain required.
