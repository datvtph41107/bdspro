package convert

import (
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/structs"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
)

func ToUser(a db.Account) structs.UserResponse {
	return structs.UserResponse{
		AccountID:       a.ID,
		PersonID:        a.PersonID,
		Email:           a.Email,
		CreatedAt:       a.CreatedAt,
		IsEmailVerified: a.IsEmailVerified,
	}
}

func ToLogin(v auth_service.LoginResult) structs.LoginResponse {
	return structs.LoginResponse{
		SessionID:             v.Session.ID.String(),
		AccessToken:           v.AccessToken,
		AccessTokenExpiresAt:  v.AccessExpiresAt,
		RefreshToken:          v.RefreshToken,
		RefreshTokenExpiresAt: v.RefreshExpiresAt,
		User:                  ToUser(v.Account),
	}
}

func ToConnections(rows []db.ListAccountConnectionHistoriesRow) []structs.ConnectionResponse {
	out := make([]structs.ConnectionResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, structs.ConnectionResponse{
			ID:        row.ID,
			SessionID: row.SessionID.String(),
			Action:    row.Action,
			IPAddress: row.IpAddress,
			UserAgent: row.UserAgent,
			CreatedAt: row.CreatedAt,
		})
	}
	return out
}
