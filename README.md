# BDSPro — Go REST API Backend

BDSPro chạy **một Gin HTTP application** với PostgreSQL, SQLC, JWT, Redis/Asynq và Docker Compose. Repository giữ một số mã gRPC/Protobuf lịch sử để đối chiếu TechSchool SimpleBank, nhưng **không khởi động gRPC Server hoặc gRPC-Gateway**. Đây là môi trường phát triển và kiểm chứng Engineering Core; chưa được tuyên bố production-ready.

**Nguồn chuẩn:** repository `datvtph41107/bdspro`, nhánh `dev`; workspace local `~/projects/bdspro`; tên database PostgreSQL **`bdspro`**; tên Docker Compose project **`bdspro`**. Không sử dụng folder clone phụ hay tên volume theo lecture.

## 1. Bắt đầu từ clone mới (WSL Ubuntu)

Yêu cầu: Git, Docker Engine hoặc Docker Desktop có WSL integration, Docker Compose V2, curl. Để chạy Go bên ngoài container và dùng test-integration: Go theo phiên bản trong `go.mod`. `sqlc`, `mockgen` và `migrate` CLI chỉ cần cho những thao tác tương ứng.

**Chỉ clone nếu `~/projects/bdspro` chưa tồn tại.** Nếu có repository cũ, kiểm tra `git status`, sao lưu code hoặc dữ liệu cần giữ rồi tự di chuyển nó ra khỏi workspace; không xóa hay clone đè vào một thư mục đang có thay đổi. Việc clone Git **không** tự sao lưu Docker volumes.

```bash
mkdir -p ~/projects
cd ~/projects
git clone --branch dev https://github.com/datvtph41107/bdspro.git
cd bdspro
git branch --show-current
git status --short --branch
cp .env.example .env
docker compose config --services
docker compose up -d --build
docker compose ps
curl -i http://127.0.0.1:18080/health
docker compose exec -T postgres psql -U root -d bdspro -c "SELECT version, dirty FROM schema_migrations;"
```

Kỳ vọng: `dev`, các service `postgres`, `redis`, `api` và HTTP `/health` trả **200**. Với DB mới, ứng dụng tự chạy migration 000001 → 000003 lúc startup; hàng cuối của `schema_migrations` có `version=3`, `dirty=false`. Không cần chạy `migrateup` thêm lần nữa sau `docker compose up`.

Nếu API chưa khởi động: `docker compose logs --tail=150 api`, tiếp theo kiểm tra `docker compose ps` và `docker compose logs postgres redis`.

> **Bảo toàn dữ liệu:** Compose sử dụng volume `bdspro-postgres-data`. Volume cũ `lecture37-postgres` (nếu từng sử dụng ở lần triển khai trước) **không được tự động xóa hoặc tự động di chuyển dữ liệu**; volume mới là một PostgreSQL instance độc lập. Không dùng `docker compose down --volumes`, `docker volume prune` hoặc lệnh xóa database nếu chưa chủ động backup. `docker compose down` chỉ tắt container và **giữ nguyên volume**.

## 2. Hai chế độ phát triển — chọn một chế độ chạy API

| Thành phần | Docker Compose (`docker compose up`) | Go chạy trên WSL (`docker compose up -d postgres redis` rồi `make server`) |
|---|---|---|
| HTTP | `http://127.0.0.1:18080` | `http://127.0.0.1:8080` |
| PostgreSQL (từ WSL) | `127.0.0.1:15432` | `127.0.0.1:15432` |
| Redis (từ WSL) | `127.0.0.1:16379` | `127.0.0.1:16379` |
| PostgreSQL trong container | `postgres:5432` | Không áp dụng cho Go host process |
| Redis trong container | `redis:6379` | Không áp dụng cho Go host process |
| Database | `bdspro` | `bdspro` |

**Chế độ A — tất cả chạy trong Compose:**

```bash
docker compose up -d --build
docker compose ps
docker compose exec -T postgres psql -U root -d bdspro -c "SELECT version, dirty FROM schema_migrations;"
docker compose logs --tail=100 api
```

**Chế độ B — PostgreSQL và Redis trong Compose, Go chạy trực tiếp trên WSL:**

```bash
docker compose stop api   # Nếu API đang chạy trong Compose
docker compose up -d postgres redis
make server
```

`make server` dùng `go run main.go` và `app.env`; nó kết nối DB cổng `15432`, Redis cổng `16379`. Không chạy đồng thời hai API processes/worker cùng nhắm một dữ liệu phát triển trừ khi chủ động thử concurrency: cả hai sẽ cùng truy cập `bdspro` và Redis queue.

Sau khi dùng chế độ B, dừng Go bằng Ctrl+C. Để trở lại Compose: `docker compose up -d --build`.

## 3. Cấu hình: `app.env`, `.env` và process environment

| Nguồn | Ai đọc? | Mục đích |
|---|---|---|
| `app.env` (được commit) | Go qua `util.LoadConfig(".")` | Giá trị phát triển **khi chạy trên WSL host**; cũng được copy vào image |
| `.env.example` (được commit) | Developer sao chép thành `.env` | Mẫu biến để Compose nội suy |
| `.env` (được gitignore) | Docker Compose | Secret/giá trị Compose; **không được Go tự đọc trực tiếp** |
| Process environment | Go/Viper và Compose | Override cấu hình cho process đang chạy |
| `docker-compose.yaml` | Compose | Host ports, internal service DNS, Compose environment |

Thứ tự cần nhớ: đối với các khóa có trong `app.env`, `util.LoadConfig` cho process environment ghi đè giá trị file. Compose đặt `DB_SOURCE`, `REDIS_ADDRESS` và `HTTP_SERVER_ADDRESS` theo container network; `app.env` mặc định dùng địa chỉ loopback của WSL. **Không** dùng `127.0.0.1:15432` làm DB host từ bên trong API container.

Cấu hình email cho Compose ở `.env`:

```dotenv
TOKEN_SYMMETRIC_KEY=replace-with-a-strong-unique-development-key
EMAIL_SENDER_NAME=BDSPro
EMAIL_SENDER_ADDRESS=
EMAIL_SENDER_PASSWORD=
```

`TOKEN_SYMMETRIC_KEY` phải có tối thiểu 32 byte; ví dụ text trên chỉ minh họa và cần thay bằng giá trị thực dài, ngẫu nhiên. Không lưu mật khẩu thật vào `app.env` đã commit. Cấu hình native Go lấy defaults từ `app.env`; muốn ghi đè một lần, truyền process environment ngay tại lệnh chạy, ví dụ `HTTP_SERVER_ADDRESS=127.0.0.1:8081 make server`. Nếu sửa khóa trong `.env` của Compose, recreate API với `docker compose up -d --force-recreate api`.

Đối với Gmail: để trống `EMAIL_SENDER_ADDRESS/PASSWORD`, API vẫn có thể đăng ký và đăng nhập; tác vụ email verification sẽ không thực sự được gửi thành công và worker có thể retry. Đăng ký thành công **không chứng minh email đã gửi**. Cơ chế Gmail hiện tại chỉ nên dùng với credential dev/test và cấu hình được cấp riêng.

## 4. Makefile — giữ tối giản như SimpleBank

Makefile chỉ giữ các lệnh Go, SQLC, mocks và migration thường dùng. Không bọc Docker Compose, network, logs, database inspection hay script kiểm thử trong Makefile.

| Lệnh | Mục đích |
|---|---|
| `make server` | `go run main.go` (host WSL, đọc `app.env`) |
| `make test` | `go test -v -cover ./...`; DB write tests được skip khi không ở môi trường test cô lập |
| `make migrateup` | Apply migrations bằng `migrate`, trỏ vào `127.0.0.1:15432/bdspro` |
| `make migratedown` | Rollback migration; không sử dụng tùy tiện với dữ liệu cần giữ |
| `make new_migration name=...` | Sinh cặp file migration mới |
| `make sqlc` | Sinh code từ SQL queries |
| `make mock` | Sinh mock Store và TaskDistributor |

Lệnh vận hành gọi trực tiếp:

```bash
docker compose up -d --build
docker compose ps
docker compose logs --tail=100 api
docker compose exec -T postgres psql -U root -d bdspro -c "SELECT version, dirty FROM schema_migrations;"
docker compose down # Giữ nguyên volume
```

Khi thay schema: `make new_migration name=add_example_field` → viết migration → `make migrateup` → `make sqlc` → `sh scripts/test-integration.sh`. Chỉ cần cài các CLI tương ứng khi sử dụng.

## 5. Kiểm thử local và CI: không ghi nhầm lên database đang sử dụng

**Nhanh, không phụ thuộc DB write:**

```bash
make test
```

HTTP unit tests dùng Gin `httptest` và mock Store. Các DB integration tests trong `db/sqlc` sẽ bị skip để bảo vệ DB phát triển `bdspro`.

**Toàn bộ test trên PostgreSQL tạm, độc lập với dữ liệu Compose:**

```bash
sh scripts/test-integration.sh
```

Lệnh này yêu cầu Docker và Go trên WSL. `scripts/test-integration.sh` tạo container tạm `bdspro-integration-postgres`, dùng database tên `bdspro`, mở cổng host `15433`, chạy migrations trong container tạm và chạy `go test -count=1 ./...`. Sau đó container tạm được dừng/xóa bằng trap, kể cả khi có test thất bại. Bộ test từ chối chạy khi local DB không ở endpoint tạm hoặc không có dấu nhận diện cluster disposable.

**GitHub Actions:** `.github/workflows/test.yml` tạo PostgreSQL 18 tạm trên `15432`, apply migrations và chạy unit + DB integration tests. `compose-smoke.yml` build cả stack, xác minh `/health`, Register, Login, quyền owner của Organization, migration version và việc gRPC listener không mở. CI có thể dùng `down --volumes` vì đó là dữ liệu tạm do runner sở hữu, không phải local development volume.

Kết quả `make test` không được gọi là “toàn bộ test PASS” nếu DB integration tests đã skip. Luôn phân biệt unit, integration, Compose smoke và production validation.

## 6. Cấu trúc source và trách nhiệm

```text
bdspro/
├── main.go                    # Config, pool, migration, Store, worker, Gin startup/shutdown
├── api/                       # Gin routes, middleware, HTTP request/response và handler tests
├── auth/                      # Register/Login use cases đang có
├── token/                     # JWT maker, payload và access/refresh token types
├── db/
│   ├── migration/             # PostgreSQL schema migrations (nguồn schema versioned)
│   ├── query/                 # SQL đầu vào của sqlc
│   ├── sqlc/                  # SQLC generated queries + Store/transaction/tests viết tay
│   └── mock/                  # Mocks được sinh
├── worker/                    # Asynq enqueue / background email verification
├── mail/                      # Email sender adapter
├── util/                      # Config, password và các helpers hiện có
├── gapi/, proto/, pb/         # Mã gRPC/Gateway lịch sử, không được startup
├── doc/                       # DBML và generated Swagger cũ (không phục vụ HTTP)
├── docs/                      # Tài liệu kỹ thuật và lịch sử nghiên cứu
├── scripts/test-integration.sh
├── Dockerfile                 # Multi-stage Go build, runtime image
├── docker-compose.yaml        # Project bdspro, PostgreSQL, Redis, Gin API
├── app.env                    # Dev defaults cho Go trên WSL
├── .env.example               # Mẫu biến Docker Compose; copy thành .env
├── Makefile                   # Lệnh phát triển/kiểm thử
├── start.sh, wait-for.sh      # Entrypoint và chờ PostgreSQL trong container
└── .github/workflows/         # Unit/integration, Compose smoke, generated source gates
```

`main.go` điều phối khởi động, không xử lý HTTP routes. `api/server.go` đăng ký Gin routes và middleware. `api/user.go` xử lý Register/Login HTTP, gọi `auth/auth.go` cho nghiệp vụ. `db/sqlc/store.go` quản lý PostgreSQL transaction và gọi SQLC queries; các SQL nằm ở `db/query`.

**Quy tắc sửa source:** bắt đầu từ use case/invariant → xác định SQL và transaction → HTTP handler/contract → tests. Không sửa files SQLC generated bằng tay; thay `db/query/*.sql` hoặc migrations rồi chạy `make sqlc`. Không thêm package/module mới chỉ vì tên trông “đúng kiến trúc”; ranh giới module sẽ được đánh giá lại sau khi từng nghiệp vụ thực tế hoàn thành.

## 7. Luồng thực thi và các endpoint hiện có

**Startup:**

```text
main()
  → util.LoadConfig(".")
  → PostgreSQL pgxpool.New/Ping
  → runDBMigration (apply pending migrations)
  → db.NewStore
  → Redis TaskDistributor + runTaskProcessor
  → runGinServer (HTTP + CORS + Gin logger/recovery)
  → graceful shutdown on SIGINT/SIGTERM
```

**Register:** `POST /users` → `api.createUser` → `auth.Register` → bcrypt → `Store.RegisterTx` → Party/Person/Account cùng transaction → enqueue verification task → HTTP 201.

**Login:** `POST /users/login` → `auth.Login` → verify password → cấp access/refresh JWT → lưu `sessions` → HTTP 200.

**Authenticated requests:** Bearer access JWT → Gin `authMiddleware` → AccountID. Với Organization, owner authorization còn được kiểm tra trong SQL operations, không chỉ dựa trên JWT.

| Method | Gin endpoint | Vai trò |
|---|---|---|
| GET | `/health` | Liveness (không xác nhận DB readiness) |
| POST | `/users` | Đăng ký |
| POST | `/users/login` | Đăng nhập / tạo session |
| GET | `/users/verify_email` | Xác minh email bằng mã |
| POST | `/tokens/renew_access` | Đổi refresh token hợp lệ lấy access JWT |
| POST | `/tokens/revoke` | Khóa refresh session |
| GET | `/users/me` | Lấy AccountID của caller |
| POST | `/organizations` | Tạo Organization cùng owner |
| GET | `/organizations` | Liệt kê Organizations của caller |
| POST | `/organizations/:id/members` | Owner thêm member |
| PATCH | `/organizations/:id` | Owner đổi tên Organization |

Thử REST bằng curl (đổi email nếu đã đăng ký):

```bash
curl -i http://127.0.0.1:18080/health

curl -i -X POST http://127.0.0.1:18080/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@example.test","password":"strong-password-123"}'

curl -i -X POST http://127.0.0.1:18080/users/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@example.test","password":"strong-password-123"}'
```

Lấy `access_token` từ login response và gửi qua `Authorization: Bearer <access-token>` tới `/users/me` hoặc các Organization endpoints. Không dán hoặc commit token thật vào source hoặc log.

## 8. Database, session và giới hạn bảo mật hiện tại

Migration hiện có: `000001_party_account_session`, `000002_email_verification`, `000003_organization_memberships`. `schema_migrations` là version state; **không sửa hoặc chạy lại bằng tay một migration đã được áp dụng**. Tạo migration mới khi thay schema. Để xem migration state, dùng `docker compose exec -T postgres psql -U root -d bdspro -c "SELECT version, dirty FROM schema_migrations;"` (chỉ đọc).

Hệ thống đã có `sessions` với `is_blocked`, refresh-token verification và revoke. Access JWT đã phát vẫn còn hiệu lực cho đến khi hết hạn; revoke hiện chỉ ngăn gia hạn. Role `owner/member` là quan hệ với từng Organization, **không phải global account role**.

**Không coi môi trường này là production-ready:** JWT signing key và PostgreSQL password trong file dev chỉ là ví dụ; chưa có TLS/reverse proxy production, secret management, rate limiting và các security controls cần được đánh giá thêm. Registration enqueue Redis bên trong PostgreSQL transaction có failure gap giữa hai hệ thống; SMTP thực tế chưa được xác nhận khi không có credentials. Các vấn đề này sẽ được review theo các luồng Authentication/Authorization, không che giấu bằng cách đánh dấu “hoàn chỉnh”.

## 9. Làm việc nhóm và nghiệm thu

Mỗi thay đổi đề xuất cần ghi nhận rõ: hành vi cần làm, nơi sở hữu, dữ liệu bị thay đổi, invariant, failure case, files ảnh hưởng và test chứng minh. Trước khi push:

```bash
make test
sh scripts/test-integration.sh    # Nếu thay SQL, transaction, auth state hoặc schema
git status --short
git diff --check
```

CI tiếp tục kiểm tra migrations, unit/database tests, SQLC/mocks canonical source và Compose smoke. Chỉ báo cáo “đã nghiệm thu” sau khi kiểm tra kết quả workflow tại commit cụ thể; **build thành công không đồng nghĩa security hoặc business correctness đã được chứng minh**.

Tài liệu lịch sử SimpleBank/lecture ở `docs/` là nguồn học và đối chiếu, không phải runtime specification. Repository, directory clone, database và Docker Compose project vận hành đều dùng tên **`bdspro`**.
