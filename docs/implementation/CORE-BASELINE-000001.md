# BDSPro — fresh CORE baseline, version 000001
Date 2026-10-10. **Feature branch only; draft until compatibility with live Listing code is resolved.**

## Decision and source state
User reports a CLEAN local PostgreSQL DB and an emptied local migrations directory. The remote `dev` branch still had five historical Listing lab migrations at SHA `80f522352bccbc2e215bf3e75cbfa2fa815ba8ea`; those are preserved in Git history and unchanged on `dev`. This branch deliberately omits ten Listing migration files and starts a new `000001_create_party_identity.{up,down}.sql` baseline, instead of packing 164 candidate tables into one init.
Other business design files (V0.4 with 164 logical candidate tables) live on design branch `architecture/canonical-database-final`, not physical migration authority.

## First durable contract
C01: at each commit, every Party has exactly one subtype (`Person XOR Organization`). Stronger subtype identity rule: a Person **cannot become Organization**, even in a single transaction. New Company = new Organization Party, with actual relationship/role modeled separately. Org `name` may change, unlike subtype identity. Account/auth and legal verification belong to later gates.

Migration creates three tables, row-level lock on Party before subtype INSERT, immediate contradiction guard, immutable subtype UPDATE/DELETE guard, and deferred Party creation constraint. Bare Party cannot commit. Business identity is not authentication; Organization row is not evidence of incorporation.

## Tests and CI
- Real PostgreSQL service (CI) imports only fresh migration 000001.
- Standard Go tests; Postgres integration tests: valid create, no-subtype commit rejection, double subtype rejection, immutability, UPDATE prohibition, Parent FK RESTRICT, explicit same-transaction kind conversion prohibition, and simultaneous attempt to add conflicting subtype with Party lock.
- Down and fresh up replay after running tests in a disposable database.
- No runtime production/Local DB touched by GitHub.
- Do not run `make dropdb`/`make migratedown` against the default old `DB_URL` without verifying DB identity and backups.

## Critical source compatibility gate before merge
Historical `main.go`, `listing/`, `db/query/listing.sql`, `db/sqlc/*` continue to refer to physical `listings`/`listing_publications`, which do not exist in this new baseline. Default Go unit tests may still pass because generated query code compiles, but **Listing HTTP routes or sqlc regenerate from clean migrations can fail**. This is intentional visibility, **not a green production integration state**. Do not merge/release until we choose to retire the Listing lab endpoint or migrate it in a separately justified capability, reconcile sqlc queries/generated code and test boot/API against the new database.
Old draft PR #2 implements C01 as migration 000006 against the historical Listing timeline. That PR is now a historical comparison, not an additive migration to merge into the fresh baseline.

## What to implement next
First verify green `000001` C01 tests and migration rollback, then decide the Listing-code retirement/replacement so `sqlc generate` reproduces cleanly. After that, develop `000002` Account as a separate migration with explicit account/person cardinality, credential/session lifecycle decisions and identity security tests; then organization Membership and authorization.
Keep BUSINESS-00 adoption/payer gate separate from engineering proof. No 164-table bulk migration.

## Evidence: PostgreSQL 18 real execution (2026-10-10)
[GitHub Actions CI run 38017389419](https://github.com/datvtph41107/bdspro/actions/runs/38017389419), job 114110595211: **SUCCESS**. Exactly one new migration `000001` applied, no Listing table present; `go test ./...` green; all C01 integration tests passed for XOR subtype, bare-Party commit rejection, dual-subtype rejection, deletion/update prohibition, same-transaction kind-switch prohibition, and competing write while Party row locked; `000001 DOWN → tables absent → UP → retest` green. Actual CI log checked. The first CI [run 38017292215](https://github.com/datvtph41107/bdspro/actions/runs/38017292215) discovered a mistaken test expected SQLSTATE 23503 for FK `ON DELETE RESTRICT`; PostgreSQL correctly returned 23001, test corrected and rerun green. **Do not confuse targeted core tests with application integration**: Listing queries/routes and sqlc regeneration still depend on schema no longer present and are merge blockers.
