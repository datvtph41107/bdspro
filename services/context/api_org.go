package context

import "github.com/datvtph41107/bdspro/models/organization"

type APIOrganization struct {
	Organization *organization.Organization
	Team         *organization.Team
}
