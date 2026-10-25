# ARC-02 — HTTP Runtime & Application Ingress

- **Baseline:** `dev@59bb950` (25/10/2026), tiếp nối [ARC-01 — Network & Trust Boundary](arc-01-network-http-ingress.md).
- **Problem:** Một request có thể tiêu thụ socket, goroutine, CPU, RAM, Redis và PostgreSQL trước khi hoàn thành. Một `IPActive` trên route **không** tự bảo vệ tất cả tài nguyên đó.
- **Intent:** Phân định chủ sở hữu transport, request context, Gin middleware, handler và resource budget; chứng minh HTTP cancellation được truyền xuống các lời gọi PostgreSQL liên quan.
- **Phạm vi source:** `api/server.go`, `api/request_limits.go`, `api/middleware_prelogin.go`, `api/organization.go`, `api/token.go`, `main.go`, `db/sqlc/store.go`, test HTTP hiện có.
- **Thay đổi ARC-02:** Ở Organization và Token handlers, chuyển những lời gọi Store đang dùng `*gin.Context` sang `ctx.Request.Context()`; thêm test không cần DB thật để chứng minh truyền tín hiệu hủy, đồng thời test `RequestIngress` chặn request có Content-Length quá hạn trước handler. Không tạo thêm service, interface, dependency, migration hoặc config.
- **Không tuyên bố:** request deadline cho mọi nghiệp vụ; giới hạn số connection/goroutine toàn cục; database cancellation cho mọi code path; production readiness; G0 WSL runtime acceptance.

## 1. Kiến trúc thực thi một HTTP request

```text
OS / Docker TCP
    ↓ net.Listen("tcp", config.HTTPServerAddress)
net/http.Server.Serve(listener)
    ├─ HTTP parser & connection state
    ├─ ReadHeaderTimeout / ReadTimeout / MaxHeaderBytes
    └─ Handler: cors.New(...).Handler(server.router)
         ├─ OPTIONS preflight có thể kết thúc tại CORS
         └─ Gin Engine / request-scoped *gin.Context
              ↓ Middleware request ID; ghi request log; Recovery
              ↓ RequestMonitor (quan sát kết quả khi downstream trả về)
              ↓ RequestIngress (ContentLength + MaxBytesReader)
              ↓ Route-specific admission (nếu có) / Authentication
              ↓ Handler -> auth operation hoặc Store
                  ├─ PostgreSQL pgxpool / transaction
                  └─ Redis, worker/SMTP (theo use case)
              ↑ response; unwinding middleware & log
    ↑ WriteTimeout / connection close
```

**Không** nhầm `net/http.Server`, `http.Request.Context()` và `*gin.Context` thành một thứ. Server xử lý network/transport. HTTP request context là cancellation + deadline + value xuyên qua I/O. Gin context giữ params, input, response writer, middleware index và các giá trị `ctx.Set()`. Handler có thể sử dụng cả hai cho những trách nhiệm khác nhau.

## 2. HTTP Runtime Owner — giá trị đã có

Trong `api/server.go:StartWithReady`:

| Cơ chế | Giá trị | Phạm vi đúng | Không bảo đảm |
| --- | --- | --- | --- |
| `ReadHeaderTimeout` | 5 giây | Thời gian đọc request headers | Không giới hạn toàn bộ handler |
| `ReadTimeout` | 15 giây | Đọc request, gồm body | Không tạo timeout cho PostgreSQL transaction |
| `WriteTimeout` | 30 giây | Deadline ghi HTTP response theo cơ chế Go | Không tự hủy goroutine / truy vấn DB vì handler chậm |
| `IdleTimeout` | 60 giây | Kết nối keep-alive đang rảnh | Không kiểm soát active request |
| `MaxHeaderBytes` | 16 KiB | Request line + header (theo net/http) | Không phải giới hạn body |
| `RequestIngress` | 1 MiB body | Từ chối Content-Length khai báo vượt mức; giới hạn stream **khi body được đọc** | Không giới hạn mọi byte TCP nếu body chưa được đọc |
| `Shutdown` | 10 giây | Thời gian cố gắng graceful shutdown sau khi context phục vụ bị hủy | Không bảo đảm mọi công việc async hoàn tất; không là per-request deadline |

`StartWithReady` bind listener trước khi báo HTTP đang lắng nghe; shutdown được gọi khi `ctx.Done()`, dùng **shutdown context độc lập** được tạo với timeout 10 giây. Không có server-side `TLSConfig`/HTTPS listener trong runtime hiện tại. Những thứ này được xét trong ARC-01 và vận hành deployment, không giải quyết bằng một Gin middleware.

## 3. Request Admission Owner — trước khi vào controller

Source:

```go
func RequestIngress() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        if ctx.Request.ContentLength > maxRequestBodyBytes {
            ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
                "error": "request body too large",
            })
            return
        }
        ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxRequestBodyBytes)
        ctx.Next()
    }
}
```

- **Declared size > 1 MiB:** middleware abort với `413` trước route handler và các middleware sau nó.
- **Chunked/unknown size:** `MaxBytesReader` phát hiện vượt ngưỡng **khi handler đọc body**; các `Pre*` dùng `ShouldBindJSON` → `invalidJSON` để ánh xạ `*http.MaxBytesError` thành `413`.
- **Malformed JSON dưới ngưỡng:** input validation trả `400`, không phải `413`.
- **CORS preflight:** `rs/cors` có thể trả response trước Gin, nên không đi qua `RequestIngress`, `RequestMonitor` hay request ID Gin.

Trình tự middleware của password login trong `server.go`: global request ID → log/recovery → RequestMonitor → RequestIngress → SufLogin wrapper → IPActiveSource → PreLogin → IPActive → loginUser. `SufLogin` và `RequestMonitor` xử lý cả chiều unwind; không diễn đạt sai thành chuỗi chạy một chiều. **IPActiveSource** chỉ tồn tại trên các route đã cấu hình, không phải lớp bảo vệ socket/connection.

## 4. Hai context — nơi phân chia trách nhiệm quan trọng nhất

| Khu vực | Nên sử dụng | Vì sao |
| --- | --- | --- |
| HTTP params, status, response, `ctx.Set/Get` | `*gin.Context` | Gắn với request, không phải context sống dài |
| DB query / Store transaction / Redis / HTTP client chấp nhận `context.Context` | `ctx.Request.Context()` | Kế thừa cancellation từ request gốc |
| Logging request | `log.Ctx(ctx.Request.Context())` | Middleware gắn logger + request ID vào request context |
| Goroutine xử lý nền độc lập | Task/job context do worker sở hữu | Không giữ `*gin.Context` sau response; không lấy request context làm lifespan của job đã nhận |
| `Context.WithTimeout` cho operation | Chỉ nếu đã xác định budget riêng và ownership | `WriteTimeout` không thay thế deadline cho DB/external API |

Go `net/http` cancel server-side request context khi kết nối client đóng, request HTTP/2 bị hủy hoặc `ServeHTTP` kết thúc. Điều đó hữu ích khi được truyền xuống các I/O operations, nhưng **không tự dừng code CPU-bound** nếu code không chủ động kiểm tra `ctx.Done()`.

**Đã phát hiện từ source:** `api/organization.go` và `api/token.go` trước ARC-02 truyền `ctx` (`*gin.Context`) trực tiếp vào Store. Gin có tùy chọn `ContextWithFallback`, nhưng `setupRouter()` hiện không bật chế độ này; do đó không nên mặc định `ctx.Done()` là request cancellation. ARC-02 chuyển tám lời gọi Store được rà soát ở hai file đó sang `ctx.Request.Context()`, không thay đổi HTTP status, SQL hoặc authorization. Các endpoint khác cần tiếp tục được rà soát khi chạm vào từng capability; không tuyên bố mọi đường gọi đã được chuyển đổi.

**Test có giá trị:** `TestARC02StoreReceivesHTTPContext` tiêm Store spy, gửi access JWT hợp lệ tới `POST /organizations`, chứng minh context mà Store nhận có request value và `Done()` nhận được cancellation. Stub trả lỗi chủ động trước database write, nên test không cần PostgreSQL thật.

## 5. Resource Consumption Map

| Tài nguyên | Bắt đầu tiêu thụ ở đâu | Bảo vệ hiện tại | Khoảng trống / hướng nghiên cứu |
| --- | --- | --- | --- |
| Socket / FD / kernel buffers | Trước handler, tại TCP accept | OS/Docker, Go idle/read deadlines | Chưa có limit kết nối đồng thời ở ingress được xác nhận |
| Goroutine / scheduler time | Go xử lý connection/request | Go scheduler + HTTP timeouts | Không có global in-flight semaphore/resource budget do BDSPro cấu hình |
| Header memory & parse CPU | `net/http` parse HTTP | `MaxHeaderBytes`, `ReadHeaderTimeout` | Không phải protection khỏi network flood |
| Body memory & decode CPU | `Pre*` / JSON binder | `RequestIngress` 1 MiB, input validation | Những route không đọc body không phát hiện exceed của stream qua parser |
| Redis commands | `IPActiveSource` và `IPActive` | Atomic Lua/quota, fail-closed auth routes | Redis latency/availability có thể gây 503; không phải HTTP transport lỗi |
| Password hashing | `auth.Login` / Register | Đứng sau admission ở route liên quan | CPU-bound bcrypt/check password không có generic request cancellation guard |
| DB pool / query | `pgxpool.New`, `Store` methods | Pool của pgx + SQL constraints, context được truyền đúng ở các flow được sửa | Không có pool sizing/latency budget tùy chỉnh được xác nhận; không suy đoán số connection max |
| Response / log | Handler và middleware unwind | `WriteTimeout`, Zerolog, route-pattern logging | Không có tracing/metrics SLO bắt buộc hiện tại |
| Task/SMTP | Redis/Asynq sau operation | Worker retries và Mailpit dev | Delivery và DB–Redis atomicity thuộc ARC-07, không phải ARC-02 |

Vì giới hạn tài nguyên phát sinh **trước IPActive**, một middleware theo IP không thể thay thế L0 Network, L1 HTTP deadlines hoặc giới hạn tài nguyên database. Đối với request đồng thời gửi tới route không có IPActive, không thể tuyên bố đã có app-wide concurrency limit.

## 6. Các đường thực thi khác nhau

**Public HTTP Login:** ingress → source admission → parse → per-identity admission → handler → auth verification/session → SQL. `Request.Context()` cần xuyên qua các I/O liên quan.

**Protected Organization:** ingress → bearer Authenticate → handler → SQL owner/membership predicate. Request context quản lý lifecycle I/O; bearer chỉ là danh tính, **không** tự là authorization.

**Background Worker:** Asynq có lifecycle và task context riêng, không đi qua Gin; không dùng `*gin.Context` cho task sống sau response. Worker retry/DB transaction là capability khác của hệ thống.

## 7. Stack decision và dependency ownership

| Lĩnh vực | Stack hiện tại | Quyết định ARC-02 |
| --- | --- | --- |
| HTTP runtime | Go `net/http.Server`, `context` | Giữ, dùng đúng context |
| Gin | `github.com/gin-gonic/gin` | Giữ route/middleware; không yêu cầu thay router |
| Body protection | `http.MaxBytesReader` | Giữ 1 MiB, nghiệm thu negative path |
| CORS | `github.com/rs/cors` | Giữ outer wrapper đã xác nhận ở ARC-01 |
| DB I/O | pgxpool + sqlc/Store | Truyền `ctx.Request.Context()` từ handler |
| Async | Asynq/Redis | Task lifecycle độc lập với Gin |
| Concurrency/rate limiter package | Chưa thêm | Chưa có số đo để chọn semaphore, circuit breaker, load shedder mới |

Không dựng `httpcore`, `BaseHandler`, `RequestPipeline` hoặc interface mới nếu không có responsibility rõ ràng.

## 8. Acceptance Matrix — cái gì được và chưa được chứng minh

| Case | Bằng chứng | Kết luận giới hạn |
| --- | --- | --- |
| Body Content-Length lớn | `TestARC02IngressRejectsDeclaredOversizeBeforeHandler` | 413 và handler chưa được gọi |
| Context từ HTTP tới DB | `TestARC02StoreReceivesHTTPContext` với Store spy | Request value/cancellation đi đến operation được test |
| Body chunked, forged proxy IP, limiter Redis | `api/request_limits_test.go` hiện hữu | Chỉ xác nhận trong test scope |
| CORS trước Gin | `api/http_boundary_test.go` từ ARC-01 | OPTIONS có thể dừng trước Gin |
| Build, migration, smoke, vet, race | CI PR / `test.yml` | Source/CI acceptance |
| TCP slowloris, số goroutine cực đại, sustained load | Chưa có đo lường có kiểm soát | **Không** tuyên bố tài nguyên đã được hard-cap |
| HTTP timeout → DB rollback mọi route | Chưa có test xuyên mọi route | **Không** tuyên bố per-use-case deadline |
| WSL network, proxy/TLS, restore | G0/ARC-01 nghiệm thu riêng | **Chưa đạt runtime acceptance** |

## 9. Các quyết định để mở, không tự ý triển khai

1. **Request deadline theo operation:** xác định SLO và behavior trước khi dùng `context.WithTimeout` cho login/DB/provider calls.
2. **Concurrency budget:** trước khi thêm semaphore, cần đo số request đồng thời, goroutine, CPU, DB pool waits và số 503/429.
3. **Graceful shutdown thực tế:** kiểm chứng kết thúc in-flight HTTP và worker trong điều kiện phù hợp; timeout 10 giây không chứng minh data delivery exactly once.
4. **Context propagation audit:** rà soát các handler/provider còn truyền `*gin.Context` như `context.Context` và những operation không check cancellation.
5. **Observability:** tiêu chuẩn metrics/tracing sau khi có SLO; trước mắt giữ log request ID và không ghi raw URL/token.

## 10. Tiếp nối roadmap

- **ARC-01:** đã xác định network/edge/trust boundary; vẫn mở WSL exposure/production edge.
- **ARC-02:** HTTP runtime + Gin execution + context/resource boundaries; code/test cụ thể.
- **ARC-03:** Request Admission & Resource Protection: `IPActiveSource`, `IPActive`, Redis Lua, thuật toán/counter/failure, test concurrent/quota.
- **ARC-04:** Authentication & Authorization (identity ≠ quyền tài nguyên).
- **ARC-05+:** Application/Transaction, Async, Observability và Operations; Business Core chưa triển khai.

## Tài liệu tham chiếu

- Go `http.Server`, `Request.Context`, `MaxBytesReader`: https://pkg.go.dev/net/http
- Gin request context and cancellation: https://gin-gonic.com/en/docs/server-config/context/
- Gin `ContextWithFallback` và `Context.Copy()`: https://github.com/gin-gonic/gin/blob/master/gin.go
- [ARC-01](arc-01-network-http-ingress.md) · [Request Lifecycle](request-lifecycle.md) · [Core Ownership Matrix](core-ownership.md) · [G0 Runbook](../operations/g0-runtime-acceptance.md)
