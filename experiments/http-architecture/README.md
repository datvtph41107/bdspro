# HTTP architecture spike — isolated from production

Status: **experimental**, not an architecture decision.

This nested Go module deliberately uses a separate `go.mod`. Neither Chi nor the
prototype Gin package is imported by BDSPro production source. C01/C02 on PR #238
is unchanged; this research branch starts at its verified HEAD.

## Hypotheses

1. Chi: native `http.Handler` composition avoids BDSPro's custom route-definition
   and mount compilation, with an extra routing dependency.
2. Gin: native groups and handlers avoid custom route compilation, but handler
   code explicitly depends on Gin.
3. ServeMux: standard-library patterns remove routing dependencies, at the
   cost of manual conventions for nested grouping.

## Equal contract

All three use `contract/contract.go` for API behavior. This isolates router
semantics, middleware placement, and path parameters; it does **not** compare
framework-native JSON binding or validation. Authentication is a toy bearer
token for deterministic tests, not production authentication.

- GET /api/v1/health → 200 `{"status":"ok"}`
- POST /api/v1/orgs/{org}/teams with Authorization: Bearer test-token
  and `{"name":"Backend"}` → 201 `{"org":"acme","name":"Backend"}`
- Unauthorized → 401; bad JSON → 400; validation → 422
- Unknown route → 404; method mismatch → 405
- NoSniff header on successful and unsuccessful responses
- Single strict JSON document, maximum 4096-byte body

## Reproducibility

From `experiments/http-architecture`:

```sh
go mod tidy
go test -race ./...
go vet ./...
go test -run '^$' -bench BenchmarkHealth -benchmem -count=5 ./...
```

Run `go test -run '^$' -bench BenchmarkHealth -benchmem -count=5 ./...`
against the **same machine, Go version, and harness** for comparable
microbenchmark numbers. Use `benchstat` when comparing distributions,
not one CI sample. This is not a replacement for production load testing.

BDSPro's existing custom Gin/Gitea hybrid baseline is tracked on PR #238;
do not claim direct performance parity between different endpoint contracts.

## Decision requirements

A canonical choice requires evidence on middleware composition, 404/405,
request identity, body limits, API compatibility, operational troubleshooting,
production integration cost, and p95/p99 under realistic request mixes.
Only then may an ADR be marked Accepted. A green test result is necessary,
not sufficient.
