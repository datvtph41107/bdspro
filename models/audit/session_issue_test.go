package audit

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRecordSessionIssuedRequiresTransaction(t *testing.T) {
	if err := RecordSessionIssued(context.Background(), 42, uuid.New()); err == nil {
		t.Fatal("session issuance audit accepted outside DB transaction")
	}
	if err := RecordSessionIssued(nil, 42, uuid.New()); err == nil {
		t.Fatal("session issuance audit accepted nil context")
	}
}
