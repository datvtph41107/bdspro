# CORE-01 → CORE-05 — Tài liệu kiến trúc, quyết định và nghiệm thu

**Ngôn ngữ chính:** tiếng Việt. **Repository:** \`datvtph41107/bdspro\`, nhánh triển khai \`dev\`.  
**Nguồn thiết kế:** canonical V0.4 gồm 164 **bảng logical ứng viên**, không mặc nhiên là schema đã cài đặt.  
**Triển khai:** Go 1.27.1, PostgreSQL 18, pgx/v5, sqlc, golang-migrate.

## A. Mục tiêu thực tế

Ta không xây bảng rời rạc chỉ để giống một sơ đồ 164 bảng. Chúng ta chứng minh một hành trình có đối tượng, tác nhân và dòng tiền/quyền phù hợp:

1. An đăng ký Account, xác minh email, đăng nhập và giữ Session.
2. An gửi yêu cầu đăng ký Công ty ABC. Tại thời điểm gửi, **chưa có quyền quản trị ABC**.
3. Một reviewer có thẩm quyền nền tảng xét hồ sơ và quyết định. Phê duyệt tạo Organization cùng Membership/Role quản trị workspace theo transaction.
4. An mời Lan. Lan chỉ được cấp Membership khi **chấp nhận** lời mời bằng Account có email đã xác minh tương ứng.
5. Lan nhận tin Zalo về căn nhà khoảng 75 m², người cung cấp chào 4,2 tỷ. Lan lưu Source Intake kèm báo giá, không tạo Property/Listing giả.
6. Ngày sau Lan ghi nhận lời báo giá 4,1 tỷ; hệ thống **giữ cả hai phát biểu**. Người ngoài ABC không được đọc hoặc ghi nhầm nguồn của ABC.

Đó là lý do Identity, Authentication, Organization và Source được triển khai thành một lát cắt xuyên suốt.

## B. Trách nhiệm của từng bảng

| Sự thật cần lưu | Thành phần | Điều nó **không** tự chứng minh |
| --- | --- | --- |
| Chủ thể nghiệp vụ ổn định | \`parties\`, \`persons\`, \`organizations\` | Khả năng đăng nhập hay quyền đại diện pháp lý |
| Thông tin hiển thị | \`person_profiles\` | Danh tính xác thực |
| Danh tính truy cập số | \`accounts\` | Quyền hành động trong bất kỳ Organization nào |
| Liên hệ và chứng cứ kiểm tra email | \`account_emails\`, \`email_verification_tokens\` | Người dùng là người đại diện tổ chức |
| Mật khẩu và phiên đăng nhập | \`password_credentials\`, \`sessions\` | Membership hoặc Role của tổ chức |
| Một đợt tham gia của Person vào Org | \`organization_memberships\` | Quyền pháp lý với doanh nghiệp |
| Quyền và phân quyền có thời gian | \`permissions\`, \`organization_roles\`, \`organization_role_assignments\`, \`organization_membership_blocks\` | Quyền bất biến sau khi bị thu hồi |
| Ý định được mời | \`organization_invitations\` | Người được mời đã chấp thuận |
| Tuyên bố đăng ký/tiếp quản | \`organization_registration_requests\`, \`organization_claim_requests\` | Tuyên bố đã được cơ quan pháp lý xác nhận |
| Quyền reviewer cấp nền tảng | \`platform_organization_reviewers\` | Người quản trị workspace nào đó tự thành reviewer |
| Một tin nguồn còn chưa xác minh | \`source_intakes\` | Property có thật/chính chủ hoặc Listing được phép đăng |
| Một lời báo giá được ghi nhận | \`source_price_reports\` | Giá hiện hành chính thức |

## C. Bất biến kỹ thuật, cơ chế và giới hạn

| Bất biến | Cơ chế bảo vệ hiện có | Trách nhiệm chưa giải quyết |
| --- | --- | --- |
| **C01:** một Party có đúng một subtype Person/Organization | \`kind\`, FK tổng hợp, constraint trigger \`DEFERRABLE INITIALLY DEFERRED\` kiểm tra lúc COMMIT | Phải tiếp tục kiểm tra thay đổi đồng thời/migration mới |
| Account tham chiếu Person; tối đa một Account mở trên mỗi Person | FK, partial UNIQUE \`closed_at IS NULL\` | Chính sách hợp nhất/khôi phục Account |
| Email login không tạo hai Account chỉ do khác hoa/thường | UNIQUE expression \`lower(btrim(email))\` | Chuẩn hoá địa chỉ quốc tế, ngăn giữ chỗ email chưa xác minh |
| Không lưu mật khẩu dạng rõ | Argon2id có salt; bảng credentials riêng | Recovery, rate limiting và chính sách đổi mật khẩu |
| Session có hạn và có thể thu hồi | Token ngẫu nhiên, chỉ lưu SHA-256, \`expires_at\`, \`revoked_at\` | Cookie/TLS/CSRF/rotation khi triển khai thực |
| Role, Membership và Permission cùng Organization | Composite FK theo Organization + kiểm tra Role/Block trong SQL | Race thu hồi quyền và thao tác đồng thời |
| Claim không tự chiếm quyền Org đã có admin | Organization row lock, primary admin uniqueness | Quy trình tranh chấp, chuyển giao quyền |
| Lời mời không tự cấp quyền, không chấp nhận lặp | UPDATE có điều kiện + transaction tạo Membership/RoleAssignment | Mời lại, lời mời chồng nhau, thành viên đã tồn tại |
| Giá báo từ nguồn không bị ghi đè bởi API | Chỉ có thao tác thêm \`source_price_reports\`, thời gian \`statement_at\` và \`recorded_at\` tách biệt | Chưa có bảo vệ immutable bằng role/DB audit cho superuser |
| Giá của nguồn Org A không bị nối sang Org B | Composite FK \`(organization_id, intake_id)\` | FK không thay authorization |
| Retry không tạo hai Source/Price giống một ý định | UNIQUE scoped key + normalized payload hash + replay lookup + transaction | TTL/lifecycle key và tương thích chuẩn hóa payload về sau |

**Điểm cần nhớ:** FK giải quyết tính hợp lệ quan hệ; Session chứng minh Account đang đăng nhập; Membership và Permission xác định quyền trong Organization. Không một cơ chế riêng nào thay thế tất cả các cơ chế còn lại.

## D. Những quyết định đã chọn và chi phí thực tế

### D01 — Write Core khác Read Convenience

Sự thật gốc phải đủ bền và có nguồn gốc. Directory, Summary, Search, CRM Lookup chỉ là projection dễ đọc. Không đặt \`display_name/avatar\` vào mọi bảng gốc vì muốn API đọc nhanh; cũng không bắt mọi HTTP handler tự UNION Person + Organization. **Chi phí:** thêm bảng/profile và projection khi nhu cầu đọc đủ rõ; **lợi ích:** phân biệt sự thật gốc với cách hiển thị.

### D02 — Party và Account tách nhau

Nếu Person trùng với Account, dễ xử lý một đăng nhập đơn giản nhưng sẽ khó diễn đạt Person đã tồn tại trước Account hoặc Account bị đóng mà sự thật nghiệp vụ vẫn tồn tại. Với mô hình tách, đăng ký cần nhiều INSERT và transaction. **Không ép mọi liên hệ CRM trở thành Person**: Contact/Inquiry có thể chỉ nằm trong phạm vi một doanh nghiệp.

### D03 — C01 phải được chứng minh bằng database

PK/FK giúp bảo vệ quan hệ con → cha nhưng không tự ép cha có đúng một con. Trigger kiểm tra trì hoãn đến COMMIT cho phép tạo Party rồi mới tạo Person/Organization trong một transaction, song không cho lưu Party không có subtype. **Chi phí:** bảo trì trigger, test sự kiện INSERT/UPDATE/DELETE, kiểm tra những lệnh đổi định danh.

### D04 — Một Account, nhiều ngữ cảnh Organization

Đăng nhập một lần, chọn workspace để làm việc; mỗi thao tác vẫn phải chứng minh Membership/Role hiện hành. Không đồng nhất Organization Owner, primary workspace admin, reviewer nền tảng và người đại diện pháp lý. **Chi phí:** thêm join/authorization và vận hành vòng đời quyền. **Rủi ro còn mở:** hai transaction ghi và thu hồi quyền chạy đua dưới READ COMMITTED.

### D05 — Đăng ký và tiếp quản doanh nghiệp là hai hành động khác nhau

Registration khai báo một doanh nghiệp mới; Claim xin quản trị doanh nghiệp đã tồn tại. Phê duyệt đòi reviewer riêng, không tự động lấy người gửi làm quản trị viên. Claim đối với doanh nghiệp đã có admin bị ngăn thay thế ngầm. **Chi phí:** tác nghiệp rà soát, xác minh chứng cứ, xử lý tranh chấp; phương án “ai claim trước được quyền” nhanh hơn nhưng có nguy cơ chiếm dụng thương hiệu.

### D06 — Source không phải Property hay Listing

Một nhân viên nhận mô tả qua Zalo không có đủ bằng chứng để khẳng định căn nhà đã xác minh, có quyền bán, được phép đăng hoặc giá chính thức. Nếu lưu thẳng \`listings\`, hệ thống sẽ trộn tin nguồn và hàng công bố. Source riêng có thêm một bảng và quá trình xác minh, nhưng giữ ngữ nghĩa sạch.

### D07 — Báo giá chỉ thêm mới, không ghi đè

Nguồn nói 4,2 tỷ rồi báo 4,1 tỷ. \`UPDATE\` vào một cột \`current_price\` làm mất sự kiện trước. Tách \`source_price_reports\` giữ lịch sử, chứng cứ và thời gian. **Chi phí:** hai lần ghi trong lần capture có giá; cần transaction/index/truy vấn lịch sử; chưa tự kết luận “giá mới nhất đáng tin”.

### D08 — Idempotency theo ý định thao tác

Browser gửi lại request khi đã COMMIT nhưng mất response. UNIQUE scoped key ngăn nhân bản; hash payload chuẩn hoá quyết định replay hợp lệ hay xung đột. **Chi phí:** lưu hash, thêm unique index, yêu cầu client tái sử dụng key khi retry và tạo key mới cho hành động mới.

### D09 — Có cân nhắc tốc độ và tính đơn giản

Auth/Organization hiện dùng pgx SQL tường minh, Source dùng sqlc; đây là sự khác biệt đã biết. Không xây sẵn toàn bộ RBAC engine, Event Sourcing, 164 migration hoặc projection chưa có tình huống trả chi phí. Khi nợ kỹ thuật tạo ra lỗi/bảo trì thực tế, xử lý bằng evidence và commit riêng, không “refactor cho đẹp” vô điều kiện.

## E. Bằng chứng nghiệm thu hiện có

| Hạng mục | Bằng chứng | Trạng thái |
| --- | --- | --- |
| 5 migration áp dụng trên PostgreSQL 18 | GitHub Actions \`38139052788\` | **Đạt trên CI** |
| \`golang-migrate\`: UP → version → DOWN 1 → UP 1 | Cùng workflow | **Đạt trên CI** |
| Go/sqlc được sinh lại không khác repository | Git diff check trong CI | **Đạt** |
| Go format/vet/build | CI | **Đạt** |
| Register → Verify → Login → Org approval → Invitation → Source | 4 integration tests | **Đạt** |
| Source/Price, FK/CHECK, retry song song, revoke, HTTP stub | 11 tests | **Đạt** |
| Khởi động server | CI smoke test | **Đạt trên CI** |
| Hợp nhất chính xác với local cũ | Chưa có output local mới | **Chưa xác minh** |
| Người dùng thao tác toàn bộ qua HTTP thật tại local | Có hướng dẫn, chưa có log local | **Chưa nghiệm thu** |
| An toàn production, pháp lý và thị trường | Chưa có bài kiểm chứng tương ứng | **Chưa nghiệm thu** |

Các test chứng minh behavior trong fixture/database CI, không tự tạo bằng chứng rằng tổ chức ngoài đời đã tồn tại, reviewer đã xác minh pháp lý hay ứng dụng chịu tải thực.

## F. Những rủi ro phải xử lý trước production

- Cơ chế gửi lại email xác minh/outbox khi SMTP lỗi sau commit Account.
- Khôi phục Account, chống đoán mật khẩu, rate limiting, chống lạm dụng endpoint đăng ký/invite/review.
- CSRF đầy đủ: Origin, \`Sec-Fetch-Site\` và \`SameSite=Lax\` chưa đủ.
- Kiểm tra quyền theo thời điểm thực: thu hồi Membership/Role cùng lúc với ghi nghiệp vụ.
- DB role tối thiểu, không chạy ứng dụng bằng DB superuser; audit, backup và migration theo lộ trình dữ liệu.
- Chứng cứ đại diện pháp lý, chuyển giao admin, xử lý tranh chấp và dữ liệu cá nhân của người cung cấp nguồn.
- Triển khai passkey, external identity và service-account thực sự: **mới có schema**, không tự động là phương thức đăng nhập hoàn thiện.

## G. Quy tắc nghiệm thu và tiếp tục

1. Không coi commit/green CI là đồng bộ local.
2. Không triển khai \`migrations/000001–000005\` lên DB cũ chứa migration Listing khi chưa đọc \`schema_migrations\` và kế hoạch chuyển đổi.
3. Đọc [báo cáo tổng kết](BAO-CAO-GIAI-DOAN-CORE-01-05.md), [phương pháp làm việc](QUY-TRINH-LAM-VIEC.md) và [hướng dẫn nghiệm thu local](LOCAL-ACCEPTANCE.md).
4. Sau khi xác nhận local, bước kinh doanh tiếp theo là **CORE-06: CRM Inquiry & Follow-up** — người khách hỏi về nguồn, nhân viên ghi nhận và theo dõi, không cần ép khách đăng ký Account.

**Lưu ý thời gian:** những tài liệu continuity trước 11/10/2026 có thể mô tả cách làm “người dùng tự viết toàn bộ code” hoặc migration Listing cũ. Đó là lịch sử giai đoạn học thực hành, không còn là mô tả đầy đủ của đợt triển khai do người dùng đã ủy quyền code trực tiếp trên \`dev\`.
