# BIZ-02 — Actor, Authority, Ownership & Resource Lifecycle

**Baseline:** source \`dev@2c1eb246d3f1275bdf12c774c5dd1d998725d142\`, ngày 25/10/2026. **Mục tiêu:** phân biệt danh tính, chủ thể, quan hệ quản lý, quyền thực thi và quyền sở hữu tài sản trước khi triển khai mô hình Property/Listing. Nối tiếp [BIZ-01](business-capability-lifecycle.md) và [ENG-GOV-03](capability-formula-catalog.md).

**Trạng thái:** đây là **source-backed contract và test evidence cho Organization hiện có**, không phải đặc tả chủ sở hữu hợp pháp của bất động sản. Không thêm schema, API, runtime, Pricing Engine, Event Bus hoặc Business Framework.

## 1. Bốn thực thể/quan hệ không được đánh đồng

| Khái niệm | Source hiện có | Ý nghĩa đúng | Không được tự suy ra |
| --- | --- | --- | --- |
| \`Account\` | accounts.id, accounts.person_id; auth/token | Định danh đăng nhập, JWT access subject = AccountID | AccountID là mã Person hoặc chủ tài sản |
| \`Person\` | parties + persons; accounts.person_id | Chủ thể con người đứng sau một account | Mọi Person đều có quyền hành động trong mọi Organization |
| \`Organization\` | parties + organizations.party_id | Thực thể tổ chức độc lập | Organization đang là chủ sở hữu bất động sản hoặc chính là tài khoản |
| \`OrganizationMembership\` | organization_memberships, role, joined_at, ended_at | Quan hệ tham gia, vai trò **quản trị trong Organization** | \`role='owner'\` là bằng chứng quyền sở hữu pháp lý đối với bất động sản |

**Luận điểm:** \`Role=owner\` hiện chỉ là authority trong phạm vi Organization. Không sử dụng lại chuỗi "owner" đó để xác nhận land title, property ownership, legal representation, brokerage mandates hoặc quyền bán/cho thuê.

## 2. Cách quyền được thực thi trên source hiện hành

- HTTP \`Authenticate\` xác minh access JWT và đặt \`AccountID\` trong context, không tự suy luận resource roles.
- \`api/organization.go\` lấy \`AccountID\` đã xác minh; khi tạo Organization, resolve \`accounts.person_id\` rồi gọi \`CreateOrganizationTx(Name, OwnerPersonID)\`.
- \`db/sqlc/tx_create_organization.go\` tạo \`Party → Organization → owner Membership\` trong một PostgreSQL transaction. Nếu insert membership lỗi, toàn transaction phải rollback.
- \`db/query/organization.sql\` giới hạn \`UpdateOrganization\` và \`AddOrganizationMember\` bằng \`EXISTS\` join accounts↔membership: actor phải đang có \`role='owner'\` với \`ended_at IS NULL\` trong Organization được chọn.
- \`ListOrganizationsForPerson\` trả danh sách theo membership active. Nó không cấp quyền write.
- \`AddOrganizationMember\` luôn cấp \`role='member'\`. Member không được tự thăng quyền thành owner chỉ bằng request body.

**Ownership authority predicate (đơn giản hóa):**

~~~text
CanManageOrganization(actorAccount, organization, dbState)
 = EXISTS active membership(
     person_id = accounts.person_id for actorAccount,
     organization_id = organization,
     role = 'owner',
     ended_at IS NULL
   )
~~~

Predicate là chính sách quyền quản trị Organization hiện có. Nó **không phải** công thức cho tài sản hay hợp đồng.

## 3. Two-track formula: Platform × Domain

| Axis | F13 — Platform FA | F14 — Domain FB |
| --- | --- | --- |
| Entry | Gin route, token.Maker, context | Account thực thi thao tác |
| Owner | PostgreSQL/sqlc Store, JWT middleware | Organization và quan hệ membership |
| Unit / Precision | Không áp dụng | Không áp dụng ở membership; không tự gán quy tắc Money |
| Transaction | \`execTx\` cho CreateOrganizationTx, PostgreSQL constraint cho insert | Hoặc cùng tạo đủ owner/resource, hoặc không tạo gì |
| Authorization | Access token hợp lệ là điều kiện cần; SQL là điểm write authority | Owner role còn active trong đúng Organization |
| Failure | FK 23503, unique 23505, SQL no rows, rollback | Không để orphan Party/Org; không nhân đôi active membership |
| Test | Integration DB disposable và HTTP/PostgreSQL đã có | Negative actor, lifecycle và concurrent writes |
| Non-goals | Không tạo DI provider/event bus | Không quyết định Property/Listing/legal title |

## 4. Các bằng chứng hiện có và mới

**BIZ-01:** \`api/organization_business_integration_test.go::TestBusinessOrganizationActorLifecycleIntegration\` kiểm tra HTTP/JWT/DB với 3 actors: owner, member, other owner; 401/403, membership active/ended và persisted name. SQL trực tiếp kết thúc membership **chỉ tạo trạng thái test**, không phải public API hoặc quyết định rằng Organization được phép ownerless.

**BIZ-02 mới:** \`db/sqlc/organization_ownership_integrity_test.go\` bổ sung hai gate:

1. \`TestCreateOrganizationTxMissingOwnerRollsBackAllEntities\` — gán OwnerPersonID không tồn tại; PostgreSQL FK phải từ chối; Party/Organization/Membership đã tạo trong transaction đều **không được commit**. Generated \`PartyID\` trong Go object khi có lỗi là ID chưa commit, không phải bằng chứng resource tồn tại.
2. \`TestConcurrentOrganizationMemberAdditionEnforcesOneActiveMembership\` — tám thao tác đồng thời cùng thêm một Person vào một Organization bằng owner thật; kỳ vọng một thành công, còn lại bị PostgreSQL unique violation; DB tồn tại đúng một active membership. Gate này xác nhận uniqueness **tại thời điểm insert**, không bảo đảm serializable behavior với đồng thời thu hồi owner hoặc transfer ownership.

**Safety:** sử dụng \`openTestDB(t)\` sẵn có, test DB chỉ hoạt động trên GitHub Actions disposable PostgreSQL hoặc môi trường local disposable có guard \`BDSPRO_INTEGRATION_DB=1\` và cluster name riêng ở :15433. Không cần tạo/cắt volume và không dùng DB sản phẩm.

## 5. Resource lifecycle: những trạng thái chưa được quyết định

Một resource trong hệ sinh thái nghiệp vụ không được mặc định có state machine giống Organization Membership:

| Miền tương lai | Cần phân biệt từ đầu | Chưa được phê duyệt |
| --- | --- | --- |
| Property | Real-world asset, representation, legal owner, delegated manager, documents | Property schema, title verification, sole vs joint ownership |
| Listing | Người tạo, actor được phép publish, owner/agent, nguồn Property, visibility | Draft/Published/Closed/Expired semantics |
| Brokerage mandate | Người ủy quyền, đơn vị đại diện, thời hạn và phạm vi ủy quyền | Authorization giao dịch bất động sản |
| Contract | Các bên, điều khoản, ký kết, revision, cancellation | Tính hợp lệ pháp lý hoặc workflow ký |
| Pricing/Fees | Unit, source amount, rounding/scale, snapshot/version | Giá tính theo m², phí, thuế, hoa hồng |
| Files/Storage | File owner, resource reference, quyền đọc/xóa, retention | Local/S3 provider hoặc cleanup cron |
| Async effects | Sau commit phát event nào, idempotency/retry, nguồn sự thật | Outbox hoặc exactly-once guarantee |

**Câu hỏi bắt buộc trước khi thêm Property:** bên nào nắm **quyền sở hữu pháp lý**, bên nào được **quản lý thực tế trên ứng dụng**, bên nào là **actor xác thực**, và tài liệu/chứng cứ nào là authoritative. Bốn vai trò có thể khác nhau. Trường hợp BDSPro chỉ muốn quản lý tin đăng mà không xác minh quyền sở hữu pháp lý cũng là một mô hình sản phẩm khác, không thể tự chọn thay người thiết kế nghiệp vụ.

## 6. Quy tắc mở rộng dùng chung cho developer

**Business Contract → Owner Mapping → Invariant → Execution → Persistence → Effects → Evidence.**

Trước khi tạo bất kỳ route, table, service, worker hay formula nào:

1. Chỉ ra actor/action/resource và owner trạng thái; phân biệt identity, management authority và legal/business ownership.
2. Xác định valid state transition, persisted invariant, trường hợp bị từ chối và rollback. Không coi \`owner\` string là universal authorization.
3. Dùng cơ chế Gin/pgx/sqlc/Asynq hiện hữu; thêm package/interface khi có capability đa entrypoint hoặc bất biến cần owner mới.
4. Tổ chức calculation thuần nếu có đơn vị số học rõ; không tự dùng floating point hoặc decimal package chỉ vì một ví dụ DRX.
5. Quyết định background effect **sau** khi chốt transaction và delivery contract; không giả định queue/event đồng bộ với commit.
6. Xác minh negative, concurrency, replay và side-effect theo tác động thật; giữ evidence và vấn đề mở trong PR hoặc runbook.
7. Nhãn maturity: **Understood** (biết contract), **Verified** (test scoped), **Operational** (đã thử runtime/recovery thực tế). Không tự nâng cấp nhãn theo CI xanh.

## 7. Quyết định và acceptance

- **Thực hiện:** bổ sung DB test owner FK rollback và concurrent membership uniqueness; giữ lại source hiện hành.
- **Hoãn:** role transfer, closing owner, ownerless Organization repair, multi-owner policy; cần business decision trước.
- **Không thực hiện:** migration, schema Property/Listing, service/controller chung, state engine, thư viện decimal/storage, production hardening.
- **Acceptance:** CI phải cho thấy **cả hai test mới RUN + PASS** trong môi trường PostgreSQL disposable; migration guard, Go tests/race, build, vet và HTTP smoke thành công; không tuyên bố WSL/production hoặc concurrency transfer ownership đã được nghiệm thu.

**Điểm chốt:** phân loại đúng authority là điều kiện cần để thiết kế đúng ownership. Tránh mở rộng những quy tắc quản trị Organization thành chứng cứ sở hữu tài sản khi chưa có một business contract về tài sản.
