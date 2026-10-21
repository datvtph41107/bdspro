# BDSPro — SimpleBank Lecture #37

**Baseline hiện hành của nhánh `dev`**: Gin API, Store + SQLC, JWT access/refresh, PostgreSQL session, mock Store và unit/integration tests. Tham chiếu: [SimpleBank #37, commit 5663a678](https://github.com/techschool/simplebank/commit/5663a678fa15e8427619a997af5ed6dbd6f31e33).

## Phạm vi kiến trúc

- `api.Server`: `config`, `store`, `tokenMaker`, `router`.
- `db.Store` interface, `db.SQLStore` implementation, `db/mock.MockStore` từ mockgen.
- `RegisterTx`: Party → Person → Account trong transaction.
- `loginUser`: password, access JWT, refresh JWT, CreateSession.
- `renewAccessToken`: xác minh refresh token và session lưu trong PostgreSQL.
- `authMiddleware`: Bearer access JWT, không dùng refresh JWT làm access JWT.
- Không có Organization/Source nghiệp vụ, Phone OTP, email verification, Swagger, gRPC hoặc error contract.

## Nâng cấp database BDSPro đang tồn tại

**Chú ý: schema Lecture #37 KHÔNG tương thích với migration 000001–000010 cũ**. Không chạy `make migrateup` trực tiếp trên DB `bdspro` cũ, vì go-migrate sẽ không tự chuyển cấu trúc.

Sau khi `git pull`, dừng ứng dụng đang kết nối `bdspro`. Trên WSL, từ root repository `dev` đã đồng bộ:

```bash
docker start postgres
CONFIRM_RESET_BDSPRO=RESET_BDSPRO_TO_LECTURE37 make resetbdspro
```

Script **tạo bản backup PostgreSQL dạng custom tại `~/bdspro-db-backups/` và kiểm tra backup trước**, sau đó thay thế database duy nhất có tên `bdspro`, apply migration 000001 mới và kiểm tra các bảng/UUID/session/version. Cả dữ liệu cũ lẫn các bảng tính năng cũ trong `bdspro` sẽ không còn trong DB mới; chúng vẫn nằm trong file backup. **Không chạy script nếu muốn giữ dữ liệu cũ trực tiếp trong DB.** Đây là thao tác chủ động tại máy local, không được thực thi từ GitHub CI.

Nếu migration thất bại, phục hồi backup đã tạo (thay `/path/to/backup.dump` bằng đường dẫn script in ra):

```bash
docker exec postgres psql -U root -d postgres -c 'DROP DATABASE bdspro WITH (FORCE);'
docker exec postgres createdb -U root -O root bdspro
docker exec -i postgres pg_restore -U root -d bdspro --no-owner --no-privileges < /path/to/backup.dump
```

Nếu DB `bdspro` chưa tồn tại, script sẽ tạo mới mà không có dữ liệu cũ để backup. Database `postgres`, `root` và các database test không bị chạm đến.

## Run + test

```bash
go test -v ./api ./token ./util
make server
```

Ví dụ HTTP:

```bash
curl -i -X POST http://127.0.0.1:8080/users -H 'Content-Type: application/json' -d '{"email":"test@example.test","password":"strong-password"}'
curl -i -X POST http://127.0.0.1:8080/users/login -H 'Content-Type: application/json' -d '{"email":"test@example.test","password":"strong-password"}'
```

API trả `access_token`, `refresh_token`, `session_id`. Dùng `POST /tokens/renew_access` với JSON `{"refresh_token":"..."}`, `GET /users/me` với `Authorization: Bearer <access_token>`.

Database integration tests bắt buộc chạy trên DB riêng:

```bash
docker exec postgres createdb -U root -O root bdspro_lecture37_test
migrate -path db/migration -database 'postgres://root:secret@127.0.0.1:5432/bdspro_lecture37_test?sslmode=disable' up
DB_TEST_URL='postgres://root:secret@127.0.0.1:5432/bdspro_lecture37_test?sslmode=disable' make testdb
```

Nếu `DB_TEST_URL` chưa đặt, DB integration tests sẽ SKIP: `go test ./...` PASS **không thay thế** nghiệm thu DB.

Để chứng minh source đồng bộ với SQLC/mock:

```bash
make sqlc
make mock
go mod tidy
go test ./...
git diff --exit-code
```

CI kiểm tra các bước trên, migration và integration tests với PostgreSQL thật. Phiên bản: `sqlc v1.30.0`, `go.uber.org/mock/mockgen v0.6.0`.

## Đồng bộ Git

```bash
git fetch origin --prune
git switch dev
git status --short --branch
git merge --ff-only origin/dev
git log -1 --oneline
```

Nhánh lưu trạng thái cũ: `archive/pre-lecture37-2026-10-21`. Lịch sử commit được bảo toàn. **Không thực hiện `git reset --hard` để xóa thay đổi chưa commit.**

## Giới hạn

Đây là baseline học tập chạy ở loopback, **không phải production authentication**. Secret trong `app.env` là development-only; PostgreSQL lưu refresh JWT dạng plaintext theo flow gốc. Trước khi public API cần xem xét revocation, token storage, rate limiting, email verification, HTTPS và các kiểm soát bảo mật tương ứng.
