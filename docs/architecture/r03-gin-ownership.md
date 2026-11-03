# R03: Gitea file ownership with Gin as the HTTP runtime

Reference: Gitea `12b20c7c9962e0be75912b290a0c5778799b0ad2`. BDSPro parent: `13db69b76321c65899c203fa4ed7b016fa14a837`.

## Decision
The requested parity is in package ownership, file naming, method placement, model behavior and operations—not an HTTP framework replacement.
Gin is the **sole** production HTTP engine. The unmounted Chi/Web and typed APIContext experiments were removed after no endpoint consumed them.

## Correspondence
| Gitea | BDSPro R03 | Runtime |
| --- | --- | --- |
| `routers/api/v1/api.go` | `routers/api/v1/api.go` | Gin route groups; preserves preexisting unversioned URL paths |
| `routers/api/v1/user/*` | `routers/api/v1/user/auth.go`, `google.go`, `phone.go` | Gin request validation, authentication and responses |
| `routers/api/v1/org/*` | `routers/api/v1/org/org.go` | Gin endpoints for organization operations |
| `models/organization/org.go` | `models/organization/org.go`, `org_user.go` | SQLC/pgx; atomic owner membership and SQL authorization |
| `modules/structs/*` | `modules/structs/auth.go`, `user.go`, `organization.go` | Public HTTP JSON types |
| `services/convert/*` | `services/convert/user.go`, `organization.go` | Convert data to public representations |
| `models/db/context.go` | `models/db/store.go` | Existing explicit pgx transaction; no hidden transaction-in-context |
| `cmd/web.go`, `routers/init.go` | Same BDSPro package paths | Preserve tested startup and graceful shutdown |

## Compatibility rules
- URL method/path matrix matches the 34-route pinned R01 contract; the `routers/api/v1` directory is source organization only, not an added `/api/v1` URL prefix.
- Preserve Gin request/middleware short-circuit ordering, verified-contact checks, rate limit failure policies, PKCE/state, OTP replay protection, SQL authorization predicate, and PostgreSQL outbox semantics.
- PostgreSQL migrations and SQLC generated files remain authoritative; no GORM runtime.
- Convert DTOs in one location; do not bind generated SQLC types directly into new HTTP contract fields.
- No new technical modules without a consumer, tests, and a measurable benefit.

## Review evidence (required before merge)
`go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, SQLC deterministic generation, migration upgrades on isolated PostgreSQL, API/OpenAPI route coverage, Docker image + HTTP smoke using Go 1.27.3.

This document describes an in-progress PR until its GitHub CI is green. No claim of framework parity or performance improvement is made.
