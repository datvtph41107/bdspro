package organization

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/datvtph41107/bdspro/modules/util"
)

type Detail struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	MembershipID string    `json:"membership_id"`
	Role         string    `json:"role"`
	JoinedAt     time.Time `json:"joined_at"`
}

type Membership struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	PersonID       string    `json:"person_id"`
	Role           string    `json:"role"`
	JoinedAt       time.Time `json:"joined_at"`
}

var (
	ErrInvalidOrganizationName = util.WithPublic(util.ErrInvalidArgument, "invalid organization name", "invalid_organization_name", "organization name must contain 1–100 characters without control characters")
	ErrOrganizationForbidden   = util.WithPublic(util.ErrPermissionDenied, "organization owner required", "organization_owner_required", "organization owner permission required")
	ErrVerifiedContactRequired = util.WithPublic(util.ErrPermissionDenied, "verified contact required", "verified_contact_required", "verified email or phone required")
	ErrMemberNotFound          = util.WithPublic(util.ErrNotExist, "organization member not found or ineligible", "member_not_found", "member not found or ineligible")
	ErrMemberExists            = util.WithPublic(util.ErrAlreadyExist, "organization member exists", "member_exists", "person is already a member")
	ErrLastOwner               = util.WithPublic(util.ErrPermissionDenied, "last eligible owner", "last_owner", "at least one eligible owner must remain")
	ErrInvalidOwnerTarget      = util.WithPublic(util.ErrInvalidArgument, "invalid owner transfer target", "invalid_owner_target", "target must be a different eligible ordinary member")
)

func NormalizeOrganizationName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 ||
		strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", ErrInvalidOrganizationName
	}
	return name, nil
}
