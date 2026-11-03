# BDSPro

Go/Gin REST monolith. PostgreSQL 18 with SQLC/pgxpool, Redis and Asynq.
One entry point (`main.go`), no gRPC, gRPC-Gateway or custom route registry.

## Source ownership (R00, Gitea-inspired)

```text
main.go                        process entry, build information and exit hook
cmd/main.go                     urfave/cli/v3 command declaration and execution
cmd/web.go                      HTTP runtime (alias: serve)
cmd/worker.go                   Asynq processor and outbox dispatch runtime
cmd/helper.go                   signals, early logging, shared setting loader
routers/init.go                 HTTP initialization
routers/api/                    Gin server, routes, handlers, middleware, response
models/db/                      SQLC/pgx generated queries and transactions
modules/setting/                INI config, common settings, log policy
modules/log/                    Gitea-style logging API, owned writers, shutdown
modules/{errors,token,util}/    shared non-HTTP logic
services/                       authentication and organization operations
services/{jobs,worker,mail,notification}/
db/{migration,query}/           canonical SQL and migration history
docker/root/etc/templates/      app.ini for first-time container setup
```

One binary has `web` and `worker` runtimes. No `internal/` tree remains. Migration remains owned by golang-migrate and `make migrateup`. SQLC remains the database access implementation; no ORM or schema ownership change is implied.

## Development

Requirements: Go 1.27.3, PostgreSQL 18, Redis 7 and golang-migrate.
Configuration: one `app.ini`, selected with `--config` (default: `docker/root/etc/templates/app.ini`).
Environment overrides use `BDS__SECTION__KEY`; no `app.env` is loaded.
Docker initializes `/data/bdspro/conf/app.ini` and `/data/bdspro/log/` using its template. Secrets must be injected externally in non-development deployments.

```sh
make migrateup
make sqlc
make swagger
make server     # API only
make worker     # background tasks and outbox delivery
go run . --version
go run . --help
go test ./...
go test -race ./...
```

Use `docker compose up --build` for PostgreSQL, Redis, Mailpit, web and worker. HTTP no longer starts the task processor; asynchronous email requires a running worker. In development the web runtime keeps the existing automatic migration behavior; run schema migrations before standalone workers in other environments.

## HTTP contract

Public success payloads are operation-specific. Public errors use a stable
code and backwards-compatible text:

```json
{"error":"invalid credentials","code":"auth.invalid_credentials"}
```

The HTTP boundary chooses what to expose; SQLC models are not public
contracts. Input, authentication, resource ownership and transaction rules
remain independent concerns.

**Core endpoints:** GET `/health`; POST `/users`, `/users/login`,
`/users/verify_email`, `/users/verify_email/resend`,
`/tokens/renew_access`, `/tokens/revoke`; POST `/users/password/reset/request`, `/users/password/reset`; GET `/users/password/reset`, `/users/verify_email`,
`/users/me`; GET/POST `/organizations`; PATCH `/organizations/:id`;
POST `/organizations/:id/members`.

**Optional provider endpoints:** POST `/users/phone/register/start`, `/users/phone/register/verify`, `/users/phone/login/start`, `/users/phone/login/verify`; POST `/users/me/google/start`,
`/users/me/phone/google/start`, `/users/me/phone/start`,
`/users/me/phone/verify`; GET `/users/google/login/start`,
`/users/google/callback`. Provider configuration determines which routes
are registered.

OpenAPI 3.0.3 is versioned at `doc/swagger/openapi.json`, with route coverage verified against native Gin registration. Swagger UI is available **only in development** at `/swagger/`
and `/swagger/openapi.json`. The running API currently serves these assets using Go `embed`.
`make swagger` restores the original `swag init` and `statik` generation
commands, producing Swagger 2.0 `doc/swagger/swagger.json` and
`doc/statik/statik.go`. The generator output is separate from the versioned
OpenAPI 3.0 `doc/swagger/openapi.json` used by the running Swagger UI.
CI checks the OpenAPI-to-Gin route coverage directly, without overwriting
the versioned API contract. The Statik output is generated, but the runtime
has not yet been switched away from Go `embed`.
Swagger does not imply an endpoint is enabled in every environment.

## Data, sessions and workers

All editable SQL belongs in `db/query`; generated SQLC files must never be
edited by hand. Use direct Queries for simple operations and a scoped
transaction for multi-write invariants. Keep PostgreSQL constraints as
the ultimate concurrency protection.

Access JWTs are stateless and short-lived. Revoking the stored refresh
session stops further renewal but does **not** invalidate an already-issued
access JWT instantly. Do not promise immediate device logout until that
contract has been implemented and verified.

Outbox delivers persisted job intents to Asynq with at-least-once semantics.
Tasks must tolerate duplicates; task types and Redis namespaces require
compatible migrations before renaming.

## Verification and limitations

CI provisions disposable PostgreSQL, checks SQLC generation, applies and
upgrades migrations, verifies Swagger assets, runs Go tests/race/vet/build.
Local integration tests require `BDS_TEST_MODE=disposable` and a dedicated
`BDS_TEST_DB_URL` referring to `bdspro_test`; never point write tests at
production or an existing application database.

A green CI run does not prove Google/SMS provider delivery, production
backup/restore, multi-dispatcher race handling or deployment rollback.
See [engineering source contract](docs/engineering/mindset-source-contract.md)
and [current architecture](docs/architecture/README.md).


## Three authentication channels
Email/password sign-up remains POST /users with a unique normalized email.
Google OAuth creates a passwordless account by verified Google subject and email;
it never generates or sends a password. A Google user can request a first
password using POST /users/password/reset/request, then use the one-time
verified-email link to set it with POST /users/password/reset. Existing refresh
sessions are revoked when a password is set or reset; already issued access
JWTs continue until expiry.

Phone signup uses POST /users/phone/register/start and /verify. Only verified
OTP creates a Party, Person, Account (email=NULL) and unique phone credential in
one PostgreSQL transaction. Login uses /users/phone/login/start and /verify.
Phone-only users do not need an email or password; linking an email later is
a separate verified-ownership workflow not yet implemented.

Phone OTP provider routes require a configured delivery channel. In development,
the local sender stores a one-time dev_code in the disposable database; use
Mailpit for password-reset and verification emails. Never expose dev_code in a
public response or production database. See docs/architecture/authentication.md.

## Authentication development guide

[Three-method authentication, OTP/Postman/Mailpit and password recovery](docs/operations/auth-three-methods.md).
