# BDSPro — CH01 / Configuration, Canonical Truth & Developer Workflow

**Status:** READY FOR REVIEW, NOT IMPLEMENTED / NO FILES ADDED.
**Trách nhiệm:** developer quyết định configuration contract; runtime nhận và validate; operator cung cấp giá trị theo môi trường; CI kiểm tra hành vi. **NEXT FIRST ACTION IN NEW CHAT**.

## 1. Tại sao CH01 là điểm bắt đầu

Developer muốn biết ngay: chạy project thế nào; DB nào được dùng; config lấy từ đâu; test có độc lập production không; command nào chỉ đọc và command nào thay đổi trạng thái; khi deploy thì secrets, timeout, startup dependencies và migrations ai sở hữu.

**Single source of truth phải theo concern**, không có "một file duy nhất" cho toàn bộ project. Cần xác định: cấu hình ý nghĩa và validation ở code, giá trị cấu hình tại môi trường triển khai, workflow commands, SQL DDL, code generation và CI checks — mỗi concern có authority riêng, không duplicate sự thật không cần thiết.

## 2. Bằng chứng từ BDSPro dev tại checkpoint

| Artifact / actor | Chức năng hiện tại đã xác minh | Authority và điều KHÔNG tự suy ra |
| --- | --- | --- |
| `main.go` | `os.Getenv("DATABASE_URL"/"APP_ENV"/"BIND_HOST"/"PORT"/"SMTP_..."/"ALLOWED_ORIGIN")`; default 127.0.0.1:8080; dev-only loopback guard; outside-dev SMTP/Origin guard; Pool/Ping/HTTP/shutdown | Đây là source **consumer/validation** của runtime config; không chứng minh nơi deploy secrets được quản lý |
| `.gitignore` | Có `.env` | Chỉ là ignore rule; **không có tracked `.env` trong tree**; không suy ra local không có config |
| `Makefile` | `DB_URL ?=` rỗng; `sqlc`/`test`/`newmigration`/`migrateup`/`migratedown`, migration targets yêu cầu `DB_URL` | Make là **command interface** chứ không tự là runtime config authority; DB_URL CLI != DATABASE_URL app |
| `sqlc.source.yaml` | `schema: "migrations"`, `queries: "db/source/query"`, generated `db/sourcesqlc` | YAML dành cho **sqlc generator**, không phải app runtime config |
| `.github/workflows/bdspro-core.yml` | PG18 service, CI DB env URLs, migrations, sqlc, Go vet/test, smoke startup | **CI/test execution contract**, không đại diện cho production runtime secrets |
| `go.mod`/`go.sum` | Go module + dependencies, Go version | Dependency/toolchain authority, không phải env settings |
| `migrations/*.sql` + DB `schema_migrations` | DDL lịch sử với tool migrate; actual DB migrations state chỉ có trên DB live | Repo migrations là intended versioned schema; **actual DB state** cần query để xác nhận |
| `README.md` / continuity | Hướng dẫn, lịch sử, recovery, cảnh báo vận hành | Docs là interface hiểu/cách làm, không thay thế code/DB/CI live |
| `.sh`/Docker Compose/`.env.example` | Không có tracked file trong Git tree tại thời điểm rà soát | Không coi vắng mặt là thiếu sót cho đến khi xác minh task/need |

**Ràng buộc quan trọng:** runtime `DATABASE_URL` vs Make `DB_URL` là một phân biệt có ý nghĩa về scope. Có thể xem xét tên thống nhất/adapter nhưng không rename mù. `make migrateup` mutate DB; `make migratedown` có thể hủy dữ liệu; `make test` thực thi tests với side effects trong DB thử nghiệm nếu env trỏ sai — luôn xác minh môi trường.

## 3. TechSchool SimpleBank làm đối chứng, KHÔNG là spec

Đã kiểm tra `techschool/simplebank` nhánh `master`: `app.env`, `util/config.go` (typed Config + Viper, file/env), `Makefile` (các target DB/migrate/sqlc/test/server), `sqlc.yaml`, `docker-compose.yaml`, `start.sh` (set -e và exec "$@"), `wait-for.sh` (chờ dependency), `main.go` (load Config, setup dependencies, DB migrations, servers/workers).

**Chuyển thành câu hỏi chứ không chuyển thành yêu cầu copy:**
- `app.env`: file ví dụ hay file chứa secret? Trường hợp nào được commit? Config values nào là demo only?
- `util/config.go`: typed Config tăng khả năng validate/IDE/documentation ra sao? Khi nào đáng dùng Viper so với Go stdlib?
- `Makefile`: gọi các tác vụ lặp lại; khi nào lệnh nên đưa vào Make target để developer không phải nhớ tham số?
- `start.sh`: dùng `exec` để process nhận signals; bdspro hiện chạy server trực tiếp và đã có shutdown; chưa cần shell script nếu không có orchestration thật.
- `wait-for.sh` và Compose: hạ tầng dev reproducible vs application `Ping`; startup ordering ≠ readiness; chống race như thế nào?
- Workflow deploy SimpleBank không là bằng chứng BDSPro cần ngay Redis/gRPC/EKS/Viper.

## 4. Canonicalization — bảng owner dự thảo

| Concern | Candidate canonical truth | Consumer / derived / override | Owner / nơi xác minh | Cần chốt |
| --- | --- | --- | --- | --- |
| Config schema + validation | Một contract trong Go (hiện phân tán ở `run()`) | ENV của process, scripts/dev tooling | App code owner; unit+startup test | Typed `Config` có đáng? key/default/required/type |
| Actual dev/test/prod settings | Runtime environment / deployment secrets/config service **TBD** | Local `.env` tùy chọn nếu chọn loader, CI env | Operator + developer | Precedence, no secrets in git, env promotion |
| CLI task names & safety | `Makefile` | Shell/local CI commands | Developer workflow owner | Target naming, help, read/write/destructive labels |
| Generator settings | `sqlc.source.yaml` | Generated code output | Data/codegen owner | Khi nào thêm sqlc cho Auth/Org |
| DB intended schema | Versioned `migrations/*.sql` | Local/CI/prod DB instances | Schema owner | Schema drift/forward-only/rollback |
| Actual DB schema/migration version | Chính database đang chạy | SQL queries và tests | DB/operator | Truy vấn `schema_migrations` trước mutation |
| Test acceptance | Go tests + CI workflow | Logs/artifacts/coverage | Reviewer/CI | Failure/fault coverage và DB isolation |
| Runtime process | `main.go`/`run()` | Compose/systemd/K8s/entrypoint (chưa xác minh) | Runtime/operator | Startup, readiness, signals, shutdown |

**Canonical không nhất thiết có nghĩa file trung tâm:** có thể chỉ là một struct Config ở Go và ENV values ở deployment; một Makefile quy định developer command; YAML cho sqlc. Tách rõ declaration/validation, provisioning, generated material và effective value.

## 5. Decisions phải giải quyết từng phần

1. **Environment vocabulary:** development/test/staging/production có nghĩa gì; `APP_ENV` được dùng ở đâu, và policy production có bị vô hiệu do typo/fallback không?
2. **Config input:** ENV trực tiếp, `.env` local, `.env.example`, config YAML, typed struct; thứ tự ưu tiên và error reporting; nếu bổ sung external library thì lý do.
3. **Secrets:** mật khẩu PostgreSQL/SMTP/session-related secrets: ai cấp, phân phối, rotate, redact; không đưa vào file tracked, Makefile hoặc CI logs.
4. **Local workflow:** từ clean clone đến run/test/sqlc/migrate; commands nào an toàn, commands nào mutate; preconditions với DB instance và branch.
5. **DB target:** khác biệt `DATABASE_URL` vs `DB_URL`, database names, schema migration target, smoke start, least privilege.
6. **Bootstrap** (CH05): Load/validate config, dependency Ping/ready, server listen, shutdown; có cần `start.sh`/Compose hay không theo dev friction.
7. **CI and release** (CH18/19): đảm bảo codegen/tests/migrations đồng nhất; production deploy workflow là unknown, không dựng trước khi rõ.
8. **Documentation ownership:** One canonical index và các docs dẫn chiếu; tránh multiple "latest" sections mâu thuẫn.

## 6. Bài review đầu tiên của chat mới — đề xuất một vấn đề duy nhất

**Question CH01.1: "Một runtime config key được khai báo, lấy giá trị, kiểm tra và sử dụng ở đâu?"**

Dùng `DATABASE_URL`:
- `main.go`: đọc biến; kiểm tra không rỗng; truyền tới `pgxpool.New`, `Ping`; lúc nào lỗi startup?
- `Makefile`: `DB_URL` cho migration target; có phải cùng database khi user chạy local? Không có proof nếu chưa xem env.
- CI: URL dành cho test DB, startup smoke; giá trị ứng dụng và migration được cấp bằng cơ chế nào?
- SimpleBank: `Config` + `LoadConfig`; đối chứng về config struct/precedence, không copy tự động.

**Output sau review CH01.1:** một truth/ownership flow nhỏ cho DB connection config, các failure tests cần có, và decision KEEP vs trích xuất typed Config vs điều chỉnh docs/workflow — không viết code trước khi người dùng quyết định.

## 7. Phương pháp làm việc từng vấn đề

`Quan sát chính xác từ source → actor và state → current contract → failure/security/dev cost → dự đoán → user tự chạy thí nghiệm → evidence → 2–3 alternatives → quyết định tương xứng → PR/commit/checkpoint nếu được duyệt`.

Không hướng dẫn chỉnh `.env` chứa secrets trong chat, không in credential thật. Không tạo hoặc chạy destructive make targets trước khi DB target đã được xác nhận. Không hỏi lại những điều đã được ghi trong checkpoint.
