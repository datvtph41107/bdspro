package sms

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTwilioCodeRequest(t *testing.T) {
	cfg := setting.SMSSettings{AccountSID: "AC" + strings.Repeat("a", 32), AuthToken: "test-only-token", SenderID: "BDSPro"}
	for _, status := range []int{201, 429, 500} {
		httpClient := &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.Scheme != "https" || r.URL.Host != "api.twilio.com" || r.Method != http.MethodPost {
				t.Fatal("unexpected provider endpoint")
			}
			u, p, ok := r.BasicAuth()
			if !ok || u != cfg.AccountSID || p != cfg.AuthToken {
				t.Fatal("missing provider credentials")
			}
			if err := r.ParseForm(); err != nil || r.Form.Get("To") != "+84912345678" ||
				r.Form.Get("From") != cfg.SenderID || !strings.Contains(r.Form.Get("Body"), "123456") {
				t.Fatal("invalid message request")
			}
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
		})}
		err := sendCode(context.Background(), httpClient, cfg, "+84912345678", "123456")
		if (err == nil) != (status == 201) {
			t.Fatal("provider status incorrectly classified")
		}
	}
}
