package structs

import "time"

// Team API shapes are independent of generated SQLC models.
type CreateTeamOption struct {
	Name string `json:"name" binding:"required,max=100"`
	Description string `json:"description" binding:"max=1000"`
}

type RenameTeamOption struct {
	Name string `json:"name" binding:"required,max=100"`
}

type AddTeamUserOption struct {
	PersonID int64 `json:"person_id" binding:"required,min=1"`
}

type Team struct {
	ID int64 `json:"id"`
	OrganizationID int64 `json:"organization_id"`
	Name string `json:"name"`
	Description string `json:"description"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeamMember struct {
	TeamID int64 `json:"team_id"`
	PersonID int64 `json:"person_id"`
	JoinedAt time.Time `json:"joined_at"`
}
