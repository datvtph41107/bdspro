# BDSPro — Quickstart hiện hành cho developer

**Branch:** `dev`; **runtime config:** `app.env` + Viper; **database:** PostgreSQL 18; **Go:** phiên bản trong `go.mod`.

## 1. Clone và xác nhận đúng nhánh

```bash
git clone --branch dev --single-branch https://github.com/datvtph41107/bdspro.git
cd bdspro
git status --short --branch
git log -1 --oneline
```

Nếu đã có local clone, trước tiên chạy `git status --short --branch`, `git fetch origin`, xem diff giữa local/remote. Chỉ `git pull --ff-only origin dev` khi không có thay đổi chưa lưu và nhánh tương thích. Không dùng `reset --hard` để đồng bộ.

## 2. PostgreSQL local disposable

```bash
make postgres
make createdb DB_NAME=bdspro
```

Nếu Docker container đã tồn tại, kiểm tra `docker ps -a` và trạng thái thay vì cố tạo mới. Trước migration xác nhận URL kết nối chỉ trỏ tới DB development:

```bash
export DB_URL='postgresql://root:secret@localhost:5432/bdspro?sslmode=disable'
psql "$DB_URL" -c 'select current_database(), current_user;'
make migrateup DB_URL="$DB_URL"
```

`DB_URL` dùng cho CLI migration; `DB_SOURCE` trong `app.env` dùng cho runtime. Hai giá trị phải cùng đích khi bạn đang thao tác trên local development. Không dùng credential `root/secret` mẫu cho production.

## 3. Chạy và kiểm tra

```bash
make sqlc
make test
make server
```

Server dùng `app.env` mặc định, DEVELOPMENT lắng nghe loopback. `go test ./...` có thể SKIP integration tests nếu không cấu hình các biến `CORE_TEST_DATABASE_URL`, `SOURCE_TEST_DATABASE_URL`, `SQLC_TEST_DATABASE_URL`.

Để chạy test tích hợp đầy đủ cần các database **riêng, có thể hủy**, đúng tên `bdspro_core_ci`, `bdspro_source_ci`, `bdspro_sqlc_ci` và chạy đủ migrations, như workflow CI. Một số fixtures dùng `TRUNCATE ... CASCADE`; không trỏ vào DB chứa dữ liệu cần giữ.

## 4. Các HTTP endpoints chính

- `POST /auth/register`: tạo tài khoản; trong local development trả verification token debug.
- `POST /auth/verify-email`: xác minh bằng token.
- `POST /auth/resend-verification`: body `{"email":"someone@example.test"}`, trả HTTP 202 chung khi email hợp lệ. Không tiết lộ email có tồn tại hay không. Trường hợp có cooldown 60 giây.
- `POST /auth/login`, `GET /auth/me`, `POST /auth/logout`.
- Các routes Organization và Source theo `api/server.go`.

**Lưu ý:** Resend có ý nghĩa khi đã cấu hình SMTP thực sự; development không có mailer vẫn dùng token debug nhận được ở lần đăng ký ban đầu. Không dùng chế độ development trên public bind.

## 5. Xác minh remote sau pull

```bash
git status --short --branch
git log -1 --oneline
git diff --exit-code
```

Không thay `db/migration/000001...` trực tiếp trên database đã được migrate. Các thao tác database có khả năng xóa dữ liệu cần được xét riêng.
