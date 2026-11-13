---
name: bdspro-backend-change
description: Use when implementing or reviewing a scoped Go backend HTTP/router, service, model, concurrency, background job or runtime behavior change. Not for schema migrations alone or literature-only research.
---

# Scoped backend change

## Prepare
1. Read `AGENTS.md`, `docs/engineering/AI_AGENT_SYSTEM.md`, current source at an exact commit and current `docs/architecture/parallel-work/` rules. Confirm what actually exists: as of 2026-11-13 `dev` contains infrastructure only; legacy Auth/OAuth/Organization HTTP endpoints were removed.
2. Record **business behavior / invariant / failure mode** and a minimal affected call graph. Read the immediate callers, storage operations, middleware chain, auth guards (if present), tests and config.
3. Establish owner lane and exact write paths. A historical PR scope is not an active lease. Cross-owner or schema changes require coordinator review and a separate scoped plan.
4. Compare with Gitea, Discourse or other source **for mechanics and tradeoffs**. Keep Go 1.27.3, Gin, SQLC/pgx, PostgreSQL 18, Redis/Asynq and REST unless the owner deliberately changes stack policy.

## Implement
- Keep one HTTP matcher/handler lifecycle. Validate on the server, respect HTTP contract, status commitment and error propagation. Never duplicate a Gin engine to imitate Chi mounting.
- Separate request transport, domain decision, transaction ownership and asynchronous effects only where the observed use case warrants the boundary; do not invent generic interfaces or unused folders.
- Propagate `context.Context` and transaction-scoped `txCtx`; never switch silently to a global DB pool mid-transaction. Preserve cancellation, deadline and rollback-only behavior.
- Treat authorization and session identity as **action-level correctness** (not only route-level middleware). Respect no-leak error behavior if actual domain contracts require it.
- For queue operations, document when external effects happen relative to DB commit and what retry/idempotency or recovery mechanism exists. Do not claim atomic delivery without an outbox or other verified bridge.
- Scope the patch to the smallest coherent behavior. Preserve or add behavior tests; do not delete tests to make code appear green.

## Verification and handoff
- Verify targeted behavior and a negative/failure case. Format changed Go files; use `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `make swagger-check` as appropriate and **only claim commands actually run**.
- Where external infrastructure is required, note environment, skipped cases, test fixtures and rollback.
- Provide exact base/HEAD and changed file paths; describe invariants, source comparisons, test output/failure, untested security/operational risks and a draft PR. Do not autonomously merge `dev`.
