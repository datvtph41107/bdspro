# Lecture #41 — BDSPro gRPC local server (Gin stays operational)

**Scope:** actual unary RPC transport and security for two existing read
operations: `GetCurrentAccount` and `ListWorkspaces`. No cloud or deployment.

## Why a new gRPC adapter?

- `proto/bdspro/v1/*.proto`: versioned typed RPC contract.
- `pb/bdspro/v1/`: generated messages and service interface.
- `gapi/server.go`: **only transport mapping**. Verify Bearer JWT, check
  PostgreSQL Session, then call existing Auth/Organization service.
- `api/`: existing HTTP/JSON Gin API and local Swagger UI remain live.
- `organization.ListWorkspaces`: same live SQL membership projection for
  both HTTP and RPC. No new User/Account table and no role JWT claim.

```text
HTTP :8080  --> Gin /auth/me, /auth/workspaces ----+
                                                    +-> same Auth/Organization -> PostgreSQL
gRPC :9090  --> gapi AccountService ----------------+
```

## Local operation

The listener **must stay on loopback**, because this lecture intentionally
uses plaintext HTTP/2 gRPC (without TLS). It is not a public API and should
not be configured for `0.0.0.0` or a WSL/host external IP.

`app.env` now contains `GRPC_SERVER_ADDRESS=127.0.0.1:9090`.
`TOKEN_SYMMETRIC_KEY` must match the Gin JWT signer.

Terminal A (REST + Swagger, existing behavior):
```bash
make server
```

Terminal B (gRPC; same DB/ENV file):
```bash
make grpc
```

No external deployment or extra DB migrations are needed.

Example using the optional local `grpcurl` CLI. First obtain a JWT from an
existing authenticated login request (see Gin Swagger at `/docs`).

```bash
grpcurl -plaintext -import-path proto -proto bdspro/v1/service_bdspro.proto \
  -H "authorization: Bearer $JWT" -d '{}' \
  127.0.0.1:9090 bdspro.v1.AccountService/GetCurrentAccount

grpcurl -plaintext -import-path proto -proto bdspro/v1/service_bdspro.proto \
  -H "authorization: Bearer $JWT" -d '{}' \
  127.0.0.1:9090 bdspro.v1.AccountService/ListWorkspaces
```

Expected first response contains `accountId` and nested `user.accountId`.
An Account with no memberships returns `workspaces: []` (or an omitted
empty repeated field in proto JSON rendering).

JWT verification is not sufficient: `gapi` checks Session status in
PostgreSQL for **every RPC**. Logout/revocation invalidates both Gin and gRPC
immediately. Missing/invalid/replayed Authorization metadata is rejected.
Neither request accepts an actor ID, tenant ID or role provided by caller.

## Security and error semantics

- Authentication: gRPC metadata `authorization: Bearer <JWT>`, exact one
  value; signer HS256/issuer/audience/expiry checks in `token.JWTMaker`,
  then `auth.AccountFromSessionHash` on PostgreSQL and subject equality.
- `Unauthenticated`: bad JWT, no active session, subject mismatch.
- `Unavailable`: DB/dependency errors, without exposing PG details.
- `Canceled`/`DeadlineExceeded`: request context cancellation.
- No transparent cookie fallback, no public gRPC listener, no duplicated
  authorization rules.
- Source permissions and organization mutation RPCs are **not added** here.
  Future RPCs must always reuse the owning application service.
- Production gRPC needs TLS and an explicit trust model; deliberately out
  of scope while the user develops locally.

## Acceptance

- `gapi/server_test.go`: actual protobuf-over-gRPC bufconn calls, valid
  JWT, wrong signature/subject, duplicate headers, revoke, error code
  mapping and response projection.
- `integration/grpc_integration_test.go`: real PostgreSQL Account Session
  and Organization membership used by **both Gin and gRPC**; logout revokes
  the identical JWT on both transports.
- `make grpc` starts a real loopback listener at configured port. SIGINT
  and SIGTERM initiate bounded graceful shutdown.
- CI must pass Go tests/race, SQLC/proto regeneration, migrations and
  Docker Compose. Gin remains intact.
- gRPC-Gateway HTTP mapping / OpenAPI v2 runtime happens at Lecture #43–45,
  not implicitly just because the annotation exists.
