package ipactive

import "github.com/datvtph41107/bdspro/modules/timeutil"

// Event is a privacy-preserving result of one atomic admission attempt.
// Admission itself never depends on asynchronous events or queue delivery.
type Event struct {
    Action     Action
    SourceID   string // SHA-256 of the trusted IP, never the raw IP
    Mode       Mode
    Allowed    bool
    Attempts   int64
    Limit      int64
    RetryAfter int64 // whole seconds, only on rejection
    OccurredAt timeutil.TimeStamp // Unix seconds for structured audit consumers
}
