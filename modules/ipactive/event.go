package ipactive

import "github.com/datvtph41107/bdspro/modules/timeutil"

type Event struct {
	Action     Action
	SourceID   string
	Mode       Mode
	Allowed    bool
	Attempts   int64
	Limit      int64
	RetryAfter int64
	OccurredAt timeutil.TimeStamp
}
