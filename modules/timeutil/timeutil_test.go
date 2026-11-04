package timeutil

import (
	"testing"
	"time"
)

func TestSecondsCeil(t *testing.T) {
	cases := []struct {
		duration time.Duration
		want     int64
	}{
		{duration: -time.Second, want: 0},
		{duration: 0, want: 0},
		{duration: time.Nanosecond, want: 1},
		{duration: time.Second - time.Nanosecond, want: 1},
		{duration: time.Second, want: 1},
		{duration: time.Second + time.Nanosecond, want: 2},
		{duration: 5 * time.Minute, want: 300},
	}
	for _, tc := range cases {
		if got := SecondsCeil(tc.duration); got != tc.want {
			t.Fatalf("SecondsCeil(%v) = %d, want %d", tc.duration, got, tc.want)
		}
	}
}

func TestExpiredIncludesExactDeadline(t *testing.T) {
	now := time.Date(2026, time.November, 4, 10, 0, 0, 0, time.UTC)
	if Expired(now, now.Add(time.Nanosecond)) {
		t.Fatal("early expiration")
	}
	if !Expired(now, now) {
		t.Fatal("exact expiry boundary accepted")
	}
	if !Expired(now, now.Add(-time.Nanosecond)) {
		t.Fatal("past expiry accepted")
	}
}
