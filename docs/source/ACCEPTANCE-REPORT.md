# CORE-05 — End-to-end design, decision impacts & acceptance register

**Status:** IMPLEMENTED ON ISOLATED FEATURE BRANCH / INTEGRATION NOT YET CERTIFIED.
**PR:** https://github.com/datvtph41107/bdspro/pull/4
**Base branch:** GitHub dev; user's Auth/Organization source is reported local-only.
**Scope:** Source Intake & Price History (NOT "all 164 canonical tables").

## 1. Business problem and concrete walkthrough

ABC employee An (Account 25, active Membership in ABC, with sources.create + sources.price_report + sources.read) receives a Zalo message saying "a ~75 m2 house is offered around VND 4.2bn". This is an illustrative intake, not official ownership, location or price proof.

**Before:** ABC has no source intake for this communication; no Property or Listing is justified.

**After capture:**
- source_intakes: one ABC-scoped row with recorder Account, Zalo channel, self-reported area, optional notes, version 1, original request scope/hash.
- source_price_reports: one ABC-scoped amount=4200000000, currency=VND, recorder, recorded_at timestamp and optional statement basis.
- **No** Property, Listing, ownership, "latest verified price", public publication, customer Account, auto-import from Zalo or CRM Contact is created.

**Next day:** An hears 4100000000 VND. APPEND another source_price_reports row. Do not UPDATE the first assertion. Response history shows both assertions; the most recently inserted report is not necessarily the most recent statement by the external source.

## 2. Comparable design options and chosen pressure

| Choice | Mechanism in this workflow | Limit/cost |
| --- | --- | --- |
| Store everything in listings | One place for reported description and price | False claim of a published market offer; cannot retain contradictory unattributed source prices |
| Source Intake + one mutable price | Fewer rows, one current number | Overwrites record of earlier seller/source statements, hides differences in timing/source basis |
| Source Intake + append-only Price Report **selected** | Separate incoming observation and its price assertions; preserves history | Two INSERTs on initial capture, a transaction, more indexes and query complexity |
| Event store for every field change | General replay mechanism | Not justified by current needs and increases reconstruction/operational cost |

The selected design is justified by the specific action "price changed after phone call", not by generic normalization fashion. It remains a candidate until the actual test and business evidence validate the expense.

## 3. Implementation files

- db/source/migrations/000001_create_source_intakes_and_price_reports.up.sql/.down.sql: versioned module migration template; composite same-org FK; exact numeric; permission catalogue; capture and price idempotency keys.
- db/source/testdata/core_prerequisites.sql: minimal schema for CI only; not canonical Identity/Auth implementation.
- sqlc.source.yaml, db/source/query/source.sql: named write/read/replay SQL.
- source/service.go: pgx transaction, validation, idempotency hash, protected queries, price history.
- source/http.go: four API routes; AccountResolver must be provided by trusted Session middleware; no header-based Account input.
- source/service_test.go: unit/integration coverage.
- .github/workflows/core05-source.yml: PostgreSQL 18 test pipeline.
- db/source/README.md and docs/source/LOCAL-INTEGRATION.md.

## 4. Technical decisions, evidence and downstream impacts

### D01: Exact SQL writes, not guessed object permissions

Both writes include current Membership/Role/Permission/Block checks for the account from the authenticated boundary. A client-supplied org path is never sufficient. Read paths scope organization+source and recheck permission. Consequence: SQL statements are longer/repeated; avoid naive global read followed by UI filtering. The exact dynamic right to approve an external publication remains separately out-of-scope.

**Unproven:** SQL READ COMMITTED read checks under concurrent revocation do not by themselves force a strict revocation-before-commit ordering. Add row locks or a permission-version strategy if required.

### D02: Composite FK for tenant consistency

source_price_reports(organization_id,intake_id) references source_intakes(organization_id,id). A price row that says org B but references org A's source fails even if the intake ID exists. This protects link integrity, not a human's permission to read/write.

### D03: Append instead of mutating source-price history

A later price report does not overwrite earlier rows. Query orders by database recorded_at plus ID. There is no "authoritative_current_price" field; source statement_at may be older than recorded_at. Read model selection policy is deferred.

### D04: Exact monetary representation

Money is a decimal string at HTTP/Go boundaries, PostgreSQL NUMERIC(19,4) in the database, with >0 and 3-uppercase-letter format check. No float64 computations or fake currency conversion. The CHECK does not prove ISO currency validity, whether price is net/gross, or trustworthy seller statement.

### D05: User intention vs retry

A capture request is scoped by (organization,actor_account,key). A price request by (organization,intake,actor_account,key). Request payload is normalized before hashing SHA-256. UNIQUE prevents duplicate writes; lookup compares hash to distinguish exact replay vs changed intent. Same-key concurrent requests serialize on the unique index; initial price is committed in the same transaction as intake.

**Costs:** additional indexes and stored request hash; client must persist/reuse Idempotency-Key on transport retry, generate a NEW key for a genuinely new report.

**Residual:** no TTL/garbage collection (intentional durable dedupe); not a general-purpose cross-workflow idempotency store; single source capture replay returns original source ID, not a comprehensive persisted HTTP response envelope.

### D06: Auth boundary is injected, not fabricated

source.NewHandler requires trusted AccountResolver from the host application's verified Session middleware. Nil/missing resolves to HTTP 401. This prevents a fake X-Account-ID test hook from silently being deployed as authentication. It also means the feature branch is not an independent full login system; wiring with local-only Auth is the explicit integration gate.

### D07: Safe branch/migration separation

The user's local Auth/Org migration sequence is not on remote dev. Putting a fictitious new global 00000N directly on dev would risk duplicate versions and inconsistent source. This feature therefore carries a module-relative 000001 SQL template and a fixture for GitHub CI, plus instructions for local migration renumbering after reconciliation. **Do not run independent version 000001 against an already-versioned local DB.**

### D08: Privacy and provenance

Capture stores user-entered source labels/summaries, not raw Zalo tokens or a web scrape. Minimize third-party PII; no automatic CRM identity merges or marketing. Before real deployment decide data retention, authorized source provenance, audit redaction and media rights.

## 5. Acceptance gates and observed evidence

| Gate | Required evidence | State as of branch creation |
| --- | --- | --- |
| Source code committed | GitHub branch file contents and commit | VERIFIED (remote feature branch) |
| SQL migration on a fresh isolated DB | PostgreSQL run output | CI configured, inspect actual run |
| sqlc generates and Go compiles | CI step output | CI configured, inspect actual run |
| Transaction atomicity | Integration test with no orphan rows | Test authored; verify run |
| Same-org authorization | Wrong-tenant, expired/blocked membership tests | Wrong-tenant/revocation tests authored; blocked case may need additional test |
| Idempotency | Same key same/changed payload, concurrency | Tests authored; verify run |
| Historical prices | Two distinct price statements preserved | Test authored; verify run |
| HTTP Auth boundary | Nil resolver 401 and authorized stub | Stub integration test authored, NOT a real Session test |
| True Session/Org integration | Local-only Auth/Org source + real middleware | BLOCKED on unpublished local source |
| Local Git and DB sync | User-supplied git status + DB version/result | NOT VERIFIABLE remotely |
| Real-user deployment acceptance | privacy review, role-revocation races, observability and load | NOT PROVEN |

## 6. Follow-up in correct dependency order

1. Review isolated CI and fix actual failures, do not infer success from green commit or PR creation.
2. Get exact local Git diff/status and canonical migration chain without overwriting user work; read docs/source/LOCAL-INTEGRATION.md.
3. Reconcile prerequisite schema and local Auth Session middleware, re-generate sqlc against real migrations.
4. Test with current user-facing Auth and cross-org roles, plus time/race semantics of membership revocation.
5. Mark integration accepted only after local tests with actual schema/runtime and record commit hashes and DB migration versions.
6. Progress to CORE-06 CRM Inquiry & Follow-up without turning source reports into verified listings.

**Never claim that a standalone CI fixture proves the remote/local application has been fully synchronized.**
