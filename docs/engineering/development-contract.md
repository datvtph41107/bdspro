# ENG-GOV-01 — Quy chuẩn viết code, thay đổi và nghiệm thu

**Phạm vi:** Mọi thay đổi mới dự kiến merge vào `dev`. Mục tiêu là ownership rõ, thay đổi tối thiểu, test có thể kiểm chứng; **không phải** thêm framework, interface hay quy trình phê duyệt máy móc.

Đọc [Core Ownership Matrix](core-ownership.md) khi đụng nhiều module; đối chiếu [Request Lifecycle](request-lifecycle.md) khi thay đổi middleware và [G0 Runbook](../operations/g0-runtime-acceptance.md) khi tác động vận hành.
Các cổng kiểm tra tự động và vòng làm việc với Git/CI được ghi tại [ENG-GOV-02 — Workspace Guardrails](workspace-guardrails.md).
Khi thiết kế một capability có runtime/worker/calculation/reuse, dùng [ENG-GOV-03 — Formula Catalog](capability-formula-catalog.md) để phân biệt cơ chế, contract, policy, failure; [Capability Workbook](capability-workbook.md) có mẫu Platform × Domain và trường hợp Register/Asynq. Các công thức là công cụ phân tích, **không tạo thêm gate bắt buộc cho thay đổi đơn giản**.
Khi thêm/chỉnh API, tham khảo [ENG-CORE-04 — Application Foundation](application-foundation.md): lấy actor đã xác minh, typed request, response session, request context, SQL ownership và tiêu chí tách package core; không ép mọi route tạo service/repository mới.
Với route params và lỗi dữ liệu, xem [ENG-CORE-05 — API Contract & Error Semantics](api-contract-error-semantics.md): chỉ dùng chung parser/driver-error recognition, còn HTTP status, bảo mật thông điệp và business conflict do operation sở hữu.

## 1. Một vòng làm việc thống nhất

```text
Problem -> Intent -> Responsibility/Owner -> Invariant
        -> Implementation -> Evidence -> Review -> Acceptance
```

1. **Xác minh source.** Đọc route, call chain, SQL, migration và test đang chạy. Không suy đoán qua tên file hoặc tài liệu reference cũ.
2. **Chốt owner.** Xác định capability, data owner, execution owner và người review PR. Kiểm tra các provider/route/actor khác có cùng quy tắc.
3. **Phát biểu invariant.** Nêu hành vi đúng, request từ chối, rollback, replay và concurrency nếu liên quan.
4. **Chỉ sửa phạm vi nhỏ nhất.** Ưu tiên cú pháp Go/Gin và SQL/sqlc sẵn có. Không tạo abstraction khi chưa chứng minh lý do.
5. **Kiểm chứng.** Test case hợp lệ, không hợp lệ, authorization, transaction, worker retry, log và concurrent request tùy thay đổi. Phân biệt unit, integration và WSL runtime.
6. **PR có bằng chứng.** Ghi rõ tác động và phần chưa kiểm thử bằng [PR Template](../../.github/PULL_REQUEST_TEMPLATE.md).
7. **Nghiệm thu chính xác.** CI xanh hỗ trợ nghiệm thu mã nguồn; WSL, external provider và backup/restore cần bằng chứng riêng. Ghi SHA đã merge và gap còn mở.

## 2. Mỗi lớp thực sự được phép làm gì?

| Thành phần | Trách nhiệm được sở hữu | Không được tự nhận là sở hữu |
| --- | --- | --- |
| Middleware trong `api/` | Admission, input shape, JWT verification, observation an toàn | Quyền owner của Organization, SQL transaction, phát token sau response |
| HTTP handler trong `api/` | Input/response, actor từ JWT, HTTP status, điều phối use case nhỏ | Bất biến dữ liệu phải được bảo vệ bằng SQL/transaction |
| `auth/` | Kiểm tra password/provider và cấp session chung | HTTP transport hoặc quyền tài nguyên nghiệp vụ |
| `token/` | JWT payload, signing và validation | Trạng thái refresh session trong DB và SQL authorization |
| `db/query/*.sql` | Queries, điều kiện ownership/authorization khi cập nhật | HTTP status codes |
| `db/migration/*.sql` | Schema, FK, UNIQUE, CHECK, index | Response mapping |
| Tệp `db/sqlc/*.go` viết tay | Store, transaction và nhiều query dùng chung một commit | Thay đổi tùy tiện tệp do sqlc generate |
| `worker/`, `mail/` | Hàng đợi, task processing, retry và delivery adapters | Tuyên bố exactly-once hoặc kiểm soát quyền HTTP không liên quan |
| `main.go`, `util/config.go` | Cấu hình, bootstrap và shutdown | Nghiệp vụ của từng resource |

**Không bắt buộc lớp Service cho mọi CRUD.** `auth.Login` là hành vi ứng dụng dùng chung thực tế; Organization handler hiện gọi Store trực tiếp. Việc thêm `OrganizationService` chỉ để đồng bộ hình thức không đủ lý do thiết kế.

**Tinh thần DRX phải đúng với hành vi:** `PreLogin` chỉ tiền xử lý request; `SufLogin` chỉ ghi nhận outcome; `IPActive` kiểm soát hạn mức; `Authenticate` xác minh identity; SQL thực thi `Authorize`. Không đổi tên hàm thành một khái niệm mà code chưa thực hiện.

## 3. Quy chuẩn thay đổi source và dữ liệu

- Làm việc từ SHA `dev` được xác minh, trên branch có phạm vi rõ. Kiểm tra `git status` trước đồng bộ; không force-push `dev`, không reset uncommitted changes.
- Migration đã áp dụng là lịch sử bất biến. Khi cần thay schema, tạo migration mới; xét dữ liệu cũ, nullable, FK/UNIQUE, rollback phù hợp và trường hợp concurrent write.
- Chỉnh `db/query/*.sql`, sau đó `make sqlc` khi query thay đổi; code sinh ra phải khớp nguồn SQL. Không sửa tay generated queries. Các file Store/transaction viết tay vẫn là source được sở hữu.
- Phân loại thay đổi có bảo toàn dữ liệu, tương thích và an toàn với volume cũ hay không. Dữ liệu PostgreSQL 14 không gắn trực tiếp vào container PostgreSQL 18; không dùng `docker compose down -v` như thao tác thường ngày.
- Process env/file config phải rõ ưu tiên. Không commit secret, JWT, OTP hoặc verification link thật vào code, log, PR.
- Endpoint mới phải định nghĩa method/path, request policy, xác thực, quyền, lỗi trả về và negative tests.
- Không tự đưa AWS, Kubernetes, microservices, outbox, framework hoặc interface mới nếu chưa có yêu cầu cụ thể và bằng chứng. Đây là nguyên tắc tương xứng, **không phải lệnh cấm vĩnh viễn**.

## 4. Evidence theo đúng mức độ tác động

| Loại thay đổi | Bằng chứng phù hợp |
| --- | --- |
| Chỉ tài liệu hoặc PR template | Đường dẫn hợp lệ, đối chiếu thực tế source, review diff, CI status |
| Middleware / Security | Test thứ tự, deny, non-leakage; Redis concurrency integration nếu thay limiter |
| Auth / JWT / Session | Valid/invalid/expiry/revoke, DB session và mọi con đường password/Google/OTP bị ảnh hưởng |
| Migration / Authorization | Migration DB test riêng; actor có quyền/không quyền; constraint và concurrent writes nếu cần |
| Worker / Email | PostgreSQL + Redis + Mailpit hoặc fake SMTP; retry/failure. Không nhận atomic handoff nếu chưa có proof |
| Docker / Runtime | Compose validation, smoke; WSL và dữ liệu volume cần nghiệm thu riêng |
| Business về sau | Actor/action/resource, owner dữ liệu, lifecycle, invariant và vertical-slice test |

Các lệnh nền tảng khi đủ dependency:

```sh
go test ./...
go build ./...
go vet ./...
docker compose config --quiet
```

CI chạy migrations/integration trên container độc lập. Một số database/E2E test **skip khi chạy local** theo chủ đích. `sh scripts/test-integration.sh` dùng PostgreSQL disposable ở `127.0.0.1:15433`, không phải DB của ứng dụng. `sh scripts/accept-g0.sh` kiểm tra Compose **đã chạy** và không tự tạo/xóa container.

## 5. Quy tắc review và acceptance

Reviewer phải trả lời được:

- Đã đọc source đang vận hành và xác định đúng owner chưa?
- Có nhân bản hoặc phá invariant giữa các provider, request route hoặc Organization membership không?
- Negative paths, concurrency, retry và lỗi được kiểm chứng tương xứng chưa?
- Test/CI thật có kết quả hay chỉ viết danh sách lệnh dự kiến?
- Có tuyên bố nhầm CI thành WSL, backup, production hoặc external-provider E2E không?
- Có đưa thay đổi không liên quan vào PR không?

**Tài liệu này không tự thiết lập GitHub CODEOWNERS, branch protection hoặc reviewer bắt buộc.** Người thực hiện PR cần chỉ ra ai review một thay đổi đa capability theo quy trình thực tế của nhóm; không bịa tên reviewer hoặc nói rằng GitHub đã tự cưỡng chế quy định chưa cấu hình.

## 6. Ví dụ nghiệm thu trên source đang có: account chưa verify email

Hiện `auth.Login` kiểm tra password rồi gọi `IssueSession`, chưa chặn bằng `IsEmailVerified`. Đó là **hiện trạng**, không phải chính sách nghiệp vụ đã được chốt.

Nếu có yêu cầu mới phải chặn account chưa verified:

1. Xác định tác động lên login, link provider, renew token, tạo Organization.
2. Quyết định chính sách theo phương thức đăng nhập hay theo account; đối chiếu `auth.Login`, `auth.IssueSession`, Google và Phone OTP.
3. Chọn **một owner chính** cho policy, không vá riêng `PreLogin` để bỏ sót provider khác.
4. Test verified/unverified trên các route liên quan và chứng minh không bypass.
5. Chỉ merge khi nghiệp vụ đã phê duyệt chính sách.

Các quyết định thay đổi ranh giới kiến trúc được ghi bằng ADR gọn như [ADR-0001](decisions/0001-core-ownership.md). Sửa lỗi nhỏ hoặc typo không cần ADR riêng.
