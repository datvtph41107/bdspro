# BDSPro

Backend bằng **Go, Gin, PostgreSQL, SQLC, Redis/Asynq**. Đường đọc source được tổ chức để developer bắt đầu từ một route, lần đến controller, service, câu SQL và response — kế thừa trải nghiệm làm việc có giá trị từ Laravel DRX, không mô phỏng cơ chế runtime của Laravel.

## Đọc source theo đúng trách nhiệm

| Thư mục | Trách nhiệm |
|---|---|
| `cmd/bdspro/main.go` | Điểm chạy ứng dụng: tín hiệu dừng, khởi tạo các thành phần và cleanup |
| `routes/` | Một nguồn khai báo HTTP Method + URL + Middleware + Controller action; dùng Gin trực tiếp |
| `app/http/controllers/auth/` | Register/Login/Session, Google OAuth, Phone OTP |
| `app/http/controllers/api/` | Các HTTP actions Organization |
| `app/http/requests/` | Body/Query typed contract theo từng nghiệp vụ |
| `app/http/request/` | Decode/validation thực hiện **tường minh trong controller** |
| `app/http/middleware/` | Token authentication, IP/source limiting, request logging và HTTP ingress |
| `app/http/resources/` | Chọn dữ liệu công khai; không để lộ password hash |
| `app/http/responses/` | Ghi JSON thành công hoặc lỗi cho HTTP |
| `app/errors/` | Những lỗi nghiệp vụ có tên, thông điệp công khai an toàn |
| `app/services/`, `auth/` | Xử lý nghiệp vụ / các hoạt động có nhiều bước |
| `db/query/`, `db/sqlc/` | SQL nguồn duy nhất; code SQLC và PostgreSQL Store |
| `db/open.go` | Kết nối PostgreSQL và chạy migrations |
| `worker/`, `jobs/` | Queue runtime và hành động của từng job |
| `app/http/server/` | Tạo HTTP engine, CORS, listener, shutdown |
| `docs/architecture/` | Quy chuẩn thiết kế hiện hành, cách làm việc và giới hạn nghiệm thu |

**Cách đọc ví dụ:**
`routes/auth.go` → `controllers/auth/controller.go` → `app/http/requests/auth/login.go` → `auth/auth.go` → `db/query/account.sql`.

Không có `routes.Handlers` map, registry, tự động tìm Controller, `ctx.Set` để tiêm Body hoặc `BaseController`. Controller dùng `request.JSON(ctx, &body)` hoặc `request.Query(ctx,&query,...)` một lần sau khi qua giới hạn request. Bộ giới hạn theo identity được gọi tường minh với dữ liệu đã parse.

## Clone nhánh tái cấu trúc

```bash
cd ~/projects
git clone --branch refactor/explicit-routes-cmd-20261026 --single-branch https://github.com/datvtph41107/bdspro.git bdspro
cd bdspro
git status --short --branch
```

Nếu đã có `~/projects/bdspro`, không clone đè hoặc xóa dữ liệu; đổi tên hay lưu bản cũ sau khi kiểm tra.

## Chạy dự án

```bash
docker compose up -d --build
docker compose ps
curl -i http://127.0.0.1:8080/health
```

Hoặc chạy mã Go trên WSL với PostgreSQL/Redis đang sẵn sàng:

```bash
make server
# tương đương: go run ./cmd/bdspro
```

Các cấu hình development ở `app.env`, biến môi trường thực thi ghi đè giá trị tương ứng. Không dùng secrets trong source cho production.

Source còn chứa `gapi/` và protobuf cho giao thức RPC, nhưng executable hiện tại **không khởi chạy gRPC server**. Google OAuth chỉ bật khi đủ cấu hình; Phone OTP chỉ bật trong development.

## Kiểm tra và độ tin cậy

Các Go tests/integration scripts cũ đã được loại bỏ theo yêu cầu tái chuẩn hóa source. CI sử dụng **build, go vet, migration check, Docker Compose validation và HTTP smoke**. Việc không có unit/integration regression tests khiến khả năng chứng minh concurrency và hành vi nghiệp vụ **giảm đáng kể**; không gọi việc CI xanh là nghiệm thu logic nghiệp vụ hoàn chỉnh.

Đọc [quy chuẩn hiện hành](docs/architecture/README.md) và [giới hạn kiểm chứng](docs/architecture/status.md). Các tài liệu dưới `docs/engineering/` phản ánh nhiều quyết định ở các giai đoạn trước và không thay thế bản đồ source hiện hành.
