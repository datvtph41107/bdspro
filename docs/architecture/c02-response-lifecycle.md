# C02.1 — Committed response contract and panic recovery

## Exact baseline and references

- BDSPro `dev@b0d3b53800a59961b1ff2248a23d739e3b228053`: `services/context.Base` wraps a `RequestSource` adapter for Gin; `routers.NormalRoutes` installs a custom recovery. Before this change, every panic attempted an additional JSON 500 even after a handler had already committed HTTP output.
- Gitea `services/context/base.go` exposes `Written()` and `WrittenStatus()` through its response writer. The feature is a response lifecycle contract, not a reason to import Chi or duplicate Gin response tracking.
- Gitea source: https://github.com/go-gitea/gitea/blob/main/services/context/base.go
- BDSPro source: https://github.com/datvtph41107/bdspro/blob/b0d3b53800a59961b1ff2248a23d739e3b228053/routers/init.go

## Owner and failure semantics

- `services/context.Base` owns the meaning of a **committed** response independent of transport; `services/context/gin.go` adapts Gin's `ResponseWriter.Written()` and `Status()`. `WrittenStatus()` is 0 until the response has been emitted, even if a status code was selected.
- `routers/init.go` owns the single runtime recovery policy. A panic before write produces one sanitized 500 JSON; a panic after write logs the panic type, aborts remaining handlers and does not append an invalid second JSON envelope or claim it can rewrite status.
- No new response middleware package, global status provider, duplicate writer, error taxonomy, or service dependency. Request logging still records the actual committed response.
- **Limit:** an already-committed HTTP 202/200 cannot be turned into 500. Clients may have observed success before panic, so this prevents corrupt output but does not guarantee rollback or detect partial side effects. Transport-level streaming/hijacked responses require separate work.

## Executable acceptance

- `TestBaseWrittenStatusMatchesGinResponse`: distinguishes response selection from actual emission.
- `TestRecoveryBeforeResponseWritesOnePublicError`: real Gin recovery produces one redacted 500 JSON.
- `TestRecoveryAfterCommittedResponseDoesNotAppendJSON`: panic after a written 202 leaves the original JSON intact and does not leak raw panic details.
- Existing response/error contracts, route parity, SQLC, PostgreSQL and worker tests must continue passing in CI.

## Follow-on programming-model gates

1. **C03 / modules/web and routing:** preserve Gin middleware semantics; identify a consumer before extending `Router[T]` or adding `BeforeRouting`/`AfterRouting`. Convert a complete Organization/Team endpoint from legacy receiver to a typed function with HTTP/JSON parity tests, not a directory rename.
2. **C04 / auth and organization guards:** keep verified first-party actor, JWT session admission, OAuth authorization-server scopes, organization membership and in-transaction SQLC owner predicates separate.
3. **C05 / installed startup:** `cmd` creates and closes DB/Redis; `routers.InitWebInstalled` composes HTTP dependencies and routes. Gitea `mustInit` is fail-fast, not dependency injection or rollback. Capture failure and cleanup evidence before re-splitting constructors.
4. **C06 / domain and models:** `models/db.WithTx` intentionally refuses nested transactions until abort/rollback semantics are designed and proven. Do not blindly copy Gitea XORM's engine-in-context contract.
5. **C07 / cleanup:** only remove legacy receivers/entrypoints after consumers and parity tests have migrated; do not introduce empty files/packages to match Gitea.

This is the first narrow runtime correction of the C02 programming-model work, **not** acceptance of C02–C07 as complete.
