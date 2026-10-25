# BDSPro

Backend Go/Gin, PostgreSQL, SQLC, JWT và Redis/Asynq. Phiên bản nền tảng được chọn theo bản stable đang hỗ trợ: **PostgreSQL 18**, **Go 1.27.3**, **Alpine 3.24**. Redis 7 được giữ tạm trong giai đoạn này để không thay đổi major version của worker khi chưa kiểm tra tương thích Redis/Asynq. Cách chạy, Dockerfile, Docker Compose, Makefile và GitHub Actions giữ bố cục đơn giản theo [TechSchool SimpleBank](https://github.com/techschool/simplebank), với các giá trị đổi thành `bdspro`.

**Runtime hiện tại:** một Gin HTTP server trên cổng 8080, Redis worker xử lý email verification, SMTP Mailpit local; Authentication gồm password/JWT/session, Google OIDC khi cấu hình credentials và Phone OTP development-only. `runGrpcServer`/`runGatewayServer` cùng protobuf/gateway generated source còn trong repository để tham khảo, **không được khởi động**. Chưa cấu hình production deployment.

## Engineering Governance — ENG-GOV-01

Quy chuẩn làm việc trên `dev` được thiết lập theo **Problem → Intent → Responsibility/Owner → Invariant → Implementation → Evidence → Acceptance**, không ép mọi nghiệp vụ phải tạo framework, interface hoặc service layer mới.

- [Core Ownership Matrix](docs/engineering/core-ownership.md): capability/data/execution/change ownership và các bất biến đã hiện diện trong source.
- [Development & Change Contract](docs/engineering/development-contract.md): quy chuẩn tổ chức code, migration, test, review và nghiệm thu.
- [ENG-GOV-02 — Workspace Guardrails](docs/engineering/workspace-guardrails.md): vòng làm việc chuẩn và kiểm tra CI chống sửa/xóa migration lịch sử.
- [ADR-0001 — Bounded Shared Core](docs/engineering/decisions/0001-core-ownership.md): lý do không sao chép toàn bộ DRX Laravel hoặc Java DRWash.
- [Request Lifecycle](docs/engineering/request-lifecycle.md): middleware thực tế của Gin HTTP. Pull request mới dùng [PR template](.github/PULL_REQUEST_TEMPLATE.md).
- [ARC-01 — Network, Edge & HTTP Ingress](docs/engineering/arc-01-network-http-ingress.md): phân định TCP/Docker, HTTP Server, CORS, Gin, trusted IP và giới hạn bảo vệ trước IPActive.
- [ARC-02 — HTTP Runtime & Application Ingress](docs/engineering/arc-02-http-runtime-ingress.md): lifecycle HTTP/Gin, request cancellation, resource budgeting và test ranh giới.
- [ARC-03 — Request Admission Evidence](docs/engineering/arc-03-request-admission-evidence.md): kiểm chứng hành vi IPActive theo malformed request, identity quota và concurrency trước khi cân nhắc thư viện mới.
- [ENG-GOV-03 — Formula Catalog](docs/engineering/capability-formula-catalog.md): 3 context, 18 công thức/mô hình ra quyết định, phạm vi reuse và evidence.
- [ENG-GOV-03 — Capability Workbook](docs/engineering/capability-workbook.md): mẫu Platform × Domain áp dụng vào Register/Asynq và ví dụ Calculation/Pricing chưa triển khai.
- [BIZ-01 — Business Capability Lifecycle](docs/engineering/business-capability-lifecycle.md): actor/action/resource, SQL ownership và một Organization vertical-slice test xuyên HTTP/JWT/PostgreSQL.
- [BIZ-02 — Ownership & Resource Lifecycle](docs/engineering/business-ownership-resource-lifecycle.md): phân biệt Account/Person/Organization/membership authority với quyền sở hữu tài sản; DB tests về rollback và concurrency.
- [ENG-CORE-04 — Application Foundation](docs/engineering/application-foundation.md): hợp đồng viết API lặp lại, authenticated actor, session response dùng chung và request context lifecycle.
- [ENG-CORE-05 — API Contract & Error Semantics](docs/engineering/api-contract-error-semantics.md): resource ID parser, PostgreSQL error recognition và giữ nguyên business-specific HTTP mapping.
- [ENG-CORE-06 — Application Operation & Module Boundary](docs/engineering/application-operation-boundary.md): phân định handler, Organization use-case orchestration và PostgreSQL data owner dựa trên source thực tế.
- [ENG-CORE-07 — Dependency & Module Ownership](docs/engineering/dependency-module-ownership.md): tách verification intent khỏi worker/Asynq, bảo vệ chiều phụ thuộc application → infrastructure bằng test.
- [ENG-CORE-08 — Session Integrity & Lifecycle](docs/engineering/session-integrity-lifecycle.md): kiểm chứng IssueSession dùng chung, Renew/Revoke, context và thất bại Phone OTP.

ENG-GOV-01 là **quy chuẩn phối hợp developer và review**, không phải triển khai Business Core, không thay đổi runtime và không tự động bổ nhiệm GitHub code owners. CI xanh không đồng nghĩa đã nghiệm thu WSL hoặc production.

## 1. Clone và chạy lần đầu

Cần Git, Docker Engine/Docker Desktop và Docker Compose V2 trên WSL Ubuntu. Nếu thư mục `~/projects/bdspro` đã tồn tại, kiểm tra và lưu thay đổi cần giữ trước; **không clone đè hoặc xóa dữ liệu/volume cũ**.

```bash
mkdir -p ~/projects
cd ~/projects
git clone -b dev https://github.com/datvtph41107/bdspro.git
cd bdspro
git status --short --branch

docker compose up -d --build
docker compose ps
curl -i http://127.0.0.1:8080/health

docker compose exec -T postgres psql -U root -d bdspro \
  -c "SELECT version, dirty FROM schema_migrations;"
```

Kết quả mong đợi: `/health` trả HTTP 200; database mới có `schema_migrations.version=5`, `dirty=false`. `main.go` tự chạy `migrate.Up()` khi API startup, nên **không cần `make migrateup` lần nữa** khi vừa chạy Compose thành công.

Nếu API không lên: `docker compose logs --tail=150 api`, `docker compose logs postgres`, `docker compose ps`.

**Lưu ý cổng 5432/6379/8080:** Compose publish `5432:5432`, `6379:6379`, `8080:8080`; PostgreSQL được cập nhật lên phiên bản ổn định 18 thay vì 14. Nếu máy bạn đang có container `postgres` hoặc API khác chiếm các cổng này, kiểm tra `docker ps` và dừng đúng container đang xung đột khi đã chắc chắn có thể dừng; không xóa dữ liệu.

**Lưu ý database:** Compose dùng PostgreSQL **18** và volume riêng `bdspro-postgres18-data`, mount vào `/var/lib/postgresql` theo Docker Official Image từ PG18. Volume `bdspro_data-volume` từng sử dụng PostgreSQL 14 **không bị xóa và không được tự động gắn lại**. Các volume PG18 trước đó như `bdspro-postgres-data` cũng được giữ nguyên. Nếu cần dữ liệu cũ, phải backup/restore hoặc thực hiện nâng cấp major version; tuyệt đối không gắn trực tiếp dữ liệu PG14 vào PG18. Clone Git chỉ tải source code, không di chuyển database.

Dừng stack mà vẫn giữ volume:

```bash
docker compose down
```

Không dùng `docker compose down -v` trên volume chứa dữ liệu muốn giữ.

## 2. Các file vận hành: ai chịu trách nhiệm gì?

| File | Trách nhiệm |
|---|---|
| `app.env` | Các giá trị cấu hình Go mặc định dành cho development |
| `util/config.go` | Viper đọc `app.env` và nhận giá trị override từ process environment |
| `docker-compose.yaml` | PostgreSQL, Redis, Mailpit và API; dùng container DNS, JSON logging và giới hạn log container |
| `Dockerfile` | Build Go binary rồi chạy bằng image Alpine tối giản |
| `start.sh` | `exec` ứng dụng |
| `wait-for.sh` | Chờ cổng PostgreSQL mở trước khi chạy API |
| `Makefile` | Các target Go, SQLC, migration, mock và Docker CLI cơ bản đúng tinh thần SimpleBank |
| `.github/workflows/test.yml` | Một workflow, một job `test`: PostgreSQL/Redis/Mailpit → migrations → tests → build/Compose check → HTTP smoke → vet → race |

**Không cần `.env` hay `.env.example` để khởi động BDSPro theo Compose hiện tại.** Go đọc `app.env`, Compose override các biến cần thiết cho kết nối container:

```text
DB_SOURCE=postgresql://root:secret@postgres:5432/bdspro?sslmode=disable
REDIS_ADDRESS=redis:6379
LOG_FORMAT=json
SMTP_ADDRESS=mailpit:1025
SMTP_AUTH=false
EMAIL_SENDER_ADDRESS=no-reply@bdspro.test
```

Khi chạy Go trên máy host, `app.env` dùng `127.0.0.1:5432` và `127.0.0.1:6379`. Docker Compose publish Redis `6379:6379` trên các host interfaces. Khi dùng WSL có thể chạy Go trực tiếp với `127.0.0.1:6379`; dừng service `api` trước (`docker compose stop api`) để tránh trùng cổng 8080 và không khởi tạo thêm Redis standalone. **Chỉ dùng cấu hình này trong môi trường development tin cậy**: Redis hiện không có xác thực và cổng 6379 có thể truy cập từ mạng ngoài nếu firewall cho phép. Tránh triển khai cấu hình này trên máy chủ có public IP.

Các khóa/mật khẩu trong `app.env` là **giá trị ví dụ cho development, tuyệt đối không sử dụng production**. Nếu cần credential SMTP thật, truyền dưới dạng process environment vào môi trường thực thi phù hợp; không commit vào `app.env` hoặc Dockerfile. Compose sử dụng Mailpit để nhận email thử nghiệm, xem tại `http://127.0.0.1:8025`. Khi chạy trực tiếp trên WSL, hãy cấu hình SMTP phù hợp bằng process environment; không commit SMTP/Google credentials thật.

## 3. Docker Compose và Makefile: hai cách sử dụng khác nhau

**Cách A — chạy cả stack với Compose (khuyên dùng khi vừa clone):**

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f api
```

Compose tự tạo network và PostgreSQL database `bdspro` qua `POSTGRES_DB`; không cần `make network`, `make postgres`, `make redis` hay `make createdb`. `depends_on` tối giản chỉ xác định thứ tự bắt đầu container, không bảo đảm Redis sẵn sàng; API tự ping Redis khi khởi động và có thể phải chạy lại nếu Redis khởi tạo chậm.

**Cách B — Go trên WSL, services được quản lý thủ công bằng Docker CLI:**

```bash
make postgres    # Chỉ áp dụng nếu đã có container standalone tên postgres
make redis       # Tạo container standalone redis lần đầu, expose :6379
make server      # Console có màu để debug tương tác, bind HTTP :8080
```

Các lệnh `make network`, `make postgres`, `make redis`, `make createdb` là đường làm việc thủ công đúng như Makefile đã chốt. `make postgres` chỉ **start một container đã tồn tại**, không tạo container mới; `make createdb` chỉ dùng khi database chưa tồn tại. Nếu bạn đã chạy Compose, không chạy song song các target này vì dễ trùng tên/cổng và tạo hai bộ dữ liệu khác nhau.

## 4. Migration, SQLC, tests và generated source

```bash
make test
make migrateup              # Cần cài migrate CLI, chỉ khi có migration chưa được áp dụng
make new_migration name=add_example_field
make sqlc                   # Cần cài sqlc
make mock                   # Cần cài mockgen
make fmt
```

`make migrateup` dùng mặc định `postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable`. Các SQL migrations versioned ở `db/migration/`, SQL inputs ở `db/query/`, code SQLC sinh ra ở `db/sqlc/`. Khi cần đổi schema, tạo migration mới; **không chỉnh sửa migration đã áp dụng**. `make migratedown`/`make migratedown1` có thể làm mất dữ liệu và không nằm trong quy trình startup thường ngày.

`make test` chạy `go test -v -cover ./...`; database write tests được skip khi chạy local thông thường để bảo vệ DB đang sử dụng. Trên GitHub Actions, tests kết nối vào PostgreSQL riêng trên runner và thực thi database integration tests. Nếu cần chạy integration local độc lập với database ứng dụng, có thể dùng `sh scripts/test-integration.sh` (Postgres tạm ở cổng 15433); không thêm target này vào Makefile tối giản.

`make proto` chỉ phục vụ tái sinh code legacy gRPC/Swagger nếu chủ động nghiên cứu phần này; runtime Gin không cần chạy nó. `make db_docs` và `make db_schema` cần cài DBDocs/DBML tools.

## 5. Cấu trúc source

```text
bdspro/
├── main.go                     # Bootstrap config, DB, migrations, worker, Gin server
├── api/                        # Gin routes, middleware, HTTP handlers và tests
├── auth/                       # Register/Login nghiệp vụ
├── token/                      # JWT access/refresh, payload và maker
├── db/
│   ├── migration/              # SQL migrations
│   ├── query/                  # SQLC input SQL
│   ├── sqlc/                   # SQLC generated code, Store, transactions, tests
│   └── mock/                   # Generated Store mocks
├── worker/                     # Redis/Asynq và email verification task
├── mail/                       # Email sender adapter
├── util/                       # Config, password, helpers
├── gapi/, proto/, pb/          # Legacy gRPC code, không chạy
├── doc/, docs/                 # DBML, Swagger legacy, kỹ thuật/tham khảo
├── scripts/                    # Optional isolated DB integration script
├── app.env
├── Dockerfile
├── docker-compose.yaml
├── Makefile
├── start.sh
├── wait-for.sh
└── .github/workflows/test.yml  # Một workflow CI duy nhất
```

## 6. API và luồng chính

| Phương thức | Path | Chức năng |
|---|---|---|
| GET | `/health` | HTTP liveness |
| POST | `/users` | Register → bcrypt → transaction Party/Person/Account → enqueue verification |
| POST | `/users/login` | Login → JWT access/refresh → PostgreSQL session |
| GET | `/users/verify_email` | Xác minh email |
| GET | `/users/me` | Bearer access-token → AccountID |
| POST | `/tokens/renew_access` | Verify refresh token + session → access JWT |
| POST | `/tokens/revoke` | Block refresh session |
| POST | `/organizations` | Tạo Organization và owner |
| GET | `/organizations` | Liệt kê Organization membership |
| POST | `/organizations/:id/members` | Owner thêm member |
| PATCH | `/organizations/:id` | Owner cập nhật thông tin Organization |
| POST | `/users/me/google/start` | JWT + password → bắt đầu liên kết Google, khi có OAuth credentials |
| GET | `/users/google/login/start` | Bắt đầu OpenID Connect Google, khi có OAuth credentials |
| GET | `/users/google/callback` | Xác minh Google và liên kết/đăng nhập |
| POST | `/users/me/phone/start` | JWT + password → bắt đầu gắn số điện thoại, development |
| POST | `/users/me/phone/verify` | Xác minh mã và gắn số, development |
| POST | `/users/phone/login/start` | Yêu cầu mã đăng nhập số điện thoại, development |
| POST | `/users/phone/login/verify` | Tiêu thụ mã OTP và phát JWT/session, development |

Smoke test bằng HTTP:

```bash
curl -i http://127.0.0.1:8080/health

curl -i -X POST http://127.0.0.1:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"newdev@example.test","password":"strong-password-123"}'

curl -i -X POST http://127.0.0.1:8080/users/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"newdev@example.test","password":"strong-password-123"}'
```

Request restrictions và IPActive/RequestMonitor đã có trên nhánh tích hợp, nhưng Google OAuth browser E2E, gửi SMS OTP thực và production hardening vẫn là các cổng nghiệm thu riêng.

## 7. GitHub Actions

Chỉ giữ `.github/workflows/test.yml`. Workflow có một job `test`, PostgreSQL 18 (database `bdspro`), setup Go từ `go.mod`, cài migrate CLI, chạy `make migrateup` và `make test`. Có các bước build/Compose validation, runtime HTTP smoke, static analysis và race detector.

Một workflow để chứng minh tests hoạt động không đồng nghĩa toàn bộ chức năng ứng dụng hoặc bảo mật production đã được nghiệm thu. Mọi thay đổi source cần được review riêng theo use case và invariant.

## 8. Kiểm soát vòng đời và nhật ký (đợt vận hành đầu tiên)

BDSPro dùng duy nhất zerolog; `LOG_FORMAT` điều khiển cách xuất log, không đổi sự kiện, field, request ID hoặc task ID. Chỉ hỗ trợ `console` và `json`. File `app.env` mặc định `LOG_FORMAT=json` để tiến trình trực tiếp/đóng gói có dữ liệu phù hợp cho công cụ xử lý. `make server` chủ động chọn console có màu khi debug tương tác:

```bash
make server                   # console có màu
make server LOG_FORMAT=json   # JSON để đối chiếu runtime
docker compose logs -f api    # JSON từ container API
```

Console hiển thị `[INF]`, `[WRN]`, `[ERR]` có màu ANSI, cùng các trường của event; JSON giữ cấu trúc key-value. `LOG_FORMAT` độc lập với `ENVIRONMENT`: Docker chủ động đặt `LOG_FORMAT=json` kể cả khi `ENVIRONMENT=development`. Banner chỉ xuất hiện trong console để không làm nhiễu JSON. Nếu pipe hoặc redirect console, ANSI có thể vẫn xuất hiện; hãy chọn JSON khi cần máy xử lý.

Không tạo thư mục `logs/` trong source. `docker compose logs -f api` xem stdout/stderr. Compose dùng `json-file` cho API với `max-size=10m`, `max-file=3`; đây là giới hạn theo dung lượng chứ không phải thời gian lưu. Gin mặc định release (không in danh sách routes), có thể bật `GIN_MODE=debug make server` khi cần. HTTP access log chỉ ghi route/path không kèm query string, tránh lộ mã xác minh. Redis expose port 6379 ra host đúng mẫu phát triển đã chốt; cần firewall khi làm việc trên mạng không tin cậy.

Trình tự kỳ vọng khi chạy ứng dụng:
`startup.begin` → `postgres.connected` → `migration.checked result=applied|no_change`
→ `redis.connected` → `task.processor.started` → `http.listening address=...` → `startup.ready`.
Nếu không bind được HTTP hoặc Redis không trả lời, không được phát `startup.ready`; tiến trình phải trả lỗi và dừng worker. `startup.ready` có `duration` đo từ đầu hàm `run()`.
SIGINT/SIGTERM chủ động đóng HTTP và worker, log `http.stopped` và `task.processor.stopped`.

**Cổng G0 — nghiệm thu runtime đang chạy trên WSL (chỉ-đọc):** Xem [docs/operations/g0-runtime-acceptance.md](docs/operations/g0-runtime-acceptance.md). Sau khi kiểm tra Git/volume và stack Compose đã chạy, dùng `sh scripts/accept-g0.sh`; script không tự khởi động container, không ghi database và không thay đổi volume. Kết quả PASS không thay thế nghiệm thu backup/restore, lifecycle thật hoặc Google/SMS E2E.

Nghiệm thu local an toàn trên clone mới:

```bash
make test
sh scripts/test-integration.sh  # Tùy chọn: PostgreSQL tạm (cổng 15433)
docker compose up -d --build
curl -i http://127.0.0.1:8080/health
docker compose logs --tail=100 api
docker compose exec -T postgres psql -U root -d bdspro \
  -c "SELECT version, dirty FROM schema_migrations;"
docker compose down        # KHÔNG thêm -v
```

Không coi HTTP liveness là database readiness liên tục: PostgreSQL/Redis được xác nhận tại khởi động,
nhưng `/health` hiện chỉ chứng minh HTTP server còn phản hồi. CI đã có migrations, Go tests, Compose config, HTTP smoke, vet và race detector;
nghiệm thu Compose, gửi email thật (thiếu SMTP credential) và bảo mật production là bước riêng.

## 9. Request Lifecycle, Google OIDC và Phone OTP

Nguồn thiết kế RequestIngress, PreLogin, IPActiveSource, IPActive, Authenticate, SQL Authorize, SufLogin và RequestMonitor: [`docs/engineering/request-lifecycle.md`](docs/engineering/request-lifecycle.md). Tham khảo [`docs/operations/request-restrictions.md`](docs/operations/request-restrictions.md), [`docs/operations/google-login.md`](docs/operations/google-login.md) và [`docs/operations/phone-otp-v1.md`](docs/operations/phone-otp-v1.md) để kiểm soát luồng và nghiệm thu riêng.

Phone OTP chỉ chạy trong development với mã lưu trong bảng `phone_otp_challenges.dev_code`; **chưa có kênh gửi SMS thực tế**. Google OAuth yêu cầu Client ID/Secret ngoài source. Không coi những luồng này là production-ready chỉ vì CI chạy thành công.
