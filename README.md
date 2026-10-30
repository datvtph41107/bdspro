# BDSPro

Go/Gin REST monolith. PostgreSQL 18 with SQLC/pgxpool, Redis and Asynq.
One entry point (`main.go`), no gRPC, gRPC-Gateway or custom route registry.

## Source ownership

```text
main.go
internal/
  api/
    server.go                   Gin engine and HTTP lifecycle
    routes.go                   Native Gin route registration
    handler/                    HTTP input and explicitly public output contracts
    middleware/                 Authentication, admission and rate limits
    response/                   Public errors and status mapping
  service/                      Business operations; does not depend on Gin
  database/                     SQLC generated files and handwritten transactions
  errors/                       Application error definitions and classifications
  config/                       Runtime settings
  token/                        JWT maker and token payload
  worker/                       Redis/Asynq runtime and outbox dispatch
  jobs/                         Background business operations
  mail/                         SMTP delivery
  notification/                 Phone delivery adapters
  util/                         Logger and small independent functions
db/
  migration/                    Append-only PostgreSQL migrations
  query/                        Editable canonical SQL queries
doc/
  swagger/                      Versioned OpenAPI JSON + Swagger UI page
docs/                            Engineering, operation and historical decisions
```

All Go code under `internal/` stays private to this module. No `pkg/`
library is needed for the current executable.

## Local development — one supported path

Prerequisites: Go 1.27.3, Docker with **Compose v2**, and Make. The local
Compose project owns PostgreSQL 18, Redis 7 and Mailpit. This path uses
the local development database `bdspro`, not the disposable test database.

```sh
make help
make dev-up        # compose up --wait: PostgreSQL/Redis healthy, Mailpit running
make dev           # run Go on host, connect to Compose services
# in a second terminal:
curl -i http://127.0.0.1:8080/health
curl -i http://127.0.0.1:8080/swagger/openapi.json
# Mailpit inbox: http://127.0.0.1:8025
make check         # before opening a PR
make dev-down      # stops dependencies; preserves named database volumes
```

`make dev` uses explicitly local database/Redis addresses and loopback HTTP.
It sends development email through Mailpit at `127.0.0.1:1025` with no SMTP
authentication. It uses `app.env` for the remaining development settings.
`main.go` automatically applies migrations in `development` only.
**Do not connect the development process to application/production data.**
Both `make dev-up` and the app can fail if ports 5432, 6379, 8025, 1025
or 8080 are already in use.

`make server` is the raw `go run .` entry point: unlike `make dev`, it does
not force local Mailpit/DB overrides and relies on `app.env`/OS settings.
`make postgres` and `make redis` are Compose-specific convenience commands;
Compose initially creates `bdspro`, so `make createdb` is not needed on first
startup. `make dev-down` uses `docker compose stop`, not `down -v`.

For changes to SQL or migrations, install **SQLC v1.30.0** and
`golang-migrate v4.18.3` separately. Run `make sqlc` after changing
`db/query/*.sql`; generated `internal/database/*.sql.go` files must be
committed with their query sources. `make migrateup` uses `DB_URL` and
must only point to an intended database. `make migratedown` and
`make migratedown1` require `CONFIRM_MIGRATION_DOWN=yes` as an explicit
safety prompt, **not** as proof that the target database is disposable.

`make check` checks formatting without modifying source, verifies
OpenAPI-to-Gin route coverage (`make swagger`), runs Go tests, race detector,
vet and build. CI additionally checks SQLC drift and migrations using a
dedicated disposable PostgreSQL database. Local integration tests run only
with `BDS_TEST_MODE=disposable` and `BDS_TEST_DB_URL` targeting
`bdspro_test` with schema already migrated; ordinary `make check`
otherwise skips those integration cases.

Outside development, migrations run in a controlled deployment step before
starting the API, secrets must be environment-specific, and neither
`make dev` nor the sample settings are production configuration.

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
and `/swagger/openapi.json`. The UI files are embedded using Go `embed`; no external CDN or Statik generator is needed.
`make swagger` runs the OpenAPI-to-Gin route coverage test. API schemas remain
versioned in `doc/swagger/openapi.json`; CI checks path/method coverage and
runtime asset serving, not automatic generation of every response schema.
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
