package organization

import (
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestMemberFromDBOwnsMembershipProjection(t *testing.T) {
	joined := time.Date(2026, 11, 9, 11, 0, 0, 0, time.UTC)
	ended := joined.Add(time.Hour)
	row := db.OrganizationMember{
		ID: 11, OrganizationID: 12, PersonID: 13,
		Role: RoleOwner, JoinedAt: joined,
		EndedAt:       pgtype.Timestamptz{Time: ended, Valid: true},
		OwnerPersonID: pgtype.Int8{Int64: 13, Valid: true},
	}
	got := MemberFromDB(row)
	if got.ID != 11 || got.OrganizationID != 12 || got.PersonID != 13 ||
		got.Role != RoleOwner || got.JoinedAt != joined ||
		got.EndedAt == nil || !got.EndedAt.Equal(ended) {
		t.Fatalf("organization member data lost: %+v", got)
	}
	// A domain entity never retains a mutable alias to the SQLC row.
	row.Role = RoleMember
	row.EndedAt.Time = time.Time{}
	if got.Role != RoleOwner || !got.EndedAt.Equal(ended) {
		t.Fatalf("domain member aliases a generated SQLC result: %+v", got)
	}
}

func TestActiveMemberFromSQLCProjection(t *testing.T) {
	joined := time.Date(2026, 11, 9, 11, 0, 0, 0, time.UTC)
	row := db.ListOrganizationMembersForOwnerRow{
		ID: 21, OrganizationID: 22, PersonID: 23,
		Role: RoleMember, JoinedAt: joined,
	}
	got := memberFromListRow(row)
	if got.ID != row.ID || got.OrganizationID != row.OrganizationID ||
		got.PersonID != row.PersonID || got.Role != row.Role ||
		got.JoinedAt != row.JoinedAt || got.EndedAt != nil {
		t.Fatalf("active member domain projection changed: %+v", got)
	}
}
