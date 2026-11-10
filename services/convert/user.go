package convert

import (
	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/structs"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
)

func ToUser(a *account_model.Account) structs.UserResponse {
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

func ToConnections(rows []account_model.Connection) []structs.ConnectionResponse {
	out := make([]structs.ConnectionResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, structs.ConnectionResponse{
			ID:        row.ID,
			SessionID: row.SessionID.String(),
			Action:    row.Action,
			IPAddress: row.IPAddress,
			UserAgent: row.UserAgent,
			CreatedAt: row.CreatedAt,
		})
	}
	return out
}

// ToSessions maps model-owned session-list values to the existing public
// `id,user_agent,client_ip,created_at,expires_at` wire representation.
func ToSessions(rows []auth_model.SessionInfo) []structs.SessionResponse {
	out := make([]structs.SessionResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, structs.SessionResponse{
			ID:        row.ID,
			UserAgent: row.UserAgent,
			ClientIP:  row.ClientIP,
			CreatedAt: row.CreatedAt,
			ExpiresAt: row.ExpiresAt,
		})
	}
	return out
}
