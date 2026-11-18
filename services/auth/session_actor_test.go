package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeSessionReader struct {
	session db.Session
	err     error
	calls   int
}

func (f *fakeSessionReader) GetSession(
	_ context.Context,
	_ uuid.UUID,
) (db.Session, error) {
	f.calls++
	return f.session, f.err
}

func TestResolveSessionActorValid(t *testing.T) {
	id := uuid.New()
	now := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	reader := &fakeSessionReader{session: db.Session{
		ID: id, AccountID: 42, ExpiresAt: now.Add(time.Hour),
	}}
	actor, err := resolveSessionActor(context.Background(), reader, id, now)
	if err != nil {
		t.Fatal(err)
	}
	if actor.AccountID != 42 || actor.SessionID != id {
		t.Fatalf("wrong actor: %+v", actor)
	}
}

func TestResolveSessionActorRejectsInactive(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	cases := []struct {
		name    string
		session db.Session
		err     error
	}{
		{"missing", db.Session{}, pgx.ErrNoRows},
		{"revoked", db.Session{ID: id, AccountID: 42, IsBlocked: true, ExpiresAt: now.Add(time.Hour)}, nil},
		{"expired", db.Session{ID: id, AccountID: 42, ExpiresAt: now}, nil},
		{"wrong identity", db.Session{ID: uuid.New(), AccountID: 42, ExpiresAt: now.Add(time.Hour)}, nil},
		{"missing account", db.Session{ID: id, ExpiresAt: now.Add(time.Hour)}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeSessionReader{session: tc.session, err: tc.err}
			actor, err := resolveSessionActor(context.Background(), reader, id, now)
			if actor != nil || !errors.Is(err, util.ErrUnauthenticated) {
				t.Fatalf("actor=%+v err=%v", actor, err)
			}
		})
	}
}

func TestResolveSessionActorPreservesDatabaseError(t *testing.T) {
	id := uuid.New()
	outage := errors.New("database unavailable")
	reader := &fakeSessionReader{err: outage}
	actor, err := resolveSessionActor(context.Background(), reader, id, time.Now())
	if actor != nil || !errors.Is(err, outage) {
		t.Fatalf("actor=%+v err=%v", actor, err)
	}
	reader.calls = 0
	actor, err = resolveSessionActor(context.Background(), reader, uuid.Nil, time.Now())
	if actor != nil || !errors.Is(err, util.ErrUnauthenticated) || reader.calls != 0 {
		t.Fatalf("nil session must reject before DB: actor=%+v err=%v calls=%d", actor, err, reader.calls)
	}
}
