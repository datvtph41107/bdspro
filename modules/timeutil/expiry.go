package timeutil

import "time"

// Expired treats an exact deadline as expired. Use in JWT and DB sessions.
func Expired(now, deadline time.Time) bool {
	return !now.Before(deadline)
}
