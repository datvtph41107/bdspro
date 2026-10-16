# Lecture #6–#10 — PostgreSQL Transactions, Locks, Isolation & GitHub Actions

Canonical target: BDSPro dev; SimpleBank historical Section 1. Official numbering:
- #6 Transaction with Store + execTx + business transaction
- #7 Transaction locking and deadlock analysis
- #8 Avoid deadlocks through globally consistent lock ordering
- #9 Isolation levels and read phenomena
- #10 GitHub Actions automation (SimpleBank #11 is Gin API, **not** Actions)

## BDSPro mapping

- `db/sqlc/store.go`: `Store` embeds generated `*Queries`, holds `*pgxpool.Pool`; `execTx` wraps `pgx.BeginFunc`. `RegisterTx` is the real multi-table transaction analogous to `TransferTx`.
- `auth/service.go`: validates input and hashes password, delegates SQL writes to `Store.RegisterTx`, then sends verification email **after** commit. A mail delivery failure does not rollback committed SQL (existing behavior).
- `db/sqlc/store_test.go`: successful COMMIT, duplicate-email rollback, five competing registrations (exactly one success).
- `db/sqlc/locking_test.go`: two accounts locked in the same global order despite reversed caller order, including concurrent transactions. Educational demonstration, not a new source of production locking requirements.
- `db/sqlc/isolation_test.go`: compares READ COMMITTED and REPEATABLE READ using two live PostgreSQL transactions and a committed update.
- GitHub Actions: disposable PostgreSQL 18 databases, migration, SQLC generate, tidy, formatting, vet, integration suite and SQLC race tests.

## Safety boundary

COMMIT tests run **only** when `SQLC_TEST_DATABASE_URL` points to the local-host database named `bdspro_sqlc_ci`; all committed test rows are deleted in a subsequent cleanup transaction, never with TRUNCATE. The developer's `bdspro` database remains suitable for rollback-only SQLC tests. These names are test guards, not authorization for a production DB.

On the developer machine, prepare an intentionally disposable PostgreSQL DB:
```sh
createdb -h localhost -U root bdspro_sqlc_ci
make migrateup DB_URL='postgresql://root:secret@localhost:5432/bdspro_sqlc_ci?sslmode=disable'
SQLC_TEST_DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro_sqlc_ci?sslmode=disable' make testsqlc
SQLC_TEST_DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro_sqlc_ci?sslmode=disable' make testsqlc-race
```

Never run destructive integration scripts with `CORE_TEST_DATABASE_URL` or `SOURCE_TEST_DATABASE_URL` pointed to `bdspro`.
