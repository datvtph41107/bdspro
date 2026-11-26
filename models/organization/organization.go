package organization

import "github.com/datvtph41107/bdspro/modules/util"

type Organization struct {
	ID   int64
	Name string
	Role string
}

func (o Organization) IsOwner() bool { return o.Role == "owner" }

var ErrOrganizationNotFound = util.WithPublic(
	util.ErrNotExist,
	"organization not found or not accessible",
	"organization_not_found",
	"organization not found",
)
