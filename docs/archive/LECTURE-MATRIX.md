# BDSPro — Ma trận lecture và hợp đồng code

Mốc SimpleBank: [techschool/simplebank](https://github.com/techschool/simplebank) commit `97f000fe58ad01a0774179ffa8884ac7784cf263`.
Mốc BDSPro: nhánh `dev`. Phần trạng thái dưới đây xác định **code đã tồn tại**, không có nghĩa mọi lecture đã triển khai xong.

## Đối chiếu theo section

| Section | SimpleBank | BDSPro | Phạm vi |
|---|---|---|---|
| 1: PostgreSQL | Migrations, sqlc, transaction | Party, Person, Account, Session và RegisterTx | Đã có |
| 2: Gin REST | HTTP handlers, middleware, test, JWT | Register, Login, Renew, Revoke, Me, Health | Đã có |
| 3: AWS/K8s | Docker, cloud deployment | Dockerfile, Compose, CI | Chỉ local, không AWS |
| 4: Session/gRPC | Refresh session, gRPC, PASETO | JWT + session + gRPC + Gateway + OpenAPI | Đã có gRPC; không dùng PASETO |
| 5: Worker | Redis/Asynq, email verification | Có worker, Gmail sender và VerifyEmailTx; cần SMTP thực tế | Đã có code, delivery ngoài CI |
| 6: Reliability | Test, security, lifecycle | Bcrypt, JWT expiry, DB unique, CI | Đã có một phần: CORS, HTTP logging, HTTP shutdown |

## Canonical request flow

```text
main -> LoadConfig -> pgxpool -> NewStore -> NewServer -> Gin (graceful HTTP stop)
  POST /users               -> RegisterTx (Party + Person + Account)
  POST /users/login         -> bcrypt -> access/refresh JWT -> CreateSession
  POST /tokens/renew_access -> VerifyToken(refresh) -> GetSession -> access JWT
  POST /tokens/revoke       -> VerifyToken(refresh) -> BlockSession
  GET  /users/me            -> authMiddleware -> AccountID
  GET  /health              -> HTTP liveness
```

## Source ownership

- `db/migration`: schema truth; không chỉnh migration đã chạy trên local.
- `db/query`: SQL definitions; `db/sqlc`: generated SQL + `store.go` chứa transaction viết tay.
- `api`: HTTP handlers, không dựng service chỉ để chuyển tiếp hàm.
- `token`: JWT signer và verifier; `util`: config và bcrypt hiện có.
- `db/mock`: gomock từ interface `Store`.

## Quy tắc Go (tham khảo Uber Go Style Guide)

- `gofmt`, tên rõ nghĩa, không interface trùng lặp hay wrapper pass-through.
- Interface dùng khi có boundary thực (`Store`, `Maker`) và cần mock, không tạo interface cho mọi struct.
- `context.Context` truyền từ HTTP tới DB; rollback/close có defer.
- Kiểm tra lỗi một lần đúng tầng chịu trách nhiệm; không trả chi tiết DB cho client.
- Chỉ validate dữ liệu có hậu quả thực: email, password, token signature/type/expiry, FK, unique email, transaction.
- Không tự phát sinh Role, Workspace, OTP, Redis, gRPC, cloud deployment.
- Thay đổi query phải giữ SQL, generated code, mock và tests đồng bộ (`make sqlc`, `make mock`).

## Vận hành

- `make server`: local Go Gin, đọc `app.env` dùng PostgreSQL `bdspro`.
- `make test`: unit tests local; integration DB chỉ chạy trên GitHub Actions.
- `docker compose up --build`: Compose DB + migrate + API tách khỏi container `postgres` local.
- `make sqlc` và `make mock`: tái sinh code từ nguồn chính.
- Revoke chỉ khóa refresh token; access JWT đã phát vẫn chạy tới hết hạn.
- Không chạy migratedown, xóa DB hay xóa volume để thử nghiệm.

## Chưa triển khai

Phone OTP, AWS/Kubernetes, gRPC partial update và gRPC RBAC chưa có implementation. Organization membership owner/member đã có. Email verification qua Redis/Gmail đã có source, cần Gmail SMTP credentials để kiểm chứng gửi thật. Không đánh đồng các mục chưa làm với yêu cầu hoàn thành khóa học.

## Đối chiếu tổ chức nguồn

- `gapi/server.go` chỉ chứa `Server/NewServer`, từng RPC ở `gapi/rpc_*.go`, phần chuyển Account ở `gapi/user.go`, HTTP/gRPC logger nằm riêng.
- `worker/distributor.go` chứa `TaskDistributor`, `worker/processor.go` chứa `TaskProcessor` và Redis lifecycle, `worker/task_send_verify_email.go` chứa queue payload và handler, `worker/logger.go` nối logging Asynq.
- `util/random.go` dùng helpers mẫu giống SimpleBank nhưng đối tượng là Person/Organization. Mã verification dùng `crypto/rand`.
