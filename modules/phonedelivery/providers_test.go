package phonedelivery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type sender interface {
	Send(context.Context, Message) (Receipt, error)
}

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderContracts(t *testing.T) {
	const secret = "secret-never-log-this"
	message := Message{Phone: "+84912345678", Code: "001234"}
	cases := []struct {
		name, host, success string
		sender              func(*http.Client) sender
		check               func(*testing.T, *http.Request)
	}{
		{"sms", "api.twilio.com", `{"sid":"SM123"}`,
			func(c *http.Client) sender {
				return Twilio{client: c, cfg: TwilioConfig{AccountSID: "AC123", AuthToken: secret, SenderID: "BDSPro"}}
			},
			func(t *testing.T, r *http.Request) {
				u, p, ok := r.BasicAuth()
				if !ok || u != "AC123" || p != secret {
					t.Fatal("missing auth")
				}
				if err := r.ParseForm(); err != nil {
					t.Fatal(err)
				}
				if r.Form.Get("To") != message.Phone || !strings.Contains(r.Form.Get("Body"), message.Code) || r.Form.Get("From") != "BDSPro" {
					t.Fatal("wrong SMS mapping")
				}
			}},
		{"zalo", "business.openapi.zalo.me", `{"error":0,"data":{"msg_id":"zalo123"}}`,
			func(c *http.Client) sender {
				return Zalo{client: c, token: secret, templateID: "approved123", otpParam: "verification_code"}
			},
			func(t *testing.T, r *http.Request) {
				if r.Header.Get("access_token") != secret || r.URL.Path != "/message/template" {
					t.Fatal("wrong Zalo endpoint/auth")
				}
				var b struct {
					Phone, TrackingID string
					TemplateID        string
					Data              map[string]string
				}
				var raw map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
					t.Fatal(err)
				}
				_ = json.Unmarshal(raw["phone"], &b.Phone)
				_ = json.Unmarshal(raw["template_id"], &b.TemplateID)
				_ = json.Unmarshal(raw["tracking_id"], &b.TrackingID)
				_ = json.Unmarshal(raw["template_data"], &b.Data)
				if b.Phone != "84912345678" || b.TemplateID != "approved123" || b.Data["verification_code"] != message.Code || len(b.TrackingID) != 32 {
					t.Fatalf("wrong Zalo payload: %+v", b)
				}
			}},
		{"telegram", "gatewayapi.telegram.org", `{"ok":true,"result":{"request_id":"telegram123"}}`,
			func(c *http.Client) sender { return TelegramGateway{client: c, token: secret} },
			func(t *testing.T, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer "+secret || r.URL.Path != "/sendVerificationMessage" {
					t.Fatal("must use Gateway, not Bot API")
				}
				var b map[string]any
				if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
					t.Fatal(err)
				}
				if b["phone_number"] != message.Phone || b["code"] != message.Code || b["ttl"] != float64(300) {
					t.Fatal("wrong Gateway payload")
				}
				if _, ok := b["code_length"]; ok {
					t.Fatal("must preserve application-owned OTP")
				}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := 200
			if tc.name == "sms" {
				status = 201
			}
			count := 0
			c := &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
				count++
				if r.Method != "POST" || r.URL.Scheme != "https" || r.URL.Host != tc.host || r.URL.RawQuery != "" {
					t.Fatal("unsafe provider request")
				}
				tc.check(t, r)
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(tc.success))}, nil
			})}
			got, err := tc.sender(c).Send(context.Background(), message)
			if err != nil || got.MessageID == "" || count != 1 {
				t.Fatalf("receipt=%+v err=%v calls=%d", got, err, count)
			}
			for _, bad := range []struct {
				status int
				body   string
			}{
				{200, `{"error":-124,"ok":false,"message":"secret-never-log-this"}`},
				{429, `{"error":-1,"code":20429,"message":"secret-never-log-this"}`},
				{500, "secret-never-log-this"},
				{200, "{}"},
				{200, strings.Repeat("x", maxResponseBytes+1)},
			} {
				count = 0
				c.Transport = testTransport(func(*http.Request) (*http.Response, error) {
					count++
					return &http.Response{StatusCode: bad.status, Body: io.NopCloser(strings.NewReader(bad.body))}, nil
				})
				_, err = tc.sender(c).Send(context.Background(), message)
				if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), secret) || count != 1 {
					t.Fatalf("must fail safely without automatic retry: %v calls=%d", err, count)
				}
			}
			c.Transport = testTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New(secret) })
			_, err = tc.sender(c).Send(context.Background(), message)
			if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), secret) {
				t.Fatalf("unsafe transport error: %v", err)
			}
		})
	}
}
