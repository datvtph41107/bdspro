package notification

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Provider acknowledgement is a transport result, not proof of phone ownership.
// A transient delivery error must never be reported as successful delivery.
func TestSMSGatewayTransientFailureThenRecovery(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("unexpected SMS gateway authorization: %q", got)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	sender := SMSGateway{URL: server.URL, Token: "test-token", Client: server.Client()}
	if err := sender.SendCode(context.Background(), "+84912345678", "123456"); err == nil {
		t.Fatal("HTTP 429 must remain a delivery failure")
	}
	if err := sender.SendCode(context.Background(), "+84912345678", "123456"); err != nil {
		t.Fatalf("subsequent provider recovery: %v", err)
	}
	if requests.Load() != 2 {
		t.Fatalf("unexpected SMS attempts: %d", requests.Load())
	}
}

func TestSMSGatewayHonorsContextDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	sender := SMSGateway{URL: server.URL, Token: "test-token", Client: server.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := sender.SendCode(ctx, "+84912345678", "123456")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("delivery timeout must propagate: %v", err)
	}
	if strings.Contains(err.Error(), "123456") {
		t.Fatal("provider error leaked OTP")
	}
}

func TestZaloDeliveryRejectsMalformedAndMissingAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"missing status", `{"message":"accepted"}`},
		{"malformed JSON", `not-json`},
		{"provider rejected", `{"error":-100,"message":"rejected"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			if err := postJSON(context.Background(), server.Client(), server.URL,
				"access_token", "test", []byte("{}"), true); err == nil {
				t.Fatal("provider response must not count as accepted")
			}
		})
	}
}
