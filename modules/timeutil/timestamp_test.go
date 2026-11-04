package timeutil

import (
	"testing"
	"time"
)

func TestTimeStampRoundTripUTC(t *testing.T) {
	original := time.Date(2026, 11, 4, 12, 30, 40, 999, time.FixedZone("ICT", 7*3600))
	stamp := TimeStampFromTime(original)
	if stamp.AsTime().Unix() != original.Unix() || stamp.AsTime().Location() != time.UTC {
		t.Fatalf("timestamp roundtrip failed: %v -> %v", original, stamp.AsTime())
	}
	if stamp.Add(60).AsTime().Unix() != original.Unix()+60 {
		t.Fatal("Add did not apply seconds")
	}
	if stamp.AddDuration(1500*time.Millisecond) != stamp.Add(1) {
		t.Fatal("subsecond conversion policy changed")
	}
	if !TimeStamp(0).IsZero() || TimeStamp(1).IsZero() {
		t.Fatal("zero timestamp semantics changed")
	}
}
