package timeutil

import "time"

func Expired(now, deadline time.Time) bool {
	return !now.Before(deadline)
}
