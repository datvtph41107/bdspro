# BDSPro — Engineering Core Operating Model / Canonical Ownership

**Checkpoint:** 2026-10-15, Asia/Bangkok · **Branch:** `dev` · **Dạng:** documentation/handoff only.
**Prior checkpoint:** [Engineering Core reorientation](../2026-10-15-engineering-core-reorientation/README.md).
**Ma trận 20 chương:** [CHAPTER-MATRIX.md](CHAPTER-MATRIX.md).
**Điểm bắt đầu CH01:** [CH01-CONFIG-AND-WORKFLOW.md](CH01-CONFIG-AND-WORKFLOW.md).
**Prompt chat mới:** [CONTINUE-PROMPT.md](CONTINUE-PROMPT.md).

> **TRỌNG TÂM MỚI NHẤT:** Nền móng về cách developer **thiết kế, viết, tổ chức, phối hợp, kiểm chứng, thay đổi, debug, bảo trì và vận hành** code BDSPro. Mỗi trách nhiệm cần một **canonical source of truth** và một **owner** được xác định; không nhầm cùng một "truth" với việc gom mọi thứ vào một file. **Bước tiếp theo: CH01 — Configuration & Developer Workflow, khảo sát `.env` / `.yaml` / Makefile / `.sh` / môi trường dev-test-prod, trước khi quyết định thay đổi.**

## 1. Trạng thái và phạm vi có hiệu lực

- Đã rà soát GitHub `datvtph41107/bdspro`, nhánh `dev`, HEAD **`111d19e1b6096ec371e9bf76e1eb34adcbcf35d9`** tại thời điểm chuẩn bị checkpoint; từ chat mới phải kiểm tra lại HEAD.
- Source thực tế có `main.go`, `auth/{http,service,smtp}.go`, `organization/{http,service}.go`, `source/{http,service,service_test}.go`, `db/source/query/source.sql`, `db/sourcesqlc/`; `Makefile`, `sqlc.source.yaml`, `.github/workflows/bdspro-core.yml`, `integration_test.go`, `party_kind_test.go`.
- Migrations hiện diện: `000001_party_identity`, `000002_account_authentication`, `000003_organization_authorization`, `000004_organization_onboarding`, `000005_source_intake`, `000006_party_kind_immutability` (mỗi migration có up/down).
- Trên **Git tree được kiểm tra**, không có `.env` hay `.sh` được track; `.gitignore` có dòng `.env`. Đây **không chứng minh máy local hoặc môi trường deploy không có file tương ứng**.
- `main.go` đọc `DATABASE_URL`, `APP_ENV`, `BIND_HOST`, `PORT`, SMTP settings và `ALLOWED_ORIGIN` qua `os.Getenv`; kiểm tra một phần preconditions; tạo `pgxpool`, Auth/Organization/Source handlers, root mux, global Origin/Sec-Fetch-Site gate, HTTP Server, signal/shutdown.
- `Makefile` có `DB_URL` (mặc định rỗng) cho **CLI migration**, `sqlc`, `test`, `newmigration`, `migrateup`, `migratedown`. `DATABASE_URL` là **runtime application variable**; phân biệt **vai trò** thay vì coi hai tên khác nhau là một bug đã chứng minh.
- `sqlc.source.yaml` quản lý cấu hình code generation từ thư mục migrations + SQL Source sang `db/sourcesqlc`. CI workflow có PostgreSQL 18, test DB riêng, migration round-trip, sqlc, gofmt/vet, integration test và server smoke; CI xanh trước đó không chứng minh triển khai production.
- `README.md` vẫn có nội dung CORE-01..05 và "next CORE-06" thuộc tiến trình cũ; các checkpoint 2026-10-15 mới là authority cho **ưu tiên công việc**. Code, DB và actual CI luôn cần xác minh live.
- Trạng thái địa phương (git worktree, secrets, PostgreSQL local, production) **chưa được xác nhận**, không khẳng định đã đồng bộ.

**Không có code/migrations/CI/Makefile/YAML/.env/.sh được sửa hay tạo trong checkpoint này.** Các chương bên dưới là **phạm vi review/ứng viên quy chuẩn**, không phải "20 modules cần viết" hoặc bản chuẩn đã được nghiệm thu.

## 2. Sự chuyển hướng tư duy qua các chặng đã thảo luận

1. Từ Party/Person/Organization, C01, SQL, raw pgx/sqlc, Auth/Organization/Source: có các implementation và test nhưng **chưa tự động có Engineering Core thống nhất**. Vấn đề `kind` + function/trigger là phản chứng về overengineering: giải thích invariant, actor, tiền đề và chi phí trước khi thêm cơ chế; mong muốn bỏ `kind`/function ở migration đầu **chưa phải thay đổi đã áp dụng**.
2. `CODE-01.1`: `main()`/`run()`, `error`, `defer` và process exit; trách nhiệm kết thúc và cleanup. `CODE-01.2`: `pgxpool.New` khác `Ping`; startup failure, timeout, resource owner và readiness. `CODE-01.3`: composition bằng `auth.New(pool, mailer)`, concrete pool vs `Mailer` interface; khi nào một abstraction có giá trị.
3. `CODE-01.4`: Organization nhận `*auth.Handler`, Source nhận callback `authentication.Account`; coupling là trade-off, không mặc định function/interface luôn tốt hơn. Laravel DRX được đánh giá cao về **quy ước rõ, dễ tìm pipeline và ownership**, nhưng không copy hình dạng của framework.
4. `AUTH-SEC`: user mở rộng phạm vi credential, email/password, opaque token, JWT/PASETO, OAuth/OIDC, Google Sign-In, session/cookie/CSRF, IPActive, rate, authorization, config. **Không chọn công nghệ token trước threat model và actor**.
5. `CORE-FLOW-01`: theo dõi `POST /organizations/{organizationID}/sources` từ HTTP server → global handler → root/module mux → identity → request validation → `Source.Capture` → SQL authorization/transaction/idempotency → response; gồm logging, metrics, auditing, debugging, cancellation, retry, tests, operational failure.
6. `CORE-FLOW-01.2`: `auth.AccountFromToken` giữ `error`, nhưng `auth.Handler.Account` rút về `(int64,bool)`; invalid token và lỗi PostgreSQL cùng biến thành false/401. Hiện **fail closed**, song mất phân loại lỗi cần cho HTTP/operations; đây là static finding cần fault-injection test trước quyết định.
7. User yêu cầu một **ma trận tác nhân / kỹ thuật có mốc chương rõ, các ràng buộc và điều kiện thay đổi**, thay vì danh sách API hoặc một kiến trúc áp đặt. Đã lập đề cương 20 chương / 5 bo mạch, chi tiết trong [matrix](CHAPTER-MATRIX.md).
8. Yêu cầu MỚI NHẤT: phong cách làm việc hàng ngày phải rõ ràng, `.env`, `.yaml`, files config, Makefile, `.sh`, dev/test/prod, SimpleBank làm tư liệu đối chứng; **one source of truth, canonicalize ownership**. Vì vậy **chuyển bước tiếp theo sang CH01 Config & Developer Workflow**, sau một khung CH00 gọn làm tiền đề. Không tự động tiếp tục AUTH-SEC hoặc CORE-FLOW với flow cũ.

## 3. Đích đến của Engineering Operating Model

Một developer nhận nhiệm vụ phải biết:
- Mục đích nghiệp vụ và đối tượng chịu trách nhiệm; business ownership khác code ownership, runtime/resource ownership và human operational ownership.
- Hợp đồng dữ liệu/credential/actor/permission nào được xác minh ở trust boundary; điều gì không được lấy từ user input.
- File nào là canonical source, file nào chỉ là adapter, generated output, environment override, command interface hoặc tài liệu dẫn xuất.
- Cách đặt tên, comment, tổ chức package/file/function, dependency direction; ở đâu thực hiện logic, ở đâu map HTTP status.
- Đường thực thi khi thành công/lỗi/hủy/retry/commit ambiguity, ai cleanup, ai log, ai debug, ai chịu trách nhiệm recovery.
- Cách chọn test, CI, review/PR, deployment, runbook phù hợp với rủi ro thay đổi và duy trì trong nhóm.

**Không diễn dịch "single source of truth" thành một file config cho tất cả môi trường.** Mỗi concern có một authority rõ; các giá trị dev/staging/prod khác nhau là dự kiến, nhưng precedence, owner, validation và promotion rules phải rõ.

## 4. Các nhóm actor/owner cần phân biệt

| Actor | Quyền và trách nhiệm cần chứng minh |
| --- | --- |
| Business actor (Person/Account/member/reviewer/Organization) | Ai thực hiện hành vi, ai sở hữu dữ liệu/tài nguyên, khi nào có quyền, khi nào bị thu hồi |
| Client/browser/mobile/integration | Gửi untrusted HTTP input/token; không quyết định actor ID hoặc quyền |
| Developer / code owner | Thiết kế contract, module/API boundaries, viết và bảo trì code; user là người viết source mặc định |
| Reviewer / maintainer | Phản chứng, review API/breaking changes, test proof, ownership và chi phí |
| Application runtime | Startup, dependency injection, HTTP ingress, context, shutdown/resource lifecycle |
| DB/schema/query owner | Data truth, migrations, constraints, transaction, migration compatibility, permissions |
| Operator/platform/CI | Env/secrets delivery, process supervision, deploy/backup/restore, monitoring, incident |
| External provider (SMTP/Google/other) | Chỉ được tin trong phạm vi hợp đồng xác thực và side effects đã kiểm chứng |

## 5. Decision discipline và gates

Mỗi chương phải có: **current evidence → problem/actor → canonical owner → target behavior → invariant/trust boundary → alternatives → cost/risks → tests/observations → KEEP/MODIFY/TRIAL/DEFER/REJECT**.

Gates:
- **Correctness/Security (P0)**: không được chờ sự cố mới đánh giá; cũng không được tuyên bố vuln từ suy đoán.
- **Engineering foundation (P1)**: tìm quy chuẩn tối thiểu giúp developer mới dự đoán nơi viết code và nơi sửa lỗi.
- **Optional optimization (P2)**: rename/refactor/framework/cache/container abstraction khi có chứng cứ lợi ích > chi phí.
- **Docs truth**: ghi nhận rõ FACT / HYPOTHESIS / DECISION / NOT VERIFIED. Green CI ≠ production readiness. Comment stale không phải source authority.
- **Implementation ownership**: người dùng trực tiếp sửa/chạy source khi bước thực hành; assistant phân tích, chỉ command/checkpoint, review output. Không tự push Go/migrations/CI/config thay user. Ngoại lệ: yêu cầu checkpoint tài liệu này được ủy quyền trực tiếp.
- **Safety**: không thao tác phá dữ liệu local, không force push/reset database, không in secrets; các lệnh có khả năng mutate phải phân loại rõ.

## 6. Reference vs authority: Laravel DRX, SimpleBank, Go và PostgreSQL

**Laravel DRX (user-owned reference):** `app/Http/Kernel.php`, `app/Providers/RouteServiceProvider.php`, `routes/api.php`, middleware `PreLoginMiddleware`/`SufLoginMiddleware`. Giá trị: middleware pipeline, route convention, dễ tìm ownership; phản chứng: middleware sau `$next` vẫn có side effects và có thể khó debug. Không tự động chuyển bdspro sang Laravel-like dirs.

**TechSchool SimpleBank (upstream public `techschool/simplebank`, `master` đã kiểm tra):**
- Có `app.env`, `util/config.go` đọc config với Viper, `Makefile`, `sqlc.yaml`, `docker-compose.yaml`, `start.sh`, `wait-for.sh`, `main.go` và workflow CI/deploy.
- Đây là đối chứng tốt để hỏi "mỗi artifact có trách nhiệm gì?": file values, typed config, command runner, lifecycle orchestration, dependency readiness, CI/CD.
- Không copy mặc định giá trị trong `app.env`/`Makefile`; upstream Makefile chứa URL dành cho demo/local; phù hợp về dạy cơ chế nhưng **không trở thành production policy của BDSPro**.
- `start.sh` dùng `exec "$@"` và `set -e`; `wait-for.sh` giải quyết chờ service ready theo cơ chế script. Không thêm script chỉ vì SimpleBank có, khi `main.go` của bdspro đã `Ping` DB và CI riêng.
- `go.mod`, `sqlc.source.yaml`, `migrations` và CI thực tế của bdspro vẫn là authority của bdspro.

**Nguồn nghiên cứu tiếp**: Go modules/layout, package names, errors, net/http, slog, context; PostgreSQL migrations/constraints; 12-Factor Config; OWASP ASVS/session/CSRF; official framework docs, production GitHub repos và Reddit/cộng đồng (phản chứng, không coi popularity là proof). Bằng chứng phải gắn với vấn đề, không áp mẫu.

## 7. Những phát hiện cụ thể còn mở

- `auth.Handler.Account()` rút lỗi infrastructure và invalid session thành boolean; cần kiểm chứng classification/fault test trước khi sửa.
- `auth/http.go` logout bỏ qua DB revocation error mà vẫn 204; kết quả client-side cookie clear khác server-side revoke.
- `source/http.go` có comment nói chưa mount Source nhưng `main.go` đã mount; tài liệu sai lệch với code.
- `auth` / `organization` / `source` khác JSON body limits và trailing JSON enforcement; cần xác định policy trước shared helper.
- `auth.New(pool, mailer)` cho phép mailer nil; prod `main.go` guard SMTP; cần contract rõ ở composition vs Service.
- Email gửi sau DB commit Register; lỗi provider không đồng nghĩa Account chưa tạo.
- Raw pgx và sqlc cùng tồn tại; chưa đủ căn cứ bắt buộc thống nhất một cách.
- `000001`/`000006` đang sử dụng `kind` và trigger/functions; user muốn review bỏ nhưng migration history/data compatibility chưa được giải quyết.
- CH01 config sources: `DATABASE_URL` runtime vs `DB_URL` Make; DEV/PROD khác hành vi; source tracked chưa có `.env` hay `.sh`; nơi vận hành thực tế local/prod chưa xác minh.
- Chưa có contract chung được xác nhận cho request ID, access logs/metrics/tracing/audit, typed config, release/incident procedures.
- Kiểm thử đã có HTTP integration, concurrency, cross-tenant, migration/CI; cần xác định test gap theo risk, không phủ nhận kết quả test đã có.

## 8. Điểm khởi động chat mới và thứ tự khai thác

**Ngay sau recovery: mở CH01 — Canonical Config & Developer Workflow**, dựa trên `main.go`, `Makefile`, `sqlc.source.yaml`, `.gitignore`, workflow CI, migrations và tham khảo SimpleBank. Đọc [CH01-CONFIG-AND-WORKFLOW.md](CH01-CONFIG-AND-WORKFLOW.md).

Đi từng câu hỏi: 1) phân biệt config runtime / command / generator / deployment; 2) map owner và truth/precedence; 3) dev/test/prod khác nhau, secrets ở đâu; 4) bootstrap/startup/health & migrate boundaries; 5) lựa chọn `.env`/`.env.example`/typed Config/`.yaml`/Makefile/`.sh`/Docker Compose với chi phí; 6) một thí nghiệm kiểm chứng bằng source hiện có; 7) decision KEEP/MODIFY/TRIAL/DEFER/REJECT. **Không tạo các file đó hoặc refactor chỉ vì có trong mẫu**.

### Link điểm bắt đầu
- [Ma trận tất cả chương](CHAPTER-MATRIX.md)
- [Hồ sơ chương CH01](CH01-CONFIG-AND-WORKFLOW.md)
- [Prompt chi tiết để dán chat mới](CONTINUE-PROMPT.md)
- [Checkpoint tư duy gốc của ngày](../2026-10-15-engineering-core-reorientation/README.md)
