# Models: canonical ownership and error contracts

This document describes **existing** BDSPro behavior and the implementation boundaries introduced in PR #125. It does not turn Gitea's XORM engine into a requirement for BDSPro.

## One source of truth per responsibility

| Owner | Source of truth | Not owned |
| --- | --- | --- |
| `db/migration/` | PostgreSQL constraints and durable storage shape | Public API errors |
| `db/query/` | Atomic SQL predicates, authorization conditions and transaction queries | HTTP response shape |
| `models/db/sqlc/` | Generated query implementations (read-only generated output) | Handwritten business decisions |
| `models/db/` | pgx pool lifecycle, one `Transactioner` contract, SQLC store, bounded SQL pagination, reusable Party/Person persistence primitive | HTTP, provider/OAuth verification |
| `models/account/` | Account creation, verified email linking and single-use email proof state transitions | Public response serialization |
| `models/auth/` | Persistent credential, session, OTP, password-reset and external identity binding invariants | Token verification by external provider, HTTP codes |
| `models/organization/` | Organization, team, member, invitation and role behavior; exact database-constraint interpretation | HTTP protocol and JSON projection |
| `services/` | Orchestration, verified external input, cross-model operations and display conversion | SQLC-generated ownership |
| `routers/api/response/` | Existing safe `{code,error}` contract and HTTP status mapping | Database invariants |

### Add or change a model operation

1. Specify the durable invariant and its authoritative SQL predicate, then update `db/query/` and migration if needed. Never edit `models/db/sqlc/*.gen.go` directly.
2. Implement the state transition in the corresponding `models/<area>/<operation>.go`. Create a transaction at the owner boundary, not in an HTTP handler. Account registration via password, phone or OIDC must all create the same Party+Person pair through `db.CreatePersonWithParty` **inside their transactions**.
3. Expose a domain-specific error from the package's `error.go` when a known SQL constraint or missing row matches the operation's invariant. Use `util.WithCause` to retain the driver cause without printing it in `Error()`. Unknown SQL errors remain unknown and must become safe HTTP 500 responses. Never classify by SQLSTATE alone.
4. The service coordinates credentials, external verification, queue and notifications. It must not duplicate SQL authorization, turn a failed proof into a success, or return a partially created entity after a rolled-back transaction.
5. The router validates request format and invokes `response.Error`, which alone maps known model errors to the existing HTTP status/code/message contract. Do not import `net/http`, Gin or response packages from models.
6. Test the positive path, denied/no-row path, relevant exact constraint, rollback, stale proof/replay and the unchanged public JSON contract. CI also enforces package import ownership and generated SQLC reproducibility.

### Why Gitea is a reference, not the runtime engine

Gitea's `models/db` and domain model packages own entity persistence and typed errors, while API contexts decide HTTP behavior. BDSPro adopts that responsibility boundary with SQLC and pgx, not Gitea's global XORM engine, `init()` registration, or a new ORM adapter. There is no parallel database access abstraction.

### SQL pagination

All team and team-member SQLC list operations use `db.ListOptions.GetSkipTake32`. It rejects `ListAll`, zero/negative page or size, oversized take and offset multiplication overflow before narrowing to SQLC's 32-bit arguments. The organization model chooses the existing `ErrTeamPaginationInvalid` public contract.

### Audit and acceptance

Model changes are not complete merely when `go test ./...` is green. Acceptance requires PostgreSQL integration on a disposable DB, race/vet/build, schema/query consistency, HTTP error regression tests and source SHA recorded in the PR.

## Public error contract (Gitea-inspired)

Explicit domain failures use `util.WithPublic(category, modelMessage, clientCode, clientMessage)` in the owning `models/<area>/error.go`. `util.ErrorForClient` recognizes this concrete declaration through wrapped error chains. Ordinary `util.WithKind`, unknown SQL/provider errors and raw categories are **not** public. `routers/api/response` has exactly one generic category-to-status mapping, without per-model switches or `models/` imports. Existing `{code,error}` values are regression-tested. Do not put driver errors or user-supplied secrets in `clientMessage`.

`models/db.Store` remains the single production SQLC store contract across account, authentication and organization. Do not introduce a second store implementation merely to resemble Gitea's XORM engine. A narrow consumer interface is appropriate only when an operation's test seam or alternate dependency genuinely needs it; it must not become a parallel database source of truth.
