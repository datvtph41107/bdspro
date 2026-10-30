package handler

import (
	"reflect"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/service"
)

func TestOrganizationAccessResourcePermissions(t *testing.T) {
	tests := []struct {
		role string
		want []string
	}{
		{role: "owner", want: []string{
			"organization.view",
			"organization.update",
			"organization.members.view",
			"organization.members.add",
		}},
		{role: "member", want: []string{"organization.view"}},
		{role: "other", want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			access := service.OrganizationAccess{
				Organization: db.Organization{PartyID: 100, Name: "Example"},
				MembershipID: 12,
				Role:         tt.role,
				JoinedAt:     time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			}
			result := newOrganizationAccessResponse(access)
			if !reflect.DeepEqual(result.Permissions, tt.want) {
				t.Fatalf("permissions = %v, want %v", result.Permissions, tt.want)
			}
			if result.Membership.Role != tt.role || result.Membership.ID != 12 || result.PartyID != 100 {
				t.Fatalf("membership information changed: %+v", result)
			}
		})
	}
}
