package organization

import (
	"errors"
	"testing"
	"time"
)

func TestTeamInviteDuration(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want time.Duration
	}{
		{0, 24 * time.Hour}, {300, 5 * time.Minute}, {86400, 24 * time.Hour},
	} {
		got, err := TeamInviteDuration(tc.in)
		if err != nil || got != tc.want {
			t.Fatal("unexpected invitation duration")
		}
	}
	for _, invalid := range []int{-1, 299, 86401} {
		if _, err := TeamInviteDuration(invalid); !errors.Is(err, ErrTeamInviteDuration) {
			t.Fatal("invalid invitation duration accepted")
		}
	}
}
