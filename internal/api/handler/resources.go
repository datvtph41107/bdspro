package handler

import (
	"time"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserResponse struct {
	AccountID       int64     `json:"account_id"`
	PersonID        int64     `json:"person_id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
	IsEmailVerified bool      `json:"is_email_verified"`
}

func newUserResponse(a db.Account) UserResponse {
	return UserResponse{
		AccountID:       a.ID,
		PersonID:        a.PersonID,
		Email:           a.Email,
		CreatedAt:       a.CreatedAt,
		IsEmailVerified: a.IsEmailVerified,
	}
}

// LoginActivityResponse never contains attempted email, credential material,
// refresh tokens, OAuth subjects or data belonging to another account.
type LoginActivityResponse struct {
	ID         int64     `json:"id"`
	Outcome    string    `json:"outcome"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	OccurredAt time.Time `json:"occurred_at"`
}

type LoginResponse struct {
	SessionID             string       `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  UserResponse `json:"user"`
}

func newLoginResponse(v service.LoginResult) LoginResponse {
	return LoginResponse{
		SessionID:             v.Session.ID.String(),
		AccessToken:           v.AccessToken,
		AccessTokenExpiresAt:  v.AccessExpiresAt,
		RefreshToken:          v.RefreshToken,
		RefreshTokenExpiresAt: v.RefreshExpiresAt,
		User:                  newUserResponse(v.Account),
	}
}

type RenewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

type OrganizationResponse struct {
	PartyID   int64     `json:"party_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func newOrganizationResponse(a db.Organization) OrganizationResponse {
	return OrganizationResponse{
		PartyID:   a.PartyID,
		Name:      a.Name,
		CreatedAt: a.CreatedAt,
	}
}

func newOrganizationListResponse(list []db.Organization) []OrganizationResponse {
	if list == nil {
		return nil
	}
	out := make([]OrganizationResponse, 0, len(list))
	for _, v := range list {
		out = append(out, newOrganizationResponse(v))
	}
	return out
}

type OrganizationMemberResponse struct {
	ID             int64              `json:"id"`
	OrganizationID int64              `json:"organization_id"`
	PersonID       int64              `json:"person_id"`
	Role           string             `json:"role"`
	JoinedAt       time.Time          `json:"joined_at"`
	EndedAt        pgtype.Timestamptz `json:"ended_at"`
}

func newOrganizationMemberResponse(a db.OrganizationMember) OrganizationMemberResponse {
	return OrganizationMemberResponse{
		ID:             a.ID,
		OrganizationID: a.OrganizationID,
		PersonID:       a.PersonID,
		Role:           a.Role,
		JoinedAt:       a.JoinedAt,
		EndedAt:        a.EndedAt,
	}
}

type OrganizationAccessMembershipResponse struct {
	ID       int64     `json:"id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type OrganizationAccessResponse struct {
	PartyID     int64                                `json:"party_id"`
	Name        string                               `json:"name"`
	CreatedAt   time.Time                            `json:"created_at"`
	Membership  OrganizationAccessMembershipResponse `json:"membership"`
	Permissions []string                             `json:"permissions"`
}

func newOrganizationAccessResponse(access service.OrganizationAccess) OrganizationAccessResponse {
	return OrganizationAccessResponse{
		PartyID:   access.Organization.PartyID,
		Name:      access.Organization.Name,
		CreatedAt: access.Organization.CreatedAt,
		Membership: OrganizationAccessMembershipResponse{
			ID:       access.MembershipID,
			Role:     access.Role,
			JoinedAt: access.JoinedAt,
		},
		Permissions: permissionsForRole(access.Role),
	}
}

// permissionsForRole preserves the current public JSON contract. This is
// display metadata only; controllers never authorize from this list.
func permissionsForRole(role string) []string {
	switch role {
	case "owner":
		return []string{"organization.view", "organization.update", "organization.members.view", "organization.members.add"}
	case "member":
		return []string{"organization.view"}
	default:
		return []string{}
	}
}

func newOrganizationAccessListResponse(accesses []service.OrganizationAccess) []OrganizationAccessResponse {
	out := make([]OrganizationAccessResponse, 0, len(accesses))
	for _, access := range accesses {
		out = append(out, newOrganizationAccessResponse(access))
	}
	return out
}

type ActiveOrganizationMemberResponse struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	PersonID       int64     `json:"person_id"`
	Role           string    `json:"role"`
	JoinedAt       time.Time `json:"joined_at"`
}

func newActiveOrganizationMembersResponse(members []db.ListOrganizationMembersForOwnerRow) []ActiveOrganizationMemberResponse {
	out := make([]ActiveOrganizationMemberResponse, 0, len(members))
	for _, member := range members {
		out = append(out, ActiveOrganizationMemberResponse{
			ID:             member.ID,
			OrganizationID: member.OrganizationID,
			PersonID:       member.PersonID,
			Role:           member.Role,
			JoinedAt:       member.JoinedAt,
		})
	}
	return out
}
