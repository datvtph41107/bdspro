package apirequest

type CreateOrganizationBody struct {
    Name string `json:"name" binding:"required"`
}
type AddOrganizationMemberBody struct {
    PersonID int64 `json:"person_id" binding:"required,min=1"`
}
type UpdateOrganizationBody struct {
    Name *string `json:"name"`
}
type OrganizationPath struct {
    ID int64 `uri:"id" binding:"required,min=1"`
}
