package auth

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
)

type rejectedProofCapture struct {
	entries []db.InsertAuthenticationFailureParams
	err     error
}

func (s *rejectedProofCapture) InsertAuthenticationFailure(_ context.Context, arg db.InsertAuthenticationFailureParams) error {
	if s.err != nil {
		return s.err
	}
	s.entries = append(s.entries, arg)
	return nil
}

func TestAuthenticationFailureMACNormalizesWithoutPlainCredential(t *testing.T) {
	store := &rejectedProofCapture{}
	key := strings.Repeat("k", 32)
	proof := RejectedProof{
		Method: FailurePassword, Reason: FailureUnknownAccount,
		Credential: " Alice@Example.Test ", ClientIP: " 203.0.113.21 ",
	}
	if err := RecordAuthenticationFailure(context.Background(), store, key, proof); err != nil {
		t.Fatal(err)
	}
	if len(store.entries) != 1 {
		t.Fatal("missing rejection")
	}
	first := store.entries[0]
	if first.AccountID.Valid || len(first.CredentialMac) != 32 || first.IpAddress != "203.0.113.21" {
		t.Fatalf("unexpected anonymous proof metadata: %+v", first)
	}
	if bytes.Contains(first.CredentialMac, []byte("alice")) {
		t.Fatal("raw identifier persisted in credential MAC")
	}
	proof.Credential = "alice@example.test"
	if err := RecordAuthenticationFailure(context.Background(), store, key, proof); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.CredentialMac, store.entries[1].CredentialMac) {
		t.Fatal("normalized credential must have stable keyed MAC")
	}
	proof.Method = FailureOIDC
	proof.Reason = FailureBadIdentity
	if err := RecordAuthenticationFailure(context.Background(), store, key, proof); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first.CredentialMac, store.entries[2].CredentialMac) {
		t.Fatal("credential MAC must be domain-separated by method")
	}
}

func TestAuthenticationFailureKeyAndAccountScoping(t *testing.T) {
	store := &rejectedProofCapture{}
	proof := RejectedProof{
		AccountID: 42, Method: FailurePhoneOTP, Reason: FailureBadOTP,
		Credential: "+84901001001", ClientIP: "malformed-ip",
	}
	if err := RecordAuthenticationFailure(context.Background(), store, "", proof); !errors.Is(err, errFailureAuditKey) {
		t.Fatalf("missing audit key must reject writes: %v", err)
	}
	if len(store.entries) != 0 {
		t.Fatal("weak or absent MAC key wrote a record")
	}
	if err := RecordAuthenticationFailure(context.Background(), store, strings.Repeat("k", 32), proof); err != nil {
		t.Fatal(err)
	}
	row := store.entries[0]
	if !row.AccountID.Valid || row.AccountID.Int64 != 42 || row.IpAddress != "" {
		t.Fatalf("known account/invalid IP not normalized: %+v", row)
	}
	store.err = errors.New("PG unavailable")
	if err := RecordAuthenticationFailure(context.Background(), store, strings.Repeat("k", 32), proof); !errors.Is(err, store.err) {
		t.Fatalf("database error should reach service policy: %v", err)
	}
}
