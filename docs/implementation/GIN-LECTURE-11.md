# Lecture #11 — Implement RESTful HTTP API in Go using Gin

Tham chiếu chính xác: TechSchool SimpleBank commit `beb83af8335d87afbf2f8bb4b678c2f9fec99fc7` (`add http api`).

## Cấu trúc và triển khai

- `api/server.go`: `Server`, `NewServer`, `gin.Default()`, routes, `Start`, `errorResponse` theo tên và tinh thần SimpleBank.
- `api/account.go`: `createAccountRequest` và `createAccount` dùng Gin binding rồi gọi `auth.Service.Register`. Không dùng các trường `owner/balance/currency` của ngân hàng vì chúng không thuộc Account của BDSPro.
- `api/server_test.go`: kiểm tra health, route fallback, request binding và chống request cross-site.
- `main.go`: khởi tạo `db.NewStore(pool)`, `api.NewServer(...)`; giữ `http.Server` để có graceful shutdown, ReadHeaderTimeout và logic Origin/CSRF cũ.
- Gin xử lý `POST /auth/register` và `GET /health`. Các route cũ của Auth, Organization và Source được giữ nguyên bằng `NoRoute(gin.WrapH(legacy))`, không bị xóa hoặc tự mở công khai.

## Giới hạn đúng theo Lecture

Đây là mốc Gin ban đầu. Chưa thay `net/http` của toàn bộ module hoặc tự tạo thêm CRUD Account công khai; `GET /accounts` không tồn tại vì chưa có contract phân quyền. Lecture #12 mới đến Viper và #13 mới đến mock HTTP DB theo thứ tự SimpleBank.

Gin giữ kiểm tra JSON unknown fields, giới hạn body 8192 bytes, mã lỗi đăng ký, bật token debug chỉ ở development và không cache response. Middleware Origin bảo vệ toàn bộ Gin router tại `main.go`.

## Kiểm chứng

```sh
make test
go test ./api -count=1 -v
go vet ./...
make server
```

CI ở `.github/workflows/ci.yml` chạy `make test` trên PostgreSQL riêng, kiểm tra `gofmt`, `go vet` và startup.
