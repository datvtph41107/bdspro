package auth

import (
	"strings"
	"testing"
	"time"
)

const sampleBrowserSessionID = "123e4567-e89b-42d3-a456-426614174000"

func TestBrowserSessionCookie(t *testing.T) {
	codec, err := NewBrowserSessionCodec(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)
	token, err := codec.IssueCookie(sampleBrowserSessionID, now, now.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	id, err := codec.ReadCookie(token, now.Add(time.Hour))
	if err != nil || id != sampleBrowserSessionID {
		t.Fatalf("valid cookie %s %v", id, err)
	}
	for _, bad := range []string{
		token + "x", strings.Replace(token, "123e4567", "223e4567", 1),
		strings.Replace(token, "v1.", "v2.", 1), "not-a-cookie", "", strings.Repeat("x", 257),
	} {
		if _, err := codec.ReadCookie(bad, now); err == nil {
			t.Errorf("accepted malformed cookie %q", bad)
		}
	}
	if _, err := codec.ReadCookie(token, now.Add(BrowserSessionTTL+time.Second)); err == nil {
		t.Fatal("expired cookie accepted")
	}
	other, _ := NewBrowserSessionCodec(strings.Repeat("q", 32))
	if _, err := other.ReadCookie(token, now); err == nil {
		t.Fatal("wrong key accepted")
	}
	if _, err := codec.IssueCookie("not-a-uuid", now, now.Add(time.Hour)); err == nil {
		t.Fatal("invalid session ID accepted")
	}
	if _, err := codec.IssueCookie(sampleBrowserSessionID, now, now); err == nil {
		t.Fatal("expired DB session accepted")
	}
	expToken, err := codec.IssueCookie(sampleBrowserSessionID, now, now.Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.ReadCookie(expToken, now.Add(10*time.Minute)); err == nil {
		t.Fatal("cookie survived DB session expiry")
	}
}
func TestBrowserCSRFBoundToSessionAndOperation(t *testing.T) {
	codec, err := NewBrowserSessionCodec(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)
	proof, err := codec.IssueCSRF(sampleBrowserSessionID, "oauth-consent", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.VerifyCSRF(proof, sampleBrowserSessionID, "oauth-consent", now); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		context, purpose, proof string
		when                    time.Time
	}{
		{"other-session", "oauth-consent", proof, now},
		{sampleBrowserSessionID, "oauth-login", proof, now},
		{sampleBrowserSessionID, "oauth-consent", proof + "x", now},
		{sampleBrowserSessionID, "oauth-consent", proof, now.Add(BrowserCSRFTokenTTL + time.Second)},
	} {
		if err := codec.VerifyCSRF(tc.proof, tc.context, tc.purpose, tc.when); err == nil {
			t.Errorf("accepted forged/mismatched proof %#v", tc)
		}
	}
	another, err := codec.IssueCSRF("anonymous-cookie-nonce", "oauth-login", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.VerifyCSRF(another, "anonymous-cookie-nonce", "oauth-login", now); err != nil {
		t.Fatal(err)
	}
	if err := codec.VerifyCSRF(another, "anonymous-cookie-nonce", "oauth-consent", now); err == nil {
		t.Fatal("reused login CSRF for consent")
	}
}
func TestBrowserCodecRejectsWeakKey(t *testing.T) {
	if _, err := NewBrowserSessionCodec("dev"); err == nil {
		t.Fatal("weak key accepted")
	}
}
