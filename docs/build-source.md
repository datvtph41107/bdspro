# Build BDSPro from source

## Requirements

Use the version declared in `go.mod` (currently Go 1.27.3).
Local services: PostgreSQL 18, Redis 7 and Mailpit for mail-flow testing.
Docker/Compose and Make are the supported development convenience tools.
`sqlc` is needed when changing SQL queries; the committed generated Go files
are checked by CI.

## Executable

```sh
make build
./bin/bdspro --help
./bin/bdspro --version
./bin/bdspro web --help
./bin/bdspro worker --help
./bin/bdspro migrate --help
```

`make build` writes **one local executable** to ignored `bin/bdspro`.
The production-image builder compiles its own executable at build time;
it does not copy `bin/` from the developer machine.

Gitea's current `make backend` produces a root executable `gitea`;
the fact that BDSPro uses `bin/bdspro` is a deliberate local output policy,
not an assertion that Gitea always uses `bin/`.
`.air/` is Air's ignored temporary state and is never checked in.

## Code and contract generation

```sh
make sqlc
make swagger-check
go build ./...
```

`db/query/*.sql` and `sqlc.yaml` generate `models/db/` Go query code.
The reviewed OpenAPI 3.0.3 file is `templates/swagger/openapi.json`.
`make swagger` now aliases `swagger-check` and verifies that every
documented path/method corresponds to a Gin route and vice versa.
**It does not regenerate schemas or operation details.** The earlier
`swag init` generated a separate Swagger 2 file not used by the running
API, and Statik generated another unused embed package; both were removed.
A genuine source-to-OpenAPI generator can be introduced after its canonical
input and reproducibility contract are decided.

See [API documentation](api/README.md) and [testing](testing.md).
