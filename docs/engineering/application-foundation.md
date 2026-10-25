# ENG-CORE-04 — Application Foundation & Repeatable API Development

**Baseline (before refactor):** BDSPro \`dev@393f7277fff47603cf4911243bf728506346e351\`, 25/10/2026.
**Intent:** Đưa các công thức ENG-GOV-03 vào **source production** với hợp đồng tái sử dụng thực sự, để việc viết API trở nên nhất quán, có thể kiểm chứng và thay đổi lâu dài. Không cần tự viết lại \`net/http\`, Gin, PostgreSQL, sqlc, Asynq hoặc tạo một framework nội bộ.

**Không phải:** quy định cứ mỗi API phải có controller/service/repository/interface, hay một package \`core\` ôm mọi thứ. Xem [Core Ownership](core-ownership.md), [Development Contract](development-contract.md) và [Business Capability Lifecycle](business-capability-lifecycle.md).

## 1. Kiến trúc hiện hữu — đâu là application-owned core?

| Responsibility | Canonical source | Ai cung cấp runtime | Cái gì được dùng lại? |
| --- | --- | --- | --- |
| Composition & lifecycle | \`main.go\`, \`api.NewServer\`, \`util/config.go\` | Go, pgx, Redis/Asynq | Khởi tạo dependency, startup/shutdown, context |
| HTTP ingress | \`api/server.go\`, \`api/request_limits.go\` | net/http, Gin, cors | Đăng ký route, header/CORS, ingress, observation |
| Pre-admission | \`api/middleware_prelogin.go\`, \`api/middleware_ipactive.go\` | Gin + Redis | Typed input đã bind; source/identity quota; fail-closed theo policy |
| Authenticated actor | \`api/middleware.go\`, \`api/actor.go\` | JWT maker + Gin context | \`authenticatedAccountID(ctx)\` sau \`Authenticate\` |
| Session representation | \`api/user.go\`: \`newLoginUserResponse\` | Go JSON/Gin | Response chung cho password/Google/Phone OTP |
| Domain execution | \`auth/\`, \`api/organization.go\`, SQL queries | Application code | Owner policy, status transition, explicit Store operations |
| Data persistence | \`db/query/\`, \`db/sqlc/\` | PostgreSQL, pgx, sqlc | Statement predicates, constraints, transaction \`execTx\` |
| Async jobs | \`worker/\` | Asynq, Redis | Task type/payload, handler, retries, monitoring |
| Verification | Go tests, disposable CI PG/Redis, G0 WSL | go test/GitHub Actions | Kiểm chứng scope tương ứng, không giả production ready |

Từ "core" ở đây chỉ những **hợp đồng và capability có một nơi sở hữu**, không đòi hỏi một thư mục vật lý tên \`core\`. \`token.Maker\`, \`db.Store\`, \`auth.IssueSession\` là core components theo nghĩa này mặc dù ở những package khác nhau.

## 2. Những thay đổi production đầu tiên (source evidence)

### 2.0. Typed JSON admission — cơ chế lõi cho một route

**Trước:** `PreLogin`, `PreRegister`, `PreToken`, `PrePhoneBind`, `PrePhoneVerify` và `PrePhoneLoginStart` sao chép cùng chu trình bind JSON, lỗi/abort, `ctx.Set`, `ctx.Next`.

**Sau:** `api/request_binding.go` sở hữu `preJSON[T](contextKey)`, còn các middleware `Pre*` vẫn giữ nguyên tên, vị trí route và typed input key:

~~~go
func PreLogin() gin.HandlerFunc {
    return preJSON[loginUserRequest](loginRequestContextKey)
}
~~~

- Bad JSON hoặc validation sai tiếp tục trả 400, abort và không chạy handler sau; input hợp lệ được lưu với kiểu đã bind.
- BDSPro không dùng `PreJSON` để xác thực người dùng, phân quyền, giới hạn lượt truy cập hoặc đổi response nghiệp vụ.
- `PreVerifyEmail` đọc query parameters nên **không** dùng chung JSON binder; Google callback cũng có hợp đồng khác.
- Nếu API không cần typed admission trước rate limiting, handler có thể bind trực tiếp; không áp `Pre*` máy móc cho mọi route.
- Không cần phản chiếu type, registry middleware hoặc code generator.

### 2.1. Authenticated actor

**Trước:** các handler Organization, Phone và Google tự đọc \`ctx.MustGet(authorizationPayloadKey).(*token.Payload).AccountID\` riêng rẽ.

**Sau:** \`api/actor.go\` cung cấp một operation cấp package:

~~~go
actorAccountID := authenticatedAccountID(ctx)
~~~

- **Precondition:** Route đã chạy \`Authenticate\` với access JWT hợp lệ. Hàm cố tình không trả \`0\` hay tạo fallback khi không có payload; thiếu middleware là lỗi route contract.
- **Không thực hiện:** kiểm tra role Organization, chuyển AccountID thành PersonID, hoặc quyết định quyền với Property. Mọi resource-specific authority vẫn ở SQL/operation.
- **Dùng lại:** \`/users/me\`, Organization handlers, Google-link, Phone-bind handlers. Không dùng trong route công khai.

### 2.2. Session response

**Trước:** password login, Google callback và Phone OTP verify dựng cùng cấu trúc \`loginUserResponse\` ở ba nơi.

**Sau:** một function thuần \`newLoginUserResponse(result auth.LoginResult)\` sở hữu cách chuyển domain session thành HTTP JSON.

- Authentication và \`IssueSession\` vẫn ở \`auth/\`; logic xác minh password/Google/OTP **không hợp nhất**, vì bản chất khác nhau.
- Chuyển một output chung tránh drift về tên JSON field, expiration, user projection; \`PasswordHash\` không được đưa ra response.
- Khi thay đổi public session contract, một nơi lập mapping và các route provider dùng lại; test kiểm tra field và không lộ password hash.

### 2.3. Request context lifecycle

Các handler truyền \`ctx.Request.Context()\` xuống các operations I/O: login \`auth.Login\` và Google Redis, Store, token verification/exchange, issue session. Gin \`*Context\` dùng cho HTTP routing/request binding/response, còn Go request context giữ propagation của cancellation, deadline, tracing và values. **Không tự tạo context.Background() trong xử lý request**.

Các phép thử nguồn có \`TestPasswordLoginPropagatesHTTPRequestContext\` để xác minh Store nhìn thấy context values của HTTP request. Điều này **không** chứng minh toàn bộ Google/OIDC provider I/O cancellation; E2E provider cần kiểm chứng riêng.

## 3. Công thức viết một API (API Change Recipe)

~~~text
Requirement / actor/action/resource
    ↓
Route & admission order
    ↓
Typed request parse + validation (Pre* nếu cần rate limit theo identity)
    ↓
Authenticate → authenticatedAccountID (chỉ route protected)
    ↓
Operation (small handler → Store hoặc auth application function)
    ↓
PostgreSQL predicate / transaction / constraint (nếu mutate state)
    ↓
Explicit mapping: success / deny / conflict / failure
    ↓
Optional async effect (nêu boundary/retry/ordering riêng)
    ↓
Positive + negative + failure tests
~~~

Một handler có thể gọi thẳng Store khi **SQL statement đã sở hữu trọn vẹn rule và atomic update**; không cần \`Service\` chỉ vì hướng tutorial đặt tên. Nhưng nếu operation phải phối hợp nhiều bước trong một transaction hoặc dùng chung giữa entrypoints, hãy đưa hành vi đó về owner thích hợp (\`auth/\`, Store transaction hoặc domain-specific service có lý do rõ).

### Checklist tối thiểu theo tác động, không phải template cứng

| Bước | Việc developer phải xác định |
| --- | --- |
| Route | Method, path, protected/public, source admission / typed parse ordering |
| Input | Binding/normalization, chuẩn field, giới hạn body/query, lỗi input |
| Identity | Ai được chứng minh qua JWT? Có cần resolve Person/Organization? |
| Authorization | Actor được phép làm action này với resource nào theo trạng thái DB hiện tại? |
| Operation | Trong handler hay function nghiệp vụ? Ai là owner của transaction? |
| Output | JSON public contract, status code và lỗi không rò rỉ secrets |
| Effects | Có enqueue/storage/cache/event? Transaction boundary và recovery ra sao? |
| Evidence | Route tests, PostgreSQL integration, concurrency/retry nếu có ảnh hưởng |
| Reuse | Shared invariant & stable behavior hay chỉ cú pháp giống nhau? |

### Một ví dụ source đã có — \`PATCH /organizations/:id\`

1. \`api/server.go\` đăng ký route ở nhóm \`Authenticate\`.
2. \`api/organization.go\` parse ID và JSON, đọc \`authenticatedAccountID\`.
3. \`db/query/organization.sql\` dùng \`EXISTS\` để kiểm tra active owner ngay trong câu UPDATE.
4. Database trả row hoặc \`pgx.ErrNoRows\` → HTTP 200 hoặc 403, không cần middleware authorization tổng quát.
5. \`api/organization_business_integration_test.go\` đối chiếu ba actors, trạng thái DB và trường hợp owner hết hiệu lực.

**Không trừu tượng hóa:** bước (3) thành hàm \`AuthorizeAllResources()\`; hệ quả bảo mật phụ thuộc action, resource và SQL transaction.

## 4. Khi nào tạo thư mục hoặc package core mới?

**Quy tắc quyết định:** một package mới nên có _stable responsibility + distinct change reason + real consumer(s) + testable contract_. Không tạo package cho mỗi tên giống Laravel.

- **Có thể giữ ngay trong \`api/\`:** lấy actor từ authenticated route, public response projection, parser helper chỉ dành cho một transport.
- **Giữ trong \`auth/\`:** xác minh chứng cứ đăng nhập, phát session; HTTP không được chứa business auth rules.
- **Giữ trong \`db/query\` và \`db/sqlc\`:** authorization predicate/transaction, constraints và các thao tác persist; không thêm DAO chỉ để bọc sqlc.
- **Giữ trong \`worker/\`:** task contract với Asynq, retry semantics; một job không biến thành event bus.
- **Chỉ cân nhắc package riêng sau này:** \`money\` nếu có nhiều nghiệp vụ tiền thật với decimal + rounding invariant chung; \`storage\` nếu có property media và yêu cầu adapter/test; \`scheduler\` nếu có recurring business job thực. Tên và vị trí package quyết định theo owner khi use case xuất hiện, không theo blueprint dựng sẵn.

### Chống lặp lại sai lệch

- Không sao chép payload JWT parsing vào mỗi handler.
- Không nhân đôi HTTP response mapping giữa password/Google/Phone.
- Không gọi \`ctx\` của Gin thay \`ctx.Request.Context()\` khi thực thi I/O.
- Không trả \`account_id=0\` khi thiếu auth payload để “tránh panic”; thiếu \`Authenticate\` phải được sửa ngay tại route registration.
- Không tạo generic \`APIHandler[T]\`/reflection router/code generator trước khi ít nhất một loạt route có invariants ổn định và đo được lợi ích.
- Không gom khác biệt nghiệp vụ như role/fee/rounding vào utility HTTP.

## 5. Chọn thời điểm refactor và kiểm chứng

**Refactor đủ căn cứ** khi (a) cơ chế đang lặp nhiều nơi và có chung invariant, (b) phải sửa cùng hành vi nhiều file, hoặc (c) sai lệch hiện hữu làm tăng rủi ro. Thay đổi đầu tiên này đúng cả (a) và (b), đặc biệt session projection và authenticated actor.

**Acceptance scope cho PR này:** sản xuất code thay đổi tối thiểu ở những handler sử dụng cơ chế chung; output/status/error mapping giữ nguyên; Go tests cho actor, session projection, HTTP context; CI build/smoke/race thành công. Không thay đổi Postgres schema, queue, auth policy, signed token rules hoặc API endpoint. Đây là một bước **Application Foundation**, không phải tuyên bố codebase đã là platform hoàn thiện.

**Hướng tiếp theo:** từ các API thực, tiếp tục đo “change amplification” (số nơi sửa khi thay cùng policy), lỗi drift và vị trí owner. Chỉ mở API error contract, resource ID parser, DI abstraction hoặc package core riêng khi nhóm tác nhân đó được ghi nhận trong một thay đổi thật.
