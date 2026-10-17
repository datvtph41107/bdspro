> **Cập nhật sau Lecture #12 (17/10/2026):** BDSPro thống nhất sáu transaction của Auth, Organization và Source với `db/sqlc/Store.ExecTx`. Đây là bản export của helper `execTx` theo SimpleBank, vì các service ở package khác cần truy cập. Callback nhận `*dbsqlc.Queries` đã gắn vào `pgx.Tx`; business rules vẫn ở service, truy vấn vẫn ở SQLC. Các đoạn mô tả `execTx` và HTTP `net/http` bên dưới là lịch sử giai đoạn trước Lecture #11, không phải trạng thái source hiện tại. Các unit/integration tests của giai đoạn tiếp theo cần xác nhận rollback, locks, quyền và idempotency. Không thay đổi schema.\n\n# Chốt giai đoạn trước Lecture #11 — SimpleBank-aligned BDSPro

Mốc tham chiếu: SimpleBank `56a8785` (Store/TransferTx); `b383211` (CI khởi tạo); `d670f28` (PostgreSQL CI).

## Cấu trúc theo bài học

- `db/migration/`: vị trí theo SimpleBank; chuyển 12 file .up/.down cũ **không thay đổi nội dung hay version**. Không chạy lại migrations đã áp dụng chỉ vì di chuyển file.
- `db/query/`, `db/sqlc/`, `sqlc.yaml`: SQLC một nguồn, một output.
- `db/sqlc/store.go`: `Store`, `NewStore`, `execTx` tự thực hiện `Begin → New(tx) → callback → Rollback/Commit`, theo tác giả SimpleBank. `RegisterTx` là transaction nghiệp vụ thật của BDSPro, tương ứng vai trò `TransferTx` chứ không sao chép chuyển tiền giả.
- `db/sqlc/store_test.go`: kiểm thử COMMIT, rollback khi trùng email, cạnh tranh 5 đăng ký và quan sát isolation READ COMMITTED/REPEATABLE READ.
- `integration_test.go`: thêm hai lời duyệt cùng `OrganizationRegistration` đồng thời; khóa `FOR UPDATE` chỉ cho một lần phê duyệt, lần thứ hai không tạo tổ chức thứ hai.
- Bỏ các file demo `locking_test.go`, `isolation_test.go` tách rời; isolation test được chuyển về `store_test.go`, locking được chứng minh ở transaction tổ chức thật.
- `.github/workflows/ci.yml`: tên `ci-test`, job `test`, chạy khi push/PR `dev`; PostgreSQL 18, `make migrateup`, `make sqlc`, `make test`, race tests và smoke startup.
- `Makefile`: các target giống workflow SimpleBank; DB_URL bắt buộc để giảm nguy cơ migrate nhầm database.

## Khác biệt có chủ đích và giới hạn

- SimpleBank dùng `database/sql`; BDSPro đang dùng `pgx/v5`, nhưng transaction flow được trình bày cùng thứ tự.
- BDSPro chưa có chuyển tiền, số dư hoặc nghiệp vụ cần khóa hai Account đối nghịch. Không dùng `TransferTx` hay `AddAccountBalance` giả. Nghiệp vụ khóa hiện tại chỉ bảo vệ duyệt một hồ sơ cùng lúc; **không chứng minh mọi luồng nghiệp vụ không thể deadlock**.
- `migratedown` ở BDSPro giữ giới hạn DOWN 1 để bảo vệ dữ liệu, không sao chép thao tác hủy toàn bộ schema mặc định từ bài giảng.
- COMMIT tests chỉ chạy với database được chủ động cung cấp đúng tên `bdspro_sqlc_ci`. `bdspro_core_ci` và `bdspro_source_ci` dùng test tích hợp có TRUNCATE. Không hướng các biến CI vào database phát triển.
- HTTP `net/http` hiện tại được giữ như nghiệp vụ đã tồn tại; **Lecture Gin chưa được triển khai**. Không đổi route hay bảo mật HTTP trong commit này.

## Chạy kiểm chứng

```sh
make sqlc
make migrateup DB_URL='postgresql://root:secret@localhost:5432/bdspro_sqlc_ci?sslmode=disable'
SQLC_TEST_DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro_sqlc_ci?sslmode=disable' make testsqlc
SQLC_TEST_DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro_sqlc_ci?sslmode=disable' make testsqlc-race
```

Các lệnh trên yêu cầu một PostgreSQL local và database test đã được chủ động tạo. Nếu schema đã ở version 6, `migrateup` không tạo bảng lần nữa.

## Đồng bộ local

Giữ nguyên các thay đổi chưa commit ở `source/service.go`, `db/sqlc/transaction_test.go` và stash trước khi pull; không reset hard, clean hay pop stash tự động. Sau khi `dev` đồng bộ và CI PASS, mốc tiếp theo là Lecture #11 — Gin HTTP API.
