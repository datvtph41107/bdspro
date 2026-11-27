# Đọc và vận hành log

## Vì sao trước đây chỉ thấy startup?

Logger chỉ ghi khi code phát event. Trả HTTP 403/503 không tự tạo log nếu không có middleware quan sát response. Mã màu ANSI xuất hiện trong file khi cùng output console có màu được ghi vào cả terminal lẫn file. Bật MODE=console,file chỉ chọn nơi ghi, không tự thêm access log hoặc SQL log.

Bản hiện tại tạo writer riêng theo đích, ghi HTTP response metadata ở router và gắn request_id cho các sự kiện SQL/security/delivery.

## Cấu hình local

Trong custom/conf/app.ini:

```ini
[log]
MODE = console,file
FORMAT = console
LEVEL = info
ROOT_PATH = data/log

[log.console]
FORMAT = console

[log.file]
FORMAT = json
FILE_NAME = bdspro.log
MAX_SIZE_MB = 20
MAX_BACKUPS = 5
MAX_AGE_DAYS = 7

[database]
LOG_SQL = slow
SLOW_QUERY_THRESHOLD = 200ms
```

Console ghi stderr, có thời gian đầy đủ và chỉ bật màu khi stderr là terminal, NO_COLOR không được đặt. File JSON không có ANSI. Nếu không đặt FORMAT riêng từng đích, format kế thừa [log] FORMAT để giữ tương thích. ROOT_PATH tương đối được tính từ application work path. Cần restart để đọc cấu hình mới.

Writer mở thử file khi startup: đường dẫn không ghi được làm startup báo lỗi. File mới tạo với quyền 0600, thư mục mới 0700; không tự thay quyền file/thư mục cũ. Khi file đạt ngưỡng, lumberjack xoay file, giữ backup theo giới hạn số lượng/tuổi; dọn backup có thể diễn ra bất đồng bộ. Không để nhiều process cùng ghi/rotate một file.

Các dòng ANSI cũ không bị sửa hoặc xóa tự động. Để đọc JSON mới trong file hỗn hợp:

```sh
tail -F data/log/bdspro.log | jq -R 'fromjson?'
```

Tìm toàn bộ sự kiện của một request bằng X-Request-ID nhận trong response:

```sh
jq -R --arg rid 'REQUEST_ID_FROM_RESPONSE' 'fromjson? | select(.request_id == $rid)' data/log/bdspro.log
```

## Đọc lỗi theo nguyên nhân

| Event/component | Thông tin dùng để debug | Vị trí sở hữu |
| --- | --- | --- |
| http.request / http | method, path, status, duration_ms, bytes, code/error/reason nếu có | routers/request_log.go |
| phone.delivery / delivery | channel, mode, accepted, reason, provider_status/code/message_id nếu có | services/phonedelivery/delivery.go |
| admission.denied / security | action, scope, attempts, limit, retry_after hoặc limiter_unavailable | services/context/admission.go |
| auth.denied / security | HTTP 401/403, code, reason nếu có | services/context/api.go |
| db.query / sql | query, query_id, duration_ms, rows, sqlstate/reason nếu lỗi | models/db/query_log.go |

Ví dụ quy trình đọc 503 “SMS delivery unavailable”: lấy request ID → tìm http.request để xác nhận status → tìm phone.delivery cùng ID → dùng reason và provider_status/code để phân biệt cấu hình, timeout hay cổng từ chối. Nếu bị từ chối trước khi gọi sender, xem reason trong access log; không phải mọi lỗi đều có event provider.

Với 429, đọc action/scope/retry_after thay vì đoán theo số điện thoại. Giới hạn theo nguồn request và theo người nhận là hai ngân sách khác nhau. Đổi kênh không làm mới ngân sách người nhận.

Với 403 browser origin, reason=browser_origin chỉ ra guard; đối chiếu Origin của trình duyệt với server.PUBLIC_ORIGIN. localhost và 127.0.0.1 là hai origin khác nhau. Không tắt guard để giải quyết vấn đề cấu hình.

SQL mode:

- off: tắt query tracer.
- slow: ghi query lỗi và query chậm từ ngưỡng cấu hình.
- all: thêm query thành công ở DEBUG; phải đặt [log] LEVEL=debug để thấy chúng.

Query log dùng tên SQLC, không chứa SQL thô, bind args hoặc PgError.Detail/Message. Tra query theo tên:

```sh
rg -n -- '-- name: CreatePhoneVerification' db/query
jq -R 'fromjson? | select(.component == "sql" and .level == "error")' data/log/bdspro.log
```

Tracer nằm trên pool pgx của ứng dụng. Nó quan sát thực thi query, không thay thế log nghiệp vụ, không bao phủ mọi lỗi Scan của caller hay các kết nối migration độc lập. SQLSTATE giúp xác định loại lỗi; muốn tái hiện phải dùng dữ liệu thử có kiểm soát.

HTTP log không lưu body, cookie, Authorization hoặc query string. Lỗi public đã chuẩn hóa được ghi để đối chiếu response. Không thêm OTP, token, mật khẩu, số điện thoại hoặc raw provider response để “debug cho đủ”. Mailpit là nơi xem OTP dev. Event vận hành security không thay thế audit_events chứa lịch sử chuyển trạng thái nghiệp vụ.

## Production

Với container, thường chỉ cần JSON stderr để hệ thống thu thập log đọc:

```ini
[log]
MODE = console
FORMAT = json
LEVEL = info

[log.console]
FORMAT = json

[database]
LOG_SQL = slow
SLOW_QUERY_THRESHOLD = 200ms
```

Nếu có [log.console] FORMAT=console cũ, phải sửa override đó. Runtime/container driver chịu trách nhiệm retention cho stderr; Compose hiện cấu hình json-file 10m/3 cho service có block logging. Đây không phải giới hạn file của ứng dụng. Trên VM dùng MODE=console,file hoặc file với rotation ở trên, mỗi process một file.

Bản này giữ một luồng JSON có component/request_id để tìm xuyên tầng. Không tạo riêng file sql/security/debug vì nhu cầu hiện tại chưa cần routing/retention riêng. component là nhãn truy vấn, chưa có cấu hình level độc lập từng component. Chưa có benchmark chứng minh thông số rotation hoặc threshold là tối ưu cho tải production.

## Bài học cụ thể từ Gitea và quyết định trong code

| Nguồn | Vấn đề thực tế | Áp dụng ở đây |
| --- | --- | --- |
| [Gitea #26137](https://github.com/go-gitea/gitea/commit/3e4a4f9c67ebc08fed0b4dff90c707ddbc608832) | Router dùng log global thay vì logger đã chọn | Gắn component đúng nơi phát event; GetLogger(name) không bỏ qua name |
| [Gitea #35676](https://github.com/go-gitea/gitea/commit/304d836a61d0ed92144f6e0a4024c4071e3d113d) | Vòng đời shutdown/logger gây thiếu writer khi còn log | Giữ logger sống tới khi CLI kết thúc; không đóng cùng một subsystem còn công việc khác |
| [Gitea #37507](https://github.com/go-gitea/gitea/commit/f049668c210d23e8177b2bb9f0ac98163cbb42ab) | Log khởi động ghi stdout làm hỏng dữ liệu gitea dump | Console ghi stderr, stdout dành cho dữ liệu lệnh |
| [Gitea logging configuration](https://docs.gitea.com/administration/logging-config/) | Phân biệt logger và output; router/xorm/access có vai trò riêng | Tách trách nhiệm phát event khỏi writer; chưa sao chép hệ registry/routing nhiều output của Gitea |
| [pgx QueryTracer](https://pkg.go.dev/github.com/jackc/pgx/v5#QueryTracer) | Có điểm quan sát query tại driver | Một tracer ở models/db, không rải log SQL trong mỗi service |
| [lumberjack v2.2.1](https://github.com/natefinch/lumberjack/tree/v2.2.1) | Writer hỗ trợ rotation theo dung lượng/backup/tuổi | Dùng implementation đã có, không tự viết thuật toán rename/cleanup |

## Vì sao tách các hàm/file này?

- manager.go quản lý logger hiện hành và đóng tài nguyên; output.go xây sink/format/rotation. Thêm một đích ghi không kéo theo sửa logic event HTTP.
- OutputOptions.Validate kiểm tra contract writer; setting.LogGlobalConfig chỉ đọc INI, resolve path và chuyển options. Không duy trì hai bộ validation rotation riêng.
- fanoutWriter vẫn thử các sink sau khi một sink lỗi. io.MultiWriter dừng ở lỗi đầu, có thể khiến file không được ghi khi terminal lỗi.
- log.For giữ request logger và thêm component; công việc nền dùng logger tiến trình. SQL/delivery/security không tự tạo request ID mới.
- APIContext.Admit ánh xạ quyết định limiter sang HTTP ở một chỗ; source admission và subject admission dùng chung Retry-After, error contract và event. Nó không chứa thuật toán rate limit.

Bằng chứng: output_test.go kiểm tra rotation, quyền file mới, tên logger và sink hỏng; query_log_test.go kiểm tra mode, correlation và loại dữ liệu được phép ghi; request_log_test.go kiểm tra 403/429/503, reason, Retry-After và dữ liệu không được log. Integration qua Mailpit kiểm tra event HTTP/SQL/delivery/admission thực tế cùng request ID. Xem các test cùng package để đối chiếu từng quyết định, không dựa riêng vào việc terminal “trông đúng”.
