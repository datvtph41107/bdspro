# Development workflow

BDSPro is one Go source tree with three CLI operations: `web`, `worker`
and `migrate`. None of these is a separate microservice or framework.

## First-time clone

```sh
git clone --branch refactor/gitea-ownership-r01 https://github.com/datvtph41107/bdspro.git bdspro-r01
cd bdspro-r01
docker compose up -d postgres redis mailpit
make config-local
# Review custom/conf/app.ini before running the application.
make migrateup
```

`make config-local` copies `custom/conf/app.example.ini` only when the
ignored local file does not exist. It never overwrites local modifications.
The example uses localhost service addresses. Credentials and signing keys
in examples are **disposable development values**, never production secrets.

## Run processes

```sh
make server     # terminal A: HTTP
make worker     # terminal B: outbox + Asynq
```

The HTTP process does not run Asynq workers. The worker does not bind HTTP.
To use automatic restart while editing Go:

```sh
make watch      # Air: rebuild bin/bdspro and restart web
```

Air is optional and configured in `.air.toml`. It watches Go code,
excludes tests and generated files, and runs the **default local config**
`custom/conf/app.ini`. For a custom `CONFIG` value use `make server
CONFIG=/absolute/file.ini`; the Air command does not inherit this Make
variable automatically.

## Verify behavior

```sh
make build
make test
make race
make vet
make swagger-check
curl -i http://127.0.0.1:8080/health
```

The examples above need running dependencies for integration tests.
When making an API change, use the [backend guidelines](guidelines-backend.md)
and [testing guide](testing.md). Do not push or merge until requested.

## How to navigate a request

`cmd/web.go` → `routers/init.go` → `routers/api/routes.go` →
`routers/api/handler/` → `services/auth` or `services/organization` →
`models/db/` → `db/query/` and `db/migration/`.

Follow the exact function calls before moving code between these owners.
