# BDSPro — PROMPT KHÔI PHỤC MỘT CHAT MỚI (2026-10-15)

Sao chép toàn bộ nội dung trong khối dưới đây thành **tin nhắn đầu tiên** của chat mới. Không cần gửi toàn bộ transcript.

~~~text
Tiếp tục dự án BDSPro theo durable checkpoint trên GitHub, KHÔNG dựa vào trí nhớ hội thoại.

Repository: datvtph41107/bdspro
Working branch: dev. Phải đọc HEAD/branch thật hiện tại; main là baseline lịch sử, không coi là release. Local Git/DB của tôi có thể khác remote, không được giả định đã đồng bộ.

BẮT BUỘC đọc và đối chiếu theo thứ tự:
1) docs/continuity/checkpoints/2026-10-15-engineering-core-reorientation/README.md (mốc MỚI chi phối trọng tâm và phương pháp)
2) docs/continuity/CHECKPOINT.md (đọc header mới; phần CORE-06/SQL-01 cũ chỉ là lịch sử)
3) docs/continuity/REPOSITORY-WORKING-CONTRACT.md
4) docs/continuity/PROPORTIONAL-ENGINEERING-STANDARD.md
5) docs/continuity/EVIDENCE-DRIVEN-REASONING.md
6) docs/continuity/PRACTICE-PROTOCOL.md
7) docs/research/go/REFERENCES.md ; ANCHOR-REPOSITORIES.md ; UBER-GO-GUIDE.md ; PACKAGE-NAMING.md
8) source thực tế ở nhánh dev: main.go, auth/http.go, auth/service.go; khi cần xem organization/http.go, source/http.go, migrations và CI.

TRỌNG TÂM HIỆN TẠI: ENGINEERING CORE, KHÔNG PHẢI BUSINESS CORE.
Tôi cần xây nền móng cách developer làm việc: cấu trúc thư mục/package/file và ownership; naming; Go idioms; Handler/Request/Response/Validation/Error; application bootstrap/dependencies; middleware và trust boundary; pgx/sqlc/SQL/migrations/transactions; security/reliability; test/CI/dev workflow; maintainability, thay đổi và vận hành. Không tự triển khai thêm Source/CRM/CORE-06, OTP hay migration mới.

PHƯƠNG PHÁP:
- Tư duy ngược từ code ĐANG CHẠY: nó đã đủ ở đâu? Lời cam kết? Failure path/chi phí nào có thật? Sau đó mới quyết định KEEP, MODIFY, TRIAL, DEFER, REJECT.
- Không tiền giả định phải refactor, tạo framework, tách sáu lớp hay Gin/Echo. Framework chỉ đáng giá khi tổng lợi ích lớn hơn chi phí; rủi ro correctness/security/data phải bảo vệ sớm khi có đường thất bại có thể chứng minh.
- Mọi quyết định lớn cần official spec, Go/Uber/Google guidelines, production GitHub repo, research/community (gồm phản biện), đối chiếu bdspro code và test. Không lấy stars/upvotes/lý thuyết thay proof.
- Giải thích Go từ bản chất và source: function/struct/closure/http.Handler/context/error/pool/transaction; tối đa một quyết định hẹp tại một thời điểm; tiếng Việt rõ, giàu bằng chứng, không giải thích một lần mười lớp.
- Trước/sau: trước kia assistant đã đi migration → handler/service → tests → module mà chưa thống nhất Engineering Core. Hiện phải ưu tiên core technical working conventions và review bằng chứng.
- Tôi là người viết/chạy source/SQL khi đến phần triển khai; assistant là người phân tích, đối chứng, chỉ rõ tác nhân/tác động, hướng dẫn kiểm chứng. Không tự sửa source, migrations, CI, reset local, thay main nếu chưa có yêu cầu cụ thể.
- Đọc thật GitHub trước khi tuyên bố trạng thái; remote docs không chứng minh máy local/DB đã sync hoặc runtime production-ready.

BƯỚC TIẾP THEO: bắt đầu ở main.go của bdspro, phân tích thật cụ thể từ main() → run() → pgxpool → constructor/wiring → ServeMux/HTTP Server → signals/shutdown; vai trò mỗi phần, ownership, Go primitive nào làm việc, chi phí/đủ dùng, tiêu chí khi cần tách file/package. Sau đó lần một POST /auth/login trọn request. Dùng Logout bỏ qua DB error hoặc sự khác biệt JSON decoder làm ví dụ reasoning, chưa sửa code. KHÔNG yêu cầu tôi kể lại toàn bộ bối cảnh đã nằm trong repo.

Trước khi bắt đầu: trả lời ngắn tình trạng đã khôi phục (trọng tâm, HEAD/source kiểm chứng, điều chưa biết, một câu hỏi kỹ thuật đầu tiên) rồi đi ngay vào bước nhỏ đầu tiên. Nếu không truy cập được repo phải nói rõ, không bịa.
~~~

## Cơ chế đồng bộ sau mốc này

1. Mỗi thay đổi quyết định lớn: cập nhật README checkpoint hoặc tạo checkpoint mới và dẫn link từ CHECKPOINT.md.
2. CONTINUATION-PROMPT.md luôn dẫn tới prompt đang có hiệu lực; phiên chat mới chỉ cần một request dán trên.
3. Trước khi khẳng định đã đồng bộ: kiểm tra HEAD dev trên GitHub và local Git status do người dùng cung cấp; GitHub write không làm thay đổi local files.
4. Bất cứ chỉ dẫn lịch sử mâu thuẫn: live source xác nhận trạng thái code; checkpoint mới xác nhận trọng tâm làm việc; đừng dùng kế hoạch CORE-06 cũ để tiếp tục.
