# BDSPro source contract — đối chiếu TechSchool SimpleBank

Nguồn tham chiếu: `techschool/simplebank` trên `master` tại commit `97f000fe58ad01a0774179ffa8884ac7784cf263`. BDSPro chủ động giữ đúng mô hình Party/Person/Account/Organization của mình.

## File-for-file ownership

| SimpleBank | BDSPro | Trách nhiệm |
|---|---|---|
| `main.go` | `main.go` | Bootstrap config, PostgreSQL, migrate, errgroup, task worker, Gateway và gRPC |
| `util/config.go`, `app.env` | Cùng đường dẫn | Viper + tên biến môi trường tương ứng, thêm `VERIFY_EMAIL_URL` |
| `util/random.go` | Cùng đường dẫn | RandomInt/RandomString/RandomEmail + Person/Organization helpers; RandomSecret dùng crypto/rand |
| `util/password.go` | Cùng đường dẫn | Bcrypt |
| `db/migration`, `db/query`, `db/sqlc`, `db/mock` | Cùng đường dẫn | Migration, SQLC, Store/SQLStore/execTx, GoMock |
| `api/` | `api/` | Gin HTTP compatibility endpoints |
| `gapi/server.go` | `gapi/server.go` | Chỉ Server và NewServer, không sở hữu listener |
| `gapi/rpc_*.go`, `gapi/user.go` | Cùng cấu trúc | RPC nghiệp vụ BDSPro và response conversion |
| `gapi/logger.go` | `gapi/grpc_logger.go`, `gapi/http_logger.go` | RPC / HTTP log tại gateway boundary |
| `worker/distributor.go` | Cùng đường dẫn | TaskDistributor, Redis client |
| `worker/processor.go` | Cùng đường dẫn | TaskProcessor, Asynq startup/shutdown |
| `worker/task_send_verify_email.go` | Cùng đường dẫn | Mã hóa payload, xử lý email verification |
| `worker/logger.go` | Cùng đường dẫn | Adapter Asynq/Redis → zerolog |
| `worker/mock/distributor.go` | Cùng đường dẫn | Mockgen từ TaskDistributor |
| `mail/sender.go` | Cùng đường dẫn | Gmail sender |
| `token/` | Cùng đường dẫn | JWT token Maker (BDSPro chọn JWT thay vì PASETO runtime) |
| `proto/`, `pb/` | Cùng đường dẫn | Protobuf source / generated gRPC Gateway |
| `doc/swagger/`, `doc/statik/` | Cùng đường dẫn | Swagger và statik generated assets |
| `Dockerfile`, `docker-compose.yaml` | Cùng đường dẫn | Multistage build, Postgres/Redis/API network |
| `start.sh`, `wait-for.sh` | Cùng đường dẫn | Entrypoint và đợi Postgres |
| `Makefile` | Cùng đường dẫn | database, migrate, sqlc, mocks, proto, server, tests |

## Bootstrap — không trùng trách nhiệm

```text
main
  -> LoadConfig + environment logging
  -> signal.NotifyContext
  -> pgxpool.New + Ping
  -> runDBMigration(MigrationURL, DBSource)
  -> db.NewStore(pool)
  -> worker.NewRedisTaskDistributor(redisOpt)
  -> errgroup.WithContext
      -> runTaskProcessor (Asynq + Gmail)
      -> runGatewayServer (protobuf Gateway + Swagger statik + Gin compatibility)
      -> runGrpcServer (grpc.NewServer + reflection)
  -> errgroup.Wait
```

Khi đóng ứng dụng: HTTP server Shutdown, gRPC GracefulStop, Asynq Shutdown, Redis client Close và PostgreSQL pool Close. Một cổng HTTP, một cổng gRPC, một PostgreSQL, một Redis.

## Khác biệt chính đáng với SimpleBank

- Không copy các bảng, truy vấn, RPC và generator cho tài khoản ngân hàng/chuyển tiền; domain BDSPro là Party, Person, Account, Session, Organization và membership.
- BDSPro giữ JWT (đã có session/refresh) thay vì thêm PASETO song song.
- Giữ phần Gin chưa di trú lên protobuf để không phá các API cũ; HTTP Gateway là điểm bắt đầu HTTP trong `main.go`, không khởi động Gin listener thứ hai.
- BDSPro có `auth/` giữ nghiệp vụ Register/Login dùng chung giữa Gin và gRPC; đó là code có một trách nhiệm, không phải service chuyển tiếp dư thừa.
- Dùng PostgreSQL 18 thay vì 14, Docker host ports 15432/18080/19090 và volume `lecture37-postgres` để bảo vệ môi trường WSL2 hiện tại.
- `util.RandomSecret` dùng bộ sinh ngẫu nhiên mật mã; không dùng `math/rand` cho mã xác thực.
- `main.go` giới hạn thời gian shutdown HTTP, xử lý lỗi trả về, đóng connection pool và Redis client rõ ràng.
- Không xây dựng hoặc giả vờ đã chạy các phần AWS, ECR, RDS, Route53, EKS, TLS production. Không triển khai SMS/OTP khi chưa có provider.

## Sự thật vận hành

`make server` đọc `app.env` và tự chạy migrations Up. `make migrateup` phục vụ DBA/manual workflow riêng; PostgreSQL kiểm soát migration đã áp dụng. Không chạy `migratedown` hay xóa volume khi có dữ liệu cần giữ.

`docker compose up --build` dùng mặc định `docker-compose.yaml`; không tạo container global tên `postgres` nên không đụng tới local standalone. API chờ `postgres:5432`, có `start.sh` và `wait-for.sh` như SimpleBank.

Test DB ghi dữ liệu chỉ chạy trên GitHub Actions disposable PostgreSQL; local `make test` bỏ qua DB integration tests. Unit, integration và Compose smoke kiểm chứng source; test thực Gmail SMTP chưa thực hiện khi thiếu credentials.
