
> **2026-11-25 governance reset:** This content is a historical snapshot/reference, not an active mandate or verified current runtime. Recheck live source and requirements. Do not resume Gitea/F40/parallel plans or infer authorization from it. Current policy: https://github.com/datvtph41107/bdspro/blob/dev/AGENTS.md (on the reset branch, use its local root AGENTS.md).

# Bằng chứng source Gitea → BDSPro refactor

**Gitea mốc:** `go-gitea/gitea@fbf986d1c7ba1b0ad0078105b3adb0a73b769295`.
**BDSPro được đối chiếu:** `datvtph41107/bdspro@c255b77d49a715cd057b4a572c2e1d1782e1da3d` (`dev` tại lúc lấy).
**ZIP:** code mục tiêu do refactor sinh ra, không phải source của Gitea; không tự nhận đã chạy trên local user.

| Gitea (source line) | BDSPro mục tiêu | Giống nhau về code | Khác có chủ đích |
| --- | --- | --- | --- |
| [`modules/setting/path.go:L86–94`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/path.go#L86-L94) | `modules/setting/path.go` | `stringWithDefault` và `.Set()` cùng implementation | CWD default, không dùng binary path |
| [`modules/setting/path.go:L103–176`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/path.go#L103-L176) | `resolvePaths()` | `tryAbsPath`, `tmpWorkPath`, `tmpCustomConf`, `readFromArgs`, `!IsSet` | Không có `--custom-path` hay `GITEA_CUSTOM` |
| [`modules/setting/path.go:L178–203`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/path.go#L178-L203) | `InitWorkPathAndCommonConfig()` | Đọc `WORK_PATH` từ default INI section, validate absolute, chuẩn hóa path | Conflict fail-fast; publish sau common validate; không có installer |
| [`modules/util/path.go:L206–217`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/util/path.go#L206-L217) | `modules/util/path.go` | `util.IsExist` dùng `os.Stat`, phân biệt missing vs I/O | Không khác |
| [`modules/setting/config_provider.go:L241–246`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/config_provider.go#L241-L246) | `configProviderLoadOptions()` | Cùng `KeyValueDelimiterOnWrite`, `IgnoreContinuation` | Ghi INI chưa có consumer |
| [`modules/setting/config_provider.go:L248–259`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/config_provider.go#L248-L259) | `NewConfigProviderFromData()` | `ini.LoadSources`, `strings.NewReader`, `ini.SnackCase` | Không `loadedFromEmpty` vì không Save() |
| [`modules/setting/config_provider.go:L261–286`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/config_provider.go#L261-L286) | `NewConfigProviderFromFile()` | `ini.Empty`, `util.IsExist`, `cfg.Append`, mapper | Missing INI -> `os.ErrNotExist`, không installer |
| [`modules/setting/config_provider.go:L135–160`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/config_provider.go#L135-L160) | `ConfigSectionKeyString()` | Default fallback, không mutate raw key khi đọc | BDSPro ConfigKey read-only + IsSet |
| [`modules/setting/setting.go:L116–126`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/setting.go#L116-L126) | `loadCommonSettingsFrom()` | `loadRunModeFrom`, `loadLogGlobalFrom` theo thứ tự | BDSPro trả typed values và validate trước publish |
| [`modules/setting/path.go:L96–100`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/setting/path.go#L96-L100) | `InitWorkPathAndCommonConfig()` | Init provider rồi common settings | BDSPro trả error và commit globals cuối |
| [`cmd/main.go:L51–64`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/cmd/main.go#L51-L64) | `cmd/main.go` | `PrepareSubcommandWithGlobalFlags`: closure và named returns | `prepareConfig` + configured logger đều có thể lỗi |
| [`cmd/main.go:L157–193`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/cmd/main.go#L157-L193) | `RunMainApp` | `usageErr`, CLI usage handler, exit 1 | Nil ErrWriter được kiểm tra |

**Thư viện đối chiếu cùng phiên bản:** commit Gitea có `github.com/urfave/cli/v3 v3.14.0`
và `gopkg.in/ini.v1 v1.67.3`; BDSPro `dev` cũng pin đúng hai phiên bản này
([Gitea go.mod](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/go.mod),
[BDSPro go.mod](https://github.com/datvtph41107/bdspro/blob/c255b77d49a715cd057b4a572c2e1d1782e1da3d/go.mod)).
Đây là bằng chứng tương ứng API thư viện, **không** chứng minh toàn bộ BDSPro đã build.

## Cơ chế có consumer thực

- `AppWorkPath` tác động relative `[log].ROOT_PATH`, ví dụ `/srv/bdspro/data/log`.
- `AppWorkPath` cũng tác động `file://` migrations path trong BDSPro target. Đây là BDSPro-specific extension được quyết định từ consumer `cmd/migrate.go`, không phải copy dòng code của Gitea.
  Tham khảo định dạng [golang-migrate file source](https://pkg.go.dev/github.com/golang-migrate/migrate/v4/source/file).
- `CustomPath` là `<effectiveWorkPath>/custom`; được dùng để dựng default config path **trước** khi đọc file,
  và được publish như root của các custom assets trong tương lai. Trong BDSPro hiện chưa có template/assets
  consumer như [`Gitea modules/templates/base.go`](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/templates/base.go#L11-L17).
- `CustomConf` phải giữ path của file đã nạp, không tự chuyển nếu `WORK_PATH` trong INI đổi.
- `CfgProvider` không expose INI parser và không cho domain code `Save`; env chỉ mutate provider riêng trước publication.
- `Server`, `API`, `Security`, `IPActive`, `Database`, `Redis` có consumer thật; `LoadSettings` publish tất cả cùng một phase sau khi validate.

## Phần KHÔNG tương đương Gitea

Gitea có installer, khả năng lưu config, `GITEA_WORK_DIR`, `GITEA_CUSTOM`, assets/templates,
`HasInstallLock`, `AppWorkPathMismatch` khi hai đường dẫn khác inode; package này không implement
những features đó vì chưa có consumer tương ứng. Nhận định 'giống hệt Gitea' về runtime là sai.
BDSPro cũng không copy `log.Fatal` của Gitea vào loader; errors được trả cho CLI.

## Ràng buộc nghiệm thu

`gofmt`, AST parse chỉ đảm bảo cú pháp; **không đủ** để khẳng định package build.
Phải chạy `go test ./modules/setting ./modules/util ./cmd` và `go test ./...`, rồi chạy binary thực.
Nếu local của bạn có các patch mới ngoài commit gốc, script cài đặt backup trước; xem `git diff` trước khi nhận kết quả.
