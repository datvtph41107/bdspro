# Vòng đời cấu hình và khởi tạo BDSPro

BDSPro học cơ chế tổ chức source của Gitea, không sao chép nghiệp vụ Git. Mục tiêu là mỗi cấu hình có một owner, mỗi subsystem được khởi tạo đúng thời điểm và mỗi tài nguyên được đóng ở đúng runtime.

## Call graph

`cmd/main.go` khai báo command và global flags. `cmd/config.go` chuẩn bị đường dẫn, đọc `app.ini`, áp dụng `BDS__SECTION__KEY` và khởi tạo logger. Các domain loader thuộc `modules/setting` cùng đọc một `CfgProvider`, không tự mở INI lần nữa.

`routers.InitWebInstalled(ctx)` điều phối Web startup: nạp settings → PostgreSQL → migration ở môi trường development → Redis → auth/OAuth/HTTP services. `routers.NormalRoutes()` là owner đăng ký routes. `common.ProtocolMiddlewares()` chạy HTTP-native `RequestID()` qua `BeforeRouting`, trước Gin route matching; access log và API recovery vẫn chạy trong Gin vì cần matched route/APIContext. `NewServer()` hoàn thiện ingress qua `WrapIngress()` trước readiness; `cmd/http.go` sở hữu listener và graceful shutdown. `Server.Close()` giải phóng tài nguyên sau khi HTTP đã drain.

`migrate` và `doctor` chỉ cần database; `cleanup` cần thêm audit retention. `worker` dùng database, Redis, queue, signing key, mailer, audit retention, không khởi tạo HTTP/OIDC.

## Nguyên tắc phát triển

Từng file có counterpart và trách nhiệm rõ ràng: `database.go`, `server.go`, `security.go`, `oauth2.go`, `redis.go`, `queue.go`, `audit.go`, `mailer.go`, `phone.go`, `ipactive.go`. Chỉ thêm hàm khi có consumer; khi cần mở rộng, quay lại owner và cập nhật call graph. BDSPro giữ các invariant riêng về signing key, trusted proxies, OAuth2, audit và giới hạn queue.

`setting.Config` là kiểu dữ liệu đầu vào cho các HTTP service hiện hữu; `WebConfig()` chiếu từ domain settings đã nạp, không đọc lại hoặc lưu thêm một bản cấu hình global.

Tham chiếu kỹ thuật: [Gitea cmd/main.go](https://github.com/go-gitea/gitea/blob/main/cmd/main.go), [Gitea modules/setting/setting.go](https://github.com/go-gitea/gitea/blob/main/modules/setting/setting.go), [Gitea routers/init.go](https://github.com/go-gitea/gitea/blob/main/routers/init.go).
