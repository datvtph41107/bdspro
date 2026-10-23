# BDSPro

Backend Go/Gin, PostgreSQL, SQLC, JWT và Redis/Asynq. Phiên bản nền tảng được chọn theo bản stable đang hỗ trợ: **PostgreSQL 18**, **Go 1.27.3**, **Alpine 3.24**. Redis 7 được giữ tạm trong giai đoạn này để không thay đổi major version của worker khi chưa kiểm tra tương thích Redis/Asynq. Cách chạy, Dockerfile, Docker Compose, Makefile và GitHub Actions giữ bố cục đơn giản theo [TechSchool SimpleBank](https://github.com/techschool/simplebank), với các giá trị đổi thành `bdspro`.

**Runtime hiện tại:** một Gin HTTP server trên cổng 8080 và Redis worker xử lý email verification. `runGrpcServer`/`runGatewayServer` cùng protobuf/gateway generated source còn trong repository để tham khảo, **không được khởi động**. Chưa cấu hình production deployment.

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

Kết quả mong đợi: `/health` trả HTTP 200; database mới có `schema_migrations.version=3`, `dirty=false`. `main.go` tự chạy `migrate.Up()` khi API startup, nên **không cần `make migrateup` lần nữa** khi vừa chạy Compose thành công.

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
| `docker-compose.yaml` | PostgreSQL, Redis, API; override đúng `DB_SOURCE` và `REDIS_ADDRESS` để dùng Docker DNS |
| `Dockerfile` | Build Go binary rồi chạy bằng image Alpine tối giản |
| `start.sh` | `exec` ứng dụng |
| `wait-for.sh` | Chờ cổng PostgreSQL mở trước khi chạy API |
| `Makefile` | Các target Go, SQLC, migration, mock và Docker CLI cơ bản đúng tinh thần SimpleBank |
| `.github/workflows/test.yml` | Một workflow, một job `test`: PostgreSQL 18 → migration → `make test` |

**Không cần `.env` hay `.env.example` để khởi động BDSPro theo Compose hiện tại.** Go đọc `app.env`, Compose override đúng hai biến:

```text
DB_SOURCE=postgresql://root:secret@postgres:5432/bdspro?sslmode=disable
REDIS_ADDRESS=redis:6379
```

Khi chạy Go trên máy host, `app.env` dùng `127.0.0.1:5432` và `127.0.0.1:6379`. Docker Compose publish Redis `6379:6379` trên các host interfaces. Khi dùng WSL có thể chạy Go trực tiếp với `127.0.0.1:6379`; dừng service `api` trước (`docker compose stop api`) để tránh trùng cổng 8080 và không khởi tạo thêm Redis standalone. **Chỉ dùng cấu hình này trong môi trường development tin cậy**: Redis hiện không có xác thực và cổng 6379 có thể truy cập từ mạng ngoài nếu firewall cho phép. Tránh triển khai cấu hình này trên máy chủ có public IP.

Các khóa/mật khẩu trong `app.env` là **giá trị ví dụ cho development, tuyệt đối không sử dụng production**. Nếu cần credential SMTP thật, truyền dưới dạng process environment vào môi trường thực thi phù hợp; không commit vào `app.env` hoặc Dockerfile. Khi không có `EMAIL_SENDER_ADDRESS/PASSWORD`, đăng ký có thể tạo tài khoản nhưng việc gửi email xác minh sẽ không thành công và worker có thể retry.

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
make server      # go run main.go, bind HTTP :8080
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

Không coi HTTPS, secret management, rate limiting hay session security là production-ready ở mốc hiện tại.

## 7. GitHub Actions

Chỉ giữ `.github/workflows/test.yml`. Workflow có một job `test`, PostgreSQL 18 (database `bdspro`), setup Go từ `go.mod`, cài migrate CLI, chạy `make migrateup` và `make test`. Không có job Compose smoke hoặc race detector.

Một workflow để chứng minh tests hoạt động không đồng nghĩa toàn bộ chức năng ứng dụng hoặc bảo mật production đã được nghiệm thu. Mọi thay đổi source cần được review riêng theo use case và invariant.

## 8. Kiểm soát vòng đời và nhật ký (đợt vận hành đầu tiên)

Development in một dòng tên `BDSPro / api`, sau đó dùng zerolog console dạng `10:31:59 INF startup.begin environment=development`:
mã mức log `INF` xanh lá, `WRN` vàng, `ERR`/`FTL` đỏ, `DBG` xanh cyan.
Chỉ mã mức log có màu ANSI khi stderr là terminal và không đặt `NO_COLOR`; khi dùng Docker, redirect hoặc pipe, log tự tắt mã màu. Còn timestamp, event name và các field đơn sắc.
Đặt `NO_COLOR=1` để tắt ANSI, hoặc dùng `ENVIRONMENT=production` để ghi JSON.
Không tạo thư mục `logs/` trong source. `docker compose logs -f api` xem stdout/stderr. Compose không cấu hình riêng logging driver/rotation. Gin mặc định release (không in danh sách routes), có thể bật `GIN_MODE=debug make server` khi cần. HTTP access log chỉ ghi route/path không kèm query string, tránh lộ mã xác minh. Redis expose port 6379 ra host đúng mẫu phát triển đã chốt; cần firewall khi làm việc trên mạng không tin cậy.

Trình tự kỳ vọng khi chạy ứng dụng:
`startup.begin` → `postgres.connected` → `migration.checked result=applied|no_change`
→ `redis.connected` → `task.processor.started` → `http.listening address=...` → `startup.ready`.
Nếu không bind được HTTP hoặc Redis không trả lời, không được phát `startup.ready`; tiến trình phải trả lỗi và dừng worker. `startup.ready` có `duration` đo từ đầu hàm `run()`.
SIGINT/SIGTERM chủ động đóng HTTP và worker, log `http.stopped` và `task.processor.stopped`.

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
nhưng `/health` hiện chỉ chứng minh HTTP server còn phản hồi. CI giản lược chỉ kiểm tra Go tests;
nghiệm thu Compose, gửi email thật (thiếu SMTP credential) và bảo mật production là bước riêng.
