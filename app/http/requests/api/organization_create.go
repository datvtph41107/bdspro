package apirequest

type CreateOrganizationBody struct {
	Name string `json:"name" binding:"required"`
}
