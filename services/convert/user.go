package convert

import (
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/structs"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
)

func ToUser(a db.Account) structs.UserResponse {
	return structs.UserResponse{
  AccountID: a.ID,
  Email: a.Email,
  CreatedAt: a.CreatedAt,
  IsEmailVerified: a.IsEmailVerified,
 }
}

func ToLogin(v auth_service.LoginResult) structs.LoginResponse {
	return structs.LoginResponse{
  SessionID: v.Session.ID.String(),
  AccessToken: v.AccessToken,
  AccessTokenExpiresAt: v.AccessExpiresAt,
  RefreshToken: v.RefreshToken,
  RefreshTokenExpiresAt: v.RefreshExpiresAt,
  User: ToUser(v.Account),
 }
}
