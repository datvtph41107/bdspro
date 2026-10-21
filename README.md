# BDSPro — Authentication baseline

Kiến trúc tham khảo [TechSchool SimpleBank Lecture #37 (commit 5663a678)](https://github.com/techschool/simplebank/commit/5663a678fa15e8427619a997af5ed6dbd6f31e33). BDSPro giữ cách tổ chức Go/Gin/SQLC/Store/Token của SimpleBank, nhưng dùng `Party → Person → Account`, login bằng email và `pgx/v5`.

## Source và trách nhiệm

- `main.go`: đọc `app.env`, kết nối PostgreSQL, tạo Store và Server.
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
```

`make server` đọc `app.env`. URL DB local: `postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable`. `make migrateup` dùng URL mặc định trên và cho phép đổi bằng `DB_URL` khi chạy. File `app.env` chỉ là mẫu local; khóa JWT trong đó tuyệt đối không dùng production.

## HTTP API

| Method | Route | Mục đích |
|---|---|---|
| POST | `/users` | Đăng ký Account + Party/Person trong transaction |
| POST | `/users/login` | Xác minh bcrypt, cấp access/refresh JWT, tạo Session |
| POST | `/tokens/renew_access` | Kiểm tra refresh JWT + Session, cấp access JWT |
| POST | `/tokens/revoke` | Thu hồi refresh session, chặn renew tiếp theo |
| GET | `/health` | HTTP liveness, không kiểm tra database |
| GET | `/users/me` | Bearer access-token; trả account ID |

JWT `Payload` chứa UUID, AccountID, TokenType, IssuedAt, ExpiredAt. Chuyển từ `jwt.RegisteredClaims` sang getter thủ công theo phong cách SimpleBank có thể làm token trước đó không tương thích: đăng nhập lại để nhận token mới.

Access JWT đã phát vẫn có hiệu lực đến khi hết hạn; revoke chỉ chặn refresh token. Không thêm migration, dùng `sessions.is_blocked` có sẵn.

Xem đối chiếu lecture và quy tắc tổ chức: [docs/LECTURE-MATRIX.md](docs/LECTURE-MATRIX.md).

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
