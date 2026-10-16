# Lecture #4 — SQLC canonical database access

Implemented with SQLC v1.31.1 and pgx/v5. This is a structural refactor; no schema migrations were modified.

## Canonical ownership
- migrations/*.up.sql: versioned PostgreSQL schema, constraints and indexes.
- db/query/*.sql: human-authored queries, including authorization predicates and lock semantics.
- sqlc.yaml: one SQLC configuration; schema migrations, queries db/query, package db, output db/sqlc.
- db/sqlc/*.go: generated Go; do not edit manually. models.go reflects known schema tables; query methods require annotated SQL.
- auth/service.go, organization/service.go, source/service.go: validation, business logic, errors and transactions.

## Query groups
- party.sql: Party roots and typed subtypes.
- account.sql: account creation, credential lookup and person lookups.
- email_verification.sql: issue, consume and mark verified.
- session.sql: issue/revoke sessions and resolve account.
- organization.sql: reviewer locks, approvals, membership, roles and invitations.
- source.sql: authorized source intake, price reports and replay-safe reads.

## Invariants retained
- Register and VerifyEmail remain atomic transactions; post-commit email delivery is separate.
- Organization reviewer FOR SHARE locks, registration/claim FOR UPDATE locks and :execrows affected-row checks remain.
- Source atomicity, tenant authority and idempotency are retained.
- HTTP routes and public response contracts are unchanged.

## Commands
    make sqlc
    go test ./... -count=1
    go vet ./...
    git diff --check

CI pins sqlc v1.31.1, regenerates code and checks generated artifacts for drift.

## Scope
Schema models cover other tables (passkeys, external identities, service accounts, etc.). SQLC does not auto-create arbitrary CRUD or business features without authored queries. Tests may use fixture SQL directly.

## Local safety
Preserve local uncommitted work before pulling. .env remains ignored. Root integration tests truncate test data, so test URLs must target disposable CI-only databases.
