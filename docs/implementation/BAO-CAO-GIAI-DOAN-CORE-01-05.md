# Báo cáo kết thúc giai đoạn 1 — Nền tảng CORE-01 → CORE-05

**Cập nhật:** 11/10/2026 · **Repository:** \`datvtph41107/bdspro\` · **Nhánh làm việc:** \`dev\`.  
**Phạm vi nghiệm thu:** phiên bản kỹ thuật đã tích hợp trên GitHub + CI. **Chưa** nghiệm thu dữ liệu/ứng dụng trên máy local của người dùng hay production.

## 1. Tại sao bắt đầu từ đây?

Mục tiêu dài hạn của BDSPro là có một lõi đủ chắc để nhiều loại chủ thể sử dụng cùng cơ chế nhận dạng, phân quyền, vòng đời quan hệ và thao tác có bằng chứng. Bài toán nghiệp vụ đầu tiên: một nhân viên trong tổ chức ghi nhận nguồn hàng nhận được bên ngoài, không biến thông tin chưa xác minh thành tài sản/tin đăng chính thức.

Để thao tác "Lan lưu một nguồn giá 4,2 tỷ" đúng đắn, hệ thống phải trả lời: Lan là Account nào, đã đăng nhập chưa, có thuộc tổ chức nào, có quyền gì tại thời điểm hành động, Source thuộc tổ chức nào, báo giá có bị nhân bản khi retry và sau này tra cứu có giữ lịch sử hay không. Vì vậy chúng ta xây **từ Identity đến workflow thực**, không triển khai 164 bảng một lượt.

## 2. Đã làm được những gì?

| Chặng | Thành phần đã triển khai | Kết quả quan sát được trong CI |
| --- | --- | --- |
| 01. Bản sắc nghiệp vụ | Party, Person, Organization, PersonProfile; ràng buộc C01 subtype | Party thiếu subtype không COMMIT hợp lệ |
| 02. Tài khoản & xác thực | Account, email/password, Argon2id, token xác minh, Session cookie, logout | Register, verify, login, me, logout qua HTTP test |
| 03. Phân quyền tổ chức | Membership, Role, Permission, RoleAssignment, Block, primary workspace admin | Người khác Organization không được xem nguồn; thu hồi/block quyền đọc |
| 04. Gia nhập tổ chức | Registration, Claim, Reviewer, Approval, Invitation, Acceptance | Người khai báo không tự duyệt; invitation chưa phải Membership |
| 05. Ghi nhận nguồn/giá | Source Intake, Price Report, composite FK, scoped permission, idempotency | Giữ 2 báo giá, rollback khi thiếu quyền, retry đồng thời không tạo trùng |
| 06. Vận hành kỹ thuật | 5 cặp migration UP/DOWN, HTTP server, SQL/sqlc, Go, Makefile, CI | Versioned migrate, generated code, vet/test và server smoke đều thành công |
| 07. Tri thức và bàn giao | Quyết định thiết kế, báo cáo, hướng dẫn chạy local, checkpoint | Tài liệu được commit trên \`dev\` |

Các schema cho \`passkeys\`, \`external_identities\`, \`service_accounts\`... là **nền dữ liệu dự phòng có giới hạn**, không có nghĩa các phương thức đó đã được triển khai hoặc kiểm thử hoàn chỉnh.

## 3. Kết quả thực thi: cái gì có bằng chứng?

### CI đã chạy thành công

- [Workflow đã nghiệm thu trên code Core](https://github.com/datvtph41107/bdspro/actions/runs/38139052788).
- 4 bài test tích hợp Account/Organization/Source và 11 test cho Source, tổng **15 test PASS**.
- PostgreSQL 18: áp dụng migration 000001–000005 thành công; CLI \`golang-migrate\` chạy \`UP → version → DOWN 1 → UP 1\`.
- \`sqlc generate\` tạo mã Go khớp file đã commit; \`go mod tidy\`/format/vet/build/tests đều qua.
- Server chạy được trên DB CI sạch; đây là smoke test, chưa thay thế HTTP manual test local.

### Những điều KHÔNG được suy diễn

- Chưa đọc/đồng bộ trực tiếp working tree hoặc PostgreSQL local của người dùng.
- Chưa chứng minh việc nâng cấp từ schema/migrations Listing cũ **trên dữ liệu có sẵn**.
- Chưa thử tải thực tế, bảo mật production, backup/khôi phục, pháp lý doanh nghiệp hoặc giá trị thị trường.
- Không phải 164 bảng canonical đã được triển khai; chỉ 5 migrations lõi theo nghiệp vụ.

## 4. Những lựa chọn kiến trúc quan trọng và tác động

### 4.1. Identity bền vững, Account riêng, Profile riêng

Một Person có thể tồn tại dưới góc độ nghiệp vụ mà chưa có đăng nhập; Account là quyền truy cập số, Profile là biểu diễn giao diện. Tách bảng tạo chi phí transaction, join và hợp đồng liên kết, nhưng tránh buộc dữ liệu nghiệp vụ biến mất/chuyển nghĩa khi đổi email hoặc đóng Account.

### 4.2. C01 là invariant database, không chỉ quy ước Go

\`parties.kind\`, composite FK và deferred trigger giúp chứng minh đúng một subtype khi COMMIT. Nếu chỉ có PK/FK, có thể để trống subtype của Party. Trigger làm tăng độ phức tạp migration và cần negative/concurrency tests, đổi lại không phụ thuộc mọi caller Go đều nhớ kiểm tra.

### 4.3. Một đăng nhập, nhiều workspace

Account chứng minh phiên đăng nhập; Membership/Role mới chứng minh quyền trong Organization. Một người làm cho nhiều công ty không nhất thiết cần nhiều mật khẩu. Giá phải trả là authorization phải xét ngữ cảnh mỗi yêu cầu, không tin Organization ID do client gửi.

### 4.4. Claim khác Registration, workspace admin khác đại diện pháp lý

Người đầu tiên gửi thông tin không tự động chiếm Organization. Reviewer là quyền nền tảng riêng và quyết định được ghi lại. Thao tác thủ công tốn thời gian hơn nhưng tránh một đường chiếm dụng tổ chức chỉ bằng việc khai báo trước. **Tuy nhiên**, review reference trong LAB chưa chứng minh đã kiểm tra giấy phép thật.

### 4.5. Source/Price không phải Listing/Property

Một lời nhắn Zalo về "75m², giá 4,2 tỷ" chỉ là thông tin có nguồn gốc từ bên ngoài. Ghi trực tiếp Listing sẽ gán ý nghĩa công bố và quyền bán chưa có bằng chứng. Tách Source khỏi Property/Listing khiến workflow xác minh rõ ràng nhưng thêm một tầng dữ liệu.

### 4.6. Append-only cho lời báo giá, transaction cho capture

Lần đầu có thể tạo Source và báo giá trong một transaction. Gặp lỗi phần giá, không để lại Source lẻ của yêu cầu có giá. Khi giá thay đổi, thêm PriceReport mới chứ không ghi đè. Cần truy vấn lịch sử và phân biệt \`statement_at\` với \`recorded_at\`.

### 4.7. Idempotency

Hai lần bấm/gửi lại do mất mạng phải trả về cùng bản ghi khi request key/payload không thay đổi; key trùng nhưng nội dung khác trả xung đột. UNIQUE + hash chuẩn hoá tăng index/chi phí xử lý nhưng giảm sai lệch nghiệp vụ khi retry đồng thời.

## 5. Công việc Git đã thực hiện

- Làm việc trực tiếp trên \`dev\` theo yêu cầu mới, phân khối bằng các commit có tên \`feat(core01)\`, \`feat(core02)\`, \`feat(core03)\`, \`feat(core04)\`, \`feat(core05)\`, \`test\`, \`ci\`, \`fix\`, \`docs\`.
- Loại bỏ runtime Listing prototype không còn migration tương ứng; giữ lịch sử Git để có thể tra cứu việc cũ.
- Chuyển hệ thống sang một chuỗi migration 000001–000005 thay vì đưa migration module riêng \`000001\` từ nhánh thử nghiệm vào.
- Commit cả mã \`db/sourcesqlc\` được sinh, \`go.mod\`, \`go.sum\`; CI phát hiện drift khi quên cập nhật file.
- Đóng PR nhánh thử nghiệm CORE-05 cũ, tránh hai nguồn implementation song song.
- Duy trì checkpoint, ghi bằng chứng chính xác theo workflow run và commit, **không gọi GitHub CI là bằng chứng đã đổi DB local**.

## 6. Nợ kỹ thuật, rủi ro và chi phí phải quản lý

**Ưu tiên trước production:** bảo mật phiên/bảo vệ CSRF; rate limiting + anti-abuse; quy trình khôi phục tài khoản; email outbox/retry; quyền reviewer cấp/thu hồi có audit; least-privilege DB; tranh chấp và chuyển giao tổ chức; dữ liệu liên hệ và retention; chạy đua giữa thu hồi quyền và ghi nghiệp vụ.

**Để ý khi phát triển tiếp:** Auth/Organization viết SQL pgx trực tiếp còn Source dùng sqlc; cần quyết định hợp nhất khi có nhu cầu rõ ràng, không refactor máy móc. \`passkeys\`, OIDC, service account vẫn chưa có runtime. Projection (Directory/Search/CRM Lookup) chưa cần tạo bằng mọi giá cho 5 migrations hiện tại.

## 7. Bảng nghiệm thu của giai đoạn

| Gate | Tình trạng | Bằng chứng/công việc cần thiết |
| --- | --- | --- |
| Code remote \`dev\` | **Đạt** | Có commit + source |
| Migration + sqlc + Go CI | **Đạt** | Workflow 38139052788 |
| Nghiệp vụ xuyên module | **Đạt trên CI** | 15 integration tests |
| Đồng bộ và test PostgreSQL local | **Chưa xác minh** | Clone/check SHA/migration version/log |
| Manual API bằng Session thật tại local | **Chưa xác minh** | Đi hết kịch bản trong LOCAL-ACCEPTANCE |
| Nghiệm thu bảo mật, pháp lý và sản phẩm | **Chưa** | Test/evidence bổ sung theo rủi ro |
| Bắt đầu CORE-06 | **Sau local gate** | Hợp đồng CRM Inquiry/Follow-up |

## 8. Hướng tiếp tục sau mốc này

**Gate ngay lập tức:** bạn clone \`dev\` vào thư mục riêng, chạy migration trên database thử nghiệm mới, chạy tests và hành trình HTTP; gửi SHA HEAD, migration version và log kết quả. Khi đó ta mới xác nhận local thực sự khớp remote.

**Giai đoạn kế tiếp CORE-06:** CRM Inquiry & Follow-up. Ví dụ khách ngoài BDSPro nhắn hỏi về nguồn #900 của ABC; Lan ghi nhận yêu cầu theo tổ chức, gán người theo dõi, thời hạn và lịch sử hành động. Không tự động tạo Account/Party cho khách, không cho dữ liệu CRM của ABC rò sang công ty khác. Thiết kế bảng và cơ chế permission tiếp tục theo phương pháp *tình huống → ràng buộc → implement → test → đánh giá tác động*.

**Tài liệu đọc tiếp:** [Quy trình làm việc](QUY-TRINH-LAM-VIEC.md), [Thiết kế chi tiết](CORE-01-TO-05.md), [Nghiệm thu local](LOCAL-ACCEPTANCE.md), [Checkpoint](../continuity/CHECKPOINT.md).
