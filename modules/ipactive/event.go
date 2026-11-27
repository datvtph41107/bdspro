package ipactive

type Event struct {
	Action     Action
	SourceID   string
	Mode       Mode
	Allowed    bool
	Attempts   int64
	Limit      int64
	RetryAfter int64
	OccurredAt int64 // Unix seconds.
}
