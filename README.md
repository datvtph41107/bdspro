# BDSPro Modular Monolith

BDSPro contains HTTP/runtime infrastructure, session-based authentication, email and phone verification, Google authentication, and Organization/Team APIs. The frontend is a development client; it does not implement every backend account-management screen.

## Source organization

- `cmd/` — CLI entry points and configuration preflight.
- `modules/` — technical subsystems: settings, logging, cryptography, connections, routing and provider transports.
- `models/db/` — SQLC execution, PostgreSQL engine, transactions and generated bindings; domain model packages expose persistence operations and types.
- `db/migration/`, `db/query/` — schema history and SQLC query definitions.
- `services/context/` — HTTP context, response/error and pagination contracts.
- `services/auth/` — authentication, challenge/session lifecycle and credential orchestration.
- `services/phonedelivery/` — enabled-channel policy and OTP delivery orchestration; HTTP provider adapters are in `modules/phonedelivery/`.
- `services/org/` — Organization membership, owner rules and Team transaction orchestration.
- `routers/` — HTTP composition, authentication/permission checks and request handlers.
- `templates/swagger/` — development Swagger UI and OpenAPI contract.

The concrete delivery boundaries, evidence and limitations are documented in [Phone delivery architecture](docs/architecture/phone-delivery.md).

## Run

Copy `custom/conf/app.example.ini` to `custom/conf/app.ini` and adjust PostgreSQL, Redis and authentication settings for your environment.

```sh
go run . --config custom/conf/app.ini config check
make server
```

The HTTP probe is `GET /api/v1/health`. Team endpoints require a session-bound access JWT and Organization membership; writes require owner membership and recheck authorization in PostgreSQL.

Database migrations and SQLC verification:

```sh
make migrateup
make sqlc-check
```

For local OTP inboxes and SMS/Zalo/Telegram configuration, see [Phone delivery setup](docs/phone-delivery.md).

For local breakpoint debugging, log correlation and profiling, see [Developer debugging](docs/developer-debugging.md) and [Logging](docs/logging.md).

## Verification

```sh
go test ./...
go vet ./...
make swagger-check
```

Frontend checks run from `frontend/`: `npm test` and `npm run build`.

With local PostgreSQL, Redis and Mailpit running, `python3 scripts/test-phone-delivery.py` exercises phone OTP through a temporary backend port. It creates expiring challenges and four inbox messages without completing accounts or contacting paid providers. See the architecture document for what these checks do and do not establish.
