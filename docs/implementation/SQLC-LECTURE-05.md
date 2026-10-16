# Lecture #5 — Test SQLC CRUD với PostgreSQL (BDSPro)

**Mốc nguồn:** `dev` / `9357e8c` (bắt đầu Lecture #5). Lấy ý tưởng từ TechSchool SimpleBank: `TestMain`, `testing`, `testify/require`, helper dữ liệu, gọi trực tiếp queries do SQLC sinh ra.

## Phạm vi

- `db/sqlc/main_test.go`: kết nối PostgreSQL khi có `SQLC_TEST_DATABASE_URL`, xác nhận DB đúng, mở transaction cho từng test, rollback sau khi test chạy.
- `db/sqlc/party_test.go`: tạo Organization Party/subtype và chứng minh composite FK ngăn ghép sai subtype.
- `db/sqlc/account_test.go`: Party → Person → Account, email primary, unique login key, password credential và account đóng.
- `db/sqlc/email_verification_test.go`: token mới, tiêu thụ một lần, token hết hạn, token không tồn tại.
- `db/sqlc/session_test.go`: phiên đang hoạt động, revoke, hết hạn, account đóng, token hash trùng.
- `db/sqlc/organization_test.go`: onboarding, reviewer, claim, membership, roles và invitation, `:execrows`.
- `db/sqlc/source_test.go`: source intake, price report, replay/idempotency, tenant authority.
- `util/random.go`: test fixture strings/emails; tuyệt đối không dùng cho security token.
- `.github/workflows/bdspro-core.yml`: chạy SQLC tests **thật** trong CI bằng `bdspro_sqlc_ci`; không skip.

## Không thay đổi

- Không thay đổi `migrations/`, các SQL query trong `db/query/`, generated code hiện có, service hoặc HTTP.
- Không thêm helper production hoặc CRUD không có nghiệp vụ chỉ để viết test.
- Không `TRUNCATE`, `DELETE`, `COMMIT`, `dropdb`, `migratedown` vào database `bdspro`.
- Các kiểm tra deferred constraint lúc `COMMIT` không được chứng minh bằng rollback tests. Bộ integration CI gốc kiểm tra phần đó.

## Lệnh local

```sh
cd ~/projects/bdspro-core-work
# Bản vá chứa các file trong archive, chép vào root repository, không chép trong thư mục con.
go mod tidy
gofmt -w db/sqlc/*_test.go util/random.go
SQLC_TEST_DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro?sslmode=disable' go test ./db/sqlc -v -count=1
SQLC_TEST_DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro?sslmode=disable' go test ./db/sqlc -count=1 -cover
# Chạy các packages khác mà không khai báo CORE_TEST_DATABASE_URL / SOURCE_TEST_DATABASE_URL trỏ vào DB dev.
go test ./... -count=1
go vet ./...
git diff --check
```

**Database test:** local chọn đúng `bdspro` theo yêu cầu của người học. Chỉ dùng một database do người học kiểm soát, không dùng production. Tất cả test SQLC viết có rollback; sequence/identity có thể tăng dù transaction rollback.

## CI

Workflow trên `dev` tạo 3 DB thử nghiệm độc lập: `bdspro_core_ci`, `bdspro_source_ci`, `bdspro_sqlc_ci`, apply migrations và chạy `go test ./... -count=1 -v`. Xác nhận trong log rằng `db/sqlc` hiện kết quả `=== RUN`, không phải `no test files` hoặc chỉ skip.

## Quy tắc PASS

- SQLC package tests chạy **PASS thực sự** trên local, không phải SKIP.
- `go test ./...`, `go vet ./...`, `git diff --check` PASS.
- CI `dev` PASS với `bdspro_sqlc_ci` và các test case mới.
- Commit thuộc sở hữu người học; tự review `git status` và tự push `origin/dev`.
