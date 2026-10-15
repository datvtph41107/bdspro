# BDSPro — Checkpoint tái định hướng Engineering Core
**Ngày:** 2026-10-15 (Asia/Bangkok)  
**Repository:** https://github.com/datvtph41107/bdspro · nhánh làm việc: dev  
**Loại mốc:** thay đổi phương pháp và trọng tâm review; chỉ tài liệu, không sửa Go/SQL/migrations/tests.  
**Trạng thái:** định hướng đã được người dùng làm rõ; các quy chuẩn cụ thể cần được kiểm chứng/chốt từng phần; không tuyên bố hoàn thành Engineering Core.  
**Prompt khôi phục:** [CONTINUE-PROMPT.md](CONTINUE-PROMPT.md)

> Mệnh đề cốt lõi: **Core hiện tại là Engineering Core — nền móng về cách thiết kế, tổ chức, viết, kiểm chứng, thay đổi và vận hành code cho developer; KHÔNG phải tiếp tục triển khai Business Core.** Dùng bdspro đang chạy được để khám phá cơ chế và áp lực thực tế; không tái cấu trúc chỉ vì một pattern/framework trông đẹp. Lợi ích phải đủ bù chi phí sở hữu; rủi ro đúng/sai, bảo mật, dữ liệu không cần đợi sự cố xảy ra mới bảo vệ.

## 1. Vì sao xuất hiện bước ngoặt

### Quá trình có thật
- Đã bắt đầu từ canonical data/Party–Person–Organization, C01 exactly-one subtype, migrations, FK, partial UNIQUE; sau đó đi vào Register/Login/VerifyEmail/Session, Organization membership/authorization, Source Intake, sqlc, HTTP và CI.
- Người dùng liên tục phê bình việc assistant chọn giải pháp trước khi lý giải cơ chế nền: thêm kind, PL/pgSQL assertion/deferred trigger, composite FK khi chưa chứng minh chi phí/lợi ích của độ phức tạp; user muốn xét lại C01 và có hướng **loại bỏ kind/trigger ở migration đầu**. Hướng mong muốn này không phải thay đổi đã áp dụng trên dev; live migrations vẫn phải đọc để xác nhận.
- Assistant có xu hướng tiến triển bằng "migration → service → handler → test PASS → module tiếp theo", sau khi có test thì coi như đủ sức mở Source/CRM. Người dùng đặt câu hỏi ngược: code vận hành được nhưng ai đã thống nhất file/package ownership, naming, Handler, Request/Response/Error, validation, transaction, SQL/sqlc, CI, runtime lifecycle, tác nhân vận hành và chi phí của mọi dev?
- User yêu cầu hiểu kỹ thuật từ Go trần trụi/standard library trước; chính người dùng thiết kế/viết code khi tới giai đoạn triển khai. Assistant chủ động tra cứu, gợi câu hỏi và phản chứng, không chỉ bày lab hoặc đem cả framework vào source.
- Sau khi khảo sát Laravel DRX, Gin/Echo và Uber Go Style Guide, nhận định thống nhất: **nguyên lý backend phổ quát** (trách nhiệm, trust boundary, side effects, correctness, error, transactions, observability) không đồng nghĩa hình thức framework; Go có idioms riêng. Framework chủ yếu đóng gói cơ chế/quy ước, đáng chọn khi tổng giá trị vượt chi phí phụ thuộc/di chuyển/bảo trì, không phải vì số API hay traffic tự động đạt ngưỡng.
- Người dùng nhấn mạnh bắt buộc nhìn ngoài thế giới: official docs, nghiên cứu, cộng đồng (kể cả phản biện), production GitHub repos, các framework; cần biết **mức ưu tiên chứng cứ**, so sánh trong cùng điều kiện và quay lại kiểm chứng bdspro.

### Trước / sau — thay đổi tư duy có thể kiểm tra
| Trước đây assistant thường ưu tiên | Từ checkpoint này phải ưu tiên |
| --- | --- |
| Đưa ra migration, API, service, testcase hoàn chỉnh | Hiểu cơ chế cơ bản, hợp đồng trách nhiệm, áp lực dẫn tới code trước khi mở rộng |
| Bảo vệ invariant tối đa bằng DB mechanism sớm | Xác định invariant, ai sở hữu, mức bảo đảm, phản chứng, chi phí; chọn cơ chế tối thiểu đủ an toàn |
| Copy/nương theo framework, pattern hoặc repo trưởng thành | Truy nguyên vấn đề đã khiến họ chọn giải pháp, đối chiếu bdspro, xem cả phản chứng |
| Mỗi module tự có helper HTTP/error/naming riêng | Xác định chính sách chung, ngoại lệ, rồi mới xét khả năng chia sẻ implementation |
| Test PASS là cửa qua module kế tiếp | Nêu test đã/chưa chứng minh, chất lượng structure, security, change impact và operation |
| Đề nghị tách file/interface/folder trước | Giữ code đủ tốt; chỉ trích xuất cấu trúc khi trách nhiệm/chi phí được chứng minh |
| Đi sâu Source/CRM và Business Core | Tạm dừng mở rộng nghiệp vụ; dùng Auth/Organization/Source làm mẫu kiểm toán Engineering Core |

**Cần giữ:** học bằng code/SQL thật; giản dị Go + PostgreSQL; nghiên cứu cơ chế; người dùng tự quan sát dự đoán–chạy–kết quả; bảo vệ dữ liệu/tenancy/security thiết yếu; không mặc định 164-table rollout. Không phủ nhận mọi code/migration cũ: đó là vật liệu để review.

## 2. Engineering Core thực sự bao gồm gì?

Đây là hệ thống quyết định/kỷ luật cho developer, **không phải package tên core, không phải một bộ template cố định**:

1. **Source structure & ownership:** thư mục/package/file có vai trò logic; ai sở hữu phạm vi nào; exports/imports và dependency direction; nơi phù hợp để thêm code; tách file ≠ tách package/import boundary.
2. **Go conventions:** naming theo call site; constructor, struct/interface/method, error wrapping/handling, giảm nesting, package surface, context, concurrency/goroutine lifecycle, resource ownership, test layout, giá trị của tính nhất quán.
3. **HTTP contracts:** server/mux, middleware, auth principal, routing, request binding/validation, body limits và trailing JSON, response/status/error semantics, logging/redaction, CSRF/origin, request cancellation.
4. **Data engineering:** schema/migration life cycle; PK/FK/UNIQUE/partial index; sqlc so với pgx/raw SQL; query naming và generated ownership; transaction scope, locking, concurrency và DB failure.
5. **Runtime/security/reliability:** configuration/secrets, application bootstrap/shutdown, DB pool, credentials/session, request identity và resource authorization, external side effect/email/SMS, timeout/retry/idempotency, audit và observability.
6. **Dev lifecycle/governance:** source inspection, predict → self-written code → actual test output, code review, CI, change compatibility, local↔remote/DB reconciliation, checkpoint/decision record và scope/quyền thay đổi.

Không nhất thiết mỗi khía cạnh là một package hoặc một tài liệu riêng. Chuẩn thành văn được "earned" sau chứng cứ, không cần sinh tám file tiêu chuẩn hay framework nội bộ ở ngày đầu.

## 3. Sự thật từ source dev cần giữ nguyên (đối chiếu lại khi sang chat mới)

**Tại thời điểm checkpoint, trước commit docs này:** remote dev = 9e0064c8bb22ca6915f7fdbf57b5095142bc7af8; 6 cặp migration 000001..000006, Go main/auth/organization/source, source.sql/sqlc, integration tests và GitHub CI lịch sử. GitHub remote ≠ máy local hay DB của user; phải kiểm tra riêng. Tài liệu cũ về CORE-06 là kế hoạch lịch sử, **không phải bước kế tiếp**.

- main.go: main() gọi run() và log.Fatal ở entrypoint; run() đọc env, tạo pgxpool + Ping, config SMTP/cookie/origin, khởi tạo auth/organization/source, ServeMux, readiness, server.Serve trong goroutine, xử lý signals và graceful shutdown. Đây là working application composition; chưa có căn cứ cần tạo cmd/, internal/ hay một Application framework.
- auth/http.go: Handler giữ *Service/config; RegisterRoutes(mux), các closure HandleFunc và RequireAccount; cookie, decode 8192 bytes, reply, Account(request), AccountContext/AccountID. Login route decode → Service.Login → cookie → JSON.
- auth/service.go: *pgxpool.Pool, Mailer interface; Register tạo Party/Person/Account/Email/password credential/token trong pgx.BeginFunc, sau commit mới gửi email; SMTP failure có thể để Account đã tạo nhưng HTTP báo lỗi. Login query credential → Argon2id verify → verified email → INSERT session; AccountFromToken đọc session hash, revoked/expiry/closed account; Logout UPDATE revoked_at.
- **Ví dụ correctness cần học:** auth/http.go Logout hiện dùng "_ = h.S.Logout(...)" rồi xóa cookie và trả 204, ngay cả khi DB UPDATE lỗi. Nếu lời cam kết là server-side revocation, đây là failure path thực sự cần review. Không đồng nghĩa mọi lỗi đều cần trả nguyên văn; cần xác định contract và timeout/commit ambiguity trước.
- **Ví dụ consistency:** auth.decode body limit 8KiB và chỉ một Decode; organization.inputJSON 16KiB và chỉ một Decode; source.decodeJSON 1MiB và kiểm tra trailing JSON qua Decode lần hai. Các endpoint chưa có chính sách JSON/response/error chung; không đồng nghĩa phải tạo httpx ngay.
- **Ví dụ dependency:** organization.Handler nhận *auth.Handler và gọi h.Auth.Account; source.NewHandler nhận AccountResolver func; auth.RequireAccount middleware tồn tại nhưng chưa được áp dụng thống nhất. organization dùng auth.PositiveID cho HTTP path parse. Source comment nói chưa mount trong main, nhưng main.go đã mount => stale comments cũng là maintainability pressure.
- Ba biến thể response/error helpers: auth.reply, organization.respond/handleErr, source.writeJSON/writeFailure; cùng chủ đề nhưng khác hành vi; xét mức khác biệt thật trước trích xuất.
- **Security/risk còn mở:** CSRF/origin, brute force/rate limit, password recovery, account lifecycle, SMTP delivery, session revocation semantics, least-privilege, reviewer audit; current tests/CI không phải production safety certification.
- **Party/C01:** migration đã có kind/assert function/triggers trên dev (phải đọc live); user đã phản biện độ phức tạp và mong xét loại bỏ; không áp đặt lại quyết định trước, không sửa migrations nếu chưa review/được yêu cầu.
- **SQL/tooling:** auth/organization dùng pgx raw SQL; source dùng sqlc trong db/sourcesqlc, SQL source trong db/source/query/source.sql, config sqlc.source.yaml. Khác biệt cần được hiểu trước khi lập standard.
- Những bằng chứng đã kiểm tra ở remote là static source + CI lịch sử; chưa được kiểm chứng bằng runtime/local trong mốc chuyển hướng này. Không nói app production ready hoặc user local synced.

## 4. Dẫn chứng nghiên cứu bên ngoài (không copy hình thức)

### Laravel DRX — repository datvtph41107/laravel-drx, branch main, đọc source ở giai đoạn đối chiếu
- Laravel cung cấp Kernel/router, middleware pipeline với code trước/sau $next, FormRequest, Eloquent, Passport OAuth2, container/provider, Event/Listener, Job/queue và response helpers.
- Registration là nhiều requests: /users → confirmation email qua queue; /confirm-email → Email verified, User ACTIVE, side-effects (CreateUserAccounts, notification, Kafka sync).
- Login route /oauth/token đi qua IPActive, pre_login, AuthenticateUser, suf_login; suf_login còn làm OTP/device/token response sau $next. Cần kiểm chứng token lifecycle nếu phát hành token trước kiểm tra bổ sung.
- KYC: KYCRequest, /otp-verify cache, /create-identity hồ sơ pending, /check-kyc gọi Blockpass rồi đổi trạng thái verified/rejected; hồ sơ nộp ≠ KYC đã verified; object authorization và lưu public document là các rủi ro cần audit.
- IPActive module tự có Define, Provider, Middleware, Service, Event/Listener, Model, Migration, Clean Command: thể hiện cách đóng gói cross-cutting concern, nhưng query COUNT rồi insert log có race, down migration sai tên bảng; module hóa ≠ correctness.
- Order HTTP tạo Order NEW và dispatch ProcessOrderRequest RabbitMQ; worker lock/cập nhật order/số dư → Redis sorted sets → ProcessOrder tìm cặp bằng Lua script → DB transaction và row lock, OrderService stored procedures và events. **Không diễn ra toàn bộ trong một HTTP request.** Các rủi ro cần audit: commit trước queue dispatch, jobs swallow exception, Redis↔DB reconciliation.
- Không sao chép cấu trúc Laravel, không tuyên bố mọi hành vi runtime khi mới đọc static code. DRX là vật liệu học cách tổ chức và cách framework cho sẵn contract, đồng thời học failure modes.

### Go / ecosystem
- Official: https://go.dev/doc/modules/layout ; https://go.dev/blog/package-names ; https://go.dev/doc/effective_go ; https://pkg.go.dev/net/http ; https://go.dev/blog/routing-enhancements .
- Uber Guide: https://github.com/uber-go/guide (package naming, error once + wrapping, avoid mutable global, limit nesting, manage goroutine lifecycle, exit in main, tests, consistency). Là style guideline, **không phải Go spec hay backend architecture authority**.
- Google Go Guide: https://google.github.io/styleguide/go/ ; conventions phải được đọc cùng Go stdlib và source.
- Mature repos: Moby, Terraform, Prometheus, etcd, NATS; chọn repo đúng vấn đề (trong docs/research/go/ANCHOR-REPOSITORIES.md), không copy cây source Kubernetes/CockroachDB làm hình mẫu của app nhỏ.
- Framework/library alternatives: net/http, Gin, Echo, Chi; router/framework chỉ đóng gói lựa chọn và developer ergonomics. Go 1.22 ServeMux hỗ trợ method/path wildcard. HTTP framework không tự giải quyết DB bottleneck, resource auth, queue recovery hay SQL correctness.
- Market/surveys, discussions/Reddit, benchmarks: giúp phát hiện tín hiệu/quan điểm đối lập. Không lấy số sao, vote, benchmark khác workload, popularity làm chứng minh fit.
- Xem thêm [REFERENCES.md](../../../research/go/REFERENCES.md) và [EVIDENCE-DRIVEN-REASONING.md](../../EVIDENCE-DRIVEN-REASONING.md).

## 5. Cách ra quyết định — bắt buộc cho thay đổi có tác động

**Không đủ để nói "best practice", "sạch hơn", "chi phí thấp hơn", "hiệu năng tốt hơn".** Cần chỉ được code/actor/flow thật, cơ chế và so sánh đối chứng. Quy trình:

1. **CURRENT:** live Git SHA + trạng thái đúng môi trường; điều gì đã chạy/được kiểm chứng; ai/nhóm developer sử dụng.
2. **CONTRACT:** function/request/resource thật đang cam kết điều gì; input/output/side effect/error boundary.
3. **PRESSURE:** lỗi thực, sai lệch chính sách, lặp code tốn thời gian, failure mode/security nghiêm trọng có đường dẫn rõ; không tự tưởng tượng quy mô.
4. **PRIMARY SOURCE:** specs/docs Go/PostgreSQL/HTTP và source implementation trả lời cơ chế; external guides/repo/cộng đồng phản chứng; nguồn và thời điểm.
5. **OPTIONS:** giữ nguyên, sửa tại chỗ, dùng helper, package/interface, library/framework; so sánh đồng điều kiện + chi phí lập trình/review/onboarding/runtime/migration/dependency/ops.
6. **EXPERIMENT:** test nhỏ + negative/concurrency/security khi phù hợp; predicted vs observed; không giả vờ chạy trên user local.
7. **DECISION:** KEEP/MODIFY/TRIAL/DEFER/REJECT + giới hạn và tín hiệu xét lại. Ghi chuẩn dùng chung chỉ khi được biện minh.
8. **DELIVERY:** user tự viết source khi đến giai đoạn; review diff và kết quả; assistant không tự mở Source/CRM/SQL migration, không gọi framework làm đích đến.

**Nguồn chứng cứ theo câu hỏi, không một bảng xếp hạng cứng:** official spec cho cơ chế, runtime bdspro cho hiệu năng thực, repo production cho tổ chức, style guide cho conventions, research cho phép đo, community cho trải nghiệm/đối lập. Mọi nguồn đều có giới hạn.

## 6. Những quyết định hiện tại — chốt phạm vi / còn mở

**Chốt hướng làm việc (không đồng nghĩa chốt implementation):**
- ENGINEERING CORE là trọng tâm; tạm dừng các bước CORE-06 CRM/Source/Listing và database business expansion.
- Tiếp tục Go standard library và net/http để học/đánh giá; **không có bằng chứng hiện tại phải chuyển Gin/Echo hoặc làm internal framework**.
- KHÔNG di chuyển main.go thành cmd/, KHÔNG tạo internal/, pkg/, common/, helpers/ hay bộ thư mục mới vì nghi thức; giữ source đang đủ để nghiên cứu.
- Kiểm tra chính source và runtime, không kết luận test PASS = tối ưu design/maintainability/production security.
- External evidence là thành phần bắt buộc cho quyết định lớn và phải có phản chứng.
- User ưu tiên tiếng Việt; một vấn đề cơ bản tại một thời điểm; giải thích cú pháp → cơ chế → vị trí trách nhiệm → tác động; không code dump, không vẽ nhiều tầng để trang trí.
- Không thay đổi code, migrations, tests, remote main, local DB trong mốc handoff này.

**Vẫn mở, cần nghiên cứu/nhận ý kiến user:**
- Exact responsibilities and dependency rules for each package/file and exported symbol.
- JSON validation rules/body policy, request/response/error contract, middleware authorization; nên dùng helper hay để cục bộ?
- Sử dụng auth.RequireAccount/AccountResolver và principal context; boundary đúng và cách kiểm soát.
- Auth Login/Logout/Registration behavior khi DB/mail lỗi; rate limit, CSRF/recovery, dependency injection/no mutable globals.
- Uniform SQL/sqlc/pgx policy, transaction ownership and generated code lifecycle.
- Quy chuẩn test, CI, code review/commit, dev lifecycle, observability, config và deployment.
- C01 kind/trigger tradeoff (quay lại khi kỹ thuật và nhu cầu cần, không tự sửa).
- Tách package hoặc dùng Gin/Echo: DEFER cho tới khi có áp lực định lượng/định tính và benchmark hợp lệ.

## 7. Bước tiếp theo — KHÔNG bắt đầu từ business tables hay cây thư mục mới

**Bước đầu: audit main.go và vòng đời của một HTTP request Auth, trên source dev hiện tại.**
1. Đọc main() / run() và phân loại config, resource creation/ownership, wiring, ServeMux, middleware, Listen/Serve, signal, shutdown/cleanup; giải thích từng primitive Go đang dùng và điều gì đã đủ tốt.
2. Truy POST /auth/login qua route → decode → Service.Login → pgx session insert → cookie/response. Tách transport, application, database, security contract; đọc code, không giả định.
3. Xem auth.RequireAccount vs organization current vs source AccountResolver: vì sao ba cách, có pressure thật không?
4. Lấy một vấn đề nhỏ có bằng chứng, ví dụ Logout bỏ qua DB error hoặc trailing JSON policy, phân loại correctness / consistency / maintainability. Trước khi "fix": xác định lời cam kết, chính sách và phản chứng.
5. So với official Go, Uber/Google, rồi repository trưởng thành đúng chủ đề, giải thích lợi/cost và KEEP/MODIFY/DEFER. Rút ra **một** quy chuẩn nhỏ, chưa commit implementation.
6. Khi user muốn và sau design review, user triển khai thử và đem output về. Cập nhật handoff theo milestone, không viết nguyên loạt migrations/services.

**Kết quả thành công của giai đoạn mới:** developer có thể trả lời "file này ở đâu, ai sở hữu, dùng cấu trúc nào, code có cơ chế nào, nếu đổi sẽ tác động gì, cần kiểm thử gì, tiêu chí ngoại lệ ra sao" mà không dựa vào pattern hay assistant như quyền lực; quy chuẩn phải giảm tổng chi phí so với cách đơn giản đang có.

## 8. Khôi phục chat mới và xử lý tài liệu lịch sử

- Prompt chính: [CONTINUE-PROMPT.md](CONTINUE-PROMPT.md), dán nguyên văn vào message đầu tiên.
- Khi một docs trước 15/10/2026 nói "next CORE-06" hoặc "continue Party migration/SQL-01" thì đó là **lịch sử** trừ khi user vừa chỉ đạo đổi. Live source/CI/DB xác nhận fact; checkpoint này xác nhận **trọng tâm và phương pháp mới**.
- Đọc [REPOSITORY-WORKING-CONTRACT.md](../../REPOSITORY-WORKING-CONTRACT.md), [PROPORTIONAL-ENGINEERING-STANDARD.md](../../PROPORTIONAL-ENGINEERING-STANDARD.md), [EVIDENCE-DRIVEN-REASONING.md](../../EVIDENCE-DRIVEN-REASONING.md), [PRACTICE-PROTOCOL.md](../../PRACTICE-PROTOCOL.md) để giữ safety và phương pháp.
- Check actual Git branch/commit trước mutate; remote dev docs update không thay đổi máy local. Không reset/force-push/overwrite; main là historical baseline không phải accepted release.
- Checkpoint phục vụ **tái tạo trạng thái làm việc và tư duy**, không phải transcript 1:1 từng token; khi đụng chi tiết chưa rõ, phải đọc source live và hỏi/kiểm chứng, không tự bịa.
