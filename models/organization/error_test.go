package organization_test

import (
	"errors"
	"testing"

	organization "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/util"
)

func TestModelErrorClassification(t *testing.T) {
	tests := []struct {
		name string
		domain error
		category error
	}{
		{"team hidden", organization.ErrTeamNotFound, util.ErrNotExist},
		{"team owner", organization.ErrTeamOwnerRequired, util.ErrPermissionDenied},
		{"team duplicate", organization.ErrTeamAlreadyExists, util.ErrAlreadyExist},
		{"invite validation", organization.ErrTeamInviteInvalid, util.ErrInvalidArgument},
		{"organization missing", organization.ErrOrganizationNotFound, util.ErrNotExist},
		{"membership duplicate", organization.ErrMembershipExists, util.ErrAlreadyExist},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.domain, tt.category) {
				t.Fatalf("%v does not classify as %v", tt.domain, tt.category)
			}
			if !errors.Is(errors.Join(tt.domain, errors.New("internal database detail")), tt.category) {
				t.Fatalf("wrapped error did not preserve classification: %v", tt.domain)
			}
		})
	}
}
