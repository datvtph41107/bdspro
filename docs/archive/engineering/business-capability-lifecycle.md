# BIZ-01 — Business Capability Lifecycle: nghiệp vụ chạy trên Engineering Core

**Baseline:** BDSPro dev@89253752441f6d7367a0dca70262a20393d96b21 (25/10/2026). **Mục tiêu:** biến tư duy Platform × Domain của [ENG-GOV-03](capability-formula-catalog.md) thành phương pháp phát triển một nghiệp vụ thực. Đây là acceptance contract và một ví dụ đã có trong source, **không phải đặc tả BĐS/Listing/Property chưa được chốt**.

**Nguyên tắc:** backend phục vụ *hành vi thực của actor trên resource thuộc phạm vi được phép*, không chỉ API CRUD. Một use case được hiểu xuyên từ HTTP ingress đến trạng thái bền vững, các tác động sau commit và recovery cần thiết.

## 1. Đặt bài toán theo thực thể, hành vi và vòng đời

Công thức mô tả một phép thực thi nghiệp vụ (các ký hiệu là mô hình tư duy, không phải phép tính số học):

~~~
BusinessOperation = Actor + Action + Resource + CurrentState
                  + Invariants + StateTransition + Effects + Evidence
~~~

- **Actor:** identity được xác thực là Account; chủ thể thực hiện tài nguyên có thể là Person/Organization. Không mặc nhiên Account là chủ sở hữu pháp lý.
- **Action:** hành vi có tên và mục đích rõ ràng (CreateOrganization, AddMember, UpdateOrganization); không sinh "Manager" chỉ vì tên gọi.
- **Resource:** Organization hiện có trong DB; Property/Listing/Contract là miền *ứng viên* chưa được thiết kế thành source of truth.
- **CurrentState:** owner/membership còn hiệu lực tại **thời điểm SQL write**, không dựa trên role tự khai báo từ body/JWT.
- **Invariant:** điều không được vi phạm (non-owner không được ghi; membership đã kết thúc không được dùng để cấp quyền).
- **Transition:** chuyển đổi có chủ ý về business state, trong transaction phù hợp. Không gom Redis queue vào PG commit.
- **Effects:** email, event, cache, worker, storage chỉ phát sinh khi nghiệp vụ cần; nêu rõ sync vs async, retry, version và idempotency.
- **Evidence:** test tình huống đúng/sai, HTTP status, database persisted rows, failure/concurrency nếu liên quan.

**Source baseline:** api/server.go định nghĩa middleware Authenticate trên protected routes; api/organization.go lấy AccountID đã verify từ JWT; db/query/organization.sql áp dụng EXISTS role='owner' AND ended_at IS NULL trong SQL write; db/sqlc/store.go và transaction code sở hữu SQL. Người dùng không cung cấp ActorAccountID tùy ý trong body.

## 2. Ecosystem logic: một thay đổi nghiệp vụ đi qua đâu?

~~~
Request / Command
  → HTTP/Gin ingress (observability, admission, validation)
  → Authenticate (access JWT → AccountID)
  → Resolve Actor (accounts.person_id, nếu use case cần)
  → Authorize for resource (SQL predicate theo trạng thái DB)
  → Domain operation (rules + transition)
  → PostgreSQL (query / transaction / constraints)
  → HTTP result
  → Optional effects (task/event/cache/storage) với delivery contract riêng
~~~

Không phải operation nào cũng cần service layer, job, cache hoặc event. Nếu nghiệp vụ chỉ cần một câu UPDATE có predicate quyền và transaction statement atomic, thêm các lớp chuyển tiếp không tạo thêm đúng đắn.

| Lớp | Runtime owner hiện có | Domain responsibility | Boundary không được vượt |
| --- | --- | --- | --- |
| HTTP / ingress | Gin, api/server.go | Contract/request validation | Không tự định nghĩa owner dữ liệu |
| Identity | token.Maker + Authenticate | Chứng minh AccountID | JWT hợp lệ không tự cấp role resource |
| Application action | api/organization.go | Map input, gọi operation, trả status | Không nhận ActorAccountID từ client |
| Data integrity | db/query/organization.sql, PostgreSQL | Owner predicate, membership lifecycle, uniqueness | SQL chỉ thấy **trạng thái tại câu lệnh**, không thay thế policy cần chốt |
| Background | Asynq worker (không tham gia Organization flow hiện có) | Chỉ khi có tác động bất đồng bộ thực tế | Không mặc định at-least-once hay exactly-once theo tên framework |
| Observability / acceptance | Tests, CI, G0 runbook | Ghi bằng chứng thực hiện | CI không tương đương WSL/production |

## 3. Vertical slice thực tế — Organization Actor Lifecycle

### Nghiệp vụ đang hỗ trợ

- Một Account tương ứng Person, có thể tạo Organization; khi tạo, owner membership được tạo trong transaction.
- Owner có thể thêm Person khác làm **member**; hệ thống không chuyển member thành owner.
- Owner có thể đổi tên Organization; member và người sở hữu Organization khác không thể đổi tên đó.
- Member có quyền nhìn Organization khi membership còn active; kết thúc membership thì không còn trong danh sách.
- Owner permission khi update/add member được xét tại SQL statement; trạng thái owner đã kết thúc sẽ không cho phép tiếp tục viết dù JWT còn hiệu lực.
- Ownership được thể hiện bằng membership role trong database, **không** phải field role trong JWT.

### Bằng chứng cần kiểm tra

| Business scenario | Expected HTTP / state | SQL invariant |
| --- | --- | --- |
| Thiếu JWT → tạo Organization | 401, không tạo bản ghi | Auth boundary |
| Account A → tạo Org A | 201, một active owner membership | Transaction create party/org/membership |
| Account C → tạo Org C | 201, một active owner membership | Phân tách resource/actor |
| C sửa Org A; B thêm member vào Org A | 403 | Owner EXISTS predicate |
| A thêm B vào Org A | 201; B nhìn được Org A | Membership insert / list |
| A thêm B lần nữa khi active | 409; không nhân đôi membership | Partial unique constraint |
| B đổi tên Org A | 403; persisted name không thay | SQL UPDATE owner predicate |
| A đổi tên Org A; C đổi tên Org C | 200; đúng resource được cập nhật | Ownership scoped write |
| Kết thúc membership B bằng SQL fixture | GET B không còn Org A | ended_at IS NULL khi list |
| Kết thúc owner A bằng SQL fixture | 403 khi update/add; Org A giữ tên cũ | ended_at IS NULL khi write |

**Test:** api/organization_business_integration_test.go::TestBusinessOrganizationActorLifecycleIntegration dùng HTTP/Gin/JWT thật và SQLStore/PostgreSQL thật. CI truyền BUSINESS_TEST_DB_URL trỏ **chỉ** vào dịch vụ PG disposable ở 127.0.0.1:5432/bdspro. Không dùng MockStore để giả lập authorization. Fixtures tạo Account bằng Store.RegisterTx không gửi email; ending membership được cập nhật **chỉ trong test**, vì source chưa có public API để kết thúc membership.

**Quan trọng:** trực tiếp kết thúc active owner có thể để lại Organization không có owner. Đây là kịch bản âm để chứng minh *access bị thu hồi theo DB state*, **không phải hành vi nghiệp vụ đã được phê duyệt**. Nếu hệ thống cần "Organization phải luôn có ít nhất một owner", đó là một invariant mới phải được xác nhận trước khi thêm migration/API; không dựa vào test fixture để coi là đã chốt.

## 4. Phát triển chức năng nghiệp vụ mới trong hệ sinh thái

Một Developer thêm một hành vi mới áp dụng quy trình **Business Contract → Execution Mapping → Database Invariant → Evidence**:

1. **Business intent:** tác nhân nào cần làm gì; một kết quả thành công và một kết quả bị từ chối; tác động đến dữ liệu nào.
2. **Data owner:** Account/Person/Organization là gì; resource là đối tượng nào; có nguồn sự thật khác phải đồng bộ không.
3. **Operation contract:** command/input/response, transition, thời điểm kiểm tra quyền; nơi transaction bắt đầu và kết thúc.
4. **Mechanism selection:** dùng Gin, SQLC, pgx, Asynq, storage/cache hiện có nếu liên quan; không nhân bản vendor runtime.
5. **Formula / precision nếu liên quan:** trước khi lưu tiền, giá, diện tích: đơn vị, scale, rounding, authoritative inputs, version policy và replay.
6. **Failure model:** invalid input, not found, conflict, concurrent write, rollback, retry, stale cache; chỉ test những trường hợp thuộc impact.
7. **Acceptance:** điều kiện có thật được kiểm tra xuyên lớp; test name/log/CI SHA; xem xét WSL khi liên quan vận hành.
8. **Consolidate:** giữ lại hàm/contract/test hữu ích, không bắt buộc tạo shared package/ADR nếu giá trị tái sử dụng không được chứng minh.

### Reuse ra sao?

- **Knowledge reuse:** hiểu JWT, SQL predicate, PG transaction; nguyên tắc áp dụng nhiều resource.
- **Code reuse:** giữ primitive, shared stable invariant; không đưa chi tiết Membership hay Property vào JWT maker.
- **Operational reuse:** mỗi feature có contract owner, failure, evidence; không tự tạo một queue semantics khác.
- **Data reuse:** quan hệ Person/Organization theo source, không tạo bảng Person hoặc Account mới cho từng provider/business module.

**Cấm hiểu sai:** nghiệp vụ giống Laravel DRX (matching, pricing, commission) không tự biến thành BDSPro requirement. Chỉ khai thác những *pattern* có thể chứng minh cần thiết.

## 5. Current limits / future decisions

- Current source chưa có full **Property/Listing/Lease/Sale** lifecycle; chưa thể phê chuẩn identity pháp lý, transfer ownership, hoa hồng, thuế, định giá hay workflow giao dịch.
- Chưa có API kết thúc membership, chuyển owner, khôi phục Organization không có owner; không giả định test SQL fixture là public contract.
- Email unverified policy, revoke semantics, G0 WSL và PG–Redis atomic handoff là những quyết định đang mở ở Core, không được tùy ý sửa kèm feature Organization.
- Testing member/owner active ở HTTP + PG **không** chứng minh toàn bộ concurrency/lock/isolation hay ngăn race giữa kết thúc membership và lệnh đang chạy; cần workload và invariant cụ thể để mở gate đó.
- Không chạy destructive operation hoặc reset volume development để tạo evidence. Dữ liệu integration được tạo trong CI disposable PostgreSQL.

**Decision:** giữ source hiện hành, dùng business vertical-slice test và contract này làm mẫu thực tế; chưa xây Product/Property/Listing module hoặc "Business Framework" cho đến khi một use case có actor/resource/rule được phê duyệt.

**Chặng nối tiếp:** [BIZ-02 — Actor, Authority, Ownership & Resource Lifecycle](business-ownership-resource-lifecycle.md) phân tích ranh giới Account–Person–Organization, quyền quản trị khác quyền sở hữu pháp lý và DB integrity tests tương ứng.
