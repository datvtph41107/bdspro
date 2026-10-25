# ENG-CORE-07 — Dependency Direction & Module Ownership

**Source baseline:** BDSPro `dev@f432a5dce1dfc1cb19d76645d63c73920fb20ef9` (25/10/2026).

**Intent:** Không chỉ ghi nguyên tắc Architecture vào tài liệu mà điều chỉnh **quan hệ import thực tế** để application logic không phụ thuộc vào queue vendor, sau đó khóa ranh giới ấy bằng tests. Kế thừa [ENG-CORE-06](application-operation-boundary.md), [ENG-GOV-03](capability-formula-catalog.md) và [Register/Asynq](../operations/register-asynq.md).

## 1. Dependency map: thực trạng đã đọc trong source

| Package / execution owner | Các phụ thuộc liên quan đã xác nhận | Trách nhiệm chính |
| --- | --- | --- |
| `main.go` | api, db/sqlc, gapi, mail, worker, util, Redis/Asynq | Composition root, startup/shutdown |
| `api` | auth, organization, db/sqlc, worker, Gin | HTTP admission, actor, response, dependency composition |
| `auth` **trước refactor** | db/sqlc, token, util, **worker**, **Asynq** | Account registration, authentication, session |
| `auth` **sau refactor** | db/sqlc, token, util; **consumer-owned `VerificationTaskScheduler`** | Yêu cầu scheduling sau khi tạo account, không biết queue runtime |
| `organization` | db/sqlc | Account → Person → Organization use cases |
| `worker` | db/sqlc, mail, util, Asynq | Task enqueue/processor và execution policy |
| `db/sqlc` | PostgreSQL/pgx/sqlc | Queries, transaction, data invariants |
| `gapi` | auth, db/sqlc, token, worker, protobuf | RPC Register dùng cùng verification scheduling port như HTTP |

**Source-backed issue:** `auth.Register` trực tiếp gọi `worker.TaskDistributor.DistributeTaskSendVerifyEmail` với `worker.PayloadSendVerifyEmail` và Asynq options `MaxRetry(10)`, `ProcessIn(10s)`, `Queue(worker.QueueCritical)`. Đây là chi tiết runtime queue đang nằm trong application auth.

**Quyết định:** đưa **scheduling intent** vào contract nhỏ do `auth` sở hữu và đưa các tùy chọn Asynq về `worker`. Không đảo ngược dependency bằng việc để `auth` import worker, và không tạo event bus, adapter registry, DI container hay package `core`.

## 2. Source sau refactor: nhu cầu khác cơ chế thực thi

```go
// auth/verification_scheduler.go — consumer-owned port
type VerificationTaskScheduler interface {
    ScheduleVerifyEmail(ctx context.Context, accountID int64) error
}

// auth/auth.go — application intent within existing RegisterTx callback
AfterCreate: func(account db.Account) error {
    return scheduler.ScheduleVerifyEmail(ctx, account.ID)
}

// worker/verification_scheduler.go — infrastructure adapter
func (s VerificationScheduler) ScheduleVerifyEmail(ctx context.Context, accountID int64) error {
    return s.distributor.DistributeTaskSendVerifyEmail(ctx,
        &PayloadSendVerifyEmail{AccountID: accountID},
        asynq.MaxRetry(10), asynq.ProcessIn(10*time.Second),
        asynq.Queue(QueueCritical))
}
```

**Composition:** Cả `api.NewServer` và `gapi.NewServer` vẫn nhận `worker.TaskDistributor` như trước, mỗi transport tạo `worker.NewVerificationScheduler(distributor)` **một lần** rồi truyền port cho `auth.Register`. `gapi/rpc_register_user.go::Register` thực sự gọi `auth.Register`, vì vậy HTTP và gRPC cùng tái sử dụng operation và scheduling intent. Generated mocks, request/response, SQL và migrations không thay đổi.

### Không thay đổi các bảo đảm hiện hành

- Enqueue vẫn được gọi trong `RegisterTx.AfterCreate`, tức là **trước PG COMMIT**.
- PostgreSQL và Redis không commit nguyên tử. `ProcessIn(10s)` là timing, không bảo đảm worker chỉ chạy sau COMMIT; rủi ro đã chứng minh ở ENG-GOV-03.
- Redis accepted task và PG rollback vẫn có thể cùng xảy ra. **Không claim outbox/exactly-once delivery**.
- Email worker dùng payload AccountID, existing queue `critical`, MaxRetry=10 và ProcessIn=10s như cũ; việc chuyển policy không thay policy.
- Application Port trả error nguyên trạng; `RegisterTx` callback failure tiếp tục làm PG rollback theo Store contract.

## 3. Khi nào thêm interface là có căn cứ?

`VerificationTaskScheduler` chỉ có **một** method, được định nghĩa bởi consumer `auth` vì:

1. **Đã có phụ thuộc sai chiều thực tế:** `auth` import worker + Asynq mặc dù nghiệp vụ chỉ cần scheduling intent.
2. **Runtime options đang làm application change amplification:** queue/MaxRetry/ProcessIn buộc sửa `auth/auth.go` thay vì `worker`.
3. **Contract stable và testable:** input AccountID và context; lỗi scheduler được trả về cho transaction owner.
4. **Không đổi dữ liệu hay semantics:** adapter reuse existing worker.TaskDistributor và bảo toàn options.

Không suy luận rằng mọi use case phải có interface: `organization.CreateForAccount` dùng trực tiếp `db.Store` hiện có, và những SQL statement kiểm tra quyền vẫn gọi thẳng từ handler.

## 4. Verification / acceptance

- `auth/verification_scheduler_test.go`: Register schedule đúng AccountID/context, bảo toàn email normalization/password hash; lỗi enqueue không bị nuốt.
- `auth/dependency_boundary_test.go`: AST kiểm tra **production imports** của package `auth`, từ chối import trực tiếp `worker` hoặc `github.com/hibiken/asynq`. Không áp đặt cùng quy tắc lên tests/transport.
- `worker/verification_scheduler_test.go`: adapter gọi distributor với payload đúng, ba tùy chọn queue (retry/delay/queue) và truyền nguyên lỗi enqueue.
- Existing RegisterTx rollback/handoff integration tests, API register, worker verify email, migrations, build/smoke/vet/race là regression gates.
- **Mức nghiệm thu:** source review + test evidence trong CI. Không giả nhận đã chứng minh retry delivery end-to-end, WSL runtime hoặc production recovery.

## 5. Future decisions / không làm

- Chưa tách `db.Store` thành hàng loạt interfaces vì chưa có bằng chứng giảm coupling; generated SQLC/MockGen Store đang là contract hiện hành.
- Không xóa gRPC/gateway chỉ vì `main.run` đang ưu tiên Gin; đã kiểm tra composition của gRPC Register nhưng chưa tuyên bố gRPC end-to-end.
- Chưa đưa `auth` vào vendor-agnostic architecture ngoài scheduling: JWT/PG vẫn là dependency có chủ ý của authentication hiện tại.
- Khi thêm event/effect mới, yêu cầu xác định **intent owner → execution adapter → transaction/delivery guarantee → failure tests** trước khi tạo contract.

**Decision:** Tập trung một ranh giới bị sai thật và kiểm chứng bằng import guard; không đồng nhất kiến trúc tốt với số lượng interface/package.
