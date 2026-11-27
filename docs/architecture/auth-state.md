# Trạng thái xác thực: tên, vòng đời và bằng chứng

## Vấn đề cần giải quyết

Trước migration 42, schema có 33 bảng ứng dụng. Hai bảng xác minh email/phone có cùng cấu trúc và vòng đời nhưng mang tên theo màn hình triển khai. Google lưu flow và ý định của người đang đăng nhập ở hai bảng quan hệ 1–1. Điều này làm người đọc phải ghép nhiều tên để hiểu một đối tượng.

Số bảng và độ dài tên không tự chứng minh thiết kế tốt/xấu. Tiêu chí của thay đổi này là: bản ghi đại diện điều gì, ai sở hữu nó, hết hạn/xóa khi nào, và ràng buộc nào thực sự khác nhau. Không sửa các migration lịch sử đã có thể được áp dụng.

## Đối chiếu nguồn

| Source đã đối chiếu | Điều quan sát được | Quyết định cho BDSPro |
| --- | --- | --- |
| [Laravel DRX: password_resets](https://github.com/datvtph41107/laravel-drx/blob/main/database/migrations/2014_10_12_100000_create_password_resets_table.php) | Tên nói mục đích, cột email/token/created_at; repo còn có các bảng OAuth tên dài | Dùng danh từ mô tả đối tượng; không lấy “ít ký tự” làm quy tắc tuyệt đối |
| [Java DRWash: Authenticode](https://github.com/datvtph41107/java-drwash/blob/main/core/src/main/java/com/drwashup/core/domain/authenticode/Authenticode.java) | Một entity có id, value dạng JSON và expireTime, hỗ trợ mã mobile/email | Có thể chia sẻ vòng đời xác minh. BDSPro vẫn dùng cột tường minh và CHECK thay vì giấu ràng buộc trong JSON |
| [Gitea: TwoFactor](https://github.com/go-gitea/gitea/blob/main/models/auth/twofactor.go) | Entity được đặt trong models/auth, có vai trò xác thực rõ ràng | Tên theo đối tượng và trách nhiệm; không biến mọi bước HTTP thành bảng. Đây là đối chiếu entity, không suy ra quy tắc đặt tên SQL của mọi framework |
| [BDSPro: lịch sử tách HTTP/service/database](https://github.com/datvtph41107/bdspro/commit/253d1d05460ad94866462c8ed161a9a74a2fced9) | Commit chủ động tách quyền sở hữu HTTP, nghiệp vụ và SQL | Giữ hướng phụ thuộc; tên thư mục lịch sử không phải lý do tạo thêm tầng trung gian |

Các ví dụ trên là bằng chứng về cách tổ chức cụ thể, không phải khảo sát “đa số cộng đồng”. Lịch sử ngắn hơn chưa chứng minh thiết kế cũ đáp ứng đầy đủ các ràng buộc browser, replay và reauthentication hiện tại.

## Schema đã triển khai

| Trước | Sau migration 42 | Lý do và ràng buộc giữ lại |
| --- | --- | --- |
| email_registration_challenges + phone_continuation_challenges | verifications | Cùng browser binding, MAC/proof, TTL, resend, số lần sai. Phân biệt bằng kind=email/phone |
| google_auth_flows + google_authenticated_intents | google_flows | Ý định link/reauth thuộc chính flow; account/session/purpose/action ở cùng bản ghi, có CHECK |
| recent_auth_proofs | auth_proofs | Bằng chứng xác thực lại để thực hiện hành động nhạy cảm; vẫn có hạn dùng và binding |
| credential_link_challenges | contact_links | Xác minh để liên kết email/phone với tài khoản đang có; giữ vòng đời và quyền riêng |
| phone_credentials | Giữ nguyên | Danh tính bền vững đã được xác minh, khác trạng thái xác minh tạm |

Kết quả là **31 bảng ứng dụng**, không tính schema_migrations. Không thêm bảng theo kênh SMS/Zalo/Telegram: kênh vận chuyển không thay đổi danh tính số điện thoại.

Tại sao vẫn có nhiều bảng auth? verifications thuộc đăng ký chưa có tài khoản; contact_links gắn người đã đăng nhập và thao tác liên kết; auth_proofs chứng minh một hành động nhạy cảm được phép; google_flows có OAuth state/PKCE/phase. Gộp tất cả thành bảng “otp” sẽ làm mất nghĩa hoặc tạo nhiều cột nullable với tổ hợp khó kiểm soát.

Hai bảng phone_otp_challenges và phone_signup_challenges không còn query được runtime hiện tại sử dụng. Migration giữ dữ liệu và đánh dấu legacy; đã bỏ query/generated code không còn nơi gọi. Chưa có căn cứ để xóa dữ liệu lịch sử hoặc tự gán browser/session cho chúng. Bước dọn tiếp cần thống kê dữ liệu, kiểm tra ứng dụng khác có dùng chung DB không và quyết định retention. Các bảng lịch sử khác cũng không được xóa chỉ vì tìm kiếm một tên hàm không có kết quả.

## Các dòng kiểm tra có giá trị gì?

| Vị trí | Quyết định cụ thể | Lỗi được ngăn / bằng chứng |
| --- | --- | --- |
| migration 42 | PRIMARY KEY(kind,id), UNIQUE(kind,proof_digest) | Hai bảng cũ có thể trùng UUID/proof; chuyển sang khóa id đơn có thể mất dữ liệu. Test cố ý tạo trường hợp trùng |
| email_verification.sql, phone_verification.sql | Mọi lookup/update/delete đều giới hạn kind | Không dùng proof hoặc xóa challenge của kênh danh tính khác; test xóa phone vẫn giữ email cùng UUID |
| verifications CHECK | MAC/proof loại trừ nhau, độ dài digest, đích nhận, failed_attempts | Giữ các invariant cũ ở DB, không chỉ trông chờ handler |
| google_flow.sql | Login yêu cầu purpose=login; authenticated result kiểm tra account/session/purpose | Flow reauth không thể bị dùng làm login ẩn danh hoặc dùng từ session khác |
| services/auth/google_authenticated.go | BindGoogleFlow kiểm tra số hàng cập nhật bằng 1 | Không coi “UPDATE không tìm thấy flow” là đã gắn ý định thành công |
| account.sql | CreateAccountWithoutEmail thay CreatePhoneOnlyAccount | Cùng query được dùng bởi phone và Google; tên phải mô tả kết quả thật |
| models/db/schema.go | validateSchema so khớp tên bảng theo version, báo missing/extra | Cùng số lượng bảng vẫn có thể sai bảng; chỉ đếm không đủ |
| migration up/down | Transaction + khóa bảng nguồn trước copy/drop | Writer không thể chèn thêm vào bảng cũ giữa lúc copy và drop; vẫn cần dừng binary cũ khi triển khai |

Không thêm generic repository hoặc interface cho từng query: SQLC đã cung cấp contract typed và transaction ở models/db. Các query đặt theo đối tượng dữ liệu; service quản lý thứ tự và quy tắc nghiệp vụ. File generated chỉ được tạo từ make generated.

validateSchema kiểm tra tên bảng và một số cột mốc lịch sử; đây không phải công cụ so sánh toàn bộ constraint/index giữa DB và SQL.

## Kiểm chứng và triển khai

TestAuthStateMigration chạy trên PostgreSQL trống do script kiểm thử tạo. Nó thực hiện 41 → 42 → 41 → 42, so sánh toàn bộ hàng mẫu trước/sau rollback; kiểm tra browser binding, tách kind, Google session/purpose, và chạy migration lần nữa không đổi gì. Test thông thường bỏ qua integration nếu không có BDSPRO_TEST_DATABASE_URL; script test-phone-delivery.py cấp DB tạm và thực sự chạy test này.

**Migration chưa được áp dụng vào DB dev hiện hữu trong đợt triển khai code này.** Development tự chạy migration khi khởi động web. Production yêu cầu chạy migrate riêng.

Quy trình triển khai khi đã sẵn sàng:

1. Sao lưu DB và dừng các web/worker cũ có thể ghi vào bảng liên quan.
2. Chạy bản mới với make migrateup; kiểm tra migration thành công trước khi mở nhận request.
3. Chạy cùng phiên bản code mới cho các tiến trình ứng dụng.
4. Thử đăng ký, resend, Google login và reauth trên môi trường đích.

Không chạy song song binary cũ/mới qua lần đổi tên bảng này. Khóa giao dịch bảo vệ copy dữ liệu, không làm các query của binary cũ tương thích schema mới. Nếu migration bị ngắt, kiểm tra transaction/schema và trạng thái dirty của migration tool; không force version khi chưa đối chiếu.

Rollback một bước ở development dùng make migratedown1 khi DB đang ở 42 và ứng dụng đã dừng; sau đó mới dùng binary cũ. CLI hiện chặn rollback production; cần quy trình phục hồi/forward fix riêng, không đổi RUN_MODE để vượt chặn. Test round-trip không thay thế thử phục hồi backup thực tế.
