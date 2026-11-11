# Chat D — API Contracts, Authentication, Authorization, OAuth2 (P03)

**Initial state: READ-ONLY SPEC / TEST DESIGN, not an active production-code lane.**
Opening a fourth chat now is possible **for source analysis only**. P03 source implementation is gated by HTTP and DB interface contracts.

## Read
Shared docs and skill, then Gitea `routers/api/v1/api.go`, `services/context/api.go`, Gitea Auth/API source, BDSPro `routers/api/v1/api.go`, `routers/web/oauth/**`, `services/auth/**` and current `models/auth/**`. Identify Gitea conventions and Laravel DRX functional/endpoint semantics where BDSPro has adopted them.

## Safe tasks before Gate H + Gate D
- Build canonical endpoint inventory (method, URL, middleware ordering, public error, session/authorization, OAuth redirect/cookie/CSRF, retry/replay behavior).
- Contrast current BDSPro routes with proposed Gitea-style functions and code ownership; distinguish code-style changes from business changes.
- Draft tests and a migration manifest listing exact files, consumer imports, needed `APIContext` and DB actor/Tx APIs.
- Produce an RFC for any signature/DB/route shape change; no production code edits while A is migrating `Router[T]`.

## After integration gates
- Create `work/gitea-d-api-auth-20261110` from fresh integrated `dev`.
- Own precisely approved `routers/api/v1/**`, `routers/web/oauth/**`, `services/auth/**`, `modules/auth/**`. Do not edit `modules/web`, `services/context` or `models/**` without explicit lease.
- Migrate HTTP routes/guards/action functions, errors, Authentication/Authorization/OAuth2 semantics and tests coherently without changing existing public API unintentionally. Preserve OAuth `/oauth/token`, `/oauth/authorize` and the canonical OAuth data contracts as agreed.
- Reverify session revocation boundary: internal sessions versus authorization codes versus OAuth access/refresh tokens remain deliberately separate unless a reviewed security change specifies otherwise.
- Full API/e2e/CI/route inventory evidence and a reviewable PR.

## START NEW CHAT — copy the following message

> Bạn là **Chat D — P03 API/Auth/AuthZ/OAuth2**, trước mắt **READ-ONLY audit + contract/spec**, chưa thực hiện source changes. BDSPro https://github.com/datvtph41107/bdspro ; Gitea https://github.com/go-gitea/gitea . Đọc `docs/architecture/parallel-work/{README,OWNERSHIP_AND_CONTRACTS,TRUST_AND_VERIFICATION,INTEGRATION_PROTOCOL,SKILL}.md` và `tasks/CHAT_D_API_AUTH.md` trên nhánh `docs/gitea-parallel-harness-20261110`. Khi `Router/APIContext` và DB verified-actor/transaction contract từ Chat A/B chưa được Chat 0 chấp nhận, tuyệt đối không sửa routers/auth production: chỉ lập endpoint inventory, source/call-graph comparison Gitea, test plan, giao diện phụ thuộc và migration RFC. Khi Chat 0 xác nhận Gate H + Gate D, tạo branch riêng từ integrated dev rồi triển khai P03. Mỗi claim có source và test evidence; không merge dev.
