package timeutil

import "time"

// TimeStamp represents Unix time in whole seconds, for external event and
// protocol boundaries. Database TIMESTAMPTZ values remain time.Time.
type TimeStamp int64

// TimeStampNow returns the current Unix timestamp in seconds.
func TimeStampNow() TimeStamp {
    return TimeStamp(time.Now().Unix())
}

// TimeStampFromTime converts a time.Time to a seconds-based timestamp.
func TimeStampFromTime(value time.Time) TimeStamp {
    return TimeStamp(value.Unix())
}

// AsTime returns the timestamp in UTC.
func (ts TimeStamp) AsTime() time.Time {
    return time.Unix(int64(ts), 0).UTC()
}

// Add adds an interval expressed in whole seconds.
func (ts TimeStamp) Add(seconds int64) TimeStamp {
    return ts + TimeStamp(seconds)
}

// AddDuration adds a duration truncated toward zero to whole seconds.
// Use time.Time directly for sub-second expiry and security decisions.
func (ts TimeStamp) AddDuration(interval time.Duration) TimeStamp {
    return ts.Add(int64(interval / time.Second))
}

// IsZero reports an unset timestamp (0), not the Unix epoch's validity.
func (ts TimeStamp) IsZero() bool {
    return ts == 0
}
