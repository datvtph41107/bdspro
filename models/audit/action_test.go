package audit

import (
	"context"
	"errors"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

type actionCapture struct {
	seen []db.InsertAuditActionParams
	err  error
}

func (s *actionCapture) InsertAuditAction(_ context.Context, p db.InsertAuditActionParams) error {
	if s.err != nil {
		return s.err
	}
	s.seen = append(s.seen, p)
	return nil
}

func TestRecordActionScopeAndTargets(t *testing.T) {
	store := &actionCapture{}
	session := uuid.New()
	if err := RecordSessionRevoked(context.Background(), store, 7, session); err != nil {
		t.Fatal(err)
	}
	if err := RecordTeamMemberRemoved(context.Background(), store, 7, 12, 55); err != nil {
		t.Fatal(err)
	}
	if len(store.seen) != 2 ||
		store.seen[0].Action != ActionSessionRevoked || store.seen[0].ActorAccountID != 7 ||
		store.seen[0].ScopeType != "account" || store.seen[0].ScopeRef != "7" ||
		store.seen[0].TargetType != "session" || store.seen[0].TargetRef != session.String() ||
		store.seen[1].Action != ActionTeamMemberRemoved || store.seen[1].ActorAccountID != 7 ||
		store.seen[1].ScopeType != "team" || store.seen[1].ScopeRef != "12" ||
		store.seen[1].TargetType != "person" || store.seen[1].TargetRef != "55" {
		t.Fatalf("wrong audited actor, scope or target: %+v", store.seen)
	}
	if err := RecordSessionRevoked(context.Background(), store, 0, session); err == nil {
		t.Fatal("anonymous actor was accepted for session revocation")
	}
	if err := RecordTeamMemberRemoved(context.Background(), store, 7, 0, 5); err == nil {
		t.Fatal("invalid team scope was accepted")
	}
	if len(store.seen) != 2 {
		t.Fatal("invalid action wrote an audit event")
	}
	store.err = errors.New("audit DB unavailable")
	if err := RecordSessionRevoked(context.Background(), store, 7, session); !errors.Is(err, store.err) {
		t.Fatalf("audit failure not forwarded for transactional rollback: %v", err)
	}
}
