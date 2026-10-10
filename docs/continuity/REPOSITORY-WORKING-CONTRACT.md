# BDSPro — Repository Working Contract (Git · PostgreSQL · Go)
**Chốt ngày:** 2026-10-10 · **Trạng thái:** quy chuẩn vận hành có hiệu lực từ khi ghi trên `dev`, có thể sửa khi bằng chứng mới bác bỏ giả định.
**Người triển khai:** user tự viết/chạy SQL, Go, Git và CI khi đến giai đoạn. Assistant phân tích, gợi ý mức tối thiểu, kiểm tra output, tìm phản chứng; không viết code, merge, CI hay thao tác destructive thay user khi chưa được yêu cầu cụ thể.

## 0. Quy tắc gốc — bản chất trước biến thể

```text
BẢN CHẤT
  → TRẠNG THÁI HIỆN CÓ
  → MỤC ĐÍCH / SỰ THẬT CẦN BẢO VỆ
  → DỰ ĐOÁN
  → THỬ NGHIỆM NHỎ DO USER TỰ VIẾT
  → OUTPUT / SAI SỐ / ĐIỀU CHƯA ĐƯỢC CHỨNG MINH
  → GIẢI PHÁP ÍT PHỨC TẠP NHẤT ĐỦ DÙNG
  → TRADE-OFF / QUYẾT ĐỊNH
  → THỰC HIỆN / REVIEW / GHI LẠI KHI CÓ GIÁ TRỊ
```

Một cơ chế được hiểu qua **nó lưu trạng thái gì, thay đổi điều gì, bảo đảm gì, không bảo đảm gì, khi nào sai, và chi phí sở hữu**. Đừng áp dụng mẫu chỉ vì gọi là "senior" hay "best practice"; cũng đừng trì hoãn tính đúng đắn của dữ liệu cốt lõi chỉ vì chưa lên production.

## 1. Những sự thật không được đánh đồng

| Khái niệm | Sự thật / cơ chế | KHÔNG chứng minh |
| --- | --- | --- |
| Git commit | Snapshot nội dung, parent(s), metadata, hash | Chương trình đã chạy đúng |
| Git branch | Con trỏ có thể di chuyển đến commit | Bản sao filesystem độc lập, môi trường deploy |
| Local working tree | File đang có trên máy, kể cả thay đổi chưa commit | GitHub đã nhận những thay đổi đó |
| Git index | Nội dung định commit | Nội dung đã được commit |
| Local `dev` | Commit tham chiếu trên máy | Bằng `origin/dev` |
| `origin/dev` | Remote-tracking ref của lần fetch cuối | GitHub hiện tại nếu chưa fetch |
| GitHub `dev` | Code và tài liệu đã push ở remote | Local working tree đã đồng bộ, DB đã migrate |
| PostgreSQL database | Schema, row, locks, catalog, migration version thực | Tương thích với Go/sqlc hiện tại |
| Canonical DBML V0.4 | 164 đối tượng logic ứng viên, quyết định thiết kế mở | 164 bảng đã triển khai, 164 migrations cần lập tức tạo |
| CI green | Những ca kiểm thử thực sự chạy đã qua | Mọi nghiệp vụ, production, concurrency đều đúng |

**Authority:** code/DB/output kiểm chứng ở **môi trường đang nói đến** > commit SHA và diff tương ứng > quyết định có bằng chứng > checkpoint > hồi ức. Không copy số commit hoặc đầu ra PostgreSQL của lab cũ sang trạng thái hiện tại.

## 2. Hai nhánh có trách nhiệm khác nhau — đã khôi phục main

**Bằng chứng GitHub 2026-10-10:**
- `main` **đã được khôi phục chính commit cuối trước khi xóa:** `3f90f7789db4dff9488f9255f909fed6695e51b9`. Đây là **snapshot lịch sử ổn định về nhận diện**, **không được tuyên bố production-ready**. Chưa merge nội dung WIP từ `dev` sang `main`.
- `dev` = nơi user phát triển, thử, commit thường xuyên. GitHub trước khi khôi phục `main` có default branch `dev`; kiểm tra thực tế trước mọi thao tác liên quan default.
- `main` = mục tiêu *được chấp nhận để giữ ổn định* sau các mốc có bằng chứng tích hợp. Hiện tại chỉ là baseline lịch sử, chưa có mốc phát hành đầu tiên.
- **Default branch là setting GitHub, không phải mức độ đáng tin của code**. Chưa yêu cầu chuyển về `main` cho đến khi có một baseline build/query/migration đáng dùng.
- Nhánh ngắn hạn chỉ khi có rủi ro cô lập cao, thay đổi song song, hotfix một bản đã phát hành, hoặc cần review độc lập; phải có mục đích và điều kiện kết thúc. **Không tạo nhánh cho mỗi bài học SQL, mỗi bảng, mỗi ngày.**
- PR #2, #3 đã đóng, không merge. Không tái sử dụng hai dòng migration `000006` và `000001` do assistant viết như triển khai hiện hành. Lịch sử commit cũ vẫn là nguồn học, không là quyền đưa code vào `dev`.

**Không đồng bộ main/dev chỉ cho cùng commit.** Khi `dev` qua được mốc ổn định, chọn **một tập thay đổi đã kiểm chứng** để đưa lên `main`, xem diff và bảo đảm nguồn, migration và runtime vẫn nhất quán. Không tùy tiện ép force push hai nhánh đang khác lịch sử.

## 3. Chu kỳ local ↔ remote tối thiểu

### Đầu phiên: đọc sự thật
```bash
git branch --show-current
git status --short --branch
git log -1 --oneline
# Khi cần biết upstream:
git fetch origin --prune
git rev-list --left-right --count dev...origin/dev
git diff --name-status dev...origin/dev
```

Khi so sánh `dev...origin/dev`, số thứ nhất trong `rev-list --left-right --count` là commit chỉ ở local dev; số thứ hai là commit chỉ ở upstream. `git fetch` **không** sửa file working tree.

### Khi viết: việc đủ nhỏ để hiểu và chứng minh
- Một unit thay đổi = một trách nhiệm rõ, không mặc định một bảng / một PR.
- Viết SQL/query/test/Go bằng tay; dự đoán kết quả, kiểm tra real output và giới hạn. Đừng chép nguyên giải pháp sẵn có trước khi có thử nghiệm.
- `git diff` thường xuyên; **chỉ stage những file/hunk thuộc quyết định đó**, review `git diff --cached` trước commit.
- Một commit tốt trả lời: *nó giải quyết cái gì, có bằng chứng nào, hậu quả nếu revert?* Không cần commit mỗi câu lệnh `psql`.

### Khi chia sẻ: push có chủ đích
- `git commit` chỉ đổi local Git; `git push origin dev` đưa commit lên GitHub; **không** áp dụng migration vào PostgreSQL.
- Trước push khi có người khác/thay đổi remote: fetch + kiểm tra divergence. Không `push --force`, `reset --hard`, `clean -fd`, xóa branch theo phản xạ.
- Remote có commit mới và local sạch: xem diff rồi `git merge --ff-only origin/dev`. Local còn sửa: bảo toàn (commit phù hợp hoặc stash chọn lọc) trước; đừng `pull` như cách thăm dò.
- Sau mỗi tác vụ Git thay đổi trạng thái, đọc output/status thực. **Local không phải thao tác mà assistant có thể nhìn hoặc khẳng định đã thực hiện**.

### Khi dừng hoặc bàn giao
- Biết working tree còn gì, HEAD local và remote nào, DB nào vừa kiểm tra, test đã chạy ở đâu, điều gì còn mở.
- Chỉ cập nhật `CHECKPOINT.md`/`DECISIONS.md` khi invariant thay đổi, có proof quan trọng, migration được chấp nhận, thay đổi hướng hoặc bàn giao lâu ngày. Không biến documentation thành log từng lệnh nhỏ.

## 4. Database/migration: lịch sử DDL khác lịch sử business

- Source migration định nghĩa sự thay đổi schema; `schema_migrations` ghi phiên bản một **DB cụ thể**; một giá nhà cập nhật tạo business row/event, **không** tạo migration.
- Schema lab disposable ≠ dữ liệu shared/pilot/production: lab có thể xóa có chủ đích; migration đã áp dụng trong shared database phải tiến hóa có khả năng giữ dữ liệu.
- Không `000001_init` nhét 164 bảng; cũng không luật cứng "mỗi bảng một migration". Gộp khi nhiều bảng/constraint cần hình thành **một invariant**; chia khi áp dụng độc lập hoặc giảm rủi ro.
- Thử trước PK → FK → UNIQUE/CHECK → transaction; chỉ khi phản chứng rõ mới dùng partial UNIQUE, deferred trigger, locking, exclusion hoặc logic application. PostgreSQL constraint bảo vệ một số tính chất của dữ liệu, **không tự xác nhận người đang yêu cầu là actor được phép**.
- Viết migration với up/down có ý nghĩa cho bước học; nhưng "down chạy được" ≠ có thể khôi phục dữ liệu đã xóa trong production. Khi hệ thống có dữ liệu, cân nhắc forward-fix, backup/restore, expand/backfill/validate/switch theo rủi ro.
- `golang-migrate force` thay phiên bản được ghi nhận; không sửa DDL sai. Khi DB dirty, đọc lỗi và trạng thái schema thực, không force để che lỗi.
- Khi query quan trọng, phân biệt **logic trả đúng kết quả** với **độ phức tạp/hiệu năng trên dữ liệu thực**. `EXPLAIN (ANALYZE, BUFFERS)` là công cụ quan sát khi có dấu hiệu/perf goal rõ; không kết luận từ vài hàng lab rằng tối ưu ở quy mô lớn.
- App tương thích phải bao gồm schema + `sqlc generate` + Go run/query (và test như phù hợp), không chỉ compile. **Hiện Go Listing cũ có thể còn tham chiếu bảng mà local DB đã reset**, đây là cổng tương thích trước khi tuyên bố Core đã runnable toàn bộ.

## 5. Mức kiểm chứng mua theo rủi ro

| Tình huống | Ít nhất cần gì | Nâng độ nghiêm khi |
| --- | --- | --- |
| Học SQL trong DB bỏ được | Lý do, dự đoán, câu truy vấn tự viết, output/error, giải thích | Bất biến nhiều dòng, time/NULL semantics, concurrent sessions |
| Một migration Core nhỏ trên DB sạch | Up + kiểm tra catalog và trường hợp sai, nhìn ảnh hưởng FK và down như bài học | Có dữ liệu phải giữ, lock/deadline hay nhiều runtime consumers |
| Go query/transaction | Compile + gọi DB thử thật + lỗi trả về + phạm vi dữ liệu | Races, retries, tenant boundary, data loss |
| Nhiều người/pilot | Kiểm tra quyền, retention, backup/restore, test lặp lại, log thiết yếu | Có traffic, release thường xuyên, lỗi tái diễn → CI do user chủ động xây |
| Production/tiền/dữ liệu nhạy cảm | Quyền và bảo vệ dữ liệu trước exposure; rollback/forward plan, giám sát, khôi phục | Throughput/rủi ro pháp lý/vận hành đòi mức cao hơn |

Đừng trì hoãn tính đúng đắn danh tính, tenant privacy hoặc bảo vệ secrets vì "mới dev"; cũng đừng tạo Kafka, microservice, elaborate CI hoặc toàn bộ feature set chỉ vì dự đoán quy mô tương lai.

## 6. Ví dụ đã có bằng chứng: học từ kết quả, không học từ khẩu hiệu

- **DB-15:** hai active membership cho (Org 6, Person 1) được INSERT. FK tồn tại **không** tự ngăn trạng thái sai.
- **DB-16:** tạo partial unique index thất bại vì dữ liệu trùng. **DB-17:** xử lý trùng và tạo index. **DB-18:** INSERT trùng active bị từ chối. Giải pháp được mua bằng một phản chứng cụ thể.
- **DB-19–20:** đóng Membership 2 và mở Membership 5 trong một transaction; `same_boundary=true`, `is_overlapping=false` với nửa khoảng `[joined_at, ended_at)`. Điều này chứng minh semantics interval SQL, **chưa xác nhận business model episode**.
- **CORE-01:** `Party XOR subtype` ở thời điểm commit **khác** bất biến `Party kind immutable over lifetime`. Hai CI draft khác nhau từng kiểm thử hai lựa chọn, không thể lấy một bộ test để mặc định lựa chọn còn lại đúng.
- **Local reset:** mười file Listing migration có trạng thái deleted; GitHub remote vẫn giữ đến khi commit/push. Đây là real-world mismatch phải được reconcile, không dùng `reset --hard` để sửa cho nhanh.

Nguồn chi tiết: `docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/DATABASE-REVIEW-AND-LAB.md` và `docs/continuity/PROPORTIONAL-ENGINEERING-STANDARD.md`.

## 7. Bài tập thực hành đầu tiên từ chính hiện trạng local

**Bài Git-00:** Người học tự dự đoán:
1. `git fetch origin` có phục hồi lại các file Listing migration đang `D` trên local không?
2. `git merge --ff-only origin/dev` có được phép khi working tree còn deletions? Điều gì làm Git từ chối?
3. `git stash push ... -- migrations` lưu gì, có tác động PostgreSQL không?
4. `git stash apply` khác `git stash pop` thế nào?
5. `git rev-list --left-right --count dev...origin/dev` đang nói về file nào hay commit nào?

**Thực hành:** Chạy read-only `git status --short --branch`, `git branch -vv`, `git fetch origin`, `git rev-list --left-right --count dev...origin/dev`. Gửi output; sau đó tự chọn cách bảo toàn deletions và đồng bộ, với assistant review. Đây là cơ chế Git cần học trước khi làm `000001`.

**Bài SQL-01 (sau khi Git local an toàn):** người học tự tạo trong DB/schema **có thể bỏ**, giải thích và thử FK/PK cho Party/Person/Organization. Thử Party không subtype và Party hai subtype; quan sát output để phân biệt constraint hiện tại với C01 cần bảo vệ. **Chưa chép trigger, chưa tạo cả 164 bảng.**

## 8. Nguồn chính thống và cách hiểu (không áp dụng máy móc)
- Git Book, "Branches in a Nutshell": https://git-scm.com/book/en/v2/Git-Branching-Branches-in-a-Nutshell — branch là movable pointer, giúp hiểu việc restore `main`.
- GitHub Flow: https://docs.github.com/en/get-started/using-github/github-flow — isolated changes/commits/PR, not a requirement to branch for every SQL test.
- GitHub protected branches: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches — reserve protective settings for branch with consequential stable releases; do not mandate CI checks before CI exists.
- GitHub default branch: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-branches-in-your-repository/changing-the-default-branch — default controls standard landing/base, not software quality.
- DORA, small batches: https://dora.dev/capabilities/working-in-small-batches/ — bounded integration and fast feedback.
- PostgreSQL constraints: https://www.postgresql.org/docs/current/ddl-constraints.html — exact SQL guarantees.
- PostgreSQL transactions: https://www.postgresql.org/docs/current/tutorial-transactions.html — atomic operation and what COMMIT/ROLLBACK changes.
- golang-migrate migration rules: https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md — versioned `up`/`down`, not a bulk canonical dump.
- golang-migrate GETTING_STARTED: https://github.com/golang-migrate/migrate/blob/master/GETTING_STARTED.md — `dirty`/version must be reconciled against real schema.
- DORA, CI: https://dora.dev/capabilities/continuous-integration/ — motivation for automation **when repeatability/collaboration pressure appears**.

## 9. Chốt
`main` có quyền tồn tại vì giúp bảo toàn và chỉ đích danh một baseline được chấp nhận, **không** vì mọi repo đều phải có `main`. `dev` có quyền tồn tại vì user cần không gian thực hành tích hợp. Hãy để hai nhánh khác nhau theo mục đích; chỉ cập nhật `main` khi có một mốc đáng ổn định. Bản thân tên branch, green test, nhiều bảng, nhiều file và quy trình dài đều **không** tạo ra giá trị nếu không bảo vệ được sự thật nghiệp vụ và cải thiện khả năng làm việc.
