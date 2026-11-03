# ENG-CORE-09 — Feature Delivery Standard

**Baseline:** `dev@4f816740d8b1c361b865f38c4e4e2e19d88f55f2`, 25/10/2026. Đây là quy trình **thực hiện một thay đổi API**, không phải framework để tạo thêm controller/service/repository. Dùng chung [Development Contract](development-contract.md), [Application Foundation](application-foundation.md), [Application Operation Boundary](application-operation-boundary.md) và [BIZ-01](business-capability-lifecycle.md).

**Deliverable:** một thay đổi có thể đọc từ yêu cầu → route → input/auth → execution/data owner → public response → test → CI. Developer không phải tự phát minh thứ tự làm việc cho từng feature, nhưng cũng không phải tạo file/abstraction khi không cần.

## 1. Chuỗi quyết định bắt buộc về mặt kỹ thuật, tương xứng với feature

| Bước | Quyết định trước khi code | Bằng chứng/source nếu cần |
| --- | --- | --- |
| **0. Source baseline** | Route và behavior hiện tại ở commit nào? Có operation tương đương chưa? | `api/server.go`, handler, SQL, tests, SHA của `dev` |
| **1. Business contract** | Actor nào yêu cầu command nào trên resource nào; trạng thái và invariant gì? | Actor/Action/Resource/CurrentState, allowed/denied case |
| **2. HTTP admission** | Route public/protected, middleware theo đúng thứ tự, loại input (body/query/path) và validation | `api/server.go`, `Pre*`, `Authenticate`, `positiveInt64Param` |
| **3. Application owner** | Một SQL statement đã đủ chưa? Hay cần chuỗi use case nhiều bước độc lập Gin? | Handler→Store trực tiếp **hoặc** handler→operation→Store |
| **4. Data owner** | Quy tắc nào phải nằm trong statement/transaction/constraint? Có PG–Redis effect không? | `db/query/*.sql`, handwritten Store transaction, migrations |
| **5. Public outcome** | JSON/status khi success/invalid/unauthorized/forbidden/conflict/failure? Có lộ secret? | Operation-specific mapping; không tự động `PgError`→HTTP |
| **6. Evidence** | Positive, negative, no-side-effect, replay, concurrency và runtime nào thật sự ảnh hưởng? | Unit/HTTP contract/PG integration/CI, ghi rõ test skip |
| **7. Review & merge** | Diff có đúng owner? CI actual RUN/PASS? Chưa chứng minh điều gì? | PR template, link CI, merge SHA, regression limit |

**Lộ trình triển khai:** xác định trước **một allowed scenario** và **một denied scenario**; map source owner, thực hiện thay đổi nhỏ nhất rồi test tương ứng. Không mở rộng sang Property/Listing chỉ vì ví dụ Organization đã chạy.

## 2. Hai kiến trúc hợp lệ cùng tồn tại trong BDSPro

### Archetype A — Operation có nhiều bước: `POST /organizations`

```text
api/server.go: Authenticate(access JWT)
  → api/organization.go: parse name + trim + authenticatedAccountID
  → organization.CreateForAccount(ctx, store, accountID, name)
      → db.Store.GetAccountByID(ctx, accountID)      [resolve Person]
      → db.Store.CreateOrganizationTx(ctx, params)  [Party + Organization + owner Membership]
  → api: 201 + Organization  |  400/401/500 theo boundary
```

**Source of truth:** không cho request body tự cấp `actor_account_id` hay `owner_person_id`. JWT access được xác thực là nguồn `AccountID`, Store tạo owner dựa trên `accounts.person_id`. `CreateOrganizationTx` sở hữu transaction PG; nó không do HTTP handler điều khiển.

**Evidence hiện có:** `organization/operations_test.go`, `db/sqlc/organization_ownership_integrity_test.go`, `api/organization_business_integration_test.go`. **Evidence thêm ở ENG-CORE-09:** `TestFeatureDeliveryCreateUsesOperationWithVerifiedActorAndContext` pin actor identity, input normalization và request context truyền qua hai Store calls, kể cả khi client gửi thêm field mạo danh.

### Archetype B — SQL statement đủ bảo vệ invariant: `POST /organizations/:id/members`

```text
api/server.go: Authenticate(access JWT)
  → api/organization.go: positiveInt64Param + body.PersonID
  → db.Store.AddOrganizationMember(ctx, orgID, personID, actorAccountID)
      → SQL: INSERT ... SELECT ... WHERE EXISTS(active owner membership)
      → PostgreSQL partial unique index: active membership cannot be duplicated
  → api: 201 | 400 | 401 | 403 | 404 | 409 | 500
```

**Source of truth:** `role='owner'` được xác nhận từ DB ở chính câu ghi, không phải field JSON, không phải JWT role và không phải utility `AuthorizeEverything`. Client không thể yêu cầu `role='owner'` qua body; SQL insert `'member'`.

**Evidence hiện có:** `api/organization_test.go`, `api/api_error_contract_test.go`, `api/organization_business_integration_test.go`, `db/sqlc/organization_ownership_integrity_test.go`. **Evidence thêm:** `TestFeatureDeliveryMemberWriteKeepsAuthorityAtSQLBoundary` pin actor từ JWT, payload quyền, context và 201/403.

**Kết luận:** **không có một đường ống cố định controller→service→repository cho mọi API**. Chọn owner theo ranh giới của invariant và số bước, không theo tên file.

## 3. Executable reference: các test buộc giữ hợp đồng

`api/feature_delivery_contract_test.go` dùng **các route thực của `NewServer`** và `Handler()`, không tự đăng ký route giả trong test:

| Test | Điều kiện được pin | Bằng chứng không đạt |
| --- | --- | --- |
| `TestFeatureDeliveryProtectedRoutesRejectBeforeStore` | `/users/me` và toàn bộ 4 Organization routes từ chối missing/wrong-purpose/malformed JWT; response 401 có request ID; không chạm Store | Mất Authenticate hoặc xử lý business trước xác minh |
| `TestFeatureDeliveryOrganizationInvalidInputStopsBeforeStore` | Bad JSON, blank name, invalid org ID, missing/negative PersonID bị 400 trước Store | Mutation được gọi với dữ liệu invalid |
| `TestFeatureDeliveryCreateUsesOperationWithVerifiedActorAndContext` | JWT AccountID → Account.PersonID → CreateOrganizationTx; trim name, ignore actor/owner fields mạo danh, pass request context | Client tự quyết định actor/owner, operation mất context |
| `TestFeatureDeliveryMemberWriteKeepsAuthorityAtSQLBoundary` | Handler chỉ truyền verified actor vào SQL operation; role là `member`; SQL denied -> 403, error mapping không đổi | Client tự cấp owner role, handler quyết định quyền từ body |

Đây là **regression contract cho code hiện hữu**, không khẳng định đầy đủ bảo mật/rate limit của mọi endpoint. Các test BIZ-01 (HTTP/JWT/PostgreSQL) và BIZ-02 (transaction/concurrency) vẫn cần chạy để kiểm chứng DB thực.

## 4. Khi nhận một yêu cầu API mới

Ví dụ một yêu cầu giả định: “người quản trị Organization có thể thay đổi thông tin hiển thị”.

- **Tìm source:** `PATCH /organizations/:id` đã làm gì; có cần thêm route hay mở rộng payload của route hiện có?
- **Vẽ bằng câu văn một allowed và một denied path:** actor có owner membership đang active được phép; member/no active owner không được phép.
- **Chọn dữ liệu authoritative:** `organizations` và `organization_memberships`, không dùng client-supplied `role`.
- **Chọn code owner:** với một `UPDATE ... WHERE EXISTS(owner)`, handler gọi Store; nếu phải tạo nhiều đối tượng với một transaction, đánh giá operation/Store transaction.
- **Chọn error/response:** không tự đổi 403 thành 404 hay 500 vì refactor; nếu thay public contract, quyết định trong PR riêng.
- **Test:** body/ID invalid không gọi Store; actor khác không sửa được; persisted state không đổi khi bị từ chối; concurrency nếu invariant bị race.
- **Kết thúc:** ghi SHA nguồn, test names, CI RUN/PASS và những ràng buộc chưa được chứng minh.

Đây là ví dụ về **phương pháp**, không phải một feature mới đã được phê duyệt. Không thêm API/DB schema “cho tương lai”.

## 5. Loại bỏ những thao tác lặp không cần thiết

- **Dùng lại:** `preJSON[T]` cho body cần pre-admission, `authenticatedAccountID`, `positiveInt64Param`, `postgresError`, `newLoginUserResponse`, `auth.IssueSession`, `db.Store` và owner-specific operations khi chúng thực sự phù hợp.
- **Không nhân bản:** HTTP JWT parsing, Account→Person mapping vào mỗi handler khi đã có organization operation, PostgreSQL constraints vào validator Go bất đồng bộ, worker delivery policy vào business logic.
- **Không tự động thêm:** `core/`, BaseController, BaseService, ORM abstraction, generic error envelope, route generator hoặc “domain event” cho mọi CRUD.
- **Đổi architecture khi có chứng cứ:** số nơi sửa cùng invariant, trùng failure handling, import ngược hoặc transaction sai owner; ghi sự kiện và test tương ứng. Đếm LOC hay số file không tự chứng minh kiến trúc tốt hơn.

## 6. Acceptance của ENG-CORE-09

- **Thay đổi:** một tài liệu quy chuẩn có ví dụ theo source, test contract cho hai route archetypes, cập nhật tài liệu discovery/review. Không đổi production Go, auth/security policy, SQL, migration, worker, API status/response.
- **Nghiệm thu CI:** 4 test mới phải **RUN/PASS**; toàn bộ Go tests, BIZ-01/BIZ-02 DB tests, migration guard, build/Compose, HTTP smoke, vet và race detector thành công.
- **Không tuyên bố:** generator hoặc scaffold hoàn chỉnh, năng suất đã tăng (chưa đo), WSL production acceptance, Property/Listing đã triển khai, non-atomic PG/Redis đã khắc phục.
- **Cách dùng ở chu kỳ sau:** mở PR theo các bước trong phần 1, chỉ link những contract/tests thực tế liên quan. Khi một API mới phát sinh lỗi lặp, sửa ở owner chung theo bằng chứng rồi cập nhật tài liệu, không mở chương mới theo hình thức.
