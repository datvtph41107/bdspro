# CORE-05 — Source Intake & Price History

**State:** isolated GitHub feature branch; integration pending the user-local canonical Auth/Organization implementation.

## Why this module exists

An authorized team employee captures an unverified real-estate source (e.g., Zalo report of ~75 m2 property at VND 4.2bn) then appends subsequent price statements. Neither source nor reported price proves property ownership, legal/advertising authority, latest confirmed price or an actual published listing. Distinguish source, property, listing and CRM inquiry.

## Implemented

- Module-relative reversible SQL migration: db/source/migrations/000001_*.up.sql/.down.sql.
- sqlc queries: db/source/query/source.sql, config: sqlc.source.yaml.
- Go service: source/service.go (transactional capture + optional price; append-only reports; exact decimal strings; scoped idempotency and resource authorization).
- HTTP: source/http.go. Authenticated account must come from a trusted session resolver, never JSON/header.
- PostgreSQL negative/integration tests: source/service_test.go; GitHub workflow: .github/workflows/core05-source.yml.
- Isolated test fixture for missing remote prerequisites: db/source/testdata/core_prerequisites.sql. This is NOT a BDSPro production migration.

## API contract

Caller Account is server-verified, scoped Organization is from path, every write and read rechecks active Membership + Role Permission + no open membership block + unclosed Account + unarchived Organization.

- POST /organizations/{organizationID}/sources — Idempotency-Key required (1..100 bytes); JSON {"channel":"zalo","label":"reported house","area_m2":"75","initial_price":{"amount":"4200000000","currency":"VND","price_basis":"asking"}}. Returns 201 {"id":N,"replayed":false}, exact retry 200 {"id":N,"replayed":true}; changed replay 409.
- GET /organizations/{organizationID}/sources/{sourceID} — protected source details.
- POST /organizations/{organizationID}/sources/{sourceID}/price-reports — Idempotency-Key; JSON {"amount":"4100000000","currency":"VND","source_basis":"phone_reconfirmation"}; exact retry 200, conflicting payload 409.
- GET /organizations/{organizationID}/sources/{sourceID}/price-reports — append-only recorded history; NOT latest confirmed official price.

Money fields are JSON strings (not float64), stored PostgreSQL NUMERIC(19,4). Initial capture and initial price commit atomically. Scope key is (organization, account, request key) for capture and (organization, source, account, request key) for price reports. SQL UNIQUE prevents duplicate rows; hash determines exact retry versus conflicting intent.

Missing verified Account -> 401; unreadable resource or different organization -> 404; malformed JSON/invalid amount -> 400; replay hash conflict -> 409. Database operational errors -> 500 without leaking SQL details.

## Safe build and test

    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
    sqlc generate -f sqlc.source.yaml
    go test ./source

Integration tests only run when SOURCE_TEST_DATABASE_URL is set; they refuse destructive fixtures outside a database named bdspro_source_ci. GitHub CI provisions PostgreSQL 18, fixture core, module schema, generates sqlc, and runs tests. The fixture tests CORE-05, NOT full Canonical C01/real user session validation.

**Migration integration:** db/source/migrations uses a module-relative 000001 number. Do NOT run its directory with golang-migrate against a database that already uses the local application's migration series. Create a newly numbered normal migration on the local branch and copy the SQL content. See docs/source/LOCAL-INTEGRATION.md.

## Architectural decisions and limitations

- Preserve reported sources independently of Property/Listing; allow unknown facts to stay unknown.
- Append price reports rather than mutating previous source assertions; recorded_at is database time and must not be interpreted as source statement time. Schema includes statement_at and received_at for subsequent API expansion.
- Composite (organization_id,intake_id) FK prevents price report cross-tenant links even when both IDs individually exist.
- Read/write rights are checked in each SQL statement; this does not prove strict serial ordering against a concurrent role revocation at PostgreSQL READ COMMITTED. Stronger admin/offboarding timing is a future acceptance gate.
- Permissions names sources.create, sources.read, sources.price_report, with membership/role/blocks contract from the local core.
- No Zalo scraping, photos/media rights, auto-publishing, cross-org sharing, dedupe by area/address, CRM Contact auto-creation, forced marketing consent, legal verification or business-market adoption claims.
- Last known GitHub dev still contains Listing code but not the user's local Auth/Organization migrations; production assembly and local synchronization are not remotely verifiable.
