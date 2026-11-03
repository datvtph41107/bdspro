package org

type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateOrganizationRequest struct {
	Name *string `json:"name" binding:"required"`
}

type AddOrganizationMemberRequest struct {
	PersonID int64 `json:"person_id" binding:"required,min=1"`
}

