# Database migrations and SQLC

The source of schema truth is `db/migration/`. `db/query/` is source SQL,
and SQLC generates `models/db/`.

```sh
make config-local
make migrateup
make migrateup1
make migratedown1
make sqlc
```

`migrateup` is implemented by `bdspro migrate` using the **same INI
database target** as `bdspro web`; the previous Makefile `DB_URL`
shortcut was removed. `migratedown` and `migratedown1` require an
explicit `RUN_MODE = development`. This guard prevents a common mistake,
not an absolute guarantee that a developer did not select production DB
credentials while in development mode.

`cmd/migrate.go` loads database-only INI settings (does not initialize Gin,
Redis, auth or worker). The migration code is in `models/db/migration.go`;
do not create a second GORM migration owner or run `AutoMigrate` on boot.

CI tests upgrading existing account and outbox rows across schema versions.
It is not a proof that every deployment can safely roll back after new
application writes. Migrations need version compatibility and a backup/
roll-forward plan in real deployments.
