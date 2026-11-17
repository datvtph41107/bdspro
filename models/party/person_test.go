package party

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type partySQLStub struct {
	statements []string
	failAt int
	failure error
}

type partyIDRow struct {
	id int64
	err error
}

func (r partyIDRow) Scan(dest ...any) error {
	if r.err != nil { return r.err }
	*dest[0].(*int64) = r.id
	return nil
}

func (s *partySQLStub) QueryRow(_ context.Context, sql string, _ ...interface{}) pgx.Row {
	s.statements = append(s.statements, sql)
	if s.failAt == 1 { return partyIDRow{err: s.failure} }
	return partyIDRow{id: 42}
}

func (s *partySQLStub) Exec(_ context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
	s.statements = append(s.statements, sql)
	if len(s.statements) == s.failAt {
		return pgconn.CommandTag{}, s.failure
	}
	return pgconn.CommandTag{}, nil
}

func (s *partySQLStub) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	panic("unexpected Query call")
}

func TestCreatePersonWithPartyUsesCallerSQLCExecutor(t *testing.T) {
	store := &partySQLStub{}
	id, err := CreatePersonWithParty(context.Background(), db.New(store))
	if err != nil || id != 42 {
		t.Fatalf("party ID=%d error=%v", id, err)
	}
	if len(store.statements) != 3 {
		t.Fatalf("statements=%d want=3", len(store.statements))
	}
	for i, table := range []string{"parties", "persons", "person_profiles"} {
		if !strings.Contains(store.statements[i], table) {
			t.Errorf("statement %d did not target %s: %q", i, table, store.statements[i])
		}
	}
}

func TestCreatePersonWithPartyStopsAtFirstFailure(t *testing.T) {
	for _, failAt := range []int{1, 2, 3} {
		failAt := failAt
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			sentinel := errors.New("database failure")
			store := &partySQLStub{failAt: failAt, failure: sentinel}
			id, err := CreatePersonWithParty(context.Background(), db.New(store))
			if id != 0 || !errors.Is(err, sentinel) {
				t.Fatalf("id=%d err=%v", id, err)
			}
			if len(store.statements) != failAt {
				t.Fatalf("statements=%d want=%d", len(store.statements), failAt)
			}
		})
	}
}
