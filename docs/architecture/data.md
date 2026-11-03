# Persistence: schema, SQLC and transaction ownership

The **only canonical schema history** is `db/migration/*.sql`.
`db/query/*.sql` defines named queries; `sqlc.yaml` generates Go code into
`models/db/`. The generated Go is committed and compared in CI.

`models/db/engine_init.go` connects to PostgreSQL, pings it, and closes a
failed pool. `models/db/store.go` provides SQLC and transaction boundaries.
`cmd/migrate.go` and development-mode `cmd/web.go` call the same versioned
SQL migration implementation using database settings loaded from INI.
`make migrateup` uses the same selected INI as `make server`.

A GORM introduction would be a *different* implementation decision:
which operations justify it, who owns the transaction, and how it respects
the same schema migration owner. Running GORM AutoMigrate beside versioned
golang-migrate is prohibited without an explicitly reviewed migration design.

A worker dispatches outbox work after records are committed, but PostgreSQL
and Redis do not share a transaction; delivery is at least once.
Do not claim exactly-once behavior from stable Asynq IDs alone.
