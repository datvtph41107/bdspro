# ENG-CORE-06 — Application Operation & Module Boundary

**Baseline:** BDSPro `dev@73bf07d2980085f0140daa4b8248de7d53cf16c8`, ngày 25/10/2026. Kế thừa [ENG-CORE-04 — Application Foundation](application-foundation.md), [ENG-CORE-05 — API Error Contract](api-contract-error-semantics.md), [BIZ-01](business-capability-lifecycle.md) và [BIZ-02](business-ownership-resource-lifecycle.md).

**Mục tiêu:** khi các HTTP handlers phối hợp cùng một **application operation** ngoài transport, đặt operation tại **một owner riêng của nghiệp vụ**, dùng lại các primitive hạ tầng đã có. Không tổ chức mỗi endpoint thành Controller → Service → Repository → Interface nếu không có yêu cầu.

## 1. Source observation và quyết định

| Source | Trước | Sau | Vì sao |
| --- | --- | --- | --- |
| `api/organization.go::createOrganization` | Bind/trim name; GetAccountByID(actorAccountID) → CreateOrganizationTx(OwnerPersonID) → HTTP 201 | Bind/trim name → `organization.CreateForAccount(ctx, store, accountID, name)` → HTTP 201 | Chuỗi Account → Person → Create Organization là ứng dụng nghiệp vụ, không phải HTTP transport |
| `api/organization.go::listOrganizations` | GetAccountByID(actorAccountID) → ListOrganizationsForPerson(account.PersonID) → HTTP 200 | `organization.ListForAccount(ctx, store, accountID)` → HTTP 200 | Hai luồng cùng có hợp đồng account/person resolution ổn định |
| `api/organization.go::addOrganizationMember` | Store.AddOrganizationMember với actor và person | **Giữ nguyên** | SQL có owner predicate, một statement; thêm service chỉ bọc lệnh |
| `api/organization.go::updateOrganization` | Store.UpdateOrganization với actor và tên | **Giữ nguyên** | Một SQL UPDATE kiểm tra active owner trong WHERE |
| `db/sqlc/tx_create_organization.go` | Party → Organization → owner membership trong PG transaction | **Giữ nguyên** | Data owner & atomicity thuộc DB Store transaction |
| `db/query/organization.sql` | Membership/owner authority, active visibility | **Giữ nguyên** | Domain invariant đang được bảo vệ tại SQL |

**Kết luận source:** nên tách orchestration cho Create và List vì cả hai cùng dùng một account→person mapping và phối hợp nhiều DB operations. **Không tách tất cả endpoint**; một SQL statement với atomic owner predicate không cần thêm tầng chuyển tiếp.

## 2. Công thức cho ranh giới module (không phải số học)

~~~text
HTTP Boundary = Transport Input + Verified Actor + Status/Response Mapping
Application Operation = Actor Resolution + Use Case Sequencing
Data Boundary = Queries + Constraints + Transaction + Durable Invariants
Execution Evidence = Unit + Integration + Failure/Concurrency as needed
~~~

**Source hiện hành:**

~~~go
// api/organization.go — transport
organization, err := orgapp.CreateForAccount(
    ctx.Request.Context(), server.store, authenticatedAccountID(ctx), name,
)

// organization/operations.go — application orchestration
personID, err := personIDForAccount(ctx, store, accountID)
if err != nil {
    return db.Organization{}, err
}
return store.CreateOrganizationTx(ctx, db.CreateOrganizationTxParams{
    OwnerPersonID: personID, Name: name,
})

// db/sqlc/tx_create_organization.go — PostgreSQL owner
// Creates Party + Organization + initial active owner membership in a transaction.
~~~

- `organization` **không phụ thuộc Gin** và **không tạo interface hoặc constructor mới**; nhận `db.Store` hiện hữu.
- `AccountID` được cấp từ authenticated route, **không chấp nhận tùy ý từ client body**.
- `personIDForAccount` là helper nội bộ cho hai operations; không tạo universal identity/ownership utility.
- Không tự định nghĩa "Organization owner" dựa trên account/person resolution. Resource authorization với Update/AddMember vẫn kiểm tra trong SQL; một JWT hợp lệ không đồng nghĩa có quyền quản lý.
- Return error nguyên bản để HTTP handler giữ **đúng status/body hiện có**. Mọi thay đổi error policy phải là quyết định riêng, không ẩn trong refactor.
- Chuyển `ctx.Request.Context()` từ HTTP vào application, rồi Store; cancellation và request-scoped values không biến mất khi đi qua module.
- Không thay đổi public routes, request validation, response JSON, migration, DB schema, authorization SQL, worker hoặc queue.

## 3. Quy tắc quyết định có tạo application module không?

Tạo module/domain package mới chỉ khi một hoặc nhiều bằng chứng thực tế cho thấy:

1. **Operation có nhiều bước phối hợp và business invariant rõ:** không thể diễn tả đầy đủ bằng một query đơn giản; HTTP không nên sở hữu orchestration.
2. **Cùng hành vi xuất hiện ở nhiều nơi:** ví dụ Create và List cùng resolve Account→Person; việc sửa mapping không phải đi sửa nhiều handlers.
3. **Có hoặc dự kiến thực tế một entrypoint thứ hai** (HTTP, worker, CLI) gọi cùng operation; nhưng không dựng module dựa trên một giả thuyết xa.
4. **Transaction/data owner không bị chuyển sai tầng:** transaction do Store sở hữu; application không thêm thao tác rollback tượng trưng.
5. **Phân tách giảm change amplification thật:** khi quyết định mapping Account→Person đổi, có một owner và test regression; chi phí đọc/điều hướng code không tăng vô nghĩa.

Không dùng "tách lớp = chất lượng". Một handler có thể trực tiếp gọi Store. Một package `organization` không được dùng làm cơ sở để tự động tạo `property`, `listing`, `transaction`, `storage` hay `core/service` khi chưa có use case.

## 4. Evidence và nghiệm thu

**New tests trong `organization/operations_test.go`:**

| Scenario | Invariant được kiểm chứng |
| --- | --- |
| CreateForAccount success | Lookup AccountID 17 → PersonID 42 → owner PersonID 42 của CreateOrganizationTx; context giữ nguyên |
| Create actor lookup failure | Không gọi transaction; không che giấu lỗi |
| Create transaction failure | Không che lỗi từ data boundary |
| ListForAccount success | Lookup actor Person 42 → ListOrganizationsForPerson(42); context giữ nguyên |
| List actor lookup failure | Không truy vấn danh sách nếu không resolve được actor |
| List query failure | Giữ nguyên error, nil result |

**Existing tests:** `api/organization_test.go` bảo vệ HTTP status, JSON và account/owner mapping; `api/organization_business_integration_test.go` kiểm chứng JWT→real SQLStore/PG và membership lifecycle; `db/sqlc/organization_ownership_integrity_test.go` kiểm chứng transaction rollback và uniqueness khi đồng thời.

**CI acceptance:** Go tests (new tests RUN/PASS), migration guard, build, Docker Compose/HTTP smoke, go vet, race detector. Nếu một test fail, không merge. Chỉ có CI disposable PostgreSQL/Redis được sử dụng: không chạy thay đổi trên WSL local/production.

## 5. Non-goals & future work

- Không tạo `OrganizationService`, `OrganizationRepository`, `OrganizationInterface`, registry, DI container hoặc generalized middleware.
- Không gom Update/AddMember vào module chỉ vì module đã tồn tại: chúng vẫn ở handler→Store vì SQL là contract chính.
- Không gọi Application Operation = Legal Ownership: Account, Person và Organization membership khác quyền sở hữu tài sản thực.
- Không thay đổi policy "Organization có luôn ít nhất một owner hay không"; test có fixture owner-ended không phải business permission.
- Nếu xuất hiện nhu cầu xuyên nhiều entrypoints (HTTP/worker) hoặc domain state machine rõ ràng, đánh giá lại số bước, ownership và transaction; refactor theo evidence, không từ bản vẽ sẵn.

**Acceptance conclusion mong đợi:** BDSPro sẽ có một module application nghiệp vụ nhỏ, source-backed, có test rõ ràng và có thể gọi độc lập HTTP; cơ chế vận hành nền tảng vẫn thuộc Gin/pgx/sqlc/Asynq.
