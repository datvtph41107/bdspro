# ARC-01 — Network, Edge & HTTP Ingress / Trust Boundary

- **Baseline:** `dev@cd1b526` (25/10/2026), BDSPro Go/Gin HTTP + Docker Compose
- **Mục đích:** xác định chính xác nơi request được tiếp nhận, nguồn dữ liệu được tin, tài nguyên bị tiêu thụ và trách nhiệm các lớp trước `IPActiveSource`.
- **Loại thay đổi:** architecture contract + HTTP-boundary tests. **Không** thay đổi Go runtime, Compose ports, SQL, Redis, migrations hay policy rate limit.
- **Trạng thái:** kiến trúc trong source được đối chiếu; nghiệm thu WSL/mạng ngoài và quyết định production ingress **chưa** thực hiện.

## 1. Câu hỏi nền tảng: request gặp lớp nào đầu tiên?

```text
Client (DNS lookup/route; cấu hình DNS không nằm trong repo)
   -> TCP connection / host OS + Windows/WSL/Docker network path
   -> Docker published port (Compose: api 8080:8080)
   -> Go net.Listen("tcp", HTTP_SERVER_ADDRESS) + http.Server.Serve
      ├─ HTTP protocol parsing / headers / deadlines
      └─ cors.New(...).Handler(server.router)  [HTTP wrapper]
           ├─ CORS preflight OPTIONS có thể dừng trước Gin
           └─ Gin Engine
                -> request ID -> request log + recovery
                -> RequestMonitor -> RequestIngress
                -> matching route pipeline (ví dụ login: SufLogin wrapper
                   -> IPActiveSource -> PreLogin -> IPActive -> loginUser)
```

**Không phải mọi request đều chạm IPActive.** `GET /health` không có IPActive; `GET /users/me` dùng Authenticate; `OPTIONS` có thể được CORS middleware trả về trước Gin; request HTTP hỏng có thể bị Go server từ chối trước khi đi đến `Handler`.

`IPActiveSource` là **admission ở mức route**, không phải firewall/TCP protection và không ngăn việc client đã mở kết nối, gửi header hoặc chiếm một phần tài nguyên HTTP server.

## 2. Ma trận lớp hạ tầng và nguồn sự thật

| Lớp | Cơ chế / nơi chạy | Trách nhiệm và nguồn tin | Điều không đảm bảo |
| --- | --- | --- | --- |
| N0 DNS/routing | OS / network ngoài repository | Chuyển hostname thành đích kết nối; routing | Không có DNS/proxy production config được xác minh trong repo |
| N1 TCP/host boundary | Windows/WSL + Docker publish | Cổng host nào được bind và ai có thể kết nối | Không chứng minh firewall/reachability từ mạng ngoài nếu chưa kiểm tra trên máy |
| N2 Go listener | `api/server.go:StartWithReady` | `net.Listen("tcp", address)`, nhận connection | Không mã hóa TLS trong server hiện tại; không có connection admission/concurrency budget riêng |
| N3 HTTP parser & deadline | `http.Server` trong `api/server.go` | Header/connection/request read & write limits | Không tự rate limit theo account/IP; timeout không phải bảo đảm mỗi use case hoàn thành đúng hạn |
| N4 CORS HTTP wrapper | `Server.Handler` → `rs/cors` | Browser origin policy; preflight có thể hoàn tất trước Gin | **Không phải Authentication**, không chặn non-browser client chỉ vì Origin không được allow |
| N5 Gin request context | `setupRouter()` | Server-issued request ID, safe request log, panic recovery, route selection | Không có authorization nếu route không áp dụng chính sách đó |
| N6 HTTP request body | `RequestIngress()` trong `api/request_limits.go` | Từ chối `ContentLength` khai báo lớn; `http.MaxBytesReader` giới hạn khi body được đọc | Không phải giới hạn mọi byte TCP được gửi nếu handler không đọc body |
| N7 Per-route admission | `IPActiveSource` ở các auth routes | Redis counter theo IP peer trước khi decode credentials | Không có mặt trên mọi route; Redis limiter không bảo vệ khỏi kết nối TCP flood |
| N8 Auth/Business | `Pre*`, `IPActive`, `Authenticate`, handler/SQL | Parse, identity, permission, business logic theo route | Không thay được các giới hạn từ N1–N6 |

Các lớp N0–N8 là **bản đồ nghiên cứu**, không yêu cầu tạo các thư mục Go tương ứng.

## 3. Cấu hình đã có (không phải đề xuất)

Trong `api/server.go`:

```go
listener, err := net.Listen("tcp", address)
httpServer := &http.Server{
    Handler:           server.Handler(),
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    16 << 10,
}
```

- `ReadHeaderTimeout`: thời gian cho phép đọc request headers.
- `ReadTimeout`: thời hạn đọc **request bao gồm body**; không đồng nghĩa tổng handler timeout.
- `WriteTimeout`: thời hạn cho việc ghi response theo ngữ nghĩa Go; không được tuyên bố tự giới hạn toàn bộ công việc có DB/external call.
- `IdleTimeout`: giữ idle keep-alive trong thời gian hữu hạn.
- `MaxHeaderBytes`: giới hạn kích cỡ header/request-line; **không** giới hạn body.

Giới hạn body hiện tại `maxRequestBodyBytes = 1 << 20` (1 MiB): nếu `ContentLength` lớn thì `RequestIngress` trả HTTP 413; với request không xác định trước độ dài, `MaxBytesReader` phát hiện khi handler đọc/parse body. `invalidJSON` ánh xạ `MaxBytesError` → 413.

`app.env`: `HTTP_SERVER_ADDRESS=0.0.0.0:8080`. `docker-compose.yaml`: `api: "8080:8080"`, `postgres: "5432:5432"`, `redis: "6379:6379"`, `mailpit: "127.0.0.1:8025:8025"`. Khi publish không chỉ định host IP, Docker thường bind mọi interface; việc bên ngoài truy cập được hay không còn tùy firewall và môi trường Docker/WSL. Đây là **rủi ro phạm vi network cần nghiệm thu**, không phải bằng chứng production hiện đã bị truy cập từ Internet.

## 4. IP identity và Trust Boundary

`router.SetTrustedProxies(nil)` trong `api/server.go` có tác dụng: `ctx.ClientIP()` dựa trên remote peer, **không** chấp nhận `X-Forwarded-For` hoặc `X-Real-IP` do client tự thêm.

**Hiện tại (direct HTTP):**

```text
peer IP của kết nối -> Gin ClientIP -> Redis IPActiveSource/monitor fingerprint
```

**Nếu sau này đặt reverse proxy:**

```text
client -> proxy -> Go: RemoteAddr = proxy IP
```

Khi đó nếu giữ `SetTrustedProxies(nil)`, mọi client phía sau một proxy có thể chia sẻ cùng rate-limit key — gây false positive/không công bằng. Tuy nhiên, đổi ngay thành tin mọi forwarded header sẽ mở khả năng giả IP để né limiter.

**Điều kiện trước khi bật trusted proxy:**

1. Chốt topology và **địa chỉ CIDR của proxy thực sự tin cậy**; không dùng wildcard.
2. Edge/proxy phải loại bỏ hoặc ghi đè các header IP từ client theo chính sách rõ ràng.
3. Đường truy cập trực tiếp vào app phải bị giới hạn để không bỏ qua edge controls.
4. Test một request giả `X-Forwarded-For` đi trực tiếp và request thật qua proxy; chứng minh `ClientIP` đúng.
5. Test rate limiting chia sẻ counter khi nhiều instance và khi nhiều client chung một NAT.

Đây là **điều kiện triển khai tương lai**; không thay đổi `SetTrustedProxies(nil)` trong ARC-01.

## 5. CORS không phải lớp bảo mật danh tính

`Server.Handler()` bọc Gin bằng `cors.New(...).Handler(...)`: allowed origins lấy từ config, allowed headers `Content-Type`/`Authorization`, credentials cho phép theo cấu hình.

- Request browser cross-origin cần CORS hợp lệ để JS đọc response.
- HTTP client không bị xác thực hoặc phân quyền chỉ nhờ CORS; non-browser có thể tự gửi `Origin`.
- Preflight `OPTIONS` được thư viện CORS xử lý **trước** Gin. Những request như vậy có thể không đi qua `RequestMonitor` hoặc nhận server-generated `X-Request-ID` như các Gin request bình thường.
- Không dùng CORS như cơ chế thay thế `Authenticate`, `Authorize` hoặc `IPActive`.

## 6. Security & failure boundary

| Tình huống | Nơi bắt đầu xử lý | Kỳ vọng/giới hạn |
| --- | --- | --- |
| TCP connection đến cổng bị đóng | Host/Docker/OS | Không đi vào Go handler |
| HTTP header quá lớn / gửi quá chậm | `http.Server` | Server có thể từ chối trước Gin; không đòi Gin luôn trả JSON |
| `ContentLength` > 1 MiB | Gin `RequestIngress` | 413, không gọi handler |
| Body chunked > 1 MiB, handler parse | `MaxBytesReader` + `invalidJSON` | 413 khi đọc vượt giới hạn |
| Forged `X-Forwarded-For` direct | Gin trusted proxy disabled | `ClientIP` không bị ghi đè bởi forwarded header |
| CORS allowed preflight | `rs/cors` | CORS response trước Gin; không phải authentication |
| CORS disallowed Origin | `rs/cors` | Không tự suy ra request backend bị cấm thực thi |
| Login burst | Redis `IPActiveSource` → `PreLogin` → `IPActive` | 429 theo policy; đây là ARC-03, không phải điều kiện N1 TCP |
| Redis limiter mất kết nối | Active auth-route admission | 503 fail-closed theo source; **không** phải lỗi Go TCP transport |
| Slow DB trong một handler | Application/DB timeout policy | `WriteTimeout` không bảo đảm rollback/cancel transaction đúng |

## 7. Ma trận stack: hiện tại, tương lai, quyết định

| Capability | Công nghệ hiện tại | Phương án tương lai / điều kiện kích hoạt | Quyết định ARC-01 |
| --- | --- | --- | --- |
| TCP / HTTP listener | Go `net.Listen`, `net/http` | Tune theo latency/resource evidence | Giữ |
| HTTP router | Gin | Không thay router để giải quyết network | Giữ |
| CORS | `github.com/rs/cors` | Rà soát origins theo môi trường/browser contract | Giữ |
| Trust source | `SetTrustedProxies(nil)` | Trusted proxy CIDR khi có proxy topology thật | Giữ |
| Reverse proxy/TLS | Chưa có trong Compose | Caddy/Nginx/Envoy khi triển khai HTTPS/domain/edge | **Chưa chọn** |
| L3/L4 / host firewall | Ngoài Go repo | Chốt theo hạ tầng deployment thực tế | **Chưa xác minh** |
| Edge rate-limit/WAF | Chưa có | Chỉ sau khi đo risk/traffic và thiết kế trust boundary | **Không thêm lúc này** |
| Route admission | Redis Lua + Gin middleware | ARC-03 nghiên cứu algorithm/quota | Giữ, không trộn với ARC-01 |

Không tạo package `network`, `edge` hoặc interface giả khi source không có dependency/behavior cần trừu tượng hóa.

## 8. Acceptance & evidence

**Đã có trên source:** `TestRequestBodySizeLimit` (declared/chunked), `TestUntrustedProxyCannotSpoofClientIP`, `TestAuthRateLimitUnavailableRejectsLogin`, các test Redis quota/concurrency trong `api/request_limits_test.go`. ARC-01 thêm một test ở **`server.Handler()` thực** để bảo vệ thứ tự CORS wrapper và Gin.

**Cần trên CI:** Go tests, Go build, static analysis, race detector, Compose config validation như workflow hiện tại.

**Cần trên WSL, chỉ đọc cấu hình và cổng:**

```sh
cd ~/projects/bdspro
git status --short --branch
docker compose config --quiet
docker compose ps
docker compose port api 8080
docker compose port redis 6379
docker compose port postgres 5432
```

Các lệnh `docker compose port` cho biết địa chỉ/port được publish, **không** chứng minh Internet reachability. Không tự gọi `docker compose up/down`, không xóa volume. Nếu stack đang chạy và đã xác minh owner, chạy G0 script `sh scripts/accept-g0.sh` theo runbook. Không thực hiện các request lỗi hàng loạt trên dữ liệu development thực vì có thể làm thay đổi Redis rate counters.

**Chưa nghiệm thu:** firewall và route thực tế Windows/WSL, reachability từ máy ngoài, TLS/certificate, reverse proxy behavior, backup/restore. Không được đánh dấu ARC-01 production-ready khi chưa có bằng chứng.

## 9. Điều kiện hoàn tất ARC-01

1. Có một bản đồ phân biệt N0–N8 và thực sự đối chiếu `server.go`, `request_limits.go`, Compose, `app.env`.
2. Có test chứng minh `rs/cors` nằm trước Gin và hoạt động đúng với origin được phép; test hiện có bảo vệ body limit và forged IP.
3. CI PR thành công; diff không thay đổi business/runtime/database.
4. Các gap về host exposure/TLS/reverse-proxy được ghi rõ, không bị suy diễn là đã đóng nghiệm thu WSL.
5. **ARC-02** tiếp tục đi sâu HTTP Runtime & Application Ingress; **ARC-03** mới đi sâu IPActive/rate algorithm.

## 10. Tham khảo kiểm chứng bên ngoài

- Go `net/http.Server`: https://pkg.go.dev/net/http#Server
- Gin Trusted Proxies: https://gin-gonic.com/en/docs/server-config/trusted-proxies/
- Docker Port Publishing: https://docs.docker.com/engine/network/port-publishing/
- OWASP API4:2023 Unrestricted Resource Consumption: https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption/
- BDSPro request pipeline: [request-lifecycle.md](request-lifecycle.md)
- BDSPro Engineering ownership: [core-ownership.md](core-ownership.md)
- G0 WSL runbook: [g0-runtime-acceptance.md](../operations/g0-runtime-acceptance.md)
