package structs

import (
	"time"

	"github.com/google/uuid"
)

// Team API shapes are independent of generated SQLC models.
type CreateTeamOption struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description" binding:"max=1000"`
}

type RenameTeamOption struct {
	Name string `json:"name" binding:"required,max=100"`
}

type AddTeamUserOption struct {
	PersonID int64 `json:"person_id" binding:"required,min=1"`
}

type Team struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TeamMember struct {
	TeamID   int64     `json:"team_id"`
	PersonID int64     `json:"person_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type CreateTeamInviteOption struct {
	Email string `json:"email" binding:"required,email"`
}

type AcceptTeamInviteOption struct {
	Token string `json:"token" binding:"required,len=64"`
}

// IssuedTeamInvite never contains the raw proof. Only the asynchronous
// worker sends that proof to the verified organization member.
type IssuedTeamInvite struct {
	ID uuid.UUID `json:"id"`
	TeamID int64 `json:"team_id"`
	InvitedEmail string `json:"invited_email"`
	ExpiresAt time.Time `json:"expires_at"`
}
