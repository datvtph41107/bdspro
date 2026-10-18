# Lecture #67 — PostgreSQL error handling (pgx)

BDSPro xử lý `pgconn.PgError` bằng **SQLSTATE + tên constraint** khi chuyển sang lỗi nghiệp vụ. Chỉ SQLSTATE `23505` không đủ cơ sở để trả 409: duplicate email và lỗi khóa duy nhất trong bảng khác có ý nghĩa khác nhau.

## Ranh giới đã triển khai

- `db/pgerrors` phân loại `PgError` bằng `errors.As` kể cả khi qua `fmt.Errorf("%w")` hoặc `errors.Join`.
- Auth `Register`: chỉ unique index `account_emails_login_key_uq` thành `ErrEmailUsed`.
- Organization `ApproveRegistration`: chỉ trùng registry identifier `organization_registry_identifiers_jurisdiction_code_registration_code_key` thành `ErrConflict`. Không nuốt mọi `23503` (FK violation) thành xung đột người dùng.
- Organization `AcceptInvitation`: các unique constraints của open membership, invitation origin, open role assignment được ánh xạ thành `ErrConflict`; lỗi constraint bất ngờ được trả lên để điều tra.
- `40001` serialization failure và `40P01` deadlock được nhận dạng **nhưng chưa tự động retry**: caller phải đảm bảo toàn bộ use case idempotent. Không retry COMMIT khi kết quả không xác định.
- `pgx.ErrNoRows` tiếp tục được ánh xạ có chủ đích theo use case (404/403, chống lộ quyền tenant).

## Điểm kiểm thử

`go test ./db/pgerrors -v` xác minh wrapped errors, joined errors, tên constraint sai, unique vs foreign key, SQLSTATE retryable, context cancellation.

Để nghiệm thu: CI migration UP-DOWN-UP, sqlc generate sạch, go test, race test và Compose smoke. Không thay schema, không thay `Store.ExecTx`, không thay Docker/AWS.
