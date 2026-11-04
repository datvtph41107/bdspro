package timeutil

import (
	"fmt"
	"time"
)

// Since returns a compact relative time for diagnostic and activity displays.
// The time units are intentionally locale-neutral; translated UI belongs at
// the presentation layer. Calendar months are not approximated as 30 days.
func Since(then, now time.Time) string {
	delta := now.Sub(then)
	future := delta < 0
	if future {
		delta = -delta
	}
	var text string
	switch {
	case delta < time.Second:
		text = "now"
	case delta < time.Minute:
		text = fmt.Sprintf("%ds", int64(delta/time.Second))
	case delta < time.Hour:
		text = fmt.Sprintf("%dm", int64(delta/time.Minute))
	case delta < 24*time.Hour:
		text = fmt.Sprintf("%dh", int64(delta/time.Hour))
	default:
		text = fmt.Sprintf("%dd", int64(delta/(24*time.Hour)))
	}
	if text == "now" {
		return text
	}
	if future {
		return "in " + text
	}
	return text + " ago"
}
