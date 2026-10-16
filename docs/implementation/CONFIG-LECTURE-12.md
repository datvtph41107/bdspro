# Lecture #12 — Load config from file and environment variables with Viper

Tham chiếu: TechSchool SimpleBank [commit 6354a89](https://github.com/techschool/simplebank/commit/6354a89f06f4c6b07760620fe11ff6cb2acc3cd6).

## Cách tổ chức giống SimpleBank

- `app.env` tại repository root — các cấu hình mặc định cho **PostgreSQL local development**.
- `util/config.go` — `Config`, `LoadConfig(path)`, Viper `AddConfigPath/SetConfigName/SetConfigType/AutomaticEnv/ReadInConfig/Unmarshal`.
- `main.go` — `util.LoadConfig(".")`, tạo PostgreSQL pool và Gin server.
- `Makefile` — giữ `server: go run .`, không export env bằng shell.
- `util/config_test.go` — đọc cấu hình file và ghi đè bằng environment.

Một `viper.New()` độc lập được sử dụng để tránh nhiễm trạng thái giữa các lần gọi/test. `BindEnv` khai báo các key có thể chỉ tồn tại ở biến môi trường để `Unmarshal` vẫn đọc được. Đây là phần điều chỉnh kỹ thuật, giữ nguyên logic SimpleBank.

## Các key

| Key | Ý nghĩa |
| --- | --- |
| ENVIRONMENT | `development` hoặc môi trường triển khai khác; chế độ debug token chỉ cho development |
| DB_DRIVER | `postgres`; BDSPro sử dụng pgx/v5 |
| DB_SOURCE | PostgreSQL connection URL |
| SERVER_ADDRESS | `host:port` HTTP Gin/server |
| ALLOWED_ORIGIN | Origin được phép cho browser mutations |
| SMTP_ADDR | Máy chủ SMTP (để trống ở development) |
| SMTP_USER | Tên đăng nhập SMTP |
| SMTP_PASSWORD | Mật khẩu SMTP, dùng secret environment khi triển khai |
| SMTP_FROM | Người gửi |
| VERIFICATION_URL | URL dùng trong verification email |

## Thứ tự ưu tiên

Biến môi trường có cùng tên **ghi đè** key trong `app.env`. Ví dụ:

```sh
# app.env dùng local bdspro theo mặc định, không cần export mỗi terminal
make server

# Override khi cần kết nối DB khác, không sửa app.env
DB_SOURCE='postgresql://root:secret@localhost:5432/bdspro_test?sslmode=disable' make server
```

Nếu dùng môi trường không phải development, phải cấu hình `SMTP_ADDR` và `ALLOWED_ORIGIN` trước khi startup. Không đưa credential thật vào app.env được commit. Các giá trị root/secret trong file chỉ là tài khoản PostgreSQL Docker LAB.

**Đổi tên so với bootstrap trước:** `DATABASE_URL` → `DB_SOURCE`, `APP_ENV` → `ENVIRONMENT`, `BIND_HOST` + `PORT` → `SERVER_ADDRESS`. `ALLOWED_ORIGIN`, `SMTP_*`, `VERIFICATION_URL` giữ nguyên. Các tích hợp tự động phải dùng tên mới.

CI sử dụng `DB_SOURCE`, `ENVIRONMENT`, `SERVER_ADDRESS` để smoke-test ứng dụng, và vẫn dùng `CORE_TEST_DATABASE_URL`/`SOURCE_TEST_DATABASE_URL`/`SQLC_TEST_DATABASE_URL` riêng cho tests, không trỏ test phá hủy dữ liệu vào DB dev.

## Kiểm chứng

```sh
go test ./util -run '^TestLoadConfig' -v -count=1
make test
make server
```

Chưa triển khai Lecture #13 (mock store) hoặc các Config cho GRPC/Redis/token không tồn tại trong bài hiện tại.
