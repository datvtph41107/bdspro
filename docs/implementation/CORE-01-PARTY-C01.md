# CORE-01 — Real implementation slice (2026-10-10)
Status: **CI-VERIFIED on GitHub Actions PostgreSQL 18 (2026-10-10)**; source remains isolated in draft PR #2, not merged to dev. Local user's PostgreSQL remains unverified.
Base: remote `dev@80f522352bccbc2e215bf3e75cbfa2fa815ba8ea`, feature branch `implementation/core01-party-c01-20261010`.
Business source: [Business-to-Core checkpoint](https://github.com/datvtph41107/bdspro/blob/architecture/canonical-database-final/docs/continuity/checkpoints/2026-10-10-business-to-core/README.md) and V0.4 canonical DBML (164 **logical** candidates).

## What was implemented
- `migrations/000006_create_party_identity.up.sql` + reverse migration.
- PostgreSQL tables: `parties`, `persons`, `organizations` and Organization name required.
- Deferred constraint triggers: after each transaction **exactly one** of Person or Organization for any surviving Party. Parent row-level `FOR UPDATE` lock on subtype writes avoids separate Person/Org concurrent transactions committing mutually incompatible facts. Attempting to move a subtype's `party_id` is rejected.
- `party/identity.go`: small transaction functions `CreatePerson` and `CreateOrganization`, no forced Account onboarding, no legal-company verification implied.
- `party/identity_integration_test.go` (build tag `integration`): valid creation, missing subtype rejection at commit, dual subtype rejection at commit, deletion of only subtype rejection, deliberate subtype replacement in **one transaction**, two competing writers cannot both commit. Guard prevents running against any DB whose name does not end `_test`.
- `.github/workflows/core01-postgres.yml`: runs original Listing migrations 000001..000005 and 000006, `go test ./...`, PostgreSQL 18 integration tests.

## Why this design, and what it costs
PK/FKs from subtypes to parties forbid nonexistent parents, but they do **not** ensure every Party has a subtype, nor do they prevent both subtype tables referencing one Party. Deferred checks permit creating one Party and its sole subtype in the same transaction, while row locks serialize changes against one Party. A stable identity is separate from frequently changing presentation, auth credentials, Organization claims, provider legitimacy and subscription.
Cost: every subtype mutation acquires a Party row lock; deferred constraint execution queries two indexed subtype PKs; four triggers and two functions add schema/operational complexity. At this scale correctness outranks negligible per-Party write overhead; measure contention before extension. Direct admin/manual SQL cannot bypass except privileged DDL, disabled triggers or superuser bypass. Restrict DB roles and migrations accordingly.

## Non-goals and open contracts
- No Account, credential, session or policy-based authority implementation in this gate; CORE-02/03 next.
- No `party_kind` stored; directory projection can UNION Person/Organization, avoiding duplicated mutable truth.
- No legal Organization registration/claim, company validation, onboarding/publisher grants, tenant CRM, individual/b2b memberships, media, Listing revisions.
- No API handler exposed for new functions; direct `pgxpool` boundary is a narrow first slice. Existing repository currently uses sqlc for Listing; generated sqlc models for new tables, party SQL queries and API integration remain **pending**. Do not hand-edit generated sqlc files without running pinned generator/version.
- No code change to `dev` until PR acceptance; feature branch does not include docs on separate design branch.

## Actual test instructions
**WARNING**: migrate only an isolated database created for testing. The Go integration helper requires database name `*_test`.
```bash
# use a disposable database such as bdspro_test, with postgresql connection available
export CORE_TEST_DATABASE_URL='postgresql://.../bdspro_test?sslmode=disable'
for f in migrations/*.up.sql; do psql "$CORE_TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"; done
go test ./... -count=1
go test -tags=integration ./party -count=1 -v
```
Use `migrate` CLI migration semantics for environments with migration history. The manual `psql` loop above is only for a brand-new disposable database, not a partially migrated/production database. No local PostgreSQL runtime was accessible to this authoring session; however a real PostgreSQL 18 GitHub Actions service executed the migration and tests successfully in run 38014866038. Repeat in the user's isolated lab before any deployment.

## Verification checklist
- [x] GitHub feature branch created based on identified remote `dev` SHA
- [x] Versioned SQL up/down authored without editing old Listing schema
- [x] Go transaction functions authored
- [x] Standard and PostgreSQL integration tests authored
- [x] CI workflow authored
- [x] CI confirmed green: run [38014866038](https://github.com/datvtph41107/bdspro/actions/runs/38014866038), job 114102785623. Six real PostgreSQL integration tests passed for C01; Go standard tests and migrations 000001..000006 passed.
- [ ] local/current PostgreSQL integration manually rerun
- [ ] sqlc generator regenerates cleanly from full schema
- [ ] PR reviewed/merged onto `dev`
- [ ] CORE-02 Account design & auth implementation

## Evidence — CI logs, observed (2026-10-10)
- PostgreSQL 18 service healthy; migrations 000001..000006 applied successfully.
- `go test ./... -count=1` green for existing packages and new `party`.
- `go test -tags=integration ./party -count=1 -v`: `TestCreatePersonAndOrganization`, `TestCommitWithoutSubtypeIsRejected`, `TestCommitWithBothSubtypesIsRejected`, `TestDeletingTheOnlySubtypeIsRejected`, `TestSwitchSubtypeWithinOneTransaction`, `TestConcurrentCompetingSubtypeWritesDoNotCommitBoth` and blank-name validation all PASS.
- Execution proof does NOT cover every serializable interleaving, deployed production state, or current local user environment. PostgreSQL up/down/re-up has now been proven on a disposable CI database.

## Second CI run: reverse migration proved (2026-10-10)
- [GitHub Actions run 38015055276](https://github.com/datvtph41107/bdspro/actions/runs/38015055276), job 114103360211, completed **SUCCESS**.
- Applied original Listing migrations and CORE-01 migration in PostgreSQL 18; Go tests, C01 integration tests all passed.
- Applied `000006_create_party_identity.down.sql`, asserted the three Party tables no longer existed, re-applied `000006_create_party_identity.up.sql`, reran real PostgreSQL C01 integration tests successfully.
- This is evidence of DDL reversibility on a clean disposable schema, NOT proof it is safe to roll back a populated production database.
