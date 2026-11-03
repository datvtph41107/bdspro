# R02 acceptance and rollback protocol

## Baseline before code change

```sh
git rev-parse HEAD
git status --short
git diff --check
go test ./...
go vet ./...
go test -race ./...
go build ./...
sqlc generate
git diff --exit-code -- models/db
docker compose config
go run ./tools/r02audit -root . -contract docs/architecture/r02/routes-r01.json
```

These commands **are required future checks**, not results claimed by this document. Run with the project's required Go toolchain (R01: Go 1.27.3) and installed tools.

## PostgreSQL acceptance

R01 integration tests commonly require `BDS_TEST_DB_URL`, `BDS_TEST_MODE=disposable`, database `bdspro_test` and applied migrations. A green test suite that skips those tests is not sufficient. Use dedicated isolated test databases/fixtures, avoid exact total-row assertions across concurrent packages, and log actually executed test names.

- Organization: success, duplicate owner/membership, owner/outsider, FK rollback and no unauthorized mutation.
- Account: email+password registration, OAuth-first, linking, credentials, password reset, email verification and token revocation.
- Phone: wrong code records failed attempt; lockout; right code; replay rejected; global uniqueness.
- Outbox: job is committed with business state; worker dispatch after commit; retry/idempotency; Redis temporarily unavailable.
- Transactions: cancellation, rollback on DB errors, exactly one transaction scope; security failures that require commit must commit explicitly.

## HTTP acceptance

Compare against the [pinned source-level inventory](routes-r01.json). For each route, test enabled/disabled feature variants, authentication requirements, verified-email/verified-contact gates, request validation, status, headers and JSON public contract. Test both `OPTIONS`/CORS and 404/405 behavior where applicable.

Do not automatically introduce `/api/v1`: the R01 API prefix is empty. Store the **old URL** in contract fixtures even if the new Go source lives under `routers/api/v1`.

## Performance acceptance

Run the exact same work on R01 and R02, separate at least:

- Unauthenticated no-DB route: `ns/op`, `B/op`, `allocs/op`.
- Authenticated endpoint: token verification, p50/p95/p99 latency.
- `PATCH /organizations/:id`: database query count, SQL duration, concurrent owner revocation case.
- Register/OTP workflow: latency, transaction duration, failure behavior.
- Mixed traffic: pool contention, memory, request cancellation and shutdown.

Avoid invented SLA thresholds. Record actual baselines, environment, sample sizes and acceptable changes before declaring success. Gitea's published throughput cannot substitute for BDSPro measurements.

## Rollback

Each PR must be independently revertible, except the final coordinated HTTP cutover. Do not drop persisted data, rewrite task types or remove a live endpoint during refactor. Preserve old migrations; new migrations are append-only and independently reviewed. No production traffic is changed until all acceptance gates pass.
