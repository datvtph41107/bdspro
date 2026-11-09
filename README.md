# BDSPro

Go/Gin REST monolith. PostgreSQL 18, SQLC/pgx, Redis 7 and Asynq.
One binary, explicit CLI modes, no internal routing facade or gRPC.

BDSPro follows Gitea's **source ownership and operational
discoverability**, not its domain model or entire folder tree.

[Package ownership review](docs/architecture/source-ownership.md) explains the
actual responsibilities of Gitea modules and the corresponding BDSPro
decisions. The review contains executable source-boundary checks:
`go test ./tests/architecture`.


## First-time development

Requirements: the Go version in `go.mod` (currently 1.27.3), Docker
Compose and Make. SQLC generator and migrations can be installed as needed.

```sh
docker compose up -d postgres redis mailpit
make config-local                   # copies app.example.ini only if missing
# inspect custom/conf/app.ini (disposable local database and auth secrets)
make migrateup
make server                         # terminal 1: HTTP API
make worker                         # terminal 2: Asynq, SQL outbox and retention
# optional PostgreSQL-only one-shot cleanup, independent of Redis:
# make cleanup
curl -i http://127.0.0.1:8080/health
```

Or run the entire development stack:

```sh
docker compose up -d --build
docker compose logs -f api worker
```

Mailpit UI: http://127.0.0.1:8025.
Swagger UI: http://127.0.0.1:8080/swagger/ (development only).

## Build, watch, test

```sh
make build          # writes ignored bin/bdspro
make watch          # Air, rebuilds/restarts only web
make test
make race
make vet
make swagger-check  # validates reviewed OpenAPI ↔ Gin routes
make sqlc           # regenerate models/db from db/query
```

`make swagger` is an alias for the current OpenAPI check; the previous
unused Swagger 2/Statik generation path is removed. This **does not**
create new OpenAPI schema content. Read [API documentation](docs/api/README.md)
before changing endpoints.

## Runtime and config

```text
main.go -> cmd/ (web | worker | migrate | cleanup)
web     -> routers/ -> services/ -> models/db/
worker  -> modules/queue/ -> services/worker/ -> services/jobs/
migrate -> modules/setting/database.go -> models/db/migration.go

modules/setting/  app.ini path, provider and validation
modules/log/      structured logs, sinks and close
modules/auth/     bcrypt and JWT signing/verification
modules/secret/   crypto-random proof values
modules/conn/     Redis connection lifecycle
```

The default local instance config is `custom/conf/app.ini` (ignored).
`custom/conf/app.example.ini` is the committed template for local setup.
The Docker runtime config is persisted at `/data/bdspro/conf/app.ini`.
All CLI commands use the same selected INI database target; `make`
does **not** inject hardcoded DB/Redis addresses.
Use `make server CONFIG=/absolute/path/app.ini` for a custom file.

## Developer documentation

Start at **[docs/README.md](docs/README.md)** for source navigation,
build and debugging instructions, modules/services/router ownership,
config precedence, Docker/Compose, CI/test contract and migration guidance.
[Architecture vs. Gitea](docs/architecture/gitea-ownership-r01.md)
records exact sources and differences; [historical docs](docs/archive/README.md)
are preserved separately.

OAuth2/OIDC uses `[oauth2]` in `app.ini`, a single configured issuer,
and `oauth2_credentials` for persistent account bindings. See
[OAuth2 model conventions](docs/architecture/r24-oauth2-model-ownership.md)
and [runtime cut-over](docs/architecture/r26-oauth2-runtime-consolidation.md).

Authentication audit events use [audit] HMAC_KEY, an independent production
secret, and RETENTION_DAYS in app.ini. See
[Audit events](docs/architecture/audit-login-events.md) for ownership and
retention limits. Retention needs the worker or a scheduled `cleanup` invocation; the one-shot
`cleanup` command operates with PostgreSQL only, independently of Redis.

This is not a claim of production readiness. Current limitations include
multi-worker outbox concurrency and retry/failure-injection E2E evidence,
production secrets management, TLS ingress and backup/restore procedures.
