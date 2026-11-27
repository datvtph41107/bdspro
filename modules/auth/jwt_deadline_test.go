package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSessionAccessTokenDeadline(t *testing.T) {
	maker, err := NewJWTMaker("disposable-test-key-not-for-production")
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	for _, tc := range []struct {
		name     string
		duration time.Duration
		deadline time.Time
		expired  bool
	}{
		{"policy shorter", time.Minute, time.Now().Add(time.Hour), false},
		{"session shorter", time.Hour, time.Now().Add(time.Minute), false},
		{"session expired", time.Hour, time.Now().Add(-time.Minute), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, payload, err := maker.CreateSessionAccessToken(1, tc.duration, sessionID, tc.deadline)
			if tc.expired {
				if !errors.Is(err, ErrExpiredToken) || raw != "" || payload != nil {
					t.Fatal("expired session issued a token")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := payload.IssuedAt.Add(tc.duration)
			if tc.deadline.Before(want) {
				want = tc.deadline
			}
			if !payload.ExpiredAt.Equal(want) {
				t.Fatalf("deadline = %v, want %v", payload.ExpiredAt, want)
			}
			decoded, err := maker.VerifyAccessToken(raw)
			if err != nil {
				t.Fatal(err)
			}
			if !decoded.ExpiredAt.Equal(want) || decoded.SessionID != sessionID {
				t.Fatal("signed token does not match canonical deadline/session")
			}
		})
	}
}
