package context

import "github.com/datvtph41107/bdspro/models/organization"

// APIOrganization carries verified resource state for this one request.
// Router actions populate the fields before typed handlers may read them.
type APIOrganization struct {
	Organization *organization.Organization
	Team         *organization.Team
}
