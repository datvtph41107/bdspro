package timeutil

import "time"

type TimeStamp int64

func TimeStampNow() TimeStamp {
	return TimeStamp(time.Now().Unix())
}

func TimeStampFromTime(value time.Time) TimeStamp {
	return TimeStamp(value.Unix())
}

func (ts TimeStamp) AsTime() time.Time {
	return time.Unix(int64(ts), 0).UTC()
}

func (ts TimeStamp) Add(seconds int64) TimeStamp {
	return ts + TimeStamp(seconds)
}

func (ts TimeStamp) AddDuration(interval time.Duration) TimeStamp {
	return ts.Add(int64(interval / time.Second))
}

func (ts TimeStamp) IsZero() bool {
	return ts == 0
}
