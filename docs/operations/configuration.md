# BDSPro configuration — trạng thái Web + migrate

Source: `modules/setting` (bootstrap, parser, loaders, canonical publisher),
`cmd/main.go` (CLI flags và error reporting). Đơn vị được phép khởi tạo DB/Redis là
`routers/init.go` và các resource owners, **không phải** `modules/setting`.

## Đường dẫn và thứ tự ưu tiên

- BDSPro dùng `os.Getwd()` làm work path mặc định, thay vì binary directory của Gitea.
- CLI `--work-path` bắt buộc tuyệt đối. `--config` tương đối được tính theo shell CWD.
- Config mặc định: `<work-path>/custom/conf/app.ini`.
- Sau khi đọc INI và env, default-section `WORK_PATH` (nếu có) thành effective `AppWorkPath`.
- CLI `--work-path` và INI/env `WORK_PATH` trỏ đến hai thư mục khác nhau ⇒ từ chối bootstrap.
  Hai đường dẫn trỏ cùng một directory qua symlink được xem là khớp nếu cả hai tồn tại.
- `CustomPath` được **tính bằng** `<effective-work-path>/custom`, không có cờ `--custom-path`.
  Nó là thư mục instance tùy chỉnh, KHÔNG phải tên file config.
- `CustomConf` **không được resolve lại** sau khi đọc `WORK_PATH`; phải biết đường dẫn
  config trước khi đọc được chính cấu hình ấy.
- `MIGRATION_URL = file://db/migration` được quy đổi sang `file://<AppWorkPath>/db/migration`.
  Nếu SQL migrations nằm ngoài instance root, truyền absolute path, ví dụ
  `MIGRATION_URL = file:///app/db/migration` trong Docker. Bản Dockerfile hiện
  COPY migrations tới `/app/db/migration` nhưng WORK_PATH là `/data/bdspro`.
- Không gọi `os.Chdir()`. Không ghi đè/apply INI vào file vật lý.

**Docker lưu ý:** `docker/setup` chỉ tạo `/data/bdspro/conf/app.ini` nếu chưa có.
Sửa Docker template không sửa config đã được persist trong Docker volume. Trên volume cũ,
chỉnh `MIGRATION_URL = file:///app/db/migration` ở file đã tồn tại trước khi dùng CLI migrate.

## Environment (in-memory overrides)

Ví dụ:

```sh
BDS____RUN_MODE=development go run . web
BDS____WORK_PATH=/srv/bdspro go run . --config /etc/bdspro/app.ini web
BDS__LOG__LEVEL=debug go run . web
BDS__LOG_0X2E_FILE__FILE_NAME=server.log go run . web
BDS__SECURITY__TOKEN_SYMMETRIC_KEY__FILE=/run/secrets/jwt_key go run . web
```

`BDS__SECTION__KEY` được apply trong RAM sau INI, trước validation.
`BDS____KEY` áp dụng default INI section. `__FILE` đọc secret từ filesystem và
bỏ đúng một newline cuối. Không ghi hoặc in nội dung secret. Duplicate override,
malformed names, secret file đọc lỗi đều dừng startup.

Đừng nhầm với Gitea Docker `environment-to-ini` có thể viết file cấu hình bền vững.

## Giá trị config có consumer runtime

| Section | Fields | Consumer |
| --- | --- | --- |
| default | `RUN_MODE`, `WORK_PATH` | startup, graceful/log path |
| `[log]`, `[log.file]` | `MODE`, `FORMAT`, `LEVEL`, `ROOT_PATH`, `FILE_NAME` | `modules/log` |
| `[server]` | listener, public origin, trusted proxies, shutdown | `cmd/web`, `routers`, `modules/graceful` |
| `[api]` | response/page/body limits | `services/context`, API |
| `[database]` | `DB_SOURCE`, `MIGRATION_URL` | `models/db`, CLI migrate |
| `[redis]` | `ADDR` | `modules/conn` |
| `[security]` | signing key, access/refresh TTL | `modules/auth` |
| `[ipactive]` | override chính sách IP | `routers.NormalHandler()` |
| `[cors]` | `ALLOWED_ORIGINS` | `setting.Server` |

Đã bỏ các loader/file không còn consumer setting hiện tại: `oauth2`, `worker`,
`queue`, `phone`, `mailer`, `audit`, `auth` projection. Không giữ placeholder global.
Worker CLI cùng queue/service worker modules được loại bỏ ở gói này để `go test ./...`
không còn phụ thuộc vào `setting.Queue` đã bị xóa.

## CLI và lỗi

- `errors.New` khi lỗi cố định, không tham số động.
- `fmt.Errorf("giai đoạn: %w", err)` khi có underlying error phải được giữ qua `errors.Is`.
- `fmt.Errorf("...%q", input)` khi lỗi cần tên file/flag nhưng không có cause.
- Loader **return error**, không `log.Fatal`, không `os.Exit`.
- `RunMainApp` báo một lần và exit code `1` cho lỗi command, phân biệt usage error.
- `main.go` đang sở hữu `cli.OsExiter` để đóng logger trước `os.Exit`.

## Canonical process startup và migration

BDSPro dùng một nguồn cấu hình hiệu lực cho mỗi process, theo tinh thần
Gitea. `setting.LoadWebSettings(args,envs)` gọi **một lần**
`LoadWebConfig` để parse/validate, rồi publish chính các
`setting.Common/Server/API/Security/Database/Redis/IPActive` globals.
`routers`, `models/db`, `modules/conn/auth/graceful` đọc từ những giá trị
đã publish, không tự đọc file và không còn truyền `WebConfig` qua nhiều tầng.
`db.Connect(ctx, source)` và `RedisManager.Client(ctx,address)` vẫn là
primitive nhận tham số có thể dùng độc lập.

`setting.LoadMigrateSettings` dùng chung parser và publish **chỉ**
Paths/Common/Database; không cần Redis, JWT hoặc Web handler.
`config check` dùng `LoadWebConfig` thuần, không publish globals, không mở
resource hoặc log writer. INI, ENV và file secret không bị ghi ngược ra đĩa.

`WebConfig`/`MigrateConfig` chỉ tồn tại như kết quả typed ngắn hạn trong
hàm loader. Không có hai global runtime authorities; publication chỉ được
phép một lần trong process. Những globals đã công bố không được mutate trong
request handling/hot reload. Không in DB source/JWT key ra log.

**Lifecycle:** DB/Redis vẫn tự quản lý connection và idempotent Close. Startup
rollback và Graceful termination có thể cùng gọi Close; không thay đổi
thuật toán shutdown chỉ vì thay đổi cách đọc cấu hình.

Docker Compose đặt `stop_grace_period: 30s` cho hammer mặc định 15s. Nếu
cấu hình hammer lớn hơn, phải tăng Docker stop deadline tương ứng.
`HTTP_SHUTDOWN_TIMEOUT` giới hạn `http.Server.Shutdown`; khi hết hạn
`http.Server.Close` đóng connections, không bảo đảm mọi handler không
hợp tác đã kết thúc. Xem
[Canonical ownership](../architecture/canonical-settings-and-ownership.md)
và [Bootstrap lifecycle](../architecture/bootstrap.md).

## Preflight không mở tài nguyên

Chạy `bdspro --config /absolute/path/app.ini config check` để kiểm tra cấu hình Web:
đọc INI, ENV/`__FILE`, xác định work path, kiểm tra known section/key và các Web loaders.
Không mở DB/Redis/listener, không khởi tạo log manager hay mutate package-level settings.
Kết quả thành công: `Configuration valid`. Đây là static check, **không xác nhận**
database/Redis sẵn sàng hay port trống.

Từ thay đổi này, unknown INI section/key và unknown `BDS__...` override đều bị từ chối.
Rà soát config đang nằm trong Docker volumes trước khi nâng cấp và loại bỏ keys cũ
không còn consumer. `WORK_PATH` trong INI vẫn tương thích nhưng không giúp định vị
INI ban đầu. Host ENV không tự truyền vào container: khai báo bằng Compose
`environment`, `env_file` hoặc secrets đúng nơi. Không in secret trong diagnostics.

## Lệnh vận hành

```sh
make config-local
go run . --config "$PWD/custom/conf/app.ini" web
# Migrate: dùng config hiện hành, cần PostgreSQL thật.
go run . --config "$PWD/custom/conf/app.ini" migrate
# Linux production: truyền rõ đường dẫn config.
./bin/bdspro --config /data/bdspro/conf/app.ini web
```

`--help`/`--version` không cần config, DB, Redis. Khi Web chạy thật, Postgres và
Redis vẫn phải sẵn sàng; settings package không giả lập resource readiness.
