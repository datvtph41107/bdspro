package source

import "time"

// The accountID parameter must originate from a verified session, not request JSON.
type PriceInput struct {
	Amount      string
	Currency    string
	StatementAt string
	PriceBasis  string
	SourceBasis string
	Note        string
}
type CaptureInput struct {
	Channel      string
	Label        string
	AreaM2       string
	AreaBasis    string
	ReceivedAt   string
	Summary      string
	InitialPrice *PriceInput
}
type CaptureResult struct {
	ID       int64
	Replayed bool
}
type PriceResult struct {
	ID       int64
	Replayed bool
}
type Source struct {
	ID                  int64
	OrganizationID      int64
	RecordedByAccountID int64
	Channel             string
	Label               string
	AreaM2              string
	AreaBasis           string
	Summary             string
	Version             int32
	RecordedAt          time.Time
	ReceivedAt          *time.Time
}
type PriceReport struct {
	ID                  int64
	IntakeID            int64
	RecordedByAccountID int64
	Amount              string
	Currency            string
	PriceBasis          string
	SourceBasis         string
	Note                string
	RecordedAt          time.Time
	StatementAt         *time.Time
}

