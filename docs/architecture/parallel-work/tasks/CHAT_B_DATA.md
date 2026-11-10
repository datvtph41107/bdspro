# Chat B — Transactions, Models, Selected Domain Services (P04, scoped P05)

**Role:** sole owner of DB/SQLC/model transaction and domain persistence pattern. **Branch:** `work/gitea-b-data-20261110` based on `dev`. The coordinator can initialize the branch; always verify its base HEAD.

## Read source before designing
- Shared [README](../README.md), [ownership](../OWNERSHIP_AND_CONTRACTS.md), [trust](../TRUST_AND_VERIFICATION.md), [integration](../INTEGRATION_PROTOCOL.md), [skill](../SKILL.md).
- Gitea `models/db/context.go`, transaction executors, domain model files and their service call sites; pin upstream Gitea commit.
- BDSPro `models/db/context.go`, SQLC-generated folder, actor/session models, `services/user/{user,update,settings}.go` if present, and all consumers of `WithStore` / `WithTx` / `WithTx2` / `TxQueries`.
- `dev` is the input baseline. Do **not** cherry-pick the unfinished HTTP PR #218 and treat it as a stable DB dependency.

## Owned scope
- **Allowed:** `models/**` (including DB context/SQLC usage, but do not hand-edit generated SQLC files), narrowly scoped `services/user/**` only when the PR names the affected file and no other lane owns it.
- **Not allowed:** `routers/**`, `services/context/**`, `modules/web/**`, `services/auth/**`, cross-domain migrations without an approved contract RFC, `go.mod`/CI outside integration lease.

## Milestones
- B1: perform a real **transaction call-site inventory**, map Tx/Queries and nested rollback-only behavior against Gitea; report which current APIs are needed or redundant. No blind rewrite.
- B2: canonicalize one transaction mechanism with SQLC/pgx retained; ensure all DB calls inside one transaction use that transaction-bound executor; verify nested failure poisons outer commit (as current behavior intends).
- B3: preserve verified actor/session authorization, rollback on revoked session and concurrent writes. Do not weaken authorization for architecture resemblance.
- B4: select limited P05 domains with existing duplicate models/services responsibility; improve domain error and method organization. No bulk renaming; no broad auth/OAuth business changes.
- B5: publish **stable DB capability contract** needed by Chat D, with compile examples. Pass Go 1.27.3 test/race/DB integration gates and handoff.

## Acceptance and failure cases
- `go test` relevant packages and `go test ./...` final HEAD; DB disposable integration, SQLC generated diff, migrations fresh schema.
- nested transaction swallowed error, rollback/no partial commit, session revoked between validation and write, concurrency and permission checks, missing store, request cancellation and meaningful domain errors.
- claims must contain before/after source and Gitea references; benchmark only if measured.
- Empty helper extraction, an extra transaction abstraction or passing `Store` everywhere without need is rejected.

## Deliverable
One draft PR to `dev`, affected paths only, no cross-owner files, complete handoff and stable interfaces for A/D. Never merge yourself.

## START NEW CHAT — copy the following message

> Bạn là **Chat B — P04/P05 BDSPro Data & Models**, làm việc độc lập với Chat A/C trên repository https://github.com/datvtph41107/bdspro ; Gitea https://github.com/go-gitea/gitea . Branch làm việc: `work/gitea-b-data-20261110` khởi nguồn từ `dev`, không dùng PR #218 làm base. Bắt buộc đọc `docs/architecture/parallel-work/{README,OWNERSHIP_AND_CONTRACTS,TRUST_AND_VERIFICATION,INTEGRATION_PROTOCOL,SKILL}.md` và `tasks/CHAT_B_DATA.md` trên nhánh `docs/gitea-parallel-harness-20261110`. Chỉ sở hữu `models/**` và những file `services/user/**` được cho phép. Bắt đầu bằng phân tích chính xác `models/db/context.go`, `WithTx`/`WithTx2`/`TxQueries`, nested rollback-only, session authority, SQLC/pgx rồi thực thi patch nhỏ có giá trị; tuyệt đối không sửa Router/context HTTP hoặc Auth/OAuth route. Đối chiếu code pattern Gitea thực tế, ghi exact SHA, test/CI + phản ví dụ bảo mật, tạo PR draft và handoff. Không merge dev.
