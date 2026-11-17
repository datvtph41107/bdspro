package structs

import "github.com/datvtph41107/bdspro/modules/optional"

// CreateTeamOption is the stable HTTP data contract, separate from
// Organization domain models and route implementations.
type CreateTeamOption struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=2000"`
}

type RenameTeamOption struct {
	Name optional.Option[string] `json:"name"`
}
