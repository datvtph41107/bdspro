# BDSPro developer documentation

This is the navigation point for **current implementation**, not a list of proposed architectures.
The source code, `go.mod`, `Makefile` and CI workflow remain authoritative when documentation disagrees.

## Start here

- [Build from source](build-source.md): prerequisites, build and executable output.
- [Develop locally](development.md): clone, config, PostgreSQL, Redis, web/worker, Air.
- [Testing and acceptance](testing.md): unit, integration, migrations, API contract, CI.
- [Backend guidelines](guidelines-backend.md): ownership, names, HTTP/SQL/worker boundaries.
- [Architecture](architecture/README.md): startup and request paths, database, service ownership.
- [Configuration](operations/configuration.md): `custom/conf`, INI precedence and secrets.
- [Docker and Compose](operations/docker.md): dev containers, volumes and first-boot setup.
- [Migrations](operations/migration.md): SQLC and schema migration decisions.
- [Logging](operations/logging.md): stdout and file modes.
- [API and Swagger](api/README.md): OpenAPI source and development-only UI.
- [Source ownership evidence](architecture/gitea-ownership-r01.md): actual comparison with Gitea.
- [Historical records](archive/README.md): older lectures, architecture plans, decisions and acceptance notes.

Gitea itself keeps developer-facing documents in its root `docs/`, including
`build-source.md`, `development.md`, `testing.md` and
`guidelines-backend.md`. BDSPro uses the same discoverability principle,
but only documents systems it actually operates. Its documentation site is
not imported as a new framework.

## Source of truth by concern

| Concern | Canonical owner |
| --- | --- |
| CLI process | `main.go`, `cmd/` |
| Configuration | `modules/setting/`, `custom/conf/app.example.ini` |
| Protocol and routes | `routers/common/`, `routers/api/` |
| Business rules | `services/auth/`, `services/organization/` |
| Database and schema | `models/db/`, `db/query/`, `db/migration/` |
| Async dispatch | `services/jobs/`, `services/worker/` |
| HTML and API assets | `templates/auth/`, `templates/swagger/` |
| Dev commands | `Makefile`, `.air.toml` |
| Image/deployment | `Dockerfile`, `docker/`, `docker-compose.yaml` |
