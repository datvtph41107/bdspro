# Test and review contract

## Developer commands

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
make swagger-check
```

Test commands alone are not a proof of concurrency safety or integration
correctness. The CI workflow `.github/workflows/test.yml` additionally
starts PostgreSQL 18 and Redis, tests an upgraded legacy database, checks
SQLC output, validates OpenAPI path/method coverage, and performs an HTTP
smoke test. Docker image build is a separate check.

## What the checks mean

| Check | Guarantee | Does not guarantee |
| --- | --- | --- |
| SQLC generate diff | committed query Go matches SQL | database semantics |
| OpenAPI coverage | documented routes and methods match Gin | every response schema is exact |
| Unit tests | the exercised input/output cases | all edge cases |
| Race detector | no observed races along exercised paths | absence of every race |
| Migration upgrade | tested fixtures preserve known records | all historical production data |
| HTTP smoke | startup, health and a few HTTP responses | full login/OTP/worker lifecycle |
| Docker build | image can be built | production hardening or crash recovery |

Tests that share a PostgreSQL database must create distinct fixtures or
isolate state; assertions on total global row counts are not reliable under
parallel package execution.

## Change review sequence

1. Identify the current owner from source and failing contract.
2. State competing explanations for the behavior.
3. Run a discriminating test before changing more folders.
4. Apply the smallest coherent change that protects the boundary.
5. Check HTTP/API schema parity, SQL migrations and asynchronous effects.
6. Record limitations and unexpected costs in the PR.

References: [Gitea testing guidance](https://github.com/go-gitea/gitea/blob/main/docs/testing.md)
and [Gitea refactoring guidance](https://github.com/go-gitea/gitea/blob/main/docs/guidelines-refactoring.md).
