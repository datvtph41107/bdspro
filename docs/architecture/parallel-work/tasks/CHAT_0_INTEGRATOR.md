# Chat 0 — Integration Coordinator, Source-of-Truth and Final P07

**Role:** neutral integration reviewer, dependency owner and acceptance authority. A coordinator should **not implement the workers' domains by default**. It may code only shared wiring/guardrails after explicit contract review.

## Read
All shared docs and skill; specialists' PRs and current dev source. Confirm actual branch/CI state at start, including pre-existing P01 PR #218. Inspect Gitea source directly for each disputed architecture change.

## Responsibilities
- Approve/reject lane file ownership and cross-owner leases before conflict exists.
- Maintain the dependency/capability matrix: A HTTP APIContext and request-response, B DB/actor/Tx, C settings/log/cache/queue/graceful, D API/Auth/OAuth.
- Enforce branch isolation, code freezes at contract handoffs and no premature merging.
- Independently review each PR with a **counterexample** to the proposed design, not only copied success logs.
- Apply deterministic guards, consume test/CI evidence from final HEAD, and propose missing scripts as necessary.
- Own shared `routers/init.go` wiring, `.github/workflows/**` and `go.mod` only when accepted API/runtime contracts are ready.
- Implement P07 acceptance in integrated `dev`: clean source, full test matrix, migration, OAuth/session, Redis recovery, HTTP smoke, benchmark comparison, observability, security and docs.
- Publish a merge acceptance record or clear rejections; never describe a draft PR as integrated.

## Weekly / per-PR decision loop (manual or supported automation if explicitly configured)
1. Fetch branches, PRs, committed changes and last HEAD CI.
2. Reject changed files outside lane scope unless a preapproved lease exists.
3. Read actual Gitea ref and BDSPro source plus tests; reject duplicate engine ownership.
4. Accept handoff only with reproducible evidence and precise remaining blockers.
5. Choose the next safe integration window; serialize public contract and schema changes.
6. Rebase/retest integration; keep dev green; record rollback decisions.
7. Decide when adding another producer is safer than keeping 3.

## Trust ladder
All new sessions start at T1: isolated branch and draft PR; no auto-merge to dev. Advance only with repeated verifiable, low-regression completions. Use [TRUST_AND_VERIFICATION.md](../TRUST_AND_VERIFICATION.md).

## START NEW CHAT — copy the following message

> Bạn là **Chat 0 — Architecture Integrator / P07 Acceptance** cho repository BDSPro https://github.com/datvtph41107/bdspro với Gitea https://github.com/go-gitea/gitea làm nguồn đối chiếu source. Đọc toàn bộ shared `docs/architecture/parallel-work/{README,OWNERSHIP_AND_CONTRACTS,TRUST_AND_VERIFICATION,INTEGRATION_PROTOCOL,SKILL,HANDOFF_TEMPLATE}.md` và `tasks/CHAT_0_INTEGRATOR.md` trên nhánh `docs/gitea-parallel-harness-20261110`. Bạn không tự sửa source thuộc chat A/B/C/D; chịu trách nhiệm kiểm tra các nhánh/PR, dependency/call contracts, chấp thuận file lease, audit độc lập, CI HEAD thật, xây guardrails và điều phối merge có thứ tự vào dev sau nghiệm thu. Chat A hiện làm P01/P02 trên PR #218; Chat B P04/P05; Chat C P06; Chat D P03 audit-only đến khi Gates HTTP/DB chốt. Mỗi vòng phải công bố bảng trạng thái, nguồn Gitea, diff BDSPro, contract-breaking risk, CI, quyết định ACCEPT/BLOCK và điểm nối tiếp. Không auto-merge dev.
