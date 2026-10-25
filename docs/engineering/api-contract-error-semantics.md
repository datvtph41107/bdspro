# ENG-CORE-05 — API Contract & Error Semantics

**Baseline:** BDSPro dev@f5a6980920d3c0cdfeaf8acf4464513c94bc5ac7 (25/10/2026). Kế thừa [ENG-CORE-04 Application Foundation](application-foundation.md), [ENG-GOV-03 Formula Catalog](capability-formula-catalog.md) và [BIZ-01 Business Capability Lifecycle](business-capability-lifecycle.md).

**Intent:** Chuẩn hóa **phần cơ chế HTTP/SQL đang lặp lại thật**, nhưng giữ quyền giải thích lỗi ở từng operation nghiệp vụ. Đây là refactor production nhỏ và evidence qua test; **không phải** một framework ErrorContract mới.

## 1. Bài toán từ source hiện hành

| Source | Trước thay đổi | Điều không được đánh mất |
| --- | --- | --- |
| api/organization.go | Hai handler cùng ParseInt(ctx.Param("id"), 10, 64) và kiểm tra id > 0 | Bad resource ID -> HTTP 400, body \`{"error":"invalid organization"}\`; không gọi Store |
| api/user.go | \`errors.As(err, *pgconn.PgError)\` để kiểm tra unique constraint của email | Chỉ \`23505 + accounts_email_login_uq\` -> 409; unique constraint khác -> 500 |
| api/organization.go | Unwrap PostgreSQL rồi phân loại \`23505/23503\` | Membership đã có -> 409; Person FK lỗi -> 404; owner không active -> 403 qua \`pgx.ErrNoRows\` |
| api/phone_otp.go | Unwrap \`23505\` khi verify phone bind | Phone conflict -> 409; các trường hợp xác minh OTP/credential giữ hành vi riêng |
| api/google.go | Unwrap \`23505\` khi link Google identity | Account linked conflict -> 409; auth/security vẫn thuộc Google handler |

**Shared Mechanism ≠ Shared Business Meaning.** Mã lỗi \`23505\` chỉ nói về unique constraint. Không được mặc định tất cả lỗi ấy luôn là HTTP 409 nếu chưa biết operation và constraint. \`pgx.ErrNoRows\` có thể là 401, 403 hoặc tình huống khác tùy identity, resource và chính sách không tiết lộ sự tồn tại resource.

## 2. Hai primitive thuộc application API owner

### 2.1. Resource identifier

\`api/request_params.go\`:

~~~go
id, validID := positiveInt64Param(ctx, "id")
if !validID {
    ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization"})
    return
}
~~~

Helper chỉ parse một positive int64 ở route params. Không quyết định tên resource, business authorization, HTTP response hoặc source DB. API caller vẫn quyết định thông điệp. \`0\`, số âm, không phải số và overflow đều không hợp lệ. Positive ID không chứng minh resource tồn tại hay actor có quyền trên resource đó.

### 2.2. PostgreSQL error introspection

\`api/postgres_error.go\`:

~~~go
if pgErr := postgresError(err); pgErr != nil && pgErr.Code == "23505" {
    // Chỉ handler hiện tại mới biết conflict này có ý nghĩa gì.
}
~~~

Helper tập trung \`errors.As\` với \`*pgconn.PgError\` (kể cả lỗi bọc bằng \`%w\`); trả \`nil\` với lỗi khác. Nó **không** phân loại lỗi domain, retry, security hay HTTP status. Handler vẫn kiểm tra \`ConstraintName\` ở trường hợp có nhiều nguồn vi phạm, ví dụ email registration.

Không đặt \`postgresError\` ở \`db/sqlc\` chỉ để dùng trong HTTP: \`pgconn.PgError\` là driver detail được API đọc tại ranh giới trả response; domain và SQL operation vẫn sở hữu tính đúng đắn của dữ liệu.

## 3. Hợp đồng lỗi phải bảo vệ (không đổi API)

| Loại outcome | Ví dụ nguồn | HTTP response hiện tại | Owner quyết định |
| --- | --- | --- | --- |
| Invalid request | Bad Organization ID, JSON validation | 400 với thông điệp resource hoặc input | HTTP handler/Pre* |
| Missing credentials | Access JWT thiếu/sai | 401 | Authenticate |
| Authorization denied | SQL owner predicate không khớp (\`pgx.ErrNoRows\`) | 403 ở Organization update/add member | Organization operation |
| Business conflict | Duplicate active membership (\`23505\`), duplicate email đúng constraint | 409 với thông điệp nghiệp vụ | Handler tương ứng |
| Missing target | FK Person trên add member (\`23503\`) | 404 “person unavailable” | Organization operation |
| Infrastructure/unknown failure | DB hoặc provider lỗi không phân loại | 500/503 theo từng handler; không gửi nguyên driver error ra client | Boundary/operation |

**Không áp đặt quy tắc rằng cứ \`pgx.ErrNoRows\` là 404.** Ví dụ: SQL update Organization dùng zero affected rows để không cung cấp quyền ghi cho người không phải owner, nên behavior hiện hành là 403. Thay thế bằng một lỗi generic có thể gây lộ sự tồn tại tài nguyên hoặc đổi contract.

## 4. Viết API tiếp theo theo framework hiện có

~~~text
Route
 → Middleware order (source quota / Pre* / identity quota / Authenticate khi cần)
 → Typed input và resource ID parsing
 → Authenticated account (không đồng nghĩa owner resource)
 → Store / application operation
 → SQL predicate / transaction / constraints
 → Operation-specific error mapping
 → Public response (không lộ secret, SQL detail)
 → Tests: allowed/denied/conflict/infra failure
~~~

**Mở package \`core/\`?** Chưa cần. \`request_params.go\` và \`postgres_error.go\` ở \`api/\` vì hiện tại chỉ HTTP transport cần chúng. Không có consumer hoặc một bất biến xuyên package nào biện minh cho package global.

**Không triển khai:** global catch-all HTTP error middleware, response envelope mới, automatic \`pgx.ErrNoRows\` mapping, logic ngầm retry, new service/repository, business role engine, migrations, DB schema hay worker logic.

## 5. Acceptance / maturity

- **Production diff:** chỉ định vị parsing/unwrap dùng chung, route/status/body/SQL/middleware order không đổi.
- **Contract tests:** \`TestPositiveInt64ParamContract\` (positive/invalid/overflow), \`TestPostgresErrorUnwrapContract\` (nil/other/PG/wrapped), \`TestOrganizationRouteIDsRejectInvalidValuesBeforeStore\`, \`TestAddOrganizationMemberPostgresErrorResponses\` và \`TestRegisterUniqueConstraintIsScopedToEmail\`.
- **Integration regression:** test BIZ-01 Organization HTTP/JWT/PostgreSQL và các test login/OTP/Google; CI Go tests, race detector, build, smoke, vet, migration guard.
- **Evidence scope:** Đây là *Verified* cho các contract đã test. Không suy luận đã chuẩn hóa hoàn toàn mọi Google/Phone failure mapping hoặc đã nghiệm thu WSL/production.
- **Bài học lặp lại:** Tập trung **mechanism** ổn định, không tập trung hóa **policy** thuộc domain. Khi xuất hiện thay đổi thực phải sửa cùng một business rule qua nhiều handler, đánh giá owner use-case trước khi tạo abstraction mới.
