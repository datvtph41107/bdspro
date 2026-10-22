# BDSPro — Core Backend

> **Runtime mặc định (Gin HTTP only):** `run()` chỉ khởi động Gin HTTP và Redis email-verification worker. `runGrpcServer()` và `runGatewayServer()` được giữ trong source nhưng không được gọi. Docker Compose không công bố cổng 9090. Các endpoint `/v1/...`, `/swagger/` và `/docs/openapi.json` thuộc Gateway không hoạt động ở chế độ này.


Kiến trúc tham khảo [TechSchool SimpleBank (78 lecture backend)](https://github.com/techschool/simplebank/blob/97f000fe58ad01a0774179ffa8884ac7784cf263/README.md). BDSPro giữ cách tổ chức Go/Gin/SQLC/Store/Token của SimpleBank, nhưng dùng `Party → Person → Account`, login bằng email và `pgx/v5`.

Xem [ma trận từng bài #0–77](docs/LECTURES-00-77.md) để phân biệt phần có code với phần chưa triển khai.

## Source và trách nhiệm

- `main.go`: bootstrap config, PostgreSQL/migrations, Redis worker và `runGinServer`; gRPC/Gateway không còn chạy trong `run()`.
- `util/config.go`: Viper `ReadInConfig` + `AutomaticEnv`, không có `BindEnv` hoặc cơ chế `.env` fallback.
- `util/password.go`: bcrypt hash/verify mật khẩu.
- `db/migration/`: schema `parties`, `persons`, `accounts`, `sessions`; migration `000001` đã áp dụng trên local thì **không sửa hoặc chạy lại bằng tay**.
- `db/query/`: các SQL query đầu vào của sqlc; `db/sqlc/` là code được sinh, ngoại trừ Store và các test thủ công.
- `db/sqlc/store.go`: `Store`/ `SQLStore`, `execTx`, `RegisterTx` (Party, Person, Account trong cùng transaction).
- `token/`: `Maker`, `Payload`, HS256 JWT, phân biệt access/refresh bằng `TokenType`; không đưa Role giả hoặc PASETO vào baseline.
- `api/`: Gin Server, Register/Login, middleware Bearer, refresh token, các unit test dùng MockStore.

## Database: duy nhất tên `bdspro`

Local dùng container `postgres` và database `bdspro`. GitHub Actions cũng tạo database tên `bdspro`, **nhưng trên PostgreSQL tạm riêng của runner**, không liên quan dữ liệu local. Không tạo `bdspro_test`, `bdspro_core`, hoặc `bdspro_lecture37_test`.

Xác minh local (chỉ đọc):

```bash
docker exec postgres pg_isready -U root -d bdspro
docker exec postgres psql -U root -d bdspro -c 'SELECT version, dirty FROM schema_migrations'
```

Nếu local đang `version=1, dirty=false`, không cần chạy `make migrateup` hoặc reset DB. Với database mới được tạo riêng, `make migrateup` sẽ áp dụng migration chưa chạy; không dùng `migratedown` tùy tiện.

`db/sqlc/main_test.go` chỉ cho phép DB integration tests ghi dữ liệu khi đang chạy trong GitHub Actions, và xác nhận `current_database() = 'bdspro'`. `make test` thông thường trên WSL chạy unit tests và skip DB integration tests để không ghi dữ liệu vào DB local. Trên CI, `make test` chạy tất cả bài test sau migration.

## Quy trình chạy

```bash
make server
make test
make sqlc
make mock
make proto
make fmt
```

`make server` đọc `app.env` và tự áp dụng go-migrate Up (bỏ qua ErrNoChange). URL DB local: `postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable`. `make migrateup` là CLI riêng; startup cũng chạy migrate Up, không nhân đôi trong Docker Compose. URL mặc định trên và cho phép đổi bằng `DB_URL` khi chạy. File `app.env` chỉ là mẫu local; khóa JWT trong đó tuyệt đối không dùng production.

## HTTP API

| Method | Route | Mục đích |
|---|---|---|
| POST | `/users` | Đăng ký Account + Party/Person trong transaction |
| POST | `/users/login` | Xác minh bcrypt, cấp access/refresh JWT, tạo Session |
| GET | `/users/verify_email` | Xác minh email một lần trong transaction |
| POST | `/tokens/renew_access` | Kiểm tra refresh JWT + Session, cấp access JWT |
| POST | `/tokens/revoke` | Thu hồi refresh session, chặn renew tiếp theo |
| GET | `/health` | HTTP liveness, không kiểm tra database |
| GET | `/users/me` | Bearer access-token; trả account ID |
| POST | `/organizations` | Tạo Organization và owner trong transaction |
| GET | `/organizations` | Các Organization mà Person đang tham gia |
| POST | `/organizations/:id/members` | Owner thêm Person làm member, kiểm tra quyền tại SQL |
| PATCH | `/organizations/:id` | Owner cập nhật tên Organization, kiểm tra quyền trực tiếp trong UPDATE SQL |

JWT `Payload` chứa UUID, AccountID, TokenType, IssuedAt, ExpiredAt. Chuyển từ `jwt.RegisteredClaims` sang getter thủ công theo phong cách SimpleBank có thể làm token trước đó không tương thích: đăng nhập lại để nhận token mới.

Access JWT đã phát vẫn có hiệu lực đến khi hết hạn; revoke chỉ chặn refresh token. Không thêm migration, dùng `sessions.is_blocked` có sẵn.

Xem đối chiếu lecture và quy tắc tổ chức: [docs/LECTURE-MATRIX.md](docs/LECTURE-MATRIX.md).

## HTTP lifecycle

Gin sử dụng request logger, recovery và `rs/cors` ở HTTP boundary. `SIGINT`/`SIGTERM` kích hoạt shutdown HTTP (tối đa 10 giây); pool PostgreSQL đóng sau khi dừng HTTP. DBML ở `doc/db.dbml`.

## Email verification (Redis/Asynq)

Đăng ký gọi `RegisterTx` và enqueue `task:send_verify_email` trong callback transaction. Worker đọc account, tạo mã dùng một lần và gửi qua Gmail SMTP. URL xác minh: `/users/verify_email?email_id=...&secret_code=...`. Cần Redis hoạt động và cấu hình `EMAIL_SENDER_*` hợp lệ để thực sự gửi email. Compose chạy kèm Redis; nếu chưa cung cấp Gmail App Password, tác vụ sẽ retry và email sẽ không được gửi. Migration `000002` bổ sung `accounts.is_email_verified` và `verify_emails`, không chỉnh migration gốc. Local đang ở migration v1 cần chạy `make migrateup` một lần trước khi chạy server phiên bản mới.

## Organization và phân quyền

Migration `000003` tạo `organizations` và `organization_memberships`. Party không có `kind` hay trigger. Transaction `CreateOrganizationTx` tạo Party, Organization và owner membership. Owner mới được thêm member bằng câu lệnh SQL `INSERT ... SELECT ... WHERE EXISTS`. Partial unique index ngăn hai membership đang hoạt động cho một cặp Organization/Person. Sau khi đồng bộ code, local cần chạy `make migrateup` một lần để áp dụng migration mới; không sửa hay chạy lại migration cũ.

## HTTP runtime

Ứng dụng mặc định chỉ cung cấp Gin REST API (ví dụ `/users`, `/users/login`, `/tokens/renew_access`, `/organizations`). Generated Protobuf/Gateway code còn trong repository nhưng không được đăng ký lên HTTP listener và không có gRPC listener. `make sqlc` và `make mock` tiếp tục tái sinh DB access và mocks. Không sử dụng Swagger JSON cũ làm API contract của Gin runtime.

## Organization partial update (Lecture 49–51)

`PATCH /organizations/:id` và `PATCH /v1/organizations/{organization_id}` hỗ trợ cùng quyền `owner`, cập nhật `name` qua `sqlc.narg()` có kiểu nullable `pgtype.Text`. gRPC chỉ nhận access JWT theo metadata `authorization: Bearer ...`; token refresh không dùng để gọi RPC này. Không tạo bảng hay service mới.

## CI

`.github/workflows/test.yml` giữ đúng pipeline SimpleBank: checkout → setup Go → install go-migrate → `make migrateup` → `make test`. PostgreSQL 18 tạm thời do GitHub Actions khởi tạo, database tên `bdspro`. Chưa triển khai AWS hoặc production deploy.

## Đồng bộ local sau khi merge

**Không clone đè lên thư mục WSL đang có chỉnh sửa chưa commit.** Clone vào thư mục mới để đối chiếu và giữ lại công việc cũ:

```bash
cd ~/projects
git clone --branch dev https://github.com/datvtph41107/bdspro.git bdspro-fresh
cd bdspro-fresh
git log -1 --oneline
git status --short --branch
```

Mã Go và các test có thể chạy bằng `go test ./...` mà không ghi dữ liệu PostgreSQL local. Nếu cần đối chiếu dữ liệu trong DB, chỉ chạy SQL `SELECT`.

## Runtime theo bố cục SimpleBank

`main.go` tạo PostgreSQL pool, chạy `runDBMigration`, tạo `Store`, `RedisTaskDistributor` và điều phối `runTaskProcessor` cùng `runGinServer` bằng `errgroup.WithContext`. Legacy gRPC/Gateway code vẫn tồn tại trong repository nhưng không được khởi động.

`docker-compose.yaml` định nghĩa các service `postgres`, `redis`, `api`. `wait-for.sh` chờ PostgreSQL, `start.sh` exec binary, Dockerfile sao chép migrations để Go tự migrate. Compose vẫn giữ volume `lecture37-postgres` và host ports riêng `15432`, `18080`, `19090` để không ảnh hưởng PostgreSQL `postgres` đang có trên local. Không chạy `docker compose down --volumes` trên môi trường chứa dữ liệu muốn giữ.

HTTP Gin sử dụng routes trong `api/server.go`; CORS cấu hình bằng `ALLOWED_ORIGINS`. Các route thuộc gRPC-Gateway không còn hoạt động trong runtime mặc định.

## Swagger và logging canonical

`doc/statik/statik.go` và `doc/swagger/bdspro.swagger.json` là generated artifacts của giai đoạn Gateway, không được phục vụ trong Gin runtime. Gin cung cấp CORS và HTTP request logging.

## Kiểm chứng bố cục SimpleBank

Xem [docs/SOURCE-MAP-SIMPLEBANK.md](docs/SOURCE-MAP-SIMPLEBANK.md) để đối chiếu từng file, ownership, bootstrap, nghiệp vụ và các khác biệt có chủ đích.
