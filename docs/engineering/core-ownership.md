# ENG-GOV-01 — Ma trận sở hữu Engineering Core

**Trạng thái:** Quy chuẩn kỹ thuật có hiệu lực sau khi được merge vào `dev`. Tài liệu mô tả ranh giới trách nhiệm của **Engineering Core V1 đang tồn tại**, không tuyên bố Business Core hoặc môi trường WSL đã nghiệm thu.

**Nguồn sự thật:** Source Go/Gin, PostgreSQL/sqlc, Redis/Asynq và migrations hiện có trong BDSPro. DRX Laravel cung cấp cách gọi tên middleware theo hành động; Java DRWash cung cấp ý tưởng chia sẻ Core; TechSchool SimpleBank cung cấp cách tổ chức thực thi Go/SQL trực tiếp. **Không repository tham chiếu nào thay thế source BDSPro.**

## 1. Bốn loại ownership phải phân biệt

- **Capability Owner — sở hữu năng lực:** package/thành phần chịu trách nhiệm chính về hành vi hoặc chính sách, chẳng hạn `auth` sở hữu việc kiểm tra thông tin đăng nhập.
- **Data Owner — sở hữu trạng thái:** nơi bảo vệ dữ liệu bền vững và bất biến; thông thường là constraints, câu lệnh SQL và transaction trong PostgreSQL.
- **Execution Owner — thực thi:** hàm đang vận hành request hoặc công việc: middleware, handler, service, Store hoặc worker.
- **Change Owner — chịu trách nhiệm thay đổi:** developer triển khai và người review được xác định cho **từng PR**. Không được coi tên package là tên người, team hoặc quy tắc phê duyệt GitHub.

Một request có thể đi qua nhiều thành phần, nhưng mỗi bất biến phải có **nơi thực thi chủ đạo**. Khi tài liệu khác source, phải kiểm tra runtime/migration và ghi rõ bất nhất; không tự giả định rằng tài liệu có thể thay thế hành vi hiện hành.

## 2. Ma trận Engineering Core hiện tại

| Capability | Source / owner hiện tại | Trách nhiệm, bất biến | Ranh giới và bằng chứng |
| --- | --- | --- | --- |
| Bootstrap & Runtime | `main.go`; `util/config.go` | Cấu hình, kết nối Postgres/Redis, migrations, khởi động worker/HTTP, readiness | Không xử lý quyền nghiệp vụ; kiểm chứng bằng startup tests, CI smoke, G0 WSL |
| Configuration & Deployment | `app.env`, `util/config.go`, Dockerfile, Compose, start/wait scripts | Ưu tiên process env, services/volumes xác định | Không commit secret; kiểm tra precedence và dữ liệu volume |
| HTTP Routing & Ingress | `api/server.go`, `api/request_limits.go` | Đăng ký route, CORS, request ID, giới hạn body, HTTP timeout, direct-peer IP | Không quyết định owner của Organization; HTTP tests |
| Auth Admission | `api/middleware_ipactive.go`, `api/middleware_prelogin.go`, `api/phone_otp.go` | `IPActiveSource` chặn lạm dụng trước parse, `Pre*` kiểm tra input, `IPActive` giới hạn theo action/identity | Rate limit không phải Authenticate/Authorize; auth route fail-closed nếu Redis limiter lỗi |
| Request Observation | `api/middleware_request_monitor.go`, `api/middleware_suflogin.go` | Ghi denial/outcome mà không lộ secret | Monitoring không được thay đổi quyền; log/detection tests |
| Authentication | `auth/auth.go`, `auth/phone_otp.go`, `api/google.go` | Password/Google/OTP phải xác minh trước `IssueSession`; cùng account identity | Google phụ thuộc config; Phone OTP chỉ development; policy email chưa xác minh vẫn chưa chốt |
| Token & Session | `token/`, `auth.IssueSession`, `api/token.go`, `db/query/session.sql` | Đúng loại và hạn JWT, kiểm tra phiên refresh trong DB, renew/revoke | Revoke hiện chỉ chặn refresh session; access JWT đã phát còn hiệu lực tới expiry |
| Identity Persistence | `db/migration/`, `db/query/account.sql`, `db/sqlc/store.go` | `accounts.id` là danh tính chung; Party/Person/Account tạo trong một PostgreSQL transaction | Không tạo account riêng cho từng provider; DB constraints/FKs là bằng chứng |
| Organization Access | `api/organization.go`, `db/query/organization.sql`, transaction DB | Actor lấy từ JWT; predicate owner/membership còn hiệu lực trong thao tác SQL | `Authenticate` không đồng nghĩa có quyền owner; SQL và constraints kiểm soát |
| Email Verification | `worker/`, `mail/`, verify-email transactions | Task, retry, verification record, verify một lần | Redis enqueue trong `RegisterTx.AfterCreate` **trước** PG COMMIT; chưa có atomic handoff |
| Schema & SQLC | `db/migration/*.sql`, `db/query/*.sql`, `sqlc.yaml`, generated `db/sqlc` | Migrations/SQL là định nghĩa chỉnh sửa; constraints bảo vệ dữ liệu | Không sửa thủ công code sqlc sinh ra; các tệp Store/transaction viết tay trong `db/sqlc` vẫn hợp lệ |
| CI & Vận hành | `.github/workflows/test.yml`, `scripts/`, `docs/operations/` | Test, build, migration, HTTP smoke, bằng chứng runtime | CI xanh không đồng nghĩa nghiệm thu WSL, khôi phục dữ liệu hay production |

**Business Core:** Property, Listing, Inquiry, Transaction chưa được triển khai/nghiệm thu theo tài liệu này. Organization hiện là nền tảng Identity/Application Access, chưa phải bằng chứng Business Core đã hình thành.

## 3. Ranh giới request — phân biệt admission, identity và quyền

Luồng password login được đăng ký trong `api/server.go` (bên trong CORS HTTP wrapper):

```text
Request ID / HTTP log / Recovery (global)
  -> RequestMonitor (quan sát response sau ctx.Next)
  -> RequestIngress (giới hạn body)
  -> SufLogin (quan sát kết quả login sau ctx.Next)
  -> IPActiveSource -> PreLogin -> IPActive
  -> loginUser -> auth.Login -> IssueSession -> SQL session
```

HTTP timeout được thiết lập trong `Server.StartWithReady`, **không phải** một Gin middleware. Chi tiết thứ tự, tính chất wrapper và xử lý response xem [Request Lifecycle](request-lifecycle.md) và `api/server.go`.

Các route Organization dùng `Authenticate` để xác lập account, sau đó SQL kiểm tra owner/membership. Route Google/Phone có thứ tự middleware và điều kiện đăng ký riêng; **không được khẳng định mọi route đều đi qua `PreLogin` hoặc cùng một chuỗi `IPActive`.**

## 4. Quy tắc ownership khi review

1. Xác định **actor, action, resource, durable state** trước khi đặt tên package.
2. `api` sở hữu input HTTP, status, response, middleware và admission. Với use case đơn giản, handler được gọi Store trực tiếp **như source hiện tại**; không ép tạo service.
3. Việc xác minh thông tin đăng nhập và phát session thuộc contract `auth`/`token`; không tự tạo đường cấp token thứ hai trong middleware/controller.
4. Nếu quyền và cập nhật dữ liệu phải nhất quán, đặt predicate authorization trong SQL/transaction phù hợp.
5. Worker/adapter sở hữu delivery; enqueue Redis thành công không chứng minh email đã được gửi.
6. Interface, package, Go module hoặc service layer mới phải có lý do thực tế: nhiều cách hiện thực, test seam, reuse hay ranh giới trách nhiệm rõ ràng.
7. Khi thay đổi xuyên capability, nêu **operation điều phối**, các owner bị ảnh hưởng và test cần bảo vệ. Không nhân bản một quy tắc thành nhiều phiên bản mâu thuẫn.

## 5. Các cổng còn mở

- **G0:** WSL Compose, backup/restore và runtime thật còn thiếu bằng chứng. Xem [G0](../operations/g0-runtime-acceptance.md).
- **G1:** nghiệm thu theo từng route, hạn mức, reverse proxy trust và denied events.
- **G2:** chốt yêu cầu PG–Redis delivery sau kiểm thử lỗi. Xem [Register/Asynq](../operations/register-asynq.md).
- **G3:** chốt chính sách unverified email, session invalidation; Google browser/SMS provider E2E còn ngoài phạm vi CI hiện tại.

Quy trình với developer: [Development Contract](development-contract.md). Lý do lựa chọn thiết kế: [ADR-0001](decisions/0001-core-ownership.md).
