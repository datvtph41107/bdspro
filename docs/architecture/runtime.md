# Runtime — Entrypoint, Database, HTTP Server, Queue và Error

## Điểm chạy rõ trách nhiệm

`cmd/bdspro/main.go` trực tiếp trình bày thứ tự nạp config/logger, PostgreSQL, migrations, Redis, SQLStore, Asynq worker, HTTP server và cleanup. Không có package `bootstrap/` hay một file `run.go` khổng lồ mang nguyên logic từ main sang.

`db/open.go` sở hữu kết nối/ping/migrations; `app/http/server/` sở hữu Gin engine/CORS/listener; `app/http/middleware/` sở hữu HTTP ingress, rate limits và giám sát request; `worker/` sở hữu Asynq server, queue và retry. `jobs/` chứa hành động gửi verification email.

Các thành phần được tạo **một lần khi chương trình bắt đầu**. Không tự lắp ghép qua `init()`, biến global service locator, reflection hoặc một DI container giả.

## Lỗi

`app/errors/` định nghĩa lỗi có tên, mã và thông điệp công khai. `app/http/responses/` ghi HTTP status/body và không xuất thẳng thông điệp từ PostgreSQL. Một lỗi không tìm thấy row không có nghĩa lúc nào cũng là 404; tình huống sai mật khẩu hay quyền owner có nghĩa khác. Không xây một mapper switch bao trùm toàn bộ nghiệp vụ.

## Job và Redis

`worker/` xử lý queue mechanics; `jobs/` xử lý task cụ thể. App chỉ yêu cầu task thông qua interface có giá trị thật (`auth.VerificationTaskScheduler`). Redis dùng cho rate-limit, Google OAuth state và Asynq; không tự gọi tất cả những dữ liệu này là cache.

## Nghiệm thu

Bản refactor chưa có test suite mới. Build và HTTP smoke không chứng minh correctness của OTP concurrency, rollback hay enqueue trước PostgreSQL commit. Những bất biến này phải được kiểm tra lại trên môi trường riêng trước khi triển khai production.
