# ARC-03 — Request Admission & Resource Protection: evidence before abstraction

**Baseline:** `dev@46566c2`, sau [ARC-02](arc-02-http-runtime-ingress.md). **Trạng thái:** khảo sát và nghiệm thu hành vi source đang có; **không** đưa vào thuật toán, thư viện, service hay security policy mới nếu thiếu failure case.

## 1. Problem / Intent

Một request đến BDSPro có thể tiêu thụ chi phí parse JSON, xác minh password, truy cập PostgreSQL/Redis và phát token. Các auth operations hiện cần giới hạn lượt thử của nguồn kết nối và danh tính. Nhưng việc thấy tên `IPActive` trong source **không đủ chứng minh** nó từ chối đúng đối tượng, đúng thứ tự, đúng khi request malformed hoặc khi nhiều request đồng thời.

**Intent của ARC-03:** kiểm chứng trực tiếp chuỗi hiện có, xác định failure modes còn mở, và chỉ thay production code nếu kiểm thử tìm ra lỗi có tác động.

## 2. Owners và trust boundaries hiện hữu

| Stage | Owner trong source | Input | Thực thi | Phạm vi/giới hạn |
| --- | --- | --- | --- | --- |
| Nguồn IP | `api/server.go` / Gin `SetTrustedProxies(nil)` | TCP remote peer | `ctx.ClientIP()` | Không tin client-injected forwarding headers; sau reverse proxy chưa cấu hình trusted list |
| Global ingress | `api/request_limits.go:RequestIngress` | HTTP Content-Length / body | 1 MiB, trả 413 | Trước các middleware riêng của route; không phải quota requests |
| Source admission | `api/middleware_ipactive.go:IPActiveSource` | `operation`, direct peer IP | Redis Lua `INCR` + TTL, 429 | Trước `PreLogin` trên password login; Redis lỗi trả 503 |
| Input shape | `api/middleware_prelogin.go:PreLogin` | JSON email/password | Bind, validation, 400 | Không kiểm tra credential/password |
| Identity admission | `api/middleware_ipactive.go:IPActive` → `allowAuthRequest` | `operation`, normalized identity, IP | Redis Lua atomic 2 counters, 429 | Redis lỗi trả 503; không phải Authentication hoặc Authorization |
| Authentication | `api/user.go:loginUser` → `auth.Login` | Credentials | Verify + IssueSession | Không phải một phần của rate limiter |
| Observation | `RequestMonitor`, `SufLogin` | HTTP outcome | log/denial counter | Quan sát response theo chiều unwind; không tự chặn bằng threat score |

`IPActiveSource` và `IPActive` ở **route-level**, không có global quota cho tất cả các route. Không mô tả một request `GET /health` như đi qua Redis limiter.

## 3. Source-derived policy snapshot

| Operation | Source quota | Parsed admission quota | Cơ chế |
| --- | --- | --- | --- |
| Register | 10 / 1h / IP | 10 / 1h / IP, 3 / 1h / normalized email | `POST /users` |
| Password login | 30 / 1m / IP | 30 / 1m / IP, 10 / 1m / normalized email | `POST /users/login` |
| Verify email | 60 / 1m / IP | 60 / 10m / IP, 10 / 10m / email ID | `GET /users/verify_email` |
| Refresh access | 60 / 1m / IP | 60 / 20m / IP, 20 / 20m / token hash | `POST /tokens/renew_access` |
| Revoke refresh | 60 / 1m / IP | 60 / 20m / IP, 20 / 20m / token hash | `POST /tokens/revoke` |

Các ngưỡng trên được chép từ `api/server.go`, **không** phải thông số đã được tuning bởi traffic measurement. Google OIDC và Phone OTP có route registration riêng theo cấu hình/environment; cần được đối chiếu độc lập trước khi tuyên bố coverage 100%.

**Redis key separation:** `bdspro:source_limit:v1:<operation>:<hashed IP>` khác namespace `bdspro:auth_limit:v1:<operation>:ip|identity:<hash>`. Hash là SHA-256 dạng hex để không đưa trực tiếp email/token vào Redis key; hash không tương đương encryption/anonymization hoàn toàn.

## 4. Technical Value Gate — kết luận theo evidence

| Giả định / vấn đề | Evidence trước ARC-03 | Việc làm tối thiểu | Phán quyết hiện tại |
| --- | --- | --- | --- |
| malformed login đi được tới parse mà không bị tính quota | Source chain có `IPActiveSource` trước `PreLogin`; thiếu test ở route chain với malformed input | Test 400, 400, 429 trên synthetic route cùng middleware order | **Thêm bằng chứng**, không đổi production |
| một email thay chữ hoa/đổi IP né identity quota | Code `strings.ToLower/TrimSpace` trong `allowAuthRequest`; đã có test Lua cấp thấp | Test `PreLogin` + IPActive với ba IP và case khác nhau | **Thêm bằng chứng**, không đổi production |
| concurrent attempts có thể vượt source IP quota | Redis Lua `INCR` atomic theo một Redis instance; trước đó đã test atomic identity script | Test 20 request concurrent với quota 4 qua Gin/Redis | **Thêm bằng chứng**, không đổi production |
| Redis outage cho phép vượt limiter | Test `TestAuthRateLimitUnavailableRejectsLogin` đã có | Giữ test 503 fail-closed; không trùng lặp | **Đã được bảo vệ trong scope test** |
| phải thay fixed-window bằng token bucket / WAF / Envoy | Chưa có tải đo, false positives, bypass hay SLA cụ thể | Không thêm dependency | **Hoãn** |
| retry-after chính xác đến giây hết hạn | Code trả nguyên `window.Seconds()` khi 429, không đọc Redis TTL | Chưa có requirement/user-impact đủ để sửa response ngay | **Rủi ro protocol/UX cần ghi nhận, không coi là thời gian chờ thực** |
| IP sau proxy được nhận diện đúng | `SetTrustedProxies(nil)` tin direct peer | ARC-01 có test chống spoofing; proxy chưa triển khai | **Không bật wildcard trust** |

**Phạm vi của test mới:** một synthetic Gin route đi qua cùng middleware implementation và Redis thật của môi trường test. Test không chạy registration/login thật, không tạo account/session hoặc ghi PostgreSQL. `REDIS_TEST_ADDRESS` không có thì integration tests sẽ skip (theo helper hiện tại); CI có Redis và cấu hình env tương ứng.

## 5. Trade-offs và failure mode

- **Fixed window:** `INCR` đầu cửa sổ gắn `EXPIRE` lần đầu. Dễ hiểu và atomic, nhưng tại ranh giới hai cửa sổ có thể cho phép burst gần gấp đôi quota định mức trong khoảng ngắn; **chưa có traffic evidence** để thay thuật toán.
- **Shared/NAT IP:** nhiều client có thể dùng chung source counter; identity counter không loại bỏ false positives của stage source. Cần đo false positive/support signals trước khi thay policy.
- **IP identity:** direct-peer tin cậy chống header spoofing nhưng sẽ thành IP proxy nếu thêm reverse proxy mà không sửa trust topology.
- **Counter cost:** lượt thử bị từ chối vẫn được tính; denial không kéo dài TTL trong Lua hiện tại. Điều này là quy tắc đang có, không tự suy ra hệ thống chống mọi kiểu DoS.
- **Fail-closed:** Redis downtime làm auth-route admission trả 503. Phải có kế hoạch reliability/monitoring trong môi trường thực, nhưng không thay bằng fail-open chỉ để tránh 503.
- **No service-layer expansion:** dùng `IPActiveSource` và `IPActive` hiện có; chỉ phân tích/đo trước khi cân nhắc shared limiter abstraction.

## 6. Acceptance

1. `TestARC03SourceAdmissionPrecedesJSONParsing` chứng minh malformed attempt được đếm và 429 xuất hiện trước input binder.
2. `TestARC03IdentityAdmissionAcrossClientIPs` chứng minh normalized email quota không bị bypass bằng chữ hoa hoặc đổi direct-peer IP.
3. `TestARC03SourceQuotaAtomicAcrossConcurrentRequests` chứng minh source quota không bị vượt trong 20 request đồng thời qua Gin/Redis.
4. Tests trước đó về Redis outage, IP spoof, identity Lua concurrency vẫn chạy; CI build, migrations, HTTP smoke, race PASS.
5. Diff không chứa production Go changes, SQL, Compose, framework hay migration. Nếu test tìm ra hành vi sai, phải sửa với root-cause explanation và test regression, không chỉ đổi expectation cho xanh.

**Giới hạn bằng chứng:** Test chạy với Redis CI/synthetic routes, không phải đo tải production, firewall/WAF, multi-region, một cụm Redis failover, hay WSL public exposure. Không tuyên bố throughput tăng hoặc threat detection đã được triển khai.

## 7. Nghiên cứu/đo tiếp theo (chưa cho phép triển khai vội)

- Chọn các operation chi phí cao và đo tài nguyên tiêu thụ/latency tại từng stage admission.
- Kiểm thử fixed-window boundary và tình huống chung NAT; ghi nhận tác động quota thực tế.
- Đối chiếu Google/Phone route matrix và retry behavior trước khi thay policy.
- Khi có proxy production: đo đúng client IP trust topology, không chấp nhận mọi forwarded header.

**ARC-04** mới đi sâu Authentication/Authorization và resource ownership; ARC-03 không đồng nhất rate limiting với policy quyền.

## Nguồn tham chiếu

- [ARC-01 — Network/Trust Boundary](arc-01-network-http-ingress.md)
- [ARC-02 — HTTP Runtime & Request Cancellation](arc-02-http-runtime-ingress.md)
- [Request Lifecycle](request-lifecycle.md)
- [Core Ownership Matrix](core-ownership.md)
- Redis Lua atomicity: https://redis.io/docs/latest/develop/programmability/eval-intro/
- OWASP API4:2023 resource consumption: https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/
