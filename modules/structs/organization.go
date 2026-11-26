package structs

import "github.com/datvtph41107/bdspro/modules/optional"

type CreateOrganizationOption struct {
	Name string `json:"name" binding:"required,max=100"`
}

type UpdateOrganizationOption struct {
	Name optional.Option[string] `json:"name"`
}

type AddOrganizationMemberOption struct {
	PersonID string `json:"person_id" binding:"required,max=19"`
}

type TransferOrganizationOwnerOption struct {
	PersonID              string `json:"person_id" binding:"required,max=19"`
	ReauthenticationProof string `json:"reauthentication_proof" binding:"required,max=128"`
}
