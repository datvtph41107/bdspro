# BDSPro — Engineering Core Full-Code Roadmap (No Deployment)

**Bản lập kế hoạch:** 2026-10-20. **Baseline được kiểm tra:** nhánh `dev`, commit `3ab1f431c887d404f807b54210031f7ce0a99128`. **Nguồn tham chiếu chương trình:** https://github.com/techschool/simplebank#backend-course-videos-golang.

> Tài liệu này định nghĩa **các cổng triển khai và nghiệm thu mã nguồn**. Nó không xác nhận các giai đoạn mới đã được code, cũng không thay đổi quyết định/schema hiện tại. Kiểm tra lại `dev` tại mỗi giai đoạn vì mã nguồn là thẩm quyền về trạng thái đã thực hiện. Các chapter CH00–CH19 hiện hành vẫn giữ vai trò review matrix; G00–G10 bên dưới là roadmap *implementation*.

## 1. Mục tiêu và phạm vi

- Xây dựng Engineering Core có thể chạy và kiểm thử **toàn bộ trên local/CI**: PostgreSQL 18 + SQLC + Go + Gin + Auth/Session/JWT v5 + đa kênh message + Outbox + Redis/Asynq + Protobuf/gRPC + Gateway + OpenAPI + RBAC + observability + graceful shutdown.
- Giữ `Account -> Person`, Organization Membership và kiểm tra quyền từ SQL; không tạo một Account cho mỗi kênh, không đưa quyền của tenant vào JWT.
- Không tạo nghiệp vụ ngân hàng giả (transfers/accounts); thay bằng flows thật của BDSPro: account onboarding, OTP, organization invitations, source workflows, notifications, admin actions.
- **Không thực hiện** AWS, ECR/RDS/Secrets Manager, EKS/Kubernetes, Route53, ingress, cert-manager, public TLS, cloud load balancers, public deployment hoặc GitHub Actions deploy. Lecture #26–36 và #71–73 = DEFER (chỉ có thể thảo luận concept, không triển khai).
- Dockerfile, Docker Compose, migration job và loopback-only networking chỉ phục vụ local development/test; không được coi là production deployment.
- Mọi thay đổi lớn phải có PR vào `dev`, test/check SQLC-generated source, báo cáo acceptance và lệnh fast-forward local. Không sửa migrations đã áp dụng để thay lịch sử.

## 2. Định nghĩa code hoàn chỉnh (Definition of Done bắt buộc)

Một gate chỉ PASS khi **tất cả** điều kiện áp dụng đã được chứng minh:

1. Problem, intent, actor, owner, trust boundary, source of truth, invariant và alternative được nêu trước khi code; không thêm interface chỉ để thỏa hình thức.
2. Database schema có forward migration + safe down cho CI, named constraints, SQLC query, **generated code đúng từ `sqlc generate`**; rollback production không được đánh đồng với SQL down.
3. Code thực thi xuyên `API / gRPC -> domain service -> DB / queue / provider adapter` với request/response thực, input validation, timeout, error mapping và cancellation.
4. Không phá login/session cũ; có backward compatibility rõ ràng hoặc test chứng minh migration của consumer. Transport không được trở thành identity hay authorization authority.
5. Positive, negative, replay, concurrency, idempotency, timeout, provider failure và security tests tùy risk; chạy CI với DB test dùng riêng và `go test -race ./...`.
6. Có hướng dẫn local từ checkout sạch, cần gì để chạy, cấu hình nào, endpoint nào, ví dụ request, rollback/retry, bí mật nào không được log, và hạn chế chưa xử lý.
7. Workflow HEAD PASS: migrations UP–DOWN–UP, `sqlc generate` và `git diff` sạch, gofmt, go vet, test, race, startup + Compose smoke (và test chuyên biệt cho worker/gRPC khi đã có).
8. GitHub PR merged vào `dev`, hậu merge CI PASS; biên bản nghiệm thu nêu evidence, trade-offs, residual risks. **Không tự nhận hoàn chỉnh khi mới tạo skeleton, mock-only, TODO hoặc code chưa chạy xuyên tầng.**

## 3. Giai đoạn tổng thể và phụ thuộc

| Gate | Tên / phạm vi | SimpleBank tham chiếu | Baseline | Điều kiện hoàn thành cụ thể |
| --- | --- | --- | --- | --- |
| **G00** | Baseline + canonical contracts | #0–25, #37–38, #53, #65–70, #74–77 | Nhiều phần đã triển khai | Khóa trạng thái `dev`, test suite, owner config/DB, map các gaps và local instructions |
| **G01** | Account & Auth Lifecycle | #15–22, #37, #49–50, #61–62, #70, #77 | Email/password + phone-login trên Account có sẵn | Phone-only onboarding + account linking + controlled recovery + session policy; không tự động bật MFA khi chưa có recovery |
| **G02** | Verification & Message Delivery Channels | #17, #21–22, #59, #61–62 | SMTP, Twilio SMS, dev SMS DB | OTP challenge độc lập transport; email/SMS/Zalo adapters; dev sinks; provider boundaries; state/retry contract |
| **G03** | HTTP Contract + OpenAPI/Swagger | #11–14, #16–18, #45, #47, #49–50, #53 | Gin + JSON error contract | API spec bao phủ routes, success/error envelope, validation, pagination/partial update, Swagger local + contract tests |
| **G04** | Reliable DB Side Effects / Transactional Outbox | #6–9, #16, #56–57 | `Store.ExecTx` & direct provider calls | Producer business changes + outbox insert 1 DB transaction, dispatcher reads post-commit, retries/idempotent delivery |
| **G05** | Redis + Asynq + Workers | #54–60, #63 | Chưa có Redis/Asynq | Queue job types, retry/backoff, delayed tasks, dedup, worker lifecycle, local Redis, fault tests and metrics |
| **G06** | Protobuf + gRPC Service | #39–42, #44, #47, #49–52 | Chưa có proto/gRPC | Versioned `.proto`, generated code reproducible, gRPC Auth/Account + at least one business use-case, interceptor auth/RBAC |
| **G07** | Gateway + Unified API Docs | #43, #45–46, #51–53, #63–64 | Gin REST only | gRPC-Gateway side-by-side with Gin, OpenAPI/Swagger, parity tests, metadata, errors, no auth bypass |
| **G08** | Security & Concurrency Hardening | #7–9, #16–22, #37, #65–67, #70, #74, #76–77 | Baseline identity/RBAC exists | Security threat tests, distributed rate policies, JWT key rotation, session recovery strategy, cross-tenant/revocation proof |
| **G09** | Observability & Quality Gates | #5, #10, #13, #18, #52–53, #57, #60, #63–64, #75 | CI, slog, request ID, race | HTTP/gRPC/worker correlation, metrics/tracing/audit decisions, failure-injection, integration coverage |
| **G10** | Local End-to-End Operation & Handover | #2–3, #10, #23–25, #38, #48, #60, #68–69, #74 | Compose PostgreSQL+API | One local Compose stack API + Redis + worker + PostgreSQL (+ optional local mail/dev channel), migration service, bounded shutdown, e2e journey and runbook |

**Thứ tự chính:** G00 → G01 → G02 → G03 → G04 → G05 → G06 → G07 → G08 → G09 → G10. Test/security/observability gates được áp dụng dọc toàn bộ quá trình, không trì hoãn các lỗi P0 đến G08–G09. Các gate nhỏ trong một giai đoạn có thể được triển khai PR tuần tự.

## 4. Chi tiết cho từng gate

### G00 — Source audit, không rewrite

- Source hiện có: `main.go` wiring, `util/config.go` Viper, `app.env`, PostgreSQL/SQLC migrations 000001–000009, Gin in `api/`, `auth/`, `organization/`, `source/`, `compose.yaml`, `.github/workflows/ci.yml`.
- Giữ `db/sqlc/Store.ExecTx`, interface tại consumer khi test cần, Session opaque backing PostgreSQL, signed JWT v5 HS256 session lookup, SQL membership authorization.
- Công việc: inventory route/module/schema/CI, xác định bugs và omissions ưu tiên P0, liệt kê lecture đã thực hiện/áp dụng khác bài; xây acceptance fixtures hiện thực. Không xóa lịch sử quyết định.
- PASS: `dev` HEAD CI xanh, baseline tests chạy ở CI, mọi gate còn lại có link docs và current-state/target-state.

### G01 — Account & Identity lifecycle

- **Business intent:** người dùng chọn Email/Password hoặc Phone OTP để vào **cùng Account**; Phone-only Registration là nghiệp vụ khác Phone Login trên số đã verified.
- Model: `Account` root, phone/email credential bindings (unique verified ownership); password credential optional theo policy; `Person` gắn Account; organization membership không xuất phát từ phương thức đăng nhập.
- Source check: `db/migration/000002` cho phép Account không có `account_emails` hay `password_credentials` (FK nằm từ các bảng credential về Account), nhưng `auth.Service.Register` hiện luôn tạo email/password. `000008` phone verification dựa trên `account_phone_id` và **phải có Account trước**, còn `000009` chỉ login số đã verified. Vì vậy G01.1 cần **challenge đăng ký trước khi Account tồn tại**, độc lập phone enrollment/login, và transaction redeem proof → tạo Party/Person/Account → verified phone binding → session. Không tái sử dụng authenticated verification hoặc tạo Account trước khi OTP được kiểm tra.
- Implement: anonymous phone-first start/verify/register, no orphan Account on failed proof, link/unlink phone/email, safe change & recovery, notification of security changes, account close, handling number reassignment and repeated enrollments; user enumeration and quota constraints.
- MFA/totp/passkey/OIDC là những **credential types hoặc step-up policy riêng**, không gọi phone OTP alone là 2FA. Nếu bật MFA thì password stage không cấp full session trước MFA; recovery codes & anti-lockout bắt buộc trước enablement.
- Proof: same account across credentials, unknown number no auto linking, concurrency unique owner, used/expired OTP no session, invalid token no access, role revocation persists for both login routes, no account discovery leak.

### G02 — Channel-independent delivery

- Phân biệt **target identifier / factor of proof / delivery channel / provider**. Zalo ZBS message gửi tới OA/Zalo recipient không tự chứng minh chủ sở hữu một số điện thoại. Zalo OAuth login là identity-provider flow riêng.
- Domain delivery contract: `purpose`, `recipient_binding`, `channel`, `provider`, `challenge_id`, `expires_at`, attempt counter, delivery receipt/status. Không biến provider status `accepted` thành `OTP verified`.
- Adapters: Dev DB delivery sink, SMTP/Email provider, SMS provider, Zalo adapter **chỉ kích hoạt khi API/approval/policy/provider credentials đã xác minh**. Nếu chưa có quyền Zalo thật: fixture/HTTP fake đầy đủ tests nhưng không tuyên bố real-world delivery.
- G01–G02 phải xác định số mã/challenge/recipient binding để không tạo insecure fallback giữa các kênh. Keys and codes never logged. Chỉ store OTP HMAC ở challenge; plaintext dev sink limited to isolated local DB.
- Recovery and channel fallback must prove the receiver is entitled to Account; don't let fallback weaken proof. Stateless interface design should follow actual adapters and avoid a premature generic notification framework.

### G03 — API contract, documentation & response

- Current Gin error envelope + request ID remains authoritative; add consistent **success** contracts and machine-consumable OpenAPI 3 docs for *all existing* endpoints.
- Test request body limits, unknown fields, validation, status codes, HTTP Cookie/JWT security, pagination where a real list needs it. On update: distinguish absent, null and zero values (SQLC nullable/optional), forbid unauthorized field mutation.
- Swagger UI serves **local-only** docs, optionally generated from OpenAPI; generated docs must be reproducible in CI. Do not introduce gRPC just to obtain Swagger.
- PASS: every implemented public route is represented in the spec with authenticated/anonymous status and examples; spec/docs consistency tests; no raw internal errors.

### G04 — Transactional outbox and durable side effects

- Define `domain DB commit` and `external delivery` as **different transactions**. Never claim `ExecTx` + direct enqueue to Redis guarantees atomic delivery.
- Migration and SQLC for outbox records, status, due time, lock/lease, retries/backoff and idempotency key; producer stores business state + intent in one PostgreSQL transaction.
- Local dispatcher scans committed rows with bounded concurrency and lease/reclaim; provider effects are **at least once**, consumer must be idempotent, handle uncertain remote success and duplicated callback.
- Choose first real case (organization invitation or non-secret notification), not raw OTP codes persisted in generic outbox/Redis. OTP requires separate ephemeral, privacy-reviewed path.
- PASS: crash before/after commit, worker down, duplicate dispatch, provider accepted/lost response, transient/permanent failure, restart recovery.

### G05 — Redis/Asynq workers

- Add local Redis Compose service, Asynq job producer+handler, delay/schedule, retries/backoff/dead letters (as supported) and task de-dup policy; PostgreSQL remains canonical event/outbox record.
- Distinguish durable business outbox, Redis queue transport, cache/limiter. Source of truth for rate limits must match desired multi-instance semantics and local topology.
- Separate worker process/lifecycle from Gin process where useful; worker shutdown drains leased tasks boundedly. Independent tests include fake provider, Redis integration, retry and duplicate idempotency.
- No cloud services or deploy targets.

### G06 — Protobuf/gRPC

- Establish Proto source owner, package/version, `buf` or reproducible `protoc` tooling, generated code path, compatibility rules; avoid maintaining two hand-written application logics.
- Start with real `GetCurrentAccount`, `ListWorkspaces`, optionally business `Source.Read` plus auth/phone login when security parity proven; both Gin and gRPC delegate same services.
- Map HTTP errors and gRPC statuses precisely; token/cookie metadata authority explicitly selected; interceptor verifies account identity, service SQL evaluates tenant permissions.
- Prove cancellation/deadline, metadata propagation, proto optional field semantics and versions. Startup health/readiness separate from HTTP, support loopback local gRPC endpoint.

### G07 — Gateway, Swagger and protocol parity

- Run REST Gin and gRPC concurrently only while preserving one authentication policy and same ownership; gRPC-Gateway exposes a selected subset with generated OpenAPI.
- Compare JSON HTTP/Gateway/gRPC: identity, validation, error classification, authorization, normalized audit, pagination and optional updates.
- Local Swagger for Gin+Gateway; explicit source authority, no duplicate schema drift. Frontend embed (#46) is an **optional documented local lab**, not a reason to change client architecture.
- Gate: unauthorized gRPC cannot bypass SQL tenant checks; JWT/session revocation works through both transports; gateway responses match approved schema.

### G08 — Security, sessions and concurrency

- Review key rotation / `kid` policy (not silently changing existing HS256), JWT v5 validity/claims and PostgreSQL session revocation; rate limit per identifier/endpoint/phone across processes only if Redis is ready.
- Authorize Account/Person/Organization/Resource actions at domain/SQL boundary, not only at Gin middleware or gRPC interceptors. Concurrency matrix: membership remove, role update, source writes, token revoke, OTP verify.
- Explicit threat models for phone change, SIM swap, phishing, Zalo account binding, replay/recovery; if MFA enabled require pending-auth proof and independent account recovery.
- Gate: fail-closed DB outage, no claims-based stale role access, synthetic attacker matrix, token rotation and cookie tests.

### G09 — Quality, observability and fault diagnosis

- Extend current JSON `slog` + request IDs into structured RPC and task logging with safe correlation; no phone/OTP raw, tokens, session secrets or tenant data in generic logs.
- Instrument relevant metrics (queue depth, dispatch lag, attempts, retries, 4xx/5xx, DB pool wait, request latency), trace context and constrained audit events; only add infra when query/debugging proves value.
- CI integration with disposable Postgres+Redis+worker+gRPC+gateway, no hidden skipped tests, gofmt/vet, generated code diff, race, fault injection, documented known limits.

### G10 — Operable local stack (NOT deploy)

- Compose `db`, migration one-shot, `api`, `redis`, `worker`, optionally local SMTP/SMTP mock/dev SMS sink and gRPC/Gateway ports **published to loopback only**.
- Boot/shutdown contract: DB health → migration complete → services ready; API/worker stop/drain independently; finite timeouts and no dropped confirmed business writes on restart.
- Developer scripts/Make targets must be minimal, explicit about destructive DB commands; config file/env override same rules in local/CI. Include smoke requests across registration → phone login → workspace → async notification → gRPC/Gateway → logout/revoke.
- PASS: clean local checkout with documented tools can run full journey with no external provider or cloud; CI provides equivalent automated proof. No promise of production-readiness.

## 5. Lecture-by-lecture coverage policy — all lectures accounted for

| Reference lessons | Gate/adaptation; không phải chép sample banking |
| --- | --- |
| **#0–10** | G00, G04, G09, G10: DBML implemented schema, PG, migrations, SQLC, CRUD tests, `ExecTx`, locks, isolation, CI |
| **#11–18** | G00, G01, G03, G09: Gin/Viper, adapter mocks, validators, Accounts/credential SQL, errors, Argon2id instead of bcrypt |
| **#19–22** | G01, G08: review JWT vs PASETO, keep JWT v5 + opaque backing session, authenticate then authorize |
| **#23–25** | G00/G10: multistage Dockerfile, networking, Compose local + service readiness; **not public deployment** |
| **#26–36** | **DEFER** AWS account, ECR, RDS, cloud secrets, EKS/K8S, Route53, Ingress/TLS, cloud deploy pipeline |
| **#37–38** | G00, G01, G10: PostgreSQL session lifecycle already implemented; catalog-driven DB docs already implemented |
| **#39–42** | G06: Proto + gRPC server/clients + selected Auth/Account APIs |
| **#43–45** | G03/G07: gRPC Gateway, metadata, Swagger/OpenAPI (REST docs can be earlier) |
| **#46** | G07 optional lab: embed static frontend without requiring frontend app redesign |
| **#47–48** | G03/G06/G10: RPC validation/errors; Go-embedded migrations explored only as lab because current migration one-shot owner remains |
| **#49–50** | G01/G03/G06: SQLC nullable partial updates, Protobuf optional presence, request masks, input ownership |
| **#51–53** | G07/G09: gRPC Authorization & structured logs; retain existing HTTP request logging |
| **#54–58** | G04/G05: Redis/Asynq, async worker, transaction-outbox safety, retries, delayed jobs |
| **#59–62** | G01/G02/G05: SMTP/mail delivery, test configuration, email token/verification already present; integrate reliable message lifecycle |
| **#63–64** | G06/G07/G09: gRPC tests with mocked DB and queue, authenticated RPC test matrices |
| **#65–70** | G00/G08/G10: SQLC v2 config, pgx driver/errors, Compose mounts, local binary tools, live RBAC checks |
| **#71–73** | **DEFER** AWS EKS security groups, gRPC public deployment, AWS cost management |
| **#74–77** | G08–G10: shutdown for HTTP+gRPC+worker, Go loop semantics tests, CORS/CSRF, JWT v5 review |

## 6. Local verification and operational handoff contract

- Each gate deliverables: `Gxx-CONTEXT.md` (why), `Gxx-DESIGN.md` (invariants/choices), source+migrations+SQLC+tests, `Gxx-ACCEPTANCE.md` (exact executed commands and CI run/commit), and `Gxx-CONTINUE.md` (next prerequisite). Keep docs brief if the gate is small.
- Local sync: `git status --short --branch`, `git switch dev`, `git fetch origin`, `git merge --ff-only origin/dev`, `go test ./...`. Never `git reset --hard` automatically; review dirty worktree before pulling.
- Isolation: `app.env` local; separate disposable `*_ci` DB names; migration CLI uses explicit DB_URL, runtime uses DB_SOURCE; dev SMS plaintext sink is manually provisioned only for isolated local DB, never in official migrations.
- Definition of *coverage* is **implemented/tested concept** not copying all SimpleBank example entities/frameworks. For skipped/deferred lecture, record reason and remaining educational alternative.

## 7. Mốc hiện tại / Next action

- **Verified baseline:** `dev` at commit `3ab1f431`; prior PR #19 added phone OTP login + opt-in dev DB SMS delivery. PostgreSQL/SQLC/Gin/Viper/JWT v5/SMTP/Role checks already exist.
- **Next implementation gate G01.1:** phone-only account onboarding + no-email requirement when a verified phone is the chosen login method. Before schema changes verify current `accounts`, `account_emails`, `password_credentials`, `account_phones` constraints and account creation transactions.
- **G01.1 acceptance:** no Account/session before OTP proof; once proven, Account/Person created transactionally with verified phone; existing verified phone logs in (no duplicate Account), unknown number can register only via explicit registration flow, MFA not incorrectly claimed, no accidental organization membership, replay and concurrent requests do not duplicate identity.
- Gate work is NOT complete just by creating this roadmap; no new auth/worker/gRPC code is claimed by this document.

## Source refs

- https://github.com/techschool/simplebank#backend-course-videos-golang
- https://github.com/datvtph41107/bdspro/tree/dev/docs/implementation
- https://github.com/datvtph41107/bdspro/blob/dev/docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/CHAPTER-MATRIX.md
