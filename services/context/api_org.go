package context

import "github.com/datvtph41107/bdspro/models/organization"

// APIOrganization owns the typed Organization/Team assigned for one request.
type APIOrganization struct {
	Organization *organization.Organization
	Team *organization.Team
	Access *organization.Access
}
