package audit

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRecordSessionRevokedRequiresTransaction(t *testing.T) {
	if err := RecordSessionRevoked(context.Background(), 42, uuid.New()); err == nil {
		t.Fatal("audit insert accepted outside transaction")
	}
}

func TestRecordSessionRevokedRejectsInvalidContext(t *testing.T) {
	if err := RecordSessionRevoked(nil, 42, uuid.New()); err == nil {
		t.Fatal("nil context accepted")
	}
}
