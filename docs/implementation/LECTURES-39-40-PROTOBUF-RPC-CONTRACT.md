# Lecture #39–40 — RPC intent and canonical Protobuf contract

**Date:** 2026-10-21. **Phase:** G06.1 (Protobuf schema + Go message generation). **No cloud/public deployment.**

## 1. Why learn gRPC when Gin already works?

The existing Gin HTTP/JSON API is useful and continues to serve its current routes.
It has a canonical OpenAPI description (`api/openapi.json`) and is not replaced.
A second communication boundary becomes useful when client/service integrations
need a **versioned, strongly typed contract**, generated bindings, deadline and
metadata conventions, or gRPC features such as streaming.

**Protocol distinction:**
- **Protobuf**: schema language + efficient typed serialization; not an HTTP
  server or a database. `proto/bdspro/v1/account.proto` owns this contract.
- **Generated `pb/`**: compile-time Go message types created by `buf generate`.
  Do not edit generated code. It does not implement authentication or SQL.
- **gRPC**: RPC transport / service dispatch (typically HTTP/2).
  `gapi/` will implement application adapters when Lecture #41–42 begins.
- **Gin**: existing REST HTTP/JSON transport that remains enabled.
- **gRPC-Gateway**: optional future HTTP-to-gRPC adapter; it does not transform
  arbitrary Gin handlers automatically.
- **OpenAPI / Swagger UI**: describes/tests HTTP contract and can exist without
  Protobuf or gRPC. It already exists in G03.1.

## 2. First RPC boundary: current Account and live Workspaces

`proto/bdspro/v1/account.proto` defines:

```proto
service AccountService {
  rpc GetCurrentAccount(GetCurrentAccountRequest)
      returns (GetCurrentAccountResponse);
  rpc ListWorkspaces(ListWorkspacesRequest)
      returns (ListWorkspacesResponse);
}
```

**Key rule:** both requests are *empty*. Client-supplied `account_id`,
`person_id`, tenant IDs or role claims must not decide the caller identity.
The future gRPC interceptor will validate `authorization: Bearer <JWT>`, then
call the **same PostgreSQL-backed** `auth.AccountFromSessionHash` as Gin.
`ListWorkspaces` must query the same live `organization.ListWorkspaces`
projection, with permissions enforced in PostgreSQL. Merely decoding the JWT
signature is not authorization.

Timestamp data uses `google.protobuf.Timestamp`, never an untyped
`string` when representing membership join time. Identifiers remain int64.
The response is a read model; the phone/email login mechanism and Account/Party
write invariants are untouched.

## 3. Package/version ownership

```text
proto/bdspro/v1/account.proto     <-- durable human-edited source
buf.yaml                         <-- Protobuf lint + breaking policy (FILE)
buf.gen.yaml                     <-- Go code generation policy
pb/bdspro/v1/account.pb.go        <-- generated Go messages
api/                              <-- existing Gin HTTP API
token/jwt_maker.go                <-- same JWT verification primitive
auth/, organization/              <-- business/application owners
db/sqlc/                          <-- PostgreSQL truth
gapi/                             <-- NOT implemented in #39–40
```

Keep `package bdspro.v1`; `go_package` points to
`github.com/datvtph41107/bdspro/pb/bdspro/v1`. Never rename fields once
clients depend on them without explicit backward-compatible migration.
Field numbers are durable: do not reuse removed numbers, use `reserved`
where required. Buf lint uses STANDARD and breaking uses FILE.

There is no second Account table, session store, role cache, or Protobuf-owned
DB model.

## 4. Local workflow and acceptance

Prerequisites for this phase: `buf` CLI v1.59.0 and
`protoc-gen-go` v1.36.10 available on PATH. CI installs them with pinned versions.

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
export PATH="$(go env GOPATH)/bin:$PATH"
buf lint
buf build
make proto
git diff --exit-code -- pb
go test ./...
```

Use `buf generate` to reproduce `account.pb.go`; that generated file
MUST be in source control. CI fails if generation changes it or adds untracked
files. No new runtime dependency is required to simply define messages.

**Gate:** CI lint+build+generation, code-gen diff clean, `go vet`,
full Go tests/race, migration and local Compose smoke. These prove the typed
contract is reproducible; they do **not** prove that a gRPC server is running.

## 5. Next lecture

**#41:** Generate `*_grpc.pb.go` using pinned
`protoc-gen-go-grpc`, implement `gapi.Server` with a live gRPC listener
bound to loopback by default, plus end-to-end RPC tests with actual Session
JWT and PostgreSQL membership authorization. **#42:** auth interceptors,
status mapping, revocation, deadlines, graceful shutdown. Only then claim
gRPC API functionality. **#43–45:** selected gRPC-Gateway endpoints and
OpenAPI parity without disturbing current Gin handlers.

The learning objective is to distinguish **protocol contract** from
**business implementation**, not to create directories called `pb` and
`gapi` for appearance.
