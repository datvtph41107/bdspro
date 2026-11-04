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


## Transaction results and composition (PR #127)

Gitea uses `models/db.WithTx2[T]` to make transaction-scoped work return a result. BDSPro's SQLC equivalent is `db.WithTxResult[T](ctx, transactioner, func(q *db.Queries) (T, error))`; `SQLStore.ExecTx` remains the sole owner of begin, commit and rollback.

- The callback must use the provided transaction-bound `q` for **all** participating SQL writes. Helpers such as `db.CreatePersonWithParty(ctx, q)` are reusable transaction-scoped operations.
- Do **not** call `store.ExecTx` or ordinary pool-backed `store.Queries` from inside the callback. That would start a second, independently committed transaction and invalidate the outer atomicity guarantee. We intentionally do **not** simulate Gitea's XORM context engine or implicit nested transactions.
- `WithTxResult` returns zero `T` on callback failure **and** failure during commit. Its returned value is not a success signal until the commit succeeds.
- Use bare `ExecTx` for operations without a result, and for security flows that **deliberately commit failure counters** before returning an OTP error. These must not be refactored into rollback-on-error flows.
- Enqueue delivery intent in the same transaction as the durable model state; deliver email/notifications asynchronously after commit.
- Tests must include rollback after a statement has succeeded, a callback failure, and a commit failure. A green test with mocked SQL is not proof of PostgreSQL atomicity.

This is a selective adaptation of Gitea's result-handling style, **not** its XORM session lifecycle or context-implicit nested transaction behavior.


## Concurrent owner authorization (PR #128)

PostgreSQL's default READ COMMITTED isolation allows a plain `EXISTS` authorization predicate to observe a previously committed owner row while a concurrent transaction is revoking that row. The write can then commit even though the revocation completed first.

`db/query/organization.sql` now locks the **exact active owner membership row** with `FOR SHARE OF owner` or `FOR SHARE OF m` in two state-changing operations: `AddOrganizationMember` and `UpdateOrganization`. If an owner revocation has already obtained the membership row lock, these writes wait and re-check the latest row version; if it is no longer active, the operation is denied.

This applies only to writes, avoiding row locks for listing and reading. The SQLC query parameters and result shapes do not change, and no extra Go authorization lookup or wrapper is introduced. The lock protects the owner authorization invariant through the statement; it does not imply that other unreviewed team or invitation mutations have been race-hardened.

Integration coverage must prove that a pending owner revocation blocks the attempted mutation, that the post-commit mutation is denied, and that authorized writes continue working when there is no conflicting revocation. A simple green sequential test is not sufficient evidence for this property.


## Team membership write authorization (PR #129)

Owner-controlled team writes use the **active owner membership** as their authority: create/rename team, add/remove team user, create/revoke team invitation. Their SQLC statements now lock that membership row with `FOR SHARE OF m` inside the existing `EXISTS` predicate. No separate Go check, duplicate repository or new transaction engine is introduced.

The recipient's **active organization membership** is additionally locked when adding a team user, creating a team invitation, and inserting a verified invited team user. Those writes cannot silently commit against a membership that was already being ended by a concurrent transaction.

Read paths are not locked. `ConsumeTeamInvite` still has a distinct multi-table `UPDATE ... FROM` shape and is deliberately not claimed as completely race-safe without its own concurrency test. The acceptance test must prove the lock dependency and the denied write after the revocation commits. No public errors or API endpoint contracts change.

This mirrors the responsibility boundary Gitea uses for persisted membership permissions while keeping BDSPro's SQLC/pgx query and schema canonical.
