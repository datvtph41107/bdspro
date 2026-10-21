# BDSPro — Authentication/Authorization baseline (SimpleBank Lecture #37)

Canonical source: branch `dev`. Reference: [SimpleBank Lecture #37, commit 5663a678](https://github.com/techschool/simplebank/commit/5663a678fa15e8427619a997af5ed6dbd6f31e33).

## Phạm vi hiện tại

- `api.Server` giữ `config`, `store`, `tokenMaker`, `router` theo phong cách SimpleBank.
- `POST /users`: tạo `Party → Person → Account` trong `db.SQLStore.RegisterTx`.
- `POST /users/login`: kiểm tra mật khẩu, phát hành access/refresh JWT, tạo phiên PostgreSQL.
- `POST /tokens/renew_access`: kiểm tra refresh JWT và session hiện hành.
- `GET /users/me`: middleware Bearer access-token bảo vệ endpoint.
- `token.Maker`, `db.Store`, sqlc, PostgreSQL migration và password helper chỉ tồn tại tại một vị trí mỗi trách nhiệm.

**Authentication** tại #37 gồm các luồng trên. **Authorization** ở mốc này là xác thực Bearer token và kiểm tra quyền truy cập vào endpoint của account hiện hành; **chưa có Role/Permission/RBAC hoặc workspace**. Không nên gọi bảo vệ endpoint là hệ thống phân quyền hoàn chỉnh.

Không có `source/`, `organization/`, `auth/`, `gapi/`, `proto/`, `pb/`, email verification, SMS OTP, SMS Outbox hay worker/outbox nghiệp vụ trong baseline. Những chức năng này thuộc các mốc triển khai sau khi có thiết kế và bài học tương ứng; **không dựng stub/mock hay package rỗng để chuẩn bị trước**.

## Test và mock: một vị trí cho mỗi trách nhiệm

```text
api/
  main_test.go        # newTestServer dùng chung trong package API
  user_test.go        # Register/Login, validation, DB failure
  token_test.go       # Renew access, blocked/expired/missing session
  middleware_test.go  # Bearer access; từ chối refresh token
db/
  mock/store.go       # MockStore DUY NHẤT; sinh từ db.Store
  sqlc/main_test.go   # Helpers cho PostgreSQL test database
  sqlc/store_test.go  # RegisterTx, commit/rollback
  sqlc/session_test.go
token/jwt_maker_test.go
util/password_test.go
```

Không có `*_test.go` tại root. Không dùng `authStub`, Source mock, OTP mock hoặc test contract của các module đã rút.

```bash
go test -v ./api ./token ./util
go vet ./...
go test ./...
```

`go test ./...` khi không có `DB_TEST_URL` **sẽ SKIP** những integration tests đòi PostgreSQL, vì vậy phải chạy trên DB test riêng để nghiệm thu dữ liệu. Ví dụ:

```bash
docker exec postgres createdb -U root -O root bdspro_lecture37_test
migrate -path db/migration -database 'postgres://root:secret@127.0.0.1:5432/bdspro_lecture37_test?sslmode=disable' up
DB_TEST_URL='postgres://root:secret@127.0.0.1:5432/bdspro_lecture37_test?sslmode=disable' make testdb
```

Hai lệnh tạo/migrate database test dành cho môi trường chưa có `bdspro_lecture37_test`; không chạy lại nếu DB test đã tồn tại và đang có schema.

Tính tái lập các file sinh tự động:

```bash
make sqlc
make mock
go mod tidy
gofmt -w api db token util main.go
git diff --exit-code
```

CI kiểm tra `sqlc v1.30.0`, `go.uber.org/mock/mockgen v0.6.0`, Go test, migration, PostgreSQL integration, HTTP Register/Login/Renew/Me smoke và generated-code drift.

## Database vận hành

`app.env` trỏ về `bdspro` trên container PostgreSQL `postgres`, không trỏ về lab DB song song. Schema hiện hành gồm `parties`, `persons`, `accounts`, `sessions` và `schema_migrations`. **Không chạy lại reset script hoặc tự ý drop DB khi đang phát triển và test.**

```bash
docker exec postgres pg_isready -U root -d bdspro
docker exec postgres psql -U root -d bdspro -c 'SELECT version, dirty FROM schema_migrations'
make server
```

Ví dụ chạy thử:

```bash
curl -i -X POST http://127.0.0.1:8080/users -H 'Content-Type: application/json' -d '{"email":"test@example.test","password":"strong-password"}'
curl -i -X POST http://127.0.0.1:8080/users/login -H 'Content-Type: application/json' -d '{"email":"test@example.test","password":"strong-password"}'
```

API login trả `access_token`, `refresh_token`, `session_id`; renew gọi `POST /tokens/renew_access` với `{"refresh_token":"..."}`; `/users/me` nhận `Authorization: Bearer <access_token>`.

## Đồng bộ Git

```bash
cd ~/projects/bdspro
git status --short --branch
git fetch origin --prune
git switch dev
git merge --ff-only origin/dev
git log -1 --oneline
```

Nhánh `archive/pre-lecture37-2026-10-21` giữ checkpoint lịch sử. Những thay đổi local cũ đã được người phát triển `stash` thì **không `stash pop` tự động** vào baseline này.

## Ranh giới bảo mật

Đây là baseline học tập chạy local, chưa phải hệ thống production. Refresh JWT hiện được lưu dạng plaintext trong PostgreSQL theo flow SimpleBank #37. Khi triển khai Internet cần thiết kế token-at-rest, rotate/revoke, rate limit, email/phone verification, HTTPS và policy authorization; không sinh OTP, SMS Outbox hay role/permission chỉ vì dự đoán sẽ dùng về sau.
