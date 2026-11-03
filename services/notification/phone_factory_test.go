package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

func TestNewPhoneCodeSenderPolicy(t *testing.T) {
	tests := []struct {
		name      string
		cfg       setting.Config
		wantKind  string
		wantError string
	}{
		{name: "development is local", cfg: setting.Config{Environment: "development", PhoneDelivery: "sms"}, wantKind: "local"},
		{name: "production disabled", cfg: setting.Config{Environment: "production", PhoneDelivery: "disabled"}},
		{name: "production empty", cfg: setting.Config{Environment: "production"}},
		{name: "SMS requires HTTPS", cfg: setting.Config{Environment: "production", PhoneDelivery: "sms", SMSGatewayURL: "http://example.com", SMSGatewayToken: "secret"}, wantError: "SMS_GATEWAY_URL"},
		{name: "SMS requires token", cfg: setting.Config{Environment: "production", PhoneDelivery: "sms", SMSGatewayURL: "https://example.com"}, wantError: "SMS_GATEWAY_URL"},
		{name: "SMS configured", cfg: setting.Config{Environment: "production", PhoneDelivery: "sms", SMSGatewayURL: "https://example.com", SMSGatewayToken: "secret"}, wantKind: "sms"},
		{name: "Zalo requires credentials", cfg: setting.Config{Environment: "production", PhoneDelivery: "zalo"}, wantError: "Zalo ZBS delivery"},
		{name: "Zalo configured", cfg: setting.Config{Environment: "production", PhoneDelivery: "zalo", ZaloAccessToken: "secret", ZaloTemplateID: "template", ZaloCodeField: "otp"}, wantKind: "zalo"},
		{name: "unknown provider rejected", cfg: setting.Config{Environment: "production", PhoneDelivery: "unknown"}, wantError: "unsupported PHONE_DELIVERY"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sender, err := NewPhoneCodeSender(tc.cfg)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v; want substring %q", err, tc.wantError)
				}
				if sender != nil {
					t.Fatalf("unexpected sender on error: %T", sender)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch tc.wantKind {
			case "":
				if sender != nil {
					t.Fatalf("unexpected provider %T", sender)
				}
			case "local":
				if _, ok := sender.(Local); !ok {
					t.Fatalf("provider = %T, want Local", sender)
				}
			case "sms":
				s, ok := sender.(SMSGateway)
				if !ok || s.Client == nil || s.Client.Timeout != 8*time.Second {
					t.Fatalf("provider = %#v, want configured SMSGateway", sender)
				}
			case "zalo":
				s, ok := sender.(ZaloZBS)
				if !ok || s.Client == nil || s.Client.Timeout != 8*time.Second {
					t.Fatalf("provider = %#v, want configured ZaloZBS", sender)
				}
			}
		})
	}
}
