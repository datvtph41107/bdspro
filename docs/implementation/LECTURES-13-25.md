# BDSPro — Lecture #13 đến #25: triển khai theo năng lực thực tế

Ngày bắt đầu: 2026-10-17. Các bài tham chiếu từ [TECH SCHOOL SimpleBank](https://github.com/techschool/simplebank#backend-course-videos-golang). Đây là tài liệu triển khai ứng dụng BDSPro, không phải source clone SimpleBank.

## Ma trận từng lecture

| Lecture | Nội dung gốc | Quyết định BDSPro | Bằng chứng / file |
|---|---|---|---|
| #13 | Mock DB để test HTTP | Mock interface Authentication tại consumer Gin; mở rộng các nhánh login, verify, session, service lỗi | `api/auth_contract_test.go` |
| #14 | Transfer API + custom validator | **Không tạo chuyển tiền giả**. Kiểm chứng validator `PositiveID`; amount/currency/decimal domain Source đã được validate trước SQL | `auth/positive_id_test.go`, `source/service.go`, `api/source.go` |
| #15 | Users table, unique, FK | **Giữ** `persons` + `accounts` + unique email và FK hiện hành, không tạo bảng `users` trùng lặp | `db/migration/000001_*`, `000002_*`, SQLC DB tests |
| #16 | DB error handling | Không phân loại mọi unique-violation là trùng email; phân biệt invalid session và lỗi DB; preserve rollback error chain | `auth/service.go`, `api/auth.go`, `db/sqlc/store.go` |
| #17 | Bcrypt passwords | **Giữ Argon2id** đã triển khai; kiểm chứng đúng/sai password, invalid encoding và salt độc lập | `auth/password_test.go` |
| #18 | gomock matcher | **Dùng behavior stub** thay vì framework mock mới: test đúng token, số lần gọi và error response | `api/auth_contract_test.go` |
| #19 | JWT vs PASETO | Đánh giá lựa chọn, giữ opaque random token + SHA256 hash DB vì cần revoke ngay | `auth/service.go`, `db/query/session.sql` |
| #20 | JWT/PASETO maker | Không sinh token ký thứ hai khi current opaque token đã chứng minh tính ngẫu nhiên, expire, revoke | `auth/password_test.go`, session SQLC tests |
| #21 | Login API trả access token | Login trả cookie opaque session; không trả raw bearer trong JSON | `api/auth.go`, `api/auth_contract_test.go` |
| #22 | Authentication middleware | Gin `requireSession` cho protected routes; cache account ID trong request, tenant authorization vẫn ở Service/SQL | `api/server.go`, `api/auth.go` |
| #23 | Multistage Docker image | Builder Go riêng, runtime Alpine non-root, không copy code/secrets hay migration tool vào API image | `Dockerfile`, `.dockerignore` |
| #24 | Docker network | Compose default network resolve service name `db`; không expose Postgres host port | `compose.yaml` |
| #25 | Docker Compose + startup order | PostgreSQL healthcheck → one-shot migrate → API; không cần `wait-for.sh` vì Compose conditions | `compose.yaml`, `.github/workflows/ci.yml` |

### Kiến trúc và trách nhiệm

- `main.go` giữ wiring một `pgxpool.Pool`.
- `db/sqlc/Store.ExecTx` giữ atomic boundary, `RegisterTx` không đổi.
- `api/Server.requireSession` chứng minh identity; không có quyền Organization mặc định. Quyền và tenant scope vẫn do SQL có điều kiện/row locks bảo vệ.
- 401 khi session không hợp lệ; 503 khi không kiểm tra được vì lỗi hạ tầng. Không báo "session expired" khi DB mất kết nối.
- API image **không tự chạy migration** trong process HTTP: Compose one-shot container nắm trách nhiệm nâng schema.
- Cố ý không thêm JWT, PASETO, bcrypt, gomock, `users` table hoặc mô hình chuyển tiền: thiếu business pressure tương ứng.

## Khởi động Docker Compose trên local mới

Yêu cầu Docker Engine và `docker compose` plugin. Tại root của checkout:

```sh
docker compose config --quiet
docker compose up --build -d
docker compose ps
curl -i http://127.0.0.1:18080/health
curl -i http://127.0.0.1:18080/ready
docker compose logs --tail=100 migrate api
```

Expected: `migrate` kết thúc code 0; `api` trả HTTP 200 cho `/ready` khi kết nối Postgres được. Local app address là `127.0.0.1:18080`; DB chỉ có DNS nội bộ `db:5432`. Với compose project, DB password mặc định `secret` **chỉ dành cho local lab**, có thể override bằng `BDS_LOCAL_DB_PASSWORD` *trước khi tạo volume lần đầu*; đổi mật khẩu env sau khi volume đã init **không** cập nhật password Postgres tự động.

### Cảnh báo dữ liệu / bảo mật quan trọng

- Compose trên đây **chỉ để phát triển trên laptop**. `ALLOW_CONTAINER_DEV_BIND=true` cố ý nới điều kiện bind bên trong container, trong khi host chỉ publish loopback. Không chạy compose này trên server public, không attach network không đáng tin, không dùng tài khoản hoặc dữ liệu thật. Ở development, đăng ký trả verification token để debug.
- `docker compose down` dừng containers nhưng **giữ named volume**. Không chạy `docker compose down -v` khi chưa chủ động chấp nhận xóa toàn bộ dữ liệu LAB.
- Không trỏ `DB_SOURCE` hoặc migration vào database local đang giữ dữ liệu thật. **Không chỉnh sửa migration lịch sử**.
- Compose không là production deployment: thiếu secret management, TLS reverse proxy, external SMTP, backup/restore, rate limits, auditing, load tests.

### Vì sao không dùng wait-for.sh?

`depends_on: service_healthy` bảo đảm Postgres đã pass healthcheck trước khi chạy migration; `depends_on: service_completed_successfully` chờ migrations trả code 0 rồi mới khởi động API. Một TCP port mở không chứng minh DB đã có schema, nên chờ mỗi socket bằng `wait-for.sh` là không đủ.

## Nghiệm thu và giới hạn

CI hiện hành chạy SQLC generate + diff, migrations, tests, race test, startup smoke; nhánh này bổ sung `docker compose config --quiet` và smoke Compose GET /ready. Chỉ đánh dấu Docker PASS khi workflow branch/PR có kết quả completed/success. Smoke không chứng minh e2e mail, tải cao hay production security.

**Local sync:** chỉ `git pull --ff-only origin dev` sau khi PR đã merge và local sạch. Nếu còn thay đổi local, review diff và bảo toàn công việc trước.
