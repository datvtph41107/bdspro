# BDSPro — Ma trận 20 chương Engineering Core

**Mục tiêu:** Bản đồ các "bo mạch" kỹ thuật và tác nhân để đào sâu từng module trách nhiệm; **không** phải một backlog bắt buộc code, sơ đồ folder, hoặc quy chuẩn đã chốt.
**Owner của quyết định:** user chốt trade-off; assistant nghiên cứu/đưa phản chứng; code owner triển khai khi được yêu cầu. Current source `dev` luôn cần đối chiếu trực tiếp.

## 0. Định nghĩa một chapter card

Mỗi chapter card lưu: (1) problem/factual state; (2) intent; (3) owners/actors; (4) authority / source of truth; (5) trust/invariants; (6) dependencies; (7) alternatives+costs; (8) proof tests / runtime observation; (9) status KEEP/MODIFY/TRIAL/DEFER/REJECT; (10) conditions to reopen.

**P0**: security/correctness gate trước live users/data; **P1**: engineering consistency và dev productivity; **P2**: optimization/renaming có tính điều kiện. Mức ưu tiên review không mặc định là thời điểm commit.

## Bo mạch A — Nền tảng vận hành (CH00–CH05)

| Chương | Tác nhân và vấn đề thực tế | Target / technical intent | Canonical owner / các ràng buộc / vì sao ở chương này |
| --- | --- | --- | --- |
| **CH00 Engineering Governance** P1 | User developer/reviewer/operator: tài liệu lịch sử và quy chuẩn chưa được gom về một operating model | Decision/acceptance protocol, change footprint, review, actor map, source authority | **Working agreement**; cần trước khi tranh luận framework, naming hay schema; không xây quy trình nặng cho chuyện nhỏ |
| **CH01 Configuration & Secrets** P0/P1 **NEXT** | `main.go` đọc `os.Getenv`; Make/CI dùng biến khác; chưa có tracked env file | Config contract: key, type, default, validation, environment/secret ownership, precedence | **Runtime config owner** và operator; dev/test/prod phải phân biệt; không lộ secret; không tự áp Viper/YAML |
| **CH02 Database Runtime & Environments** P1 | `pgxpool.New/Ping/Close`, CI DBs, local chưa chứng minh | DB identity/naming, connection parameters, pool sizing, isolation, privileges, timeouts | **DB runtime owner**; database local/test/prod không được vô tình dùng chung; đừng trộn DB_URL command với DATABASE_URL runtime |
| **CH03 Migrations & Schema Evolution** P0/P1 | 6 migration pairs, Make migrate, CI up/down/up, lịch sử live chưa được xác minh trên local | DDL lifecycle, forward-fix, ownership, CI gate, rollback/data recovery, migration naming | **Schema owner**, migration history là state; không sửa migration đã áp lên DB thật mà không có compatibility plan |
| **CH04 DB Semantic Naming & Invariants** P1/P2 | Party kind/functions, subtype, Account, Org, Source; yêu cầu xét lại `kind` và naming | Đặt tên DB/schema/table/column/constraint/index theo intent, cost-aware invariant proof | **Domain+schema owner**; rename database vật lý ≠ rename table/column; kiểm chứng constraints/SQL/migration cost |
| **CH05 Bootstrap / Startup / Shutdown** P1 | `run()` sở hữu Pool, routes, signal/shutdown, health/ready | Startup conditions, wiring, resource ownership, graceful termination, readiness | **Application runtime owner**; startup != request lifetime; không bắt buộc `cmd/`, container hay script bootstrap |

**Luận cứ**: Nếu developer chưa biết process đọc giá trị từ đâu, DB nào, schema version nào, ai chịu trách nhiệm cleanup, mọi feature sau dễ bị sai môi trường hoặc sai ownership. Config file, Make targets và migrations khác nhau về authority.

## Bo mạch B — Source organization & HTTP (CH06–CH09)

| Chương | Tác nhân và vấn đề thực tế | Target / technical intent | Canonical owner / ràng buộc / vì sao |
| --- | --- | --- | --- |
| **CH06 Module / Package Architecture** P1 | `auth`, `organization`, `source`; Organization nhận `*auth.Handler` còn Source callback | Package API/import direction, code ownership, dependency boundaries, cost of change | **Module owner**; interface khi có hành vi/consumer cần, không copy layering pattern |
| **CH07 Naming / Comments / Go Idioms** P1 | `New` vs `NewService`, `Handler.S`, stale `source.NewHandler` comment | Names tại call site, exported API, comment WHY/contracts, errors, structs, test naming | **Code owner/reviewer**; đổi tên có giá trị đọc/bảo trì đủ bù cost; tránh documentation sai |
| **CH08 HTTP Routing / Middleware / Pipeline** P0/P1 | root `ServeMux` + module router + global Origin gate; `RequireAccount` có nhưng chưa dùng thống nhất | Full request flow, route precedence, middleware scope/order, trust boundary | **HTTP/platform owner**; route protected không bị vô ý bypass; phản chứng từ Laravel DRX middleware |
| **CH09 API Request / Response Contract** P1 | Khác limits 8KB/16KB/1MB, trailing JSON checks khác nhau, error helpers khác nhau | Parse/validate/body limit/status/error envelope/backward compatibility | **HTTP adapter owner**; phân biệt malformed input, business invalid, identity, infra; không tạo shared helper sớm |

**Luận cứ**: Naming/file location là kết quả của contracts và responsibilities; route/middleware rules ảnh hưởng bảo mật nên không thể xem là style thuần túy.

## Bo mạch C — Identity, Security, Authorization (CH10–CH13)

| Chương | Tác nhân và vấn đề thực tế | Target / technical intent | Canonical owner / ràng buộc / vì sao |
| --- | --- | --- | --- |
| **CH10 Identity & Authentication** P0 | `accounts`/password/email verify; passkeys/external identities schema chưa đủ flows | Credential ownership, proof, login/recovery, OIDC linking, service account trust | **Identity owner**; Account ≠ Person ≠ legal representative; credentials không được tiết lộ |
| **CH11 Session / Cookies / Token Strategy** P0 | Opaque 32-byte token+hash, DB session 24h, cookie; Logout ignores DB error | Session issue/expiry/revoke/rotate, CSRF/cookie deployment, JWT/PASETO trade-offs | **Auth runtime owner**; keep fail closed; differentiate session invalid vs database unavailable |
| **CH12 Authorization / Tenant Boundary** P0 | Membership/role/permissions, reviewer and scoped SQL checks | Actor→organization→resource→action; privilege/revocation/cross-tenant invariant | **Domain security + DB**; authenticated không mặc định authorized; test tenant/race paths |
| **CH13 Edge Security / Abuse Controls** P0 | Origin/Sec-Fetch-Site; chưa xác minh rate limiting toàn đường login | CSRF, IPActive/proxy trust, throttling, brute force, input limits, threat telemetry | **HTTP/security operations**; IP là tín hiệu, không phải identity; cần tránh false blocks |

**Luận cứ**: Phải xác định proof, threat model và resource authority rồi mới quyết định JWT/PASETO/OAuth, Redis, rate middleware.

## Bo mạch D — Data behavior & failure (CH14–CH16)

| Chương | Tác nhân và vấn đề thực tế | Target / technical intent | Canonical owner / ràng buộc / vì sao |
| --- | --- | --- | --- |
| **CH14 SQL, SQLC, Queries & Transactions** P1 | Auth/Org raw pgx vs Source sqlc; permissions trong SQL; transactions Service | Data access contract, query naming, SQL ownership, race/locks, query generation | **Persistence + domain owner**; không yêu cầu SQLC cho mọi truy vấn trước cost-benefit |
| **CH15 Failure, Retry, Idempotency & Side Effects** P0/P1 | Source idempotency; Register commits then SMTP; HTTP response may fail after commit | Retry/commit ambiguity, cancellation, external effects, outbox threshold, recovery | **Behavior + DB + external provider owner**; không cho retry tạo duplicate, không báo rollback sai |
| **CH16 Error Semantics & Ownership** P0/P1 | `AccountFromToken` error → `Handler.Account` bool; infra failure → 401; HTTP mappings khác nhau | Where error arises, classification, preserve/wrap, client exposure, metrics/logging | **Boundary owner**; fail closed, nguyên nhân dùng debug không bị xóa/tiết lộ |

**Luận cứ**: Đúng SQL ≠ đúng vòng đời hành vi khi cancel/retry/DB failure. Cần proof bằng tests và fault scenarios.

## Bo mạch E — Observability, Quality & Human Operations (CH17–CH19)

| Chương | Tác nhân và vấn đề thực tế | Target / technical intent | Canonical owner / ràng buộc / vì sao |
| --- | --- | --- | --- |
| **CH17 Logs, Metrics, Traces & Audit** P0/P1 | Startup log nhưng chưa thấy request correlation/causal logging chuẩn trong các đường đọc | Request IDs, duration/status, error cause, trace propagation, audit, secret redaction | **Runtime/operations**; logs phục vụ điều tra, không log raw cookie/password/token; avoid duplicates |
| **CH18 Test Strategy & CI Gates** P1 | Unit/integration/HTTP/concurrency/CI migrations+sqlc; fault gaps cần review | Tests theo boundary: config, handler, auth, DB, migration, race, fault; coverage theo risk | **Dev/reviewer/CI owner**; green test chỉ chứng minh kịch bản chạy; deterministic DB isolation |
| **CH19 Delivery / Runbooks / Work Habits** P1 | Make/CI/docs, local/remote chưa được chứng minh đồng bộ; thiếu production workflow contract | Daily dev commands, PR reviews, secrets/env promotion, release/rollback, incident & handoff | **Developer + operator**; destructive command opt-in; một lệnh có owner, preconditions, observable effect |

**Luận cứ**: Developer viết tốt nhưng không debug/release/recover được thì core vẫn chưa đủ; hình thức process phải tương xứng quy mô thực tế.

## 6. Tương quan, vòng review, trạng thái

**Cross-cutting (xét song song):** CH16 Error đi cùng CH08/CH11/CH17; CH15 Retry cùng CH14/CH18; CH01 Config đi cùng CH02/CH05/CH19; CH04 Naming liên quan CH03/CH14. Không hoàn thành máy móc theo ID.

**Vòng khai thác đề xuất**:
1. **Baseline / canonical ownership**: CH00 rất ngắn → **CH01 NEXT** với `main.go`/Make/YAML/CI và SimpleBank; CH05 bootstrap, CH02 DB envs.
2. **Schema and state**: CH03, CH04, CH14, CH15.
3. **HTTP & identity**: CH06–CH13, CH16.
4. **Observability/verification/delivery**: CH17–CH19 (review sớm những gate P0, không hoãn đến cuối).

**Cổng nghiệm thu của từng chương**:
- Có sơ đồ actor / source-of-truth / owner và current-vs-target.
- Dự đoán một hoặc nhiều failure/retry paths; tìm bằng chứng từ code, spec, test và một phản chứng/alternative.
- Đề xuất thay đổi tối thiểu *hoặc* chứng minh nên KEEP.
- Ghi tác động lên source, docs, DB, migrations, CI, dev/test/prod và backwards compatibility.
- Chỉ chuyển thành code sau khi user đồng ý/chủ động thực hiện; không tự tạo package, YAML, shell, framework, migration.

**Status khi checkpoint**: TẤT CẢ là review topics / hypotheses; không tự ghi "standard approved". Existing code/tests/CI được ghi là baseline, không bị phủ định chỉ vì mở ma trận này.
