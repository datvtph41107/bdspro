# Quy trình làm việc BDSPro — Từ lý luận đến commit có bằng chứng

**Hiệu lực từ 11/10/2026.** Quy trình này mô tả phương pháp thực tế cho giai đoạn triển khai Core và các module tiếp theo. Các file practice/continuity trước đây là **lịch sử và nguồn tham chiếu phương pháp**, không phủ nhận yêu cầu mới: assistant được giao triển khai code, migration, test, docs trực tiếp lên remote \`dev\`; người dùng tiếp nhận/kiểm tra local sau từng module lớn.

## 1. Đích đến

Không tạo sơ đồ cho đẹp, không chạy theo 164 bảng cùng lúc, không thỏa mãn với build xanh không có nghiệp vụ. Mỗi module phải có:

- **Người nào** thực hiện hành động và đang đại diện cho đối tượng nào.
- **Sự thật nào** được ghi, đọc hoặc thay đổi; trạng thái **trước → thao tác → sau**.
- **Bất biến** nào DB giữ được, và cái gì DB không bảo đảm.
- **Chi phí, rủi ro và phương án khác** trong điều kiện tương đương.
- **Bằng chứng chạy thật**: migration, sqlc, Go, PostgreSQL, âm tính (negative) và concurrency khi liên quan.
- **Dấu vết**: commit, CI run, quyết định, checkpoint và đường quay lại.

Tất cả giải thích nghiệp vụ, lập luận và tài liệu chính viết **bằng tiếng Việt**. Tên code/CLI/SQL định danh kỹ thuật giữ nguyên tiếng Anh.

## 2. Nhịp làm việc cho một module lớn

\`\`\`text
HIỂU BÀI TOÁN / BUSINESS SCENE
  → đọc canonical và source thực tế
  → tách các sự thật và vòng đời
  → nêu bất biến + phương án A/B/C
  → chọn baseline nhỏ nhất bảo vệ đúng nghiệp vụ
  → migration / SQL / Go / HTTP / tests
  → CI PostgreSQL, test âm tính và concurrency
  → sửa lỗi dựa trên output, không suy đoán
  → commit theo từng khối việc hoàn chỉnh trên dev
  → cập nhật quyết định, chi phí, rủi ro
  → checkpoint + tài liệu nghiệm thu
  → người dùng đồng bộ và nghiệm thu local
  → giai đoạn tiếp theo
\`\`\`

**Mảng lớn** không có nghĩa một commit khổng lồ. Một mảng lớn phải đủ hành vi để một người thật sử dụng được; trong đó vẫn chia commit theo trách nhiệm sao cho xem \`git log\` hiểu tiến trình.

## 3. Vai trò người dùng và trợ lý

### Trợ lý phụ trách

- Kiểm tra HEAD/\`dev\`, code thật, migration thật và canonical hiện hành trước khi chỉnh sửa.
- Thực thi code/SQL/Go/HTTP/tests/docs trên **nhánh \`dev\`**, commit theo khối công việc; không mở nhánh song song trừ khi người dùng thay đổi quyết định.
- Chạy/đọc CI khi có điều kiện; ghi đúng kết quả đạt/trượt, sửa dựa trên log, và không báo xong khi chưa có bằng chứng.
- Phân tích quyết định: tác nhân, dữ liệu mẫu, query, constraint, alternative, chi phí, rủi ro, điểm chưa chứng minh.
- Giữ README/báo cáo/checkpoint bằng tiếng Việt, cập nhật khi có chuyển pha.
- Không giả vờ đã thao tác máy local của người dùng; không tự merge dữ liệu cũ hay đoán migration version local.

### Người dùng phụ trách

- Quyết định nghiệp vụ và đánh giá mức độ phù hợp thực tế; có thể yêu cầu thay đổi canonical.
- Đồng bộ/check source từ remote, chạy PostgreSQL local và API khi giai đoạn kết thúc; cung cấp output hoặc lỗi thực tế.
- Xác nhận những chi tiết chỉ local mới biết: worktree đang sửa, database có dữ liệu, migration history, hệ thống Docker và các thao tác kinh doanh thật.
- Có thể yêu cầu chuyển sang chế độ thực hành: tự viết từng migration/Go để học cơ chế. Khi đó trợ lý hướng dẫn và review thay vì viết toàn bộ.

**Quy tắc ưu tiên:** chỉ thị mới, tường minh của người dùng về cách thực hiện thay thế mô tả tác nghiệp lịch sử; tuy nhiên vẫn giữ nguyên nguyên tắc bằng chứng, tính đúng, an toàn Git/DB và khả năng giải thích.

## 4. Mẫu phân tích một quyết định

Ví dụ: Lan làm nhân viên ABC, nhận qua Zalo một nguồn nhà 75 m² được báo giá 4,2 tỷ.

| Thành phần | Câu hỏi phải trả lời |
| --- | --- |
| Actor | Lan đăng nhập bằng Account nào, Membership nào, Role nào? |
| Đối tượng | ABC sở hữu quyền quản lý Source; ai chỉ cung cấp thông tin? |
| Trước | Source chưa có; báo giá cũng chưa có |
| Thao tác | Lưu nguồn và giá ban đầu trong một transaction |
| Sau | Một Source và một PriceReport; chưa có Property/Listing |
| Bất biến | Không cross-tenant; không Source lẻ khi Price thất bại; không duplicate khi retry |
| Phương án khác | Lưu thẳng Listing? Một cột giá hiện hành? Event store? |
| Chi phí | 2 bảng, 2 INSERT, unique index/hash, authorization SQL |
| Phản chứng | Báo giá muộn, role bị thu hồi song song, cùng key khác payload |
| Evidence | DDL/FK/CHECK, Go test, SQLSTATE, CI run, database thực |
| Giới hạn | Không chứng minh nguồn hàng hoặc giá đã được xác minh |

Không cần viết dài cho từng syntax nhỏ; độ sâu tùy mức độ rủi ro và hậu quả.

## 5. Nguyên tắc kiến trúc và kỹ thuật

**Write Core ≠ Read Convenience.** Không nhét fields hiển thị vào gốc nghiệp vụ chỉ vì query giao diện cần nhanh; đồng thời không bắt mọi handler tự viết logic Person-vs-Organization nếu read projection là lựa chọn phù hợp.

**Identity ≠ Authentication ≠ Authorization.** Person/Org là chủ thể, Account/Session là danh tính truy cập, Membership/Role/Permission là quyền theo ngữ cảnh. Subscription/package không tự chứng minh quyền đại diện hoặc quyền sở hữu.

**Một FK không phải quyền truy cập.** Phải có actor verified, scope Organization, kiểm tra quyền hiện hành, bảo vệ dữ liệu khi hai actor chạy đồng thời.

**Sự thật chưa xác minh phải được ghi là chưa xác minh.** Source không bằng Listing; báo giá được báo không bằng giá chính thức; email verified không bằng người được xác minh pháp lý.

**Lặp lại có chủ đích và giao dịch nguyên tử.** Khi retry dễ tạo lỗi tiền/dữ liệu, dùng idempotency; khi một nghiệp vụ gồm nhiều INSERT phụ thuộc, dùng transaction; không dựng event store khi chưa có sức ép thực tế.

**Không đánh đổi local cho remote.** Không \`reset --hard\`, \`clean -fd\`, \`force push\`, \`migrate force\`, \`dropdb\` khi chưa có nguyên nhân, phân tích tác động và bằng chứng có thể phục hồi. Clone riêng để nhận source mới khi local chưa đồng nhất.

## 6. Quy tắc commit và CI

Mỗi commit lớn nên có một tiêu đề đúng trách nhiệm:

- \`feat(core06): ...\` — tính năng/nghiệp vụ mới.
- \`test(core06): ...\` — bằng chứng và phản chứng.
- \`fix(core06): ...\` — sửa nguyên nhân lỗi đã quan sát.
- \`docs(core06): ...\` — quyết định, status, hướng dẫn nghiệm thu.
- \`ci(core06): ...\` — kiểm thử tái lập.

Không chia commit chỉ theo một thao tác terminal, nhưng cũng không gom schema/auth/http/docs của nhiều tầng không liên quan vào một commit. **CI xanh xác nhận commit đó**. Khi commit mới xuất hiện phải xác minh lại CI của HEAD mới; log CI cũ không tự động chứng minh HEAD mới.

## 7. Nghiệm thu có phân tầng

| Tầng | Đã chứng minh điều gì? |
| --- | --- |
| Source commit | File thật có trong GitHub \`dev\` |
| Migration/schema CI | DDL chạy trên PostgreSQL CI sạch |
| Go/SQL CI | Biên dịch, truy vấn, test, race cơ bản theo case đã viết |
| Tích hợp nghiệp vụ | Đúng hành trình Account → Organization → Source |
| Local | Đúng worktree, database/version/cấu hình trên máy người dùng |
| Production | Bảo mật, sự cố, vận hành, pháp lý, privacy, tải thật và người dùng thực |

**Không được suy luận từ tầng thấp lên tầng cao.** Báo cáo phải tách "Đạt trên CI", "Chưa xác minh ở local" và "Chưa đạt production".

## 8. Cách tiếp tục một cuộc trò chuyện

Câu \`Tiếp tục\` có nghĩa tiếp tục từ **checkpoint mới nhất**, không tự quay về review lý thuyết hoặc khởi tạo lại schema. Trình tự:

1. Đọc \`docs/continuity/CHECKPOINT.md\`, \`README.md\`, \`docs/implementation/BAO-CAO-GIAI-DOAN-CORE-01-05.md\`, tài liệu module và HEAD \`dev\`.
2. Kiểm tra điều đã được chứng minh với CI/commit hiện tại. Không nhầm thông tin trong checkpoint lịch sử.
3. Kiểm tra gate kế tiếp (ngay sau CORE-05: đồng bộ local và manual acceptance; sau đó thiết kế/triển khai CORE-06).
4. Nếu phụ thuộc local, đưa **lệnh an toàn tối thiểu** và yêu cầu output cụ thể; nếu có thể giải quyết trên remote, chủ động code/tests/docs theo ủy quyền hiện tại.
5. Trả về trạng thái mới, quyết định, commit/CI mới và đúng một bước tiếp theo có thể thực hiện.

## 9. Hạng mục ngay tiếp theo

**GATE-LOCAL-01:** clone \`dev\` sang thư mục riêng, đối chiếu \`git status/HEAD\`, chạy 000001–000005 trên database mới, kiểm tra \`sqlc/go test\`, chạy thủ công hành trình HTTP. Không xóa bất kỳ local code cũ.

**CORE-06 (sau gate):** CRM Inquiry & Follow-up — Contact/Inquiry trong scope Org; ghi sự quan tâm tới một Source, phân công follow-up, ghi hoạt động/hẹn tiếp theo, permission và privacy phù hợp. Không tự tạo Party/Account cho mỗi khách từ Zalo. Sau mỗi quyết định, ghi vào report/changelog và thử phản chứng.

**Trạng thái dài hạn:** canonical V0.4 vẫn là tập ứng viên cần kiểm chứng; CMS/SEO/Marketing, packages, quyền pháp lý/ủy quyền, Property/Listing, GIS, search projection... triển khai chỉ khi workflow thật đòi hỏi.
