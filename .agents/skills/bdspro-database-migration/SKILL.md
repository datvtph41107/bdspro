---
name: bdspro-database-migration
description: Use before changing or reviewing BDSPro PostgreSQL SQL schema, db/migration files, db/query SQLC queries, SQLC-generated bindings, or transaction contracts. Focus on data safety and deploy compatibility.
---

# Safe DB and SQLC evolution

1. Read `AGENTS.md`, current `README.md`, `Makefile`, `models/db/context.go`, `db/migration/`, `db/query/` and affected callers; pin exact branch SHA and validate current ownership/lease. BDSPro's 2026-11-12 source reset did **not** reset database schema or erase data.
2. Define current schema, intended behavior, read/write paths, index/query requirements, existing data and rollback risk. Identify whether a fresh database and existing database require different checks.
3. Design an **expand → migrate/read compatibility → contract** sequence when live old/new code may overlap; do not assume Discourse's Rails pre/post-deploy migration mechanics exist in BDSPro.
4. Evaluate lock duration, large-table rewrites, index creation, transaction boundaries, foreign keys, uniqueness, defaults, concurrency, nullability and backfill cost. For destructive changes require explicit migration/release approval and recovery plan. Never recommend irreversible data rollback as if restoring binaries restores data.
5. Keep schema SQL, SQLC queries and generated bindings coherent. Use the **actual** `Makefile` workflow: `make sqlc-check` regenerates generated SQLC code and checks `models/db/sqlc` and `models/db/sqlc_types.gen.go` for diffs; inspect the working tree afterward.
6. Create tests for fresh-schema migration, old-data compatibility, negative case, nested `WithTx` rollback-only, cancellation and transaction context propagation where relevant. Run infrastructure-dependent checks only in configured disposable environments; record what wasn't run.
7. Handoff: migration filenames, exact changed queries/generated artifacts, source SHA, lock/data risk, compat timeline, test evidence, explicit reviewer/owner approval and rollback limitations.

This skill does not authorize migrations or production writes; no deletion or data movement without a reviewed scope.
