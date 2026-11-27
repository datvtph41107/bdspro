package timeutil

import "time"

// SecondsCeil rounds a positive duration up to whole seconds (for example,
// 500ms becomes 1s). Non-positive durations return zero.
// Division before addition avoids overflowing time.Duration at its upper bound.
func SecondsCeil(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	seconds := int64(duration / time.Second)
	if duration%time.Second != 0 {
		seconds++
	}
	return seconds
}
