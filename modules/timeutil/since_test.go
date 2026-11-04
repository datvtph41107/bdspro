package timeutil

import (
    "testing"
    "time"
)

func TestSince(t *testing.T) {
    now := time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)
    cases := []struct { diff time.Duration; want string }{
        {0, "now"},
        {time.Millisecond, "now"},
        {time.Second, "1s ago"},
        {59*time.Second, "59s ago"},
        {time.Minute, "1m ago"},
        {time.Hour, "1h ago"},
        {24*time.Hour, "1d ago"},
        {-5*time.Minute, "in 5m"},
    }
    for _, tc := range cases {
        if got := Since(now.Add(-tc.diff), now); got != tc.want {
            t.Errorf("Since(%s) = %q; want %q", tc.diff, got, tc.want)
        }
    }
}
