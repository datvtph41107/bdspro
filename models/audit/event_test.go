package audit

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
)

type auditStoreFake struct {
	entries []db.InsertAuditEventParams
	err error
}

func (s *auditStoreFake) InsertAuditEvent(_ context.Context, arg db.InsertAuditEventParams) error {
	if s.err != nil {
		return s.err
	}
	s.entries = append(s.entries, arg)
	return nil
}

func TestRecordLoginRejectedMACAndScope(t *testing.T) {
	store := &auditStoreFake{}
	key := strings.Repeat("k", 32)
	event := LoginRejected{
		Method: MethodPassword, Reason: ReasonUnknownAccount,
		Identifier: " Alice@Example.Test ", IP: " 203.0.113.21 ",
	}
	if err := RecordLoginRejected(context.Background(), store, key, event); err != nil {
		t.Fatal(err)
	}
	first := store.entries[0]
	if first.AccountID.Valid || len(first.SubjectMac) != 32 || first.IpAddress != "203.0.113.21" {
		t.Fatalf("anonymous audit event has wrong account or IP: %+v", first)
	}
	event.Identifier = "alice@example.test"
	if err := RecordLoginRejected(context.Background(), store, key, event); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.SubjectMac, store.entries[1].SubjectMac) {
		t.Fatal("email should normalize to the same HMAC")
	}
	event.Method = MethodOIDC
	event.Reason = ReasonInvalidIdentity
	event.Identifier = "https://issuer.example.test\x00Subject-A"
	if err := RecordLoginRejected(context.Background(), store, key, event); err != nil {
		t.Fatal(err)
	}
	event.Identifier = "https://issuer.example.test\x00subject-a"
	if err := RecordLoginRejected(context.Background(), store, key, event); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(store.entries[2].SubjectMac, store.entries[3].SubjectMac) {
		t.Fatal("OIDC subject is case-sensitive")
	}
}

func TestRecordLoginRejectedInvalidKeyAndDatabaseError(t *testing.T) {
	store := &auditStoreFake{}
	event := LoginRejected{
		AccountID: 42, Method: MethodPhoneOTP, Reason: ReasonInvalidOTP,
		Identifier: "+84901001001", IP: "malformed-ip",
	}
	if err := RecordLoginRejected(context.Background(), store, "", event); !errors.Is(err, ErrMissingKey) {
		t.Fatalf("missing key accepted: %v", err)
	}
	if len(store.entries) != 0 {
		t.Fatal("weak key allowed a write")
	}
	key := strings.Repeat("k", 32)
	if err := RecordLoginRejected(context.Background(), store, key, event); err != nil {
		t.Fatal(err)
	}
	if !store.entries[0].AccountID.Valid || store.entries[0].AccountID.Int64 != 42 || store.entries[0].IpAddress != "" {
		t.Fatalf("account scope or IP normalization incorrect: %+v", store.entries[0])
	}
	store.err = errors.New("postgres unavailable")
	if err := RecordLoginRejected(context.Background(), store, key, event); !errors.Is(err, store.err) {
		t.Fatalf("audit store errors must be returned for service policy: %v", err)
	}
}
