# ENG-GOV-03 — Capability Workbook (Platform × Domain)

**Purpose:** Mẫu làm việc cho một capability có thật, sử dụng đồng thời [CTX-01/02/03 — Formula Catalog](capability-formula-catalog.md). Mỗi ô chỉ điền khi liên quan; không bắt buộc mỗi operation có service, interface, event, scheduler hay một ADR.

**Baseline:** đã đối chiếu source dev tại e7db09ee4e78cc6609c5d6eff2740e08d74c5cef (25/10/2026). Code có thể thay đổi: **source và test hiện hành thắng tài liệu**.

## 1. Mẫu một capability (copy vào PR hoặc issue khi thật sự cần)

| Dimension | Câu hỏi phải trả lời | Gợi ý bằng chứng |
| --- | --- | --- |
| Problem / decision | Tác nhân, thất bại, chi phí hoặc yêu cầu thực là gì? | Repro, requirement, observed log |
| Boundary / owner | Capability owner, data owner, execution owner là ai? | Source call chain, SQL, migration |
| Platform FA (F13) | Contract, runtime, lifecycle, dependencies, lỗi/retry/cancellation? | Runtime test, dependency docs, runbook |
| Domain FB (F14) | Input, đơn vị, precision, validation, state transition, persisted state? | Domain examples, SQL constraints, tests |
| Reuse (F03/F17) | Có shared invariant, stable behavior, compatible failure semantics? | Compare old/new call paths, alternative |
| Implementation (F07–F09) | Ai phát lệnh? Logic ở đâu? Transaction bao phủ gì? Effects nào đi sau? | Files/functions, sequence, DB checks |
| Evidence (F15) | Unit, integration, CI, WSL, external E2E nào thực sự đạt? | Test name, CI URL, SHA, skip/limit |
| Decision (F04–F06) | Implement / reuse / monitor / defer / reject? | Lý do, non-goals, review |

**Nguyên tắc:** từ tên một capability không suy ra tự động có service/repository/interface. Từ một job đã dispatch không suy ra exactly-once. Từ một SQL transaction không suy ra atomic commit giữa PostgreSQL và Redis.

## 2. Ví dụ đã tồn tại — Register → Redis/Asynq → Verify Email

### Source / thực thi hiện hành

- **HTTP entry:** api/server.go đăng ký POST /users với IPActiveSource → PreRegister → IPActive → api.createUser (api/user.go).
- **Application:** auth.Register tại auth/auth.go normalize email, hash password và gọi Store.RegisterTx.
- **Data boundary:** db/sqlc/store.go execTx tạo parties → persons → accounts trong **một PostgreSQL transaction**. Trước khi callback trả về, nó chưa COMMIT.
- **Cross-resource effect:** RegisterTx.AfterCreate gọi TaskDistributor.DistributeTaskSendVerifyEmail, sử dụng Asynq và Redis với ProcessIn(10s), MaxRetry(10), critical queue.
- **Worker:** worker/processor.go đăng ký task:send_verify_email với Asynq; worker/task_send_verify_email.go lấy account, tái sử dụng/tạo verification token qua Store và gửi email qua mail.EmailSender.
- **Verify:** HTTP GET /users/verify_email dùng DB verify flow để đánh dấu account đã được xác minh.
- **Docs gốc:** [Register/Asynq checkpoint](../operations/register-asynq.md) giữ nguyên quyết định không triển khai outbox lúc chưa chốt reliability requirements.

### Áp dụng các công thức

| ID liên quan | Kết quả áp dụng được xác thực từ source |
| --- | --- |
| F01/F07/F08 | Mechanism = pgx transaction + Asynq queue; policy = đăng ký một account / gửi email; runtime owner = PostgreSQL cho commit, Asynq cho task; *không* có transaction xuyên Redis/PG |
| F03/F17 | auth.Register và auth.IssueSession là những hành vi auth application; không hợp nhất worker với HTTP handler vì lifecycle/retry khác |
| F09 | Command = register; validation = PreRegister và hash; transition = tạo account; persistence = transaction PostgreSQL; effect = enqueue và gửi email |
| F13 | Task contract = AccountID payload; lifecycle = schedule → queue → worker → retry/complete; dependencies = Redis/PG/SMTP; errors = Asynq handler return |
| F14 | Input = email/password; domain state = Party/Person/Account/verification; **unit/precision không áp dụng** |
| F15 | CI có tests và worker failure behavior trong scope đã được định nghĩa; PG–Redis atomic handoff **chưa được chứng minh** |
| F16/F18 | Bài học tích lũy = ranh giới hai data owners; cần chốt desired delivery guarantee trước khi lựa chọn một giải pháp phức tạp hơn |

### Bảng failure semantics phải được hiểu đúng

| Điểm lỗi | Hành vi có thể kết luận | Không được suy ra |
| --- | --- | --- |
| Hash password thất bại | auth.Register trả lỗi, chưa gọi RegisterTx | Đã thử nghiệm mọi loại lỗi hash |
| PostgreSQL lỗi trước callback | Transaction rollback, không chủ động gọi enqueue trong callback | Mọi loại side effect đều được hoàn tác |
| Redis enqueue trả lỗi | Callback trả lỗi làm PG rollback; Redis có thể đã chấp nhận job nếu gặp network ambiguity | PG và Redis rollback nguyên tử |
| Redis enqueue thành công, PG COMMIT thất bại | Job có thể tồn tại mà account không commit | ProcessIn(10s) bảo đảm không bao giờ race |
| Worker query account trước commit | Query có thể không thấy account và trả lỗi, phụ thuộc timing | Worker được đảm bảo luôn chạy sau COMMIT |
| SMTP gửi xong nhưng phản hồi thất lạc | Asynq có thể retry và gửi lại email | Exactly-once mail delivery |
| Verify token đã dùng hoặc account đã verified | Logic worker và verification flow phải tuân thủ trạng thái hiện tại | Không cần test replay hoặc concurrent flow |

**Quyết định tại baseline:** giữ code hiện hành, theo dõi rủi ro non-atomic handoff; **chưa** tự thêm outbox, event bus, cron hoặc provider. Nếu có requirement “một account commit phải có verification task bền vững”, mới mở trade-off atomic handoff và integration gate phù hợp.

**Evidence tiếp theo nếu cần nâng reliability:** fail Redis enqueue, fail PG COMMIT, replay job, SMTP ambiguous timeout, duplicate task, concurrent verification; phân biệt test dùng fake với Redis/PG/SMTP thật. Không ghi token, OTP, secret vào log.

## 3. Ví dụ tham chiếu chưa triển khai — Calculation / Pricing

Từ DRX main: app/Utils/BigNumber.php đóng gói BCMath; app/Facades/AmalCalculator.php thể hiện phép tính payment/bonus; app/Http/Services/OrderService.php có price/quantity/fees và state transition. Những thứ này **chưa tồn tại dưới dạng Pricing Engine hoặc Money primitive trong BDSPro**.

**Khi BDSPro có nghiệp vụ giá/phí thực**, trước khi thêm package hãy xác định:

| Trục | Quyết định cần làm rõ |
| --- | --- |
| Input | Giá niêm yết, số lượng hoặc căn cứ tính phí nào là authoritative? |
| Unit | VND, USD, diện tích m², %, quantity có thể nhân/chia với nhau thế nào? |
| Representation | integer smallest unit hay decimal string/numeric; không âm thầm dùng float cho tiền |
| Precision | scale ở input, intermediate result, output, persisted NUMERIC |
| Rounding | HALF_UP/HALF_EVEN/FLOOR; quy tắc từng phép hay cuối công thức |
| Version | Biểu phí thay đổi, kết quả lịch sử có phải tái hiện được? |
| Persistence | Giá trị dẫn xuất được tính lại hay snapshot; một nguồn sự thật là gì? |
| Safety | Zero/negative, overflow, concurrency, update conflict và rollback |
| Platform | Pure calculation không phụ thuộc HTTP/DB; Store transaction chỉ khi thật sự cần lưu kết quả |
| Evidence | Golden vectors, property-based cases, PostgreSQL representation, replay version cũ |

Ví dụ **chỉ minh họa công thức**, chưa phải quy tắc giá được BDSPro chấp nhận:

~~~text
Payable = Quantize(Base + Fee - Discount, currency scale, rounding mode)
~~~

Không dùng công thức minh họa này để ghi tiền thật khi chưa chốt đơn vị, rounding, versioning và invariants. Tương tự **Order Matching Engine DRX chỉ là tham chiếu nghiên cứu**, không phải lý do xây matching engine cho bất động sản.

## 4. Lộ trình tích lũy theo evidence, không theo số package

| Chặng học | Platform track | Domain track | Điều kiện để áp dụng vào code |
| --- | --- | --- | --- |
| Composition | main.go / NewServer, Asynq startup/shutdown | Một use case có owner rõ | Hiện trạng thiếu contract thực, không phải tên “provider” |
| Exact calculation | Decimal/rounding/SQL NUMERIC | Giá, phí, chiết khấu có business specification | Golden test vectors + representation được chốt |
| Transaction | pgx, sqlc, FK/UNIQUE, locks | Concurrent writes, quyền sở hữu | DB integration test với failure/negative |
| Async & event | Asynq delivery/retry, Redis, SMTP | Notification hoặc nghiệp vụ hậu giao dịch | Delivery guarantee và replay/idempotency cụ thể |
| Resource lifecycle | Storage/cache/scheduler (nếu cần) | Listing media, expiry, materialization | Owner, invalidation, cleanup và recovery đã xác định |
| Operations | CI, disposable integration, WSL G0 | End-to-end use case | Evidence level được ghi đúng; không đánh đồng CI với production |

## 5. Acceptance cho việc bổ sung workbook

- **Đã làm:** ghi catalog, định nghĩa ranh giới, áp dụng một capability thực vào cả FA/FB, định nghĩa ví dụ tính toán tương lai.
- **Không làm:** không đổi production Go/SQL, migrations, job retry policy, precision, API contract, composer/vendor dependencies hoặc runtime G0.
- **Review docs:** mọi mệnh đề mô tả hiện trạng phải đối chiếu source; khi source đổi, cập nhật workbook hoặc ghi rõ baseline.
- **Nghiệm thu code:** tuân theo [ENG-GOV-01](development-contract.md); mức Operational yêu cầu evidence runtime thực, không do workbook tự cấp.
