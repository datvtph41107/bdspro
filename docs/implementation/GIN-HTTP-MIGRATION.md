# BDSPro — Chuyển toàn bộ HTTP layer sang Gin

## Phạm vi

Tham chiếu TechSchool SimpleBank từ bài Gin (#11), áp dụng thống nhất cách tổ chức API theo `api/server.go`, `api/account.go` và các handler `api/*.go`. Từ thay đổi này, không còn sử dụng các `http.Handler` riêng trong `auth/http.go`, `organization/http.go`, `source/http.go`.

| Package | Trách nhiệm |
| --- | --- |
| `api/` | Gin router, request binding, response, cookie, session HTTP, quyền truy cập request |
| `auth/` | Register/Verify/Login/Logout, password hashing, phiên đăng nhập |
| `organization/` | Registration, reviewer, claim, invitations, membership, permissions |
| `source/` | Source intake, idempotency và lịch sử giá |
| `db/sqlc/` | SQLC queries và transactional Store |
| `main.go` | LoadConfig, pool, services và graceful shutdown |

## Ranh giới bảo toàn

- HTTP paths và phương thức không đổi, bao gồm `/auth/*`, `/organizations/*`, `/platform/organizations/*`, `/health`, `/ready`.
- Cookie `bdspro_session` trong development và `__Host-bdspro_session` trong non-development; `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`. Không sử dụng header giả làm account ID.
- Mọi thao tác thay đổi state áp dụng chung `Origin` và `Sec-Fetch-Site` guard, không chỉ Account.
- JSON binding bằng Gin, nhưng vẫn giới hạn body 8 KiB (Auth), 16 KiB (Organization) và 1 MiB (Source), không nhận unknown field hoặc nhiều JSON documents.
- Thao tác Organization tiếp tục xác thực session, ủy quyền trong service và row lock trong PostgreSQL; Source giữ kiểm tra tenant ở SQL/Service.
- Idempotency-Key và phản hồi `201`/replayed `200`/conflict `409` không thay đổi.
- `main.go` vẫn sử dụng `http.Server` với graceful shutdown; Gin là `http.Handler` của server, không yêu cầu bỏ tính năng shutdown để chạy `router.Run`.

## Kiểm thử

- `api/server_test.go`: đầy đủ route Gin, JSON validation, cross-site và 401 không có session.
- `integration_test.go`: vòng đời đăng ký/xác minh/đăng nhập/cookie/logout bằng Gin; test nghiệp vụ thực qua Organization và Source.
- `source/service_test.go`: tiếp tục test tầng nghiệp vụ, không còn HTTP handler test trong package Source.
- GitHub Actions `ci.yml` chạy PostgreSQL độc lập, `make test`, race detector, smoke startup.

## Local

```sh
git switch dev
git status --short
git fetch origin
git pull --ff-only origin dev

make test
make server
```

Đọc kỹ trạng thái trước pull nếu local đang có `source/service.go`, `db/sqlc/transaction_test.go` hoặc stash cũ. Không reset, clean hoặc pop stash tự động.

Đây là thay đổi tổ chức HTTP layer, không phải thay đổi schema, data migration hay chính sách xác thực.
