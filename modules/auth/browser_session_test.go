package auth

import (
	"errors"
	"testing"
	"time"
)

func TestBrowserTimeBoundaries(t *testing.T) {
	codec, err := NewBrowserSessionCodec("disposable-browser-test-key-0123456789")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1800000000, 0)
	session := "12345678-1234-1234-1234-123456789abc"
	for _, tc := range []struct {
		name     string
		deadline time.Time
		want     time.Time
	}{
		{"short session", now.Add(time.Minute), now.Add(time.Minute)},
		{"long session", now.Add(24 * time.Hour), now.Add(8 * time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := codec.IssueCookie(session, now, tc.deadline)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := codec.ReadCookie(raw, tc.want.Add(-time.Second)); err != nil || got != session {
				t.Fatal("valid browser binding rejected", err)
			}
			if _, err := codec.ReadCookie(raw, tc.want); !errors.Is(err, ErrBrowserProofInvalid) {
				t.Fatal("expired browser cookie accepted", err)
			}
		})
	}
	csrf, err := codec.IssueCSRF(session, "test", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.VerifyCSRF(csrf, session, "test", now.Add(299*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := codec.VerifyCSRF(csrf, session, "test", now.Add(300*time.Second)); !errors.Is(err, ErrBrowserProofInvalid) {
		t.Fatal("expired CSRF accepted")
	}
	if err := codec.VerifyCSRF(csrf, session, "other", now); !errors.Is(err, ErrBrowserProofInvalid) {
		t.Fatal("wrong purpose accepted")
	}
}
