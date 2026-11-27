# OTP qua SMS, Zalo và Telegram

Tài liệu áp dụng cho code trong repo này. Đây là đề xuất triển khai phù hợp với ứng dụng tại Việt Nam, không phải khẳng định mọi nhóm dev đều dùng cùng một nhà cung cấp.

## Thiết kế đã triển khai

`services/auth` tạo mã ngẫu nhiên, lưu MAC của OTP, kiểm tra hạn dùng, giới hạn thử sai và ràng buộc challenge với browser/tài khoản. `services/phonedelivery` áp dụng chính sách kênh, giới hạn gửi đồng thời và log; package này sở hữu interface tiêu thụ adapter:

```go
type Sender interface {
    Send(context.Context, Message) (Receipt, error)
}
```

`Message` gồm phone E.164 và OTP. Channel thuộc service. `Receipt` có provider/message ID nếu cổng trả về. `modules/phonedelivery` chứa các adapter HTTP độc lập với setting, auth, DB và router. `routers.NormalHandler` tạo service một lần từ cấu hình đã kiểm tra; router truyền dependency tường minh vào hàm nghiệp vụ auth. Thêm nhà cung cấp SMS Việt Nam bằng adapter mới và nối trong constructor service; giữ nguyên thuật toán OTP và API frontend.

Xem [quyết định kiến trúc và bằng chứng kiểm thử](architecture/phone-delivery.md) để đối chiếu từng tầng, nguồn tham khảo và giới hạn.

| Kênh trên giao diện | Adapter thật | Dev local |
| --- | --- | --- |
| SMS | Twilio Programmable Messaging | Email mô phỏng trong Mailpit |
| Zalo | ZBS Template Message qua số điện thoại | Email mô phỏng trong Mailpit |
| Telegram | Telegram Gateway | Email mô phỏng trong Mailpit |

Bot API của Telegram dùng chat ID. Gateway mới là API phù hợp để gửi mã đến số điện thoại đã đăng ký Telegram. Backend truyền mã tự sinh vào Gateway; bước kiểm tra mã vẫn ở backend. [Telegram Gateway API](https://core.telegram.org/gateway/api)

Người dùng chọn kênh mà họ sử dụng với đúng số điện thoại đó. Danh sách kênh chỉ phản ánh cấu hình server, không dò tài khoản của người nhận trên Zalo/Telegram. Không tự gọi API kiểm tra khả năng gửi hoặc tự gửi qua tất cả kênh.

## Chạy local

Mailpit hiện có SMTP ở `127.0.0.1:1025`, inbox ở http://localhost:8025. Nếu chưa chạy container, dùng `docker compose up -d mailpit`; nếu container của project Docker khác đang dùng các cổng này, dùng container hiện có.

Trong `custom/conf/app.ini`:

```ini
RUN_MODE = development

[mailer]
SMTP_PROTOCOL = smtp
SMTP_ADDRESS = 127.0.0.1:1025
FROM = security@bdspro.test
USERNAME =
PASSWORD =

[phone_delivery]
MODE = mailpit
DEFAULT_CHANNEL = sms
CHANNELS = sms,zalo,telegram
```

Nếu API cũng chạy trong mạng Compose, SMTP_ADDRESS là `mailpit:1025`. Chế độ mailpit chỉ chấp nhận development và SMTP local: loopback/localhost hoặc hostname mailpit. Production cấu hình nhầm chế độ này sẽ bị từ chối khi khởi động.

Chạy từ root repo:

```bash
go run . --config custom/conf/app.ini config check
make server
```

Chạy frontend trong terminal khác:

```bash
cd frontend
npm run dev
```

Mở http://localhost:5173, chọn Continue with phone number, nhập số và chọn kênh. Trong Mailpit, tìm tiêu đề `[DEV sms]`, `[DEV zalo]` hoặc `[DEV telegram]`. Địa chỉ nhận dạng `zalo.849xxxxxxxx@phone.invalid` chỉ là mailbox thử, không phải địa chỉ người dùng.

OTP không trả trong JSON API hoặc access log. Mailpit là nơi đọc OTP ở dev. Không có nhánh luôn chấp nhận mã cố định.

## Hợp đồng API

`GET /api/v1/auth/phone/channels` trả:

```json
{"channels":["sms","zalo","telegram"],"default_channel":"sms","development":true}
```

`POST /api/v1/auth/phone/start`:

```json
{"phone":"0912345678","country_code":"VN","channel":"zalo"}
```

Sau 202, response có `challenge_id`, `resend_after` và `delivery_channel`. Browser phải giữ cookie. `POST /auth/phone/verify` vẫn nhận challenge_id/code như trước.

`POST /api/v1/auth/phone/resend` có thể đổi kênh:

```json
{"challenge_id":"UUID_FROM_START","channel":"sms"}
```

Resend lấy số điện thoại từ challenge đã ràng buộc cookie; client không được đổi đích nhận. Khi bỏ channel, server dùng DEFAULT_CHANNEL; frontend luôn gửi kênh đang chọn. Channel không phải thuộc tính danh tính, nên không cần migration DB cho thay đổi này. Response chỉ cho biết kênh vừa được chấp nhận gửi.

`POST /api/v1/me/phone/start` cũng hỗ trợ channel, vẫn yêu cầu đăng nhập và reauthentication proof. Luồng liên kết không được bỏ qua xác thực lại.

Các POST từ browser vẫn cần Origin hợp lệ và `X-BDS-Request: 1`. OpenAPI đã được cập nhật trong `templates/swagger/openapi.json`.

## Kết nối cổng thật

Chuyển `[phone_delivery] MODE = live`. Chỉ liệt kê kênh đã có tài khoản và credentials trong CHANNELS; DEFAULT_CHANNEL phải thuộc danh sách đó. Mỗi kênh bật thiếu cấu hình sẽ làm preflight thất bại.

### SMS/Twilio

```ini
[phone_delivery]
MODE = live
DEFAULT_CHANNEL = sms
CHANNELS = sms

[sms]
ACCOUNT_SID = <account-sid>
AUTH_TOKEN = <auth-token>
SENDER_ID = <approved-brand>
```

Adapter dùng endpoint Messages của Twilio và mẫu nội dung hiện có trong `providers.go`. Với Việt Nam, Twilio yêu cầu đăng ký alphanumeric sender và các mẫu nội dung; cần gửi đúng mẫu đã được nhà cung cấp duyệt. [Hướng dẫn đăng ký tại Việt Nam](https://help.twilio.com/articles/360040339113)

Nếu chọn nhà cung cấp SMS trong nước, yêu cầu họ cấp tài liệu API, sandbox, mã lỗi, quy tắc nhận diện request trùng và quy trình đăng ký mẫu. Viết adapter Sender và test HTTP tương ứng; không đưa URL/token tùy ý từ frontend xuống backend.

### Zalo

Tạo OA/ứng dụng phù hợp, cấp quyền gửi tin qua số điện thoại và duyệt mẫu xác thực. Lấy token đại diện OA. Adapter hiện hỗ trợ mẫu có một tham số OTP; đặt OTP_PARAM đúng tên tham số đã duyệt. [Quy trình kết nối](https://zalo.solutions/business-message/guidelines/en/gui-api)

```ini
[phone_delivery]
MODE = live
DEFAULT_CHANNEL = zalo
CHANNELS = zalo

[zalo]
ACCESS_TOKEN = <oa-access-token>
TEMPLATE_ID = <approved-authentication-template-id>
OTP_PARAM = otp
```

Request dùng `phone`, `template_id`, `template_data`, `tracking_id` và header access_token. HTTP 200 chưa đủ: adapter yêu cầu error=0 và msg_id. [Hợp đồng API chính thức](https://docs.zaloplatforms.com/docs/ZBS/gui-tin-template-qua-sdt/api-gui-tin-qua-sdt/api-gui-tin)

Token OA có hạn sử dụng; cần quy trình refresh/rotation. Bản hiện tại đọc token lúc khởi động, chưa tự refresh OAuth. Khi đổi token cần restart/redeploy. Trước khi vận hành liên tục, tích hợp refresh token vào hệ quản lý secret hoặc một token source có refresh; không chờ token hết hạn rồi để đăng nhập lỗi. [Vòng đời token](https://zalo.solutions/business-message/guidelines/en/gui-api)

Zalo còn có chế độ development gửi tới quản trị viên ứng dụng/OA. Adapter hiện tại dùng hợp đồng gửi thường; thử local bằng Mailpit, thử hợp đồng cổng bằng tài khoản/mẫu đã chuẩn bị. [Chế độ thử của Zalo](https://docs.zaloplatforms.com/docs/ZBS/gui-tin-template-qua-sdt/api-gui-tin-qua-sdt/api-gui-tin-su-dung-development-mode)

### Telegram Gateway

Tạo tài khoản Telegram Gateway, lấy Gateway access token, chuẩn bị billing theo tài khoản và kiểm thử số của bạn. Không dùng token BotFather.

```ini
[phone_delivery]
MODE = live
DEFAULT_CHANNEL = telegram
CHANNELS = telegram

[telegram_gateway]
ACCESS_TOKEN = <gateway-access-token>
```

Adapter gửi phone_number E.164, code và ttl=300 tới sendVerificationMessage; kiểm tra cả HTTP và ok/result.request_id. Người nhận phải dùng số đó trên Telegram. Không cần chat_id, bot hoặc liên kết chat cho luồng Gateway này. [API Gateway](https://core.telegram.org/gateway/api)

## Cấu hình credentials

Ví dụ production bật đủ ba kênh:

```ini
[phone_delivery]
MODE = live
DEFAULT_CHANNEL = sms
CHANNELS = sms,zalo,telegram
```

Các section [sms], [zalo], [telegram_gateway] tương ứng vẫn phải cấu hình. Production cũng cần các cấu hình HTTPS, database, session và Google callback phù hợp của ứng dụng; đổi MODE của phone_delivery không thay thế việc cấu hình toàn bộ server.

Không commit credentials. Repo đã hỗ trợ biến môi trường hoặc đọc từ secret file, ví dụ:

```bash
export BDS__SMS__AUTH_TOKEN__FILE=/run/secrets/twilio_auth_token
export BDS__ZALO__ACCESS_TOKEN__FILE=/run/secrets/zalo_access_token
export BDS__TELEGRAM_GATEWAY__ACCESS_TOKEN__FILE=/run/secrets/telegram_gateway_token
```

Các file trên do bạn/deployment tạo với quyền đọc phù hợp; code không tạo chúng. Giá trị được đọc khi startup.

## Cách triển khai hiệu quả cho nhóm nhỏ

1. Hằng ngày: Mailpit và fake HTTP trong test, không phụ thuộc credentials hay phí gửi.
2. CI: chạy test contract cho từng adapter, test schema/config và build frontend.
3. Staging: bật từng cổng thật, thử số do nhóm kiểm soát; ghi nhận tỷ lệ nhận mã và thời gian nhận thực tế.
4. Mở cho người dùng: cho chọn các kênh đã kiểm chứng. Với khách hàng Việt Nam, đánh giá SMS và Zalo trên nhóm khách hàng mục tiêu; Telegram là lựa chọn bổ sung khi người dùng có tài khoản. Đây là khuyến nghị triển khai, không giả định ai cũng có một ứng dụng cụ thể.
5. Chỉ thêm hàng đợi/fallback tự động khi có nhu cầu và có thiết kế chống gửi trùng. Trước khi chọn cổng dựa trên giá, so sánh chi phí trên một lượt xác minh thành công, độ trễ và khả năng hỗ trợ khi lỗi.

## Rate limit, lỗi và khả năng quan sát

Các giới hạn SMS_DESTINATION_COOLDOWN/HOUR/DAY, SMS_GLOBAL_DAY và SMS_VERIFY_DESTINATION cũ hiện áp dụng chung cho mọi kênh. Giữ tên config để tương thích. Đổi kênh không làm mới ngân sách gửi hay số lần thử OTP.

Client tự chọn kênh khác và resend sau cooldown. Không tự retry/fallback khi timeout: chưa biết provider đã nhận tin chưa, gửi lần hai có thể phát sinh phí hoặc hai mã khác nhau. Nhà cung cấp chấp nhận request không đồng nghĩa thiết bị đã nhận OTP.

Log `phone.delivery` có channel/mode/accepted, reason/provider_status/provider_code khi có lỗi, provider_message_id khi có. Dùng request_id từ HTTP response để đối chiếu. Không ghi OTP, số điện thoại hay token trong event này; không ghi nguyên văn lỗi/response của provider.

Gửi tin diễn ra sau khi DB commit challenge/rotate OTP, kế thừa luồng hiện có. Nếu cổng lỗi, API trả 503; challenge có thể vẫn tồn tại và mã cũ sau resend có thể đã mất hiệu lực. DB và nhà cung cấp không có transaction chung; bản này không cam kết gửi đúng một lần. Người dùng đợi cooldown rồi yêu cầu mã mới.

Bản này chưa lưu delivery receipts vào DB, chưa nhận webhook, chưa tự refresh Zalo token, chưa quản lý ngân sách tiền theo giá từng nhà cung cấp. Shared rate limit và giới hạn số request đang xử lý đã có. Khi cần số liệu giao tin/billing chính xác, thêm lưu receipt và webhook có xác minh theo tài liệu từng cổng; không chỉ đếm HTTP 202.

## Kiểm chứng

```bash
go test ./...
go vet ./...
make swagger-check
cd frontend
npm test
npm run build
```

Kiểm thử local từ root repo khi PostgreSQL, Redis và Mailpit đang chạy:

```bash
python3 scripts/test-phone-delivery.py
```

Script biên dịch backend vào thư mục tạm, chọn cổng trống, buộc mọi kênh qua Mailpit và tự dừng tiến trình. Nó kiểm tra gửi/verify ba kênh, mã sai, dùng lại mã sau verify, cooldown chung, đổi SMS sang Zalo sau cooldown và mã cũ sau resend. Có chờ khoảng 61 giây để kiểm tra cooldown thực. Chỉ tạo challenge tạm và tin Mailpit; không hoàn tất tạo tài khoản, không gửi SMS/Zalo/Telegram thật. Test dùng config local và bỏ các BDS__ overrides từ shell để tránh vô tình dùng cổng thật.

Contract tests dùng fake HTTP kiểm tra cấu trúc request và xử lý lỗi; chúng không chứng minh credentials/template thật đã được nhà cung cấp phê duyệt. Cần kiểm thử live sau khi có tài khoản và secret.
