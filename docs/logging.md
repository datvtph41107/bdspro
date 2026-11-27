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

Console ghi stderr, hiển thị giờ/phút/giây ở format console và chỉ bật màu khi stderr là terminal, NO_COLOR không được đặt. File JSON không có ANSI. Nếu không đặt FORMAT riêng từng đích, format kế thừa [log] FORMAT để giữ tương thích. ROOT_PATH tương đối được tính từ application work path. Cần restart để đọc cấu hình mới.

Writer mở thử file khi startup: đường dẫn không ghi được làm startup báo lỗi. File mới tạo với quyền 0600, thư mục mới 0700; không tự thay quyền file/thư mục cũ. Khi file đạt ngưỡng, lumberjack xoay file, giữ backup theo giới hạn số lượng/tuổi; dọn backup có thể diễn ra bất đồng bộ. Không để nhiều process cùng ghi/rotate một file.

Các dòng ANSI cũ không bị sửa hoặc xóa tự động. Để đọc JSON mới trong file hỗn hợp:

```sh
tail -F data/log/bdspro.log | jq -R 'fromjson?'
```

Tìm toàn bộ sự kiện của một request bằng X-Request-ID nhận trong response:

```sh
jq -R --arg rid 'REQUEST_ID_FROM_RESPONSE' 'fromjson? | select(.request_id == $rid)' data/log/bdspro.log
```

## Console compact (developer)

Khi `[log.console] FORMAT = console`, terminal ưu tiên thông tin theo loại event, không in toàn bộ JSON dưới dạng key=value. Thời gian chỉ hiển thị giờ/phút/giây (không thêm độ chính xác giả); file JSON vẫn giữ timestamp gốc.

Ví dụ minh họa (giá trị giả lập):

```text
10:30:05 [D] sql: CreateSession 1.382ms rows=1 rid=8b6424f1
10:30:05 [I] router: completed POST /api/v1/auth/login 200 8.732ms rid=8b6424f1
10:30:10 [W] security: admission.denied action=login scope=source retry_after=60 rid=8b6424f1
10:30:15 [E] delivery: sms accepted=false 5000ms reason=timeout provider_status=503 rid=8b6424f1
```

- UUID trong response X-Request-ID và JSON giữ nguyên. rid là 8 ký tự đầu để đọc nhanh; có thể trùng. Luôn dùng UUID đầy đủ để truy vấn chính xác.
- HTTP chỉ ẩn bytes ở console; SQL có tên chỉ ẩn query_id. Query unnamed giữ fingerprint. Các trường lỗi, reason, SQLSTATE, provider metadata vẫn hiện.
- Event không nhận diện được giữ định dạng key=value. Không thêm logging dữ liệu nhạy cảm.
- File JSON và file text giữ nguyên dữ liệu/format trước đây; không áp dụng compact cho file. Production JSON không đổi. Không có thay đổi về level, router, request context, database tracer hoặc writer lifecycle.
- Tham khảo [Gitea router logger](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/web/routing/logger.go#L118-L123) và [Zerolog FormatPrepare](https://github.com/rs/zerolog/blob/v1.34.0/console.go#L138-L158). Chỉ học cách trình bày; không sao chép logging engine.

Kiểm tra format và bảo toàn JSON bằng:

```sh
go test ./modules/log -run 'TestCompactConsolePreservesOriginalEvent|TestFullTextFileKeepsMetadata' -v -count=1
go test ./modules/log ./modules/setting ./routers ./models/db -count=1
```

## Caller & source location

Logger manager đăng ký một Zerolog hook chung ở logger khởi tạo sớm và logger sau cấu hình. Hook tìm frame ứng dụng đầu tiên bên ngoài modules/log và Zerolog bằng runtime.CallersFrames, không áp dụng một caller skip cố định cho tất cả wrapper.

JSON có hai field bổ sung: caller (đường dẫn tương đối theo package và số dòng) và caller_func (tên hàm). Ví dụ minh họa:

```text
10:30:05 [D] sql: CreateSession 1.382ms rows=1 rid=8b6424f1
10:30:05 models/db/query_log.go:68:(*queryLogger).TraceQueryEnd() [E] sql: CreateSession 2.4ms reason=postgres_error sqlstate=23505 rid=8b6424f1
```

Console compact chỉ hiện caller cho log nghiệp vụ/startup hoặc warning/error. HTTP/SQL thông thường (trace/debug/info) không lặp lại vị trí instrumentation. File JSON vẫn giữ caller nếu xác định được; không lưu đường dẫn tuyệt đối máy build. File text cũng có metadata mới. Caller là nơi phát event, không phải nguồn gốc lỗi hay tên HTTP handler. Nếu không xác định được frame ứng dụng, không gán một caller giả. Hook không ghi thêm error body, SQL, credential hay stacktrace.

Kiểm tra:

```sh
go test ./modules/log -run 'TestApplicationSource|TestIsLoggingFrame|TestCallerLoggerPaths|TestCompactCallerVisibility' -v -count=1
go test ./modules/log -count=1
```

Tham khảo [Gitea LoggerImpl.Log](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/log/logger_impl.go), [Zerolog event hooks](https://github.com/rs/zerolog/blob/v1.34.0/event.go#L141-L147). Không thay đổi router, error policy, log level, request ID hay cơ chế ghi output.

## HTTP request lifecycle

- http.started: DEBUG, trước khi gọi router. Ẩn ở INFO.
- http.slow: WARN, một lần sau 3 giây nếu timer chạy trước khi hoàn tất. Không hủy request, không đọc response hoặc request data từ goroutine timer.
- http.request: giữ contract hoàn tất (status, bytes, duration_ms, reason, level theo status). Console đổi thành router: completed.
- http.failed: ERROR nếu panic thoát ra khỏi handler. Không ghi raw panic, không bịa HTTP status; panic được truyền tiếp để recovery hiện hành xử lý. Panic đã bị middleware khác recover không được tự động phát hiện.
- handler và handler_source: action/provider gần nhất đã vào pipeline, không phải stack trace, không chắc là endpoint cuối. Guard chặn request thì tên guard có giá trị. 404 không match không bịa handler. Metadata lấy khi compile route, request giữ con trỏ atomic.
- Slow callback kết thúc trước event cuối; không thêm registry hoặc goroutine quét toàn bộ request như Gitea. Chi phí một timer/request chưa benchmark. Long polling chưa có phân loại riêng. Không thêm request query string, body, cookie, token, IP hoặc raw panic.

Ví dụ minh họa:

```text
10:30:05 [D] router: started GET /api/v1/health rid=8b6424f1
10:30:08 [W] router: slow GET /api/v1/health 3001ms @ v1.Health rid=8b6424f1
10:30:09 [I] router: completed GET /api/v1/health 200 4000ms @ v1.Health rid=8b6424f1
```

```sh
go test ./routers -run 'TestRequestLifecycle|TestRequestLoggingErrorsAndPrivacy' -v -count=1
go test ./modules/log -run 'TestRouterLifecycleConsole|TestCompact' -v -count=1
go test -race ./routers ./modules/web ./modules/log -count=1
```

Tham khảo [Gitea request record manager](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/web/routing/logger_manager.go) và [Gitea handler metadata](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/web/handler.go#L130-L164).

## Error diagnostics & stacktrace

APIErrorAuto (unexpected errors only) and APIErrorInternal keep their existing event names, HTTP status and public response. They now attach error_chain (at most 12 nodes, depth for joined errors), error_type, optional SQLSTATE, and error_chain_truncated if the walk exceeds its limit. No raw error text, SQL parameters, PostgreSQL detail, or panic value is added.

WithOperation annotates selected login/session failures without changing Error(), errors.Is or errors.As. Operation IDs must be static developer-owned labels; never pass request data. Other services can still return ordinary errors; the type chain is useful but does not reconstruct the original call path.

At DEBUG level, unexpected API errors include up to 12 application stack_frames from the reporting point. Escaping panics include these frames before rethrow at any enabled error level. These are active application frames, not a captured origin stack for returned errors. Compact console omits the chain and stack array, exposes operation labels and SQLSTATE when available. JSON or full text output retains diagnostic details; use file JSON or JSON console to inspect stack_frames. No new settings or log events for ordinary API errors. Existing database and HTTP result logging remains unchanged.

Example (illustrative):
```text
[E] sql: CreateSession 2.4ms reason=postgres_error sqlstate=23505 rid=8b6424f1
[E] http.application_error operations="auth.login.transaction > auth.session.create" sqlstate=23505 rid=8b6424f1
[E] router: completed POST /api/v1/auth/login 500 8ms @ auth.Login rid=8b6424f1
```

```sh
go test ./modules/log -run 'TestDescribeError|TestErrorDetailsPrivacyAndStack|TestDiagnosticConsole' -v -count=1
go test ./routers ./services/auth ./modules/log -count=1
```

Reference: [Gitea API internal error helper](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/services/context/api.go) and [stack formatting](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/log/event_format.go#L245-L252). BDSPro deliberately does not adopt raw error message logging.

## Phân luồng level và lọc console

Mỗi đích có LEVEL riêng, kế thừa [log] LEVEL nếu không khai báo. Logger tạo event từ level thấp nhất của các đích đang bật; đích không bật không ảnh hưởng. Do đó stack_frames cho lỗi API được bật khi ít nhất một đích đang bật dùng DEBUG. Không có override runtime; cần restart.

Ví dụ dev: terminal chỉ hiện HTTP thông thường và startup, file giữ debug tất cả component:

```ini
[log]
MODE = console,file
LEVEL = info
ROOT_PATH = data/log
[log.console]
FORMAT = console
LEVEL = info
COMPONENTS = default,http
[log.file]
FORMAT = json
LEVEL = debug
FILE_NAME = bdspro.log
[database]
LOG_SQL = all
```

COMPONENTS là allowlist chính xác, phân biệt hoa thường, tên dạng chữ thường, số (không ở đầu), dấu gạch ngang hoặc gạch dưới. Để trống là tất cả; default khớp event không có component. Bộ lọc chỉ áp dụng dưới WARN; WARN/ERROR vẫn đi qua nếu đạt LEVEL của console. Ví dụ LEVEL=error không hiện WARN dù thuộc allowlist. Cấu hình sai bị từ chối khi startup. File không áp dụng component filter; không thay đổi retention hoặc JSON schema. Với console JSON, quy tắc lọc vẫn áp dụng. Không bật filter khi cần stream JSON đầy đủ cho collector.

Để tập trung SQL, đổi console LEVEL=debug và COMPONENTS=sql; để tập trung IAM, dùng COMPONENTS=security,delivery,http. Đây là các nhãn hiện có, không tự tạo thêm log. LOG_SQL=all vẫn cần để phát query thành công. Muốn xem riêng file theo mục đích, truy vấn luồng JSON chung:

```sh
tail -F data/log/bdspro.log | jq -Rc --unbuffered 'fromjson? | select(.component == "sql")'
tail -F data/log/bdspro.log | jq -Rc --unbuffered 'fromjson? | select(.level == "error")'
tail -F data/log/bdspro.log | jq -Rc --unbuffered 'fromjson? | select(.component == "security")'
```

Cơ chế: Zerolog LevelWriter cung cấp level cùng payload, SyncWriter giữ đồng bộ, fanoutWriter thử tất cả sink kể cả khi một sink lỗi. filteredSink bỏ event trước formatter, trả len(p),nil cho event bị lọc. Chỉ console có allowlist và event dưới WARN mới cần giải mã component. Không tái serialize JSON, không tạo file riêng, không thêm registry hoặc queue. Chi phí giải mã component chưa benchmark. Tham khảo [Zerolog FilteredLevelWriter](https://github.com/rs/zerolog/blob/v1.34.0/writer.go) và [Gitea logger settings](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/setting/log.go).

```sh
go test ./modules/log -run 'TestComponentFilter|TestDestination|TestFileLevelOverride|TestLevelFanout' -v -count=1
go test ./modules/setting -run 'TestLog' -v -count=1
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
- all: thêm query thành công ở DEBUG; phải có ít nhất một đích bật LEVEL=debug để sinh event; đích cần xem cũng phải ở DEBUG.

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

Bản này giữ một luồng JSON có component/request_id để tìm xuyên tầng. Không tạo riêng file sql/security/debug vì nhu cầu hiện tại chưa cần routing/retention riêng. component là nhãn truy vấn và dùng cho allowlist console; level độc lập theo đích, không theo từng component. Chưa có benchmark chứng minh thông số rotation hoặc threshold là tối ưu cho tải production.

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

## Debugger integration

See [Developer debugging](developer-debugging.md) for Delve / VS Code on WSL, a read-only loopback HTTP/log smoke test, and offline formatter benchmarks/profiles. Do not expose debugger or pprof endpoints in the application. Running with breakpoints can change timing, so do not interpret that session's latency as performance evidence.
