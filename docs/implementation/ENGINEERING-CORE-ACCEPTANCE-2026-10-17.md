# BDSPro — Báo cáo nghiệm thu Engineering Core trước Lecture #13

**Ngày:** 2026-10-17. **Nhánh:** `dev`. **Phạm vi:** Auth, Gin boundary, SMTP, Organization invitation, SQLC, cấu hình/onboarding.  
**Thẩm quyền:** source và CI HEAD hiện tại là chuẩn kiểm tra; tài liệu này không xác nhận production-ready.

## 1. Cơ sở thực tế

- Baseline `4396fa95` chạy CI PASS nhưng HTTP Logout bỏ qua lỗi thu hồi session.
- `auth/smtp.go` trả lỗi `Write()` mà bỏ qua lỗi `Close()` ở DATA acknowledgment.
- `RegisterTx` có thể COMMIT trước khi SMTP từ chối; Account tồn tại nhưng người dùng không có con đường recovery.
- `AcceptPendingInvitation` chưa kiểm tra `organizations.archived_at`.
- Gin `Server` giữ một `Store` không dùng nhưng kiểm tra readiness qua `authentication.Pool`.
- Các test HTTP trước đây không có seam để giả lập các nhánh lỗi Service độc lập DB.
- Makefile `createdb` mặc định `bdspro_sqlc_ci` trong khi runtime `app.env` cần `bdspro`.

## 2. Các thay đổi được triển khai

| Nhóm | Thay đổi | Bằng chứng cần nghiệm thu |
| --- | --- | --- |
| Logout | HTTP trả 503 khi thu hồi session lỗi; không xóa cookie trong response lỗi; không cookie vẫn 204 | `TestLogoutHonorsRevocationFailure`, `TestAuthCookieLifecycle` |
| SMTP | Trả lỗi `DataWriter.Close()`; giới hạn SMTP I/O bằng deadline 15 giây (hoặc deadline ngắn hơn của context) | `TestWriteSMTPDataChecksFinalServerResponse` |
| Config | Ngoài development yêu cầu sender hợp lệ và verification URL HTTPS, kiểm tra trước khi nhận request | `TestLoadConfigValidation`, `TestLoadProductionConfig` |
| Recovery | `POST /auth/resend-verification`; 202 chung cho trạng thái tài khoản, 400 email lỗi; khóa theo account_email trước khi kiểm tra cooldown 60 giây và quota tối đa 6 token trong 24 giờ | `TestRegistrationDeliveryFailureCanRecover`, `TestConcurrentResendDoesNotBypassCooldown`, `TestResendDailyTokenCap`, HTTP stub tests |
| Organization | Từ chối nhận lời mời nếu Organization archived | `TestOrganizationMembershipAndInvitationQueries` |
| Gin | Consumer-defined `Authentication` interface, readiness dùng pool trực tiếp, bỏ Store field không dùng | `TestCreateAccountServiceOutcomes`, `TestServerHealthAndReadiness` |
| Onboarding | `DB_NAME` mặc định `bdspro`, hướng dẫn quickstart mới | `LOCAL-QUICKSTART.md` |

**Không thay:** sáu migrations, Party C01, `Store.ExecTx`, `Store.RegisterTx`, mô hình dữ liệu Auth/Organization/Source, transaction isolation.

## 3. Bất biến và hợp đồng

**Logout:** 204 chỉ khi không có session cookie hoặc backend chấp nhận revoke. Lỗi backend: 503. Không thể suy ra backend rollback khi connection mất sau UPDATE; retry phải idempotent.

**Register:** PostgreSQL COMMIT trước SMTP; email không thuộc transaction. Lỗi SMTP không xóa Account. Người dùng có thể yêu cầu gửi lại sau cooldown. Endpoint không trả trạng thái có/không có account.

**Resend:** Token sinh ngẫu nhiên, chỉ lưu hash, hết hạn 30 phút. SQL chỉ tạo token cho account chưa đóng và email chưa verified. Có khóa hàng account_email giữ qua transaction tới khi tạo token, nên nhiều requests đồng thời cùng địa chỉ được tuần tự hóa khi tính quota. Chưa có rate limit IP/global hoặc bảo vệ phân tích thời gian phản hồi (timing enumeration).

**Organization:** Invitation chỉ được chấp nhận khi chưa resolved, chưa hết hạn, account đang mở, email verified trùng và Organization chưa archived. SQL vẫn bảo vệ scope role/membership.

**Transaction:** Query thuộc giao dịch phải dùng callback `q` của `ExecTx`; không được gọi `s.store`/pool bên ngoài callback cho các ghi đòi hỏi nguyên tử. Không yêu cầu tạo `XxxTx` hay interface riêng cho mọi use case.

## 4. Giới hạn chưa được chứng minh — không được tuyên bố production-ready

- Chưa có per-IP/global rate limiting chống abuse tại public endpoints; resend chỉ giới hạn token theo email.
- Chưa có hệ thống delivery bền vững (outbox/queue) hay quan sát được giao hàng thực tế; SMTP 250 chỉ là máy chủ nhận thư chấp nhận.
- Chưa có giao thức đồng bộ thu hồi membership/role với các transactions đang chạy; SQL READ COMMITTED kiểm tra quyền theo statement snapshot.
- Chưa có API quản trị đầy đủ cho thu hồi Role/Membership; một số test thao tác SQL trực tiếp.
- Chưa xác minh tải thực, p95/p99, EXPLAIN/ANALYZE hay các điều kiện deployment Internet.
- Chưa nghiệm thu toàn bộ future schema như passkeys/OIDC/service accounts.
- CI green không chứng minh dữ liệu local được migrate hay server production đáp ứng mọi threat model.

## 5. Điều kiện nghiệm thu

Đối chiếu workflow `.github/workflows/ci.yml` tại commit cuối. CI cần chạy:

1. PostgreSQL 18, áp dụng migrations, kiểm tra UP-DOWN-UP migration cuối.
2. `sqlc generate`, gofmt, `go mod tidy`, `go vet`, diff generated không đổi.
3. `go test -v -cover ./...` với ba disposable PostgreSQL CI databases; không chấp nhận SKIP ngầm.
4. SQLC race tests, HTTP smoke startup.
5. Kiểm tra logs không có FAIL; trạng thái GitHub Actions completed/success.

## 6. Quyết định để tiếp tục Lecture #13

- Dùng interface nhỏ tại phía Gin cho **behavior của Auth**; không tạo `Store interface` toàn bộ chỉ vì muốn mock.
- Giữ Store concrete và `ExecTx` hiện tại; `NewStore(pool)` là wrapper rẻ, không mở pool mới.
- Lecture #13 tập trung mock các handler HTTP đúng lớp; Organization/Source interfaces chỉ xuất hiện khi có các test nghiệp vụ tương ứng cần cô lập.
- Không tạo framework, `UnitOfWork`, RepositoryFactory hoặc tái cấu trúc Party chỉ vì muốn nhất quán vẻ ngoài.

## 7. Hướng đồng bộ local

Xem [LOCAL-QUICKSTART.md](LOCAL-QUICKSTART.md). Kiểm tra trạng thái và chỉ fast-forward khi local sạch. Không tự động chạy DOWN migration, reset/drop database hay `git reset --hard`.

**Kết luận:** Baseline đủ để tiếp tục học Lecture #13 khi CI HEAD PASS; đây là nghiệm thu cho thay đổi trong phạm vi liệt kê, không phải giấy chứng nhận public production.
