package apirequest

type AddOrganizationMemberBody struct {
	PersonID int64 `json:"person_id" binding:"required,min=1"`
}
