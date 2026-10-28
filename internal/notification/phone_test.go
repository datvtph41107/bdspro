package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSMSGatewaySendsCodeToConfiguredGateway(t *testing.T) {
	const token = "sms-token"
	var received bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("invalid SMS gateway authentication or method")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body struct {
			Phone   string `json:"phone"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.Phone != "+84912345678" || !strings.Contains(body.Message, "123456") {
			t.Errorf("invalid gateway payload")
		}
		received = true
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	gateway := SMSGateway{URL: server.URL, Token: token, Client: server.Client()}
	if err := gateway.SendCode(context.Background(), "+84912345678", "123456"); err != nil {
		t.Fatal(err)
	}
	if !received {
		t.Fatal("gateway did not receive OTP")
	}
}

func TestZaloZBSProviderAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		response  string
		wantError bool
	}{
		{"accepted", http.StatusOK, `{"error":0}`, false},
		{"provider rejected", http.StatusOK, `{"error":-100}`, true},
		{"HTTP unavailable", http.StatusBadGateway, `{"error":0}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			// postJSON is the transport common to SMS and Zalo. ZaloZBS uses
			// the fixed, official API endpoint at runtime.
			err := postJSON(context.Background(), server.Client(), server.URL,
				"access_token", "fake-token", []byte("{}"), true)
			if (err != nil) != tc.wantError {
				t.Fatalf("err=%v, wantError=%v", err, tc.wantError)
			}
		})
	}
}
