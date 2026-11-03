// Package structs defines public HTTP API shapes, not persistence entities.
package structs

import "time"

type CreateOrgOption struct {
	Name string `json:"name" binding:"required"`
}

type EditOrgOption struct {
	Name *string `json:"name" binding:"required"`
}

type AddOrgUserOption struct {
	PersonID int64 `json:"person_id" binding:"required,min=1"`
}

type Organization struct {
	PartyID   int64     `json:"party_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type OrganizationMembership struct {
	ID       int64     `json:"id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type OrganizationAccess struct {
	PartyID     int64                  `json:"party_id"`
	Name        string                 `json:"name"`
	CreatedAt   time.Time              `json:"created_at"`
	Membership  OrganizationMembership `json:"membership"`
	Permissions []string               `json:"permissions"`
}

type OrganizationMember struct {
	ID             int64      `json:"id"`
	OrganizationID int64      `json:"organization_id"`
	PersonID       int64      `json:"person_id"`
	Role           string     `json:"role"`
	JoinedAt       time.Time  `json:"joined_at"`
	EndedAt        *time.Time `json:"ended_at"`
}

type ActiveOrganizationMember struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	PersonID       int64     `json:"person_id"`
	Role           string    `json:"role"`
	JoinedAt       time.Time `json:"joined_at"`
}

