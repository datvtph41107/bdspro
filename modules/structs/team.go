package structs

import "github.com/datvtph41107/bdspro/modules/optional"

type CreateTeamOption struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=2000"`
}

type RenameTeamOption struct {
	Name optional.Option[string] `json:"name"`
}

type CreateTeamInviteOption struct {
	Recipient string `json:"recipient" binding:"required,max=254"`
	ExpiresIn int    `json:"expires_in"`
}

type AcceptTeamInviteOption struct {
	InviteToken string `json:"invite_token" binding:"required,max=128"`
}
