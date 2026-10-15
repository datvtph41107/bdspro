# BDSPro — Prompt khôi phục nguyên ngữ cảnh (2026-10-15 · Engineering Operating Model)

Sao chép phần giữa dấu ~~~ bên dưới làm tin nhắn đầu tiên ở chat mới. Không cần kể lại lịch sử.

~~~text
Tiếp tục BDSPro từ CHECKPOINT MỚI NHẤT lưu trên GitHub; không dùng trí nhớ cuộc chat trước để thay thế việc đọc source thật.

REPO: datvtph41107/bdspro
BRANCH: dev
CHECKPOINT:
docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/README.md
MA TRẬN 20 CHƯƠNG:
docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/CHAPTER-MATRIX.md
CHƯƠNG ĐANG MỞ:
docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/CH01-CONFIG-AND-WORKFLOW.md
PROMPT ĐẦY ĐỦ:
docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/CONTINUE-PROMPT.md

Hãy trực tiếp đọc BỐN tài liệu trên GitHub, xác minh HEAD dev và các file thực tế:
main.go, Makefile, .gitignore, sqlc.source.yaml, .github/workflows/bdspro-core.yml, go.mod, migrations/000001–000006; khi cần xem auth/http.go, auth/service.go, organization/http.go, source/http.go, source/service.go, source.sql và tests.
Đọc thêm checkpoint tư duy 2026-10-15-engineering-core-reorientation/README.md; continuity/REPOSITORY-WORKING-CONTRACT.md, EVIDENCE-DRIVEN-REASONING.md, PROPORTIONAL-ENGINEERING-STANDARD.md, ENGINEERING-WORKFLOW.md. Các roadmap CORE-06/Business Core/SQL-01 trước đây là lịch sử, không phải ưu tiên hiện tại. Local Git/DB/production chưa được chứng minh đồng bộ remote.

MỤC ĐÍCH:
Xây ENGINEERING CORE thành một hệ điều hành làm việc cho developer và operator: naming, code/file/package ownership, comments, request lifecycle, middleware, credentials/session/security, Business/Authorization boundary, Go errors/context, DB/SQL/sqlc, migrations, config, startup/shutdown, logging/metrics/tracing/audit, testing/CI, debug/recovery/PR/work habits. Mỗi concern có một CANONICAL SOURCE OF TRUTH, OWNER, CONTRACT, test/operational proof và trade-off. Không ép mọi sự thật vào một file và không áp kiến trúc/framework có sẵn.

ĐÃ CÓ ma trận 20 chương / 5 bo mạch CH00–CH19. Đây là map review, KHÔNG phải 20 module phải viết; ưu tiên dựa trên risk và dependency. User rất thích sự nhất quán và khả năng dự đoán của Laravel DRX về routes, middleware, code location; muốn học từ TechSchool SimpleBank (app.env, util/config.go, Makefile, sqlc.yaml, docker-compose.yaml, start.sh, wait-for.sh), KHÔNG sao chép Viper/shell/Redis/Grpc/architecture chỉ vì mẫu có.

ƯU TIÊN MỚI NHẤT — BẮT ĐẦU NGAY CH01:
Canonical Configuration & Developer Workflow / Engineering Work Habits.
Làm rõ .env/.env.example/.yaml/Makefile/.sh/config struct, dev/test/prod, secret delivery, env precedence, CLI tasks, DB URL, startup, process ownership và ownership of truth.
BDSPro git tree được kiểm tra mới có Makefile, sqlc.source.yaml, CI workflow và .gitignore ghi .env; không có .env/.sh tracked. main.go đọc DATABASE_URL, APP_ENV, SMTP, ALLOWED_ORIGIN, BIND_HOST, PORT; Makefile dùng DB_URL cho migrations; CI dùng DB riêng. Chưa kết luận đó là bug hay cần thêm file. Chưa xác minh local settings.
Bước kỹ thuật đầu tiên CH01.1: lần dấu DATABASE_URL từ source/main.go qua pool/Ping và so với DB_URL Makefile/CI; thiết kế bảng authority-owner-consumer-precedence và đề xuất validation/dev workflow tối thiểu. Đặt luận điểm/luận cứ và phản chứng, theo một vấn đề nhỏ tại một thời điểm.

TƯ DUY / CÁCH PHẢN HỒI:
- Chính xác source nào là fact, điều gì là inference, open question, approved decision. Không tuyên bố đã chốt standard khi mới review.
- Từng mốc: từ cơ chế Go/DB/OS cơ bản → trách nhiệm và actor → failure modes / trust boundary → tài liệu chính thức → source/repo/community và phản chứng → trade-off (complexity, runtime, developer productivity, maintenance) → KEEP / MODIFY / TRIAL / DEFER / REJECT → kiểm chứng.
- Không đưa một loạt abstraction, folder mẫu, 100 files hay tổng hợp lý thuyết dài thay vì đi sâu source.
- User tự viết/chạy Go, SQL và commands; assistant hướng dẫn/checkpoint/review. KHÔNG đổi source, migrations, YAML, Makefile, CI hay .env nếu chưa được ủy quyền cụ thể; yêu cầu checkpoint docs hiện tại đã được ủy quyền.
- Không destructive commands, không reset local, không in secrets. Nếu nguồn ngoài/actual code thay đổi, đối chiếu live.
- Trả lời bằng tiếng Việt, rõ ví dụ và ownership; tránh hỏi lại thông tin đã lưu.

BÂY GIỜ:
(1) Tóm tắt đã khôi phục từ GitHub và những gì vẫn chưa chứng minh.
(2) Đưa ra ma trận canonical truth/owner của config các files hiện có.
(3) Bắt đầu CH01.1 với DATABASE_URL vs Make DB_URL; theo một cơ chế cụ thể, nêu test/quan sát và quyết định tạm thời. Không mở CRM hoặc bắt đầu AUTH/JWT trước.
~~~
